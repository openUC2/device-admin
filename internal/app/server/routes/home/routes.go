// Package home contains the route handlers related to the app's home screen.
package home

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/sargassum-world/godest"

	ipc "github.com/openUC2/device-admin/internal/app/ipc/boot"
	sc "github.com/openUC2/device-admin/internal/clients/sidecar"
	sd "github.com/openUC2/device-admin/internal/clients/systemd"
)

type Handlers struct {
	r godest.TemplateRenderer

	sdc *sd.Client
	scc *sc.Client

	l godest.Logger
}

func New(r godest.TemplateRenderer, sdc *sd.Client, scc *sc.Client, l godest.Logger) *Handlers {
	return &Handlers{
		r:   r,
		sdc: sdc,
		scc: scc,
		l:   l,
	}
}

func (h *Handlers) Register(er godest.EchoRouter) {
	er.GET(h.r.BasePath, h.HandleHomeGet())
	if h.r.BasePath != "/" {
		er.GET(strings.TrimSuffix(h.r.BasePath, "/"), h.HandleHomeGet())
	}
	// boot
	er.POST(h.r.BasePath+"boot", h.HandleBootPost())
}

type HomeViewData struct {
	Hostname string
	Port     string
}

func getHomeViewData(host string) (vd HomeViewData, err error) {
	split := strings.Split(host, ":")
	const expectedComponents = 2
	if len(split) > expectedComponents {
		return HomeViewData{}, errors.Errorf(
			"unable to split host '%s' into a hostname and a port", host,
		)
	}
	vd.Hostname = split[0]
	if len(split) == expectedComponents {
		vd.Port = split[expectedComponents-1]
	}
	return vd, nil
}

func (h *Handlers) HandleHomeGet() echo.HandlerFunc {
	t := "home/index.page.tmpl"
	h.r.MustHave(t)
	return func(c echo.Context) error {
		// Run queries
		homeViewData, err := getHomeViewData(c.Request().Host)
		if err != nil {
			return err
		}
		// Produce output
		return h.r.CacheablePage(c.Response(), c.Request(), t, homeViewData, struct{}{})
	}
}

func (h *Handlers) HandleBootPost() echo.HandlerFunc {
	return func(c echo.Context) error {
		// Parse params
		state := c.FormValue("state")
		redirectTarget := c.FormValue("redirect-target")

		// Run queries
		ctx := c.Request().Context()
		switch state {
		default:
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf(
				"invalid boot state %s", state,
			))
		case "soft-rebooted":
			if err := shutdownViaSidecar(ctx, "SoftReboot", h.scc, h.l); err != nil {
				if sdErr := h.sdc.SoftReboot(ctx); err != nil {
					return errors.Wrapf(
						sdErr, "couldn't soft-reboot through sidecar (%s) or directly", err.Error(),
					)
				}
				h.l.Warnf("soft-rebooted directly after failure to soft-reboot through sidecar", err)
			}
			// Redirect user
			return c.Redirect(http.StatusSeeOther, redirectTarget)
		case "rebooted":
			if err := shutdownViaSidecar(ctx, "Reboot", h.scc, h.l); err != nil {
				if sdErr := h.sdc.Reboot(ctx); err != nil {
					return errors.Wrapf(
						sdErr, "couldn't reboot through sidecar (%s) or directly", err.Error(),
					)
				}
				h.l.Warnf("rebooted directly after failure to reboot through sidecar", err)
			}
			// Redirect user
			return c.Redirect(http.StatusSeeOther, redirectTarget)
		case "powered-off":
			if err := shutdownViaSidecar(ctx, "Poweroff", h.scc, h.l); err != nil {
				if sdErr := h.sdc.Poweroff(ctx); err != nil {
					return errors.Wrapf(
						sdErr, "couldn't power-off through sidecar (%s) or directly", err.Error(),
					)
				}
				h.l.Warnf("powered-off directly after failure to power-off through sidecar", err)
			}
			// Redirect user
			return c.Redirect(http.StatusSeeOther, redirectTarget)
		}
	}
}

func shutdownViaSidecar(ctx context.Context, method string, scc *sc.Client, l godest.Logger) error {
	conn, err := scc.Open(ctx)
	if err != nil {
		return errors.Wrap(err, "couldn't open connection to sidecar")
	}
	defer func() {
		if conn == nil {
			return
		}
		if err := conn.Close(); err != nil {
			l.Error(errors.New("couldn't close connection to sidecar"))
		}
	}()
	switch method {
	default:
		return errors.Errorf("unknown sidecar method %s", method)
	case "SoftReboot":
		if err := ipc.SoftReboot().Call(ctx, conn); err != nil {
			return errors.Wrapf(err, "couldn't call sidecar's %s method", method)
		}
	case "Reboot":
		if err := ipc.Reboot().Call(ctx, conn); err != nil {
			return errors.Wrapf(err, "couldn't call sidecar's %s method", method)
		}
	case "Poweroff":
		if err := ipc.Poweroff().Call(ctx, conn); err != nil {
			return errors.Wrapf(err, "couldn't call sidecar's %s method", method)
		}
	}
	return nil
}
