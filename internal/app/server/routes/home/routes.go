// Package home contains the route handlers related to the app's home screen.
package home

import (
	"context"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/sargassum-world/godest"
	"github.com/sargassum-world/godest/handling"
	"github.com/sargassum-world/godest/turbostreams"

	sh "github.com/openUC2/machine-admin/internal/app/server/handling"
	"github.com/openUC2/machine-admin/internal/clients/identity"
	"github.com/openUC2/machine-admin/internal/clients/tailscale"
	"github.com/openUC2/machine-admin/internal/clients/versioning"
)

type Handlers struct {
	r godest.TemplateRenderer

	ic  *identity.Client
	vc  *versioning.Client
	tsc *tailscale.Client

	l godest.Logger
}

func New(
	r godest.TemplateRenderer, ic *identity.Client, vc *versioning.Client, tsc *tailscale.Client,
	l godest.Logger,
) *Handlers {
	return &Handlers{
		r:   r,
		ic:  ic,
		vc:  vc,
		tsc: tsc,
		l:   l,
	}
}

func (h *Handlers) Register(er godest.EchoRouter, tr turbostreams.Router) {
	er.GET(h.r.BasePath, h.HandleHomeGet())
	if h.r.BasePath != "/" {
		er.GET(strings.TrimSuffix(h.r.BasePath, "/"), h.HandleHomeGet())
	}
	tr.SUB(h.r.BasePath, sh.AllowTSSub())
	if h.r.BasePath != "/" {
		tr.SUB(strings.TrimSuffix(h.r.BasePath, "/"), sh.AllowTSSub())
	}
	tr.PUB(h.r.BasePath, h.HandleHomePub())
	if h.r.BasePath != "/" {
		tr.PUB(strings.TrimSuffix(h.r.BasePath, "/"), h.HandleHomePub())
	}
}

func (h *Handlers) HandleHomeGet() echo.HandlerFunc {
	t := "home/index.page.tmpl"
	h.r.MustHave(t)
	return func(c echo.Context) error {
		// Run queries
		homeViewData, err := getHomeViewData(c.Request().Context(), h.vc, h.ic, h.tsc)
		if err != nil {
			return err
		}
		// Produce output
		return h.r.CacheablePage(c.Response(), c.Request(), t, homeViewData, struct{}{})
	}
}

type HomeViewData struct {
	ForkliftVersioning versioning.Forklift
	MachineName        string
	Hostname           string
	TailscaleDNS       string

	IsStreamPage bool
}

func getHomeViewData(
	ctx context.Context, vc *versioning.Client, ic *identity.Client, tsc *tailscale.Client,
) (vd HomeViewData, err error) {
	vd.ForkliftVersioning, err = vc.GetForklift()
	if err != nil {
		return vd, err
	}

	vd.MachineName, _ = ic.GetMachineName()
	vd.Hostname, _ = ic.GetHostname()
	vd.TailscaleDNS, _ = getTailscaleDNSName(ctx, tsc)

	return vd, nil
}

func getTailscaleDNSName(ctx context.Context, tsc *tailscale.Client) (name string, err error) {
	status, err := tsc.GetStatus(ctx)
	if err != nil {
		return "", errors.Wrap(err, "couldn't get tailscale daemon status")
	}
	selfStatus := status.Self
	if selfStatus == nil {
		return "", nil
	}
	return selfStatus.DNSName, nil
}

func (h *Handlers) HandleHomePub() turbostreams.HandlerFunc {
	t := "home/index.page.tmpl"
	h.r.MustHave(t)
	return func(c *turbostreams.Context) error {
		// Publish periodically
		const pubInterval = 10 * time.Second
		return handling.RepeatImmediate(c.Context(), pubInterval, func() (done bool, err error) {
			// Run queries
			vd, err := getHomeViewData(c.Context(), h.vc, h.ic, h.tsc)
			if err != nil {
				return false, err
			}
			if vd.ForkliftVersioning == (versioning.Forklift{}) {
				return false, nil // don't output empty info if there was a read-write race condition!
			}
			// Produce output
			vd.IsStreamPage = true
			return false, sh.PublishPageReload(c, h.r, t, vd)
		})
	}
}
