// Package routes contains the route handlers for the web server.
package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/sargassum-world/godest"

	"github.com/openUC2/device-admin/internal/app/deviceadmin/client"
	"github.com/openUC2/device-admin/internal/app/deviceadmin/routes/assets"
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

func (h *Handlers) Register(er godest.EchoRouter, em godest.Embeds) error {
	assets.RegisterStatic(h.r.BasePath, er, em)
	assets.NewTemplated(h.r).Register(er)
	home.New(h.r).Register(er)
	identity.New(h.r).Register(er)
	internet.New(h.r, h.globals.NetworkManager).Register(er)
	h.remote = remote.New(h.r, h.globals.Tailscale)
	if err := h.remote.Register(er); err != nil {
		return errors.Wrap(err, "couldn't register handlers for remote routes")
	}
	storage.New(h.r, h.globals.UDisks2, h.globals.Base.Logger).Register(er)
	osconfig.New(h.r).Register(er)
	return nil
}

func (h *Handlers) TrailingSlashSkipper(c echo.Context) bool {
	return h.remote.TrailingSlashSkipper(c)
}

func (h *Handlers) GzipSkipper(c echo.Context) bool {
	return h.remote.GzipSkipper(c)
}
