// Package remote contains the route handlers related to remote access.
package remote

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
	er.GET(h.r.BasePath+"remote", h.HandleRemoteGet())
}

type RemoteViewData struct{}

func getRemoteViewData() (vd RemoteViewData, err error) {
	return vd, nil
}

func (h *Handlers) HandleRemoteGet() echo.HandlerFunc {
	t := "remote/index.page.tmpl"
	h.r.MustHave(t)
	return func(c echo.Context) error {
		// Run queries
		remoteViewData, err := getRemoteViewData()
		if err != nil {
			return err
		}
		// Produce output
		return h.r.CacheablePage(c.Response(), c.Request(), t, remoteViewData, struct{}{})
	}
}
