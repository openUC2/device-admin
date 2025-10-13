// Package home contains the route handlers related to the app's home screen.
package home

import (
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
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
	er.GET(h.r.BasePath, h.HandleHomeGet())
	if h.r.BasePath != "/" {
		er.GET(strings.TrimSuffix(h.r.BasePath, "/"), h.HandleHomeGet())
	}
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
	t := "home/main.page.tmpl"
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
