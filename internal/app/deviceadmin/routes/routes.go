// Package routes contains the route handlers for the web server.
package routes

import (
	"github.com/sargassum-world/godest"

	"github.com/openUC2/device-admin/internal/app/deviceadmin/client"
	"github.com/openUC2/device-admin/internal/app/deviceadmin/routes/assets"
	"github.com/openUC2/device-admin/internal/app/deviceadmin/routes/home"
	"github.com/openUC2/device-admin/internal/app/deviceadmin/routes/identity"
	"github.com/openUC2/device-admin/internal/app/deviceadmin/routes/internet"
	"github.com/openUC2/device-admin/internal/app/deviceadmin/routes/osconfig"
	"github.com/openUC2/device-admin/internal/app/deviceadmin/routes/remote"
)

type Handlers struct {
	r       godest.TemplateRenderer
	globals *client.Globals
}

func New(r godest.TemplateRenderer, globals *client.Globals) *Handlers {
	return &Handlers{
		r:       r,
		globals: globals,
	}
}

func (h *Handlers) Register(er godest.EchoRouter, em godest.Embeds) {
	assets.RegisterStatic(er, em)
	assets.NewTemplated(h.r).Register(er)
	home.New(h.r).Register(er)
	identity.New(h.r).Register(er)
	internet.New(h.r, h.globals.NetworkManager).Register(er)
	remote.New(h.r).Register(er)
	osconfig.New(h.r).Register(er)
}
