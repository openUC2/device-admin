// Package identity contains the route handlers related to identity access.
package identity

import (
	"github.com/labstack/echo/v4"
	"github.com/sargassum-world/godest"
)

type Handlers struct {
	r godest.TemplateRenderer
}

func New(r godest.TemplateRenderer) *Handlers {
	return &Handlers{
		r: r,
	}
}

func (h *Handlers) Register(er godest.EchoRouter) {
	er.GET(h.r.BasePath+"identity", h.HandleIdentityGet())
}

type IdentityViewData struct{}

func getIdentityViewData() (vd IdentityViewData, err error) {
	return vd, nil
}

func (h *Handlers) HandleIdentityGet() echo.HandlerFunc {
	t := "identity/main.page.tmpl"
	h.r.MustHave(t)
	return func(c echo.Context) error {
		// Run queries
		identityViewData, err := getIdentityViewData()
		if err != nil {
			return err
		}
		// Produce output
		return h.r.CacheablePage(c.Response(), c.Request(), t, identityViewData, struct{}{})
	}
}
