// Package internet contains the route handlers related to internet access.
package internet

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
	er.GET("/internet", h.HandleInternetGet())
}

type InternetViewData struct{}

func getInternetViewData() (vd InternetViewData, err error) {
	return vd, nil
}

func (h *Handlers) HandleInternetGet() echo.HandlerFunc {
	t := "internet/main.page.tmpl"
	h.r.MustHave(t)
	return func(c echo.Context) error {
		// Run queries
		internetViewData, err := getInternetViewData()
		if err != nil {
			return err
		}
		// Produce output
		return h.r.CacheablePage(c.Response(), c.Request(), t, internetViewData, struct{}{})
	}
}
