// Package osconfig contains the route handlers related to OS configuration management.
package osconfig

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
	er.GET("/os-config", h.HandleOSConfigGet())
}

type OSConfigViewData struct{}

func getOSConfigViewData() (vd OSConfigViewData, err error) {
	return vd, nil
}

func (h *Handlers) HandleOSConfigGet() echo.HandlerFunc {
	t := "os-config/main.page.tmpl"
	h.r.MustHave(t)
	return func(c echo.Context) error {
		// Run queries
		osConfigViewData, err := getOSConfigViewData()
		if err != nil {
			return err
		}
		// Produce output
		return h.r.CacheablePage(c.Response(), c.Request(), t, osConfigViewData, struct{}{})
	}
}
