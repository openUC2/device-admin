// Package remote contains the route handlers related to remote access.
package remote

import (
	"context"
	"fmt"
	"net/http"
	"net/netip"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/sargassum-world/godest"

	ts "github.com/openUC2/device-admin/internal/clients/tailscale"
)

type Handlers struct {
	r   godest.TemplateRenderer
	tsc *ts.Client
}

func New(r godest.TemplateRenderer, tsc *ts.Client) *Handlers {
	return &Handlers{
		r:   r,
		tsc: tsc,
	}
}

func (h *Handlers) Register(er godest.EchoRouter) error {
	er.GET(h.r.BasePath+"remote", h.HandleRemoteGet())
	// assistance
	er.POST(h.r.BasePath+"remote/assistance", h.HandleAssistancePost())
	// tailscale
	tsws, err := h.tsc.InitWebServer(h.r.BasePath + "remote/tailscale")
	if err != nil {
		return err
	}
	er.GET(h.r.BasePath+"remote/tailscale/", echo.WrapHandler(tsws))
	er.GET(h.r.BasePath+"remote/tailscale/*", echo.WrapHandler(tsws))
	er.POST(h.r.BasePath+"remote/tailscale/*", echo.WrapHandler(tsws))
	er.PATCH(h.r.BasePath+"remote/tailscale/*", echo.WrapHandler(tsws))
	return nil
}

func (h *Handlers) TrailingSlashSkipper(c echo.Context) bool {
	// Tailscale's web UI page assumes that the web page has a trailing slash for loading its JS and
	// CSS assets from relative paths:
	return c.Request().URL.Path == h.r.BasePath+"remote/tailscale/"
}

func (h *Handlers) GzipSkipper(c echo.Context) bool {
	// We skip gzip for the Tailscale web GUI because its HTTP handler already does its own gzip
	// compression, and HTTP clients assume that responses don't have two or more layers of gzipping:
	return strings.HasPrefix(c.Request().URL.Path, h.r.BasePath+"remote/tailscale/")
}

type RemoteViewData struct {
	State ts.State
	// HealthProblems []string
	IPs     []netip.Addr
	DNSName string
	// Tags    []string
	Online bool
	// KeyExpiration time.Time
	NetworkName string
}

func getRemoteViewData(ctx context.Context, tc *ts.Client) (vd RemoteViewData, err error) {
	status, err := tc.GetStatus(ctx)
	if err != nil {
		return vd, errors.Wrap(err, "couldn't get tailscale daemon status")
	}
	vd.State = ts.State(status.BackendState)
	// vd.HealthProblems = status.Health
	vd.IPs = status.TailscaleIPs
	selfStatus := status.Self

	if selfStatus != nil {
		vd.DNSName = selfStatus.DNSName
		// if tags := selfStatus.Tags; tags != nil {
		// 	vd.Tags = selfStatus.Tags.AsSlice()
		// }
		vd.Online = selfStatus.Online
		// if selfStatus.KeyExpiry != nil {
		// 	vd.KeyExpiration = *selfStatus.KeyExpiry
		// }
	}

	tailnet := status.CurrentTailnet
	if tailnet != nil {
		vd.NetworkName = tailnet.Name
	}

	return vd, nil
}

func (h *Handlers) HandleRemoteGet() echo.HandlerFunc {
	t := "remote/index.page.tmpl"
	h.r.MustHave(t)
	return func(c echo.Context) error {
		// Run queries
		remoteViewData, err := getRemoteViewData(c.Request().Context(), h.tsc)
		if err != nil {
			return err
		}
		// Produce output
		return h.r.CacheablePage(c.Response(), c.Request(), t, remoteViewData, struct{}{})
	}
}

func (h *Handlers) HandleAssistancePost() echo.HandlerFunc {
	return func(c echo.Context) error {
		// Parse params
		state := c.FormValue("state")
		redirectTarget := c.FormValue("redirect-target")

		// Run queries
		switch state {
		default:
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf(
				"invalid connection profiles state %s", state,
			))
		case "enabled":
			deviceAuthKey := c.FormValue("device-authentication-key")
			if err := h.tsc.Provision(c.Request().Context(), deviceAuthKey); err != nil {
				return err
			}
			// Redirect user
			return c.Redirect(http.StatusSeeOther, redirectTarget)
		case "disabled":
			if err := h.tsc.Deprovision(c.Request().Context()); err != nil {
				return err
			}
			// Redirect user
			return c.Redirect(http.StatusSeeOther, redirectTarget)
		}
	}
}
