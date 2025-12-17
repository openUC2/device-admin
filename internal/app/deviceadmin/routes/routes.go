// Package routes contains the route handlers for the web server.
package routes

import (
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/sargassum-world/godest"
	"github.com/sargassum-world/godest/handling"
	"github.com/sargassum-world/godest/turbostreams"

	"github.com/openUC2/device-admin/internal/app/deviceadmin/client"
	dah "github.com/openUC2/device-admin/internal/app/deviceadmin/handling"
	"github.com/openUC2/device-admin/internal/app/deviceadmin/routes/assets"
	"github.com/openUC2/device-admin/internal/app/deviceadmin/routes/cable"
	"github.com/openUC2/device-admin/internal/app/deviceadmin/routes/home"
	"github.com/openUC2/device-admin/internal/app/deviceadmin/routes/identity"
	"github.com/openUC2/device-admin/internal/app/deviceadmin/routes/internet"
	"github.com/openUC2/device-admin/internal/app/deviceadmin/routes/osconfig"
	"github.com/openUC2/device-admin/internal/app/deviceadmin/routes/remote"
	"github.com/openUC2/device-admin/internal/app/deviceadmin/routes/storage"
)

type Handlers struct {
	r       godest.TemplateRenderer
	globals *client.Globals

	remote *remote.Handlers
}

func New(r godest.TemplateRenderer, globals *client.Globals) *Handlers {
	return &Handlers{
		r:       r,
		globals: globals,
	}
}

func (h *Handlers) Register(er godest.EchoRouter, tsr turbostreams.Router, em godest.Embeds) error {
	tsh := h.globals.Base.TSBroker.Hub()
	l := h.globals.Base.Logger

	assets.RegisterStatic(h.r.BasePath, er, em)
	assets.NewTemplated(h.r).Register(er)
	cable.New(
		h.r, h.globals.Base.ACSigner, h.globals.Base.TSBroker, l,
	).Register(er)
	home.New(h.r).Register(er)
	identity.New(h.r).Register(er)
	internet.New(h.r, tsh, h.globals.NetworkManager, l).Register(er, tsr)
	h.remote = remote.New(h.r, h.globals.Tailscale)
	if err := h.remote.Register(er, tsr); err != nil {
		return errors.Wrap(err, "couldn't register handlers for remote routes")
	}
	storage.New(h.r, h.globals.UDisks2, l).Register(er)
	osconfig.New(h.r).Register(er)

	tsr.SUB(h.r.BasePath+"refresh", dah.AllowTSSub())
	tsr.PUB(h.r.BasePath+"refresh", h.HandleRefreshPub())

	tsr.MSG(h.r.BasePath+"*", dah.HandleTSMsg(h.r))
	tsr.UNSUB(h.r.BasePath+"*", turbostreams.EmptyHandler)
	return nil
}

func (h *Handlers) HandleRefreshPub() turbostreams.HandlerFunc {
	return func(c *turbostreams.Context) error {
		// Make change trackers
		initialized := false

		// Parse params
		ctx := c.Context()
		// iface := c.Param("iface")

		// Run queries
		// Publish periodically
		const pubInterval = 2 * time.Second
		return handling.RepeatImmediate(ctx, pubInterval, func() (done bool, err error) {
			if !initialized {
				// We just started publishing because a page added a subscription, so there's no need to
				// send the devices list again - that page already has the latest version
				initialized = true
				return false, nil
			}

			// Publish changes
			c.Publish([]turbostreams.Message{{Action: turbostreams.ActionRefresh}}...)
			return false, nil
		})
	}
}

func (h *Handlers) TrailingSlashSkipper(c echo.Context) bool {
	return h.remote.TrailingSlashSkipper(c)
}

func (h *Handlers) GzipSkipper(c echo.Context) bool {
	return h.remote.GzipSkipper(c)
}
