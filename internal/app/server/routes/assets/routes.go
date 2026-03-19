// Package assets contains the route handlers for assets which are static for the server
package assets

import (
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sargassum-world/godest"
)

const (
	AppURLPrefix    = "app/"
	StaticURLPrefix = "static/"
	FontsURLPrefix  = "fonts/"
)

type TemplatedHandlers struct {
	r godest.TemplateRenderer
}

func NewTemplated(r godest.TemplateRenderer) *TemplatedHandlers {
	return &TemplatedHandlers{
		r: r,
	}
}

func (h *TemplatedHandlers) Register(er godest.EchoRouter) {
	er.GET(h.r.BasePath+AppURLPrefix+"app.webmanifest", h.getWebmanifest())
	er.GET(h.r.BasePath+AppURLPrefix+"offline", h.getOffline())
}

func RegisterStatic(basePath string, er godest.EchoRouter, em godest.Embeds) {
	const (
		day  = 24 * time.Hour
		week = 7 * day
		year = 365 * day
	)

	// TODO: serve sw.js with an ETag!
	er.GET(
		basePath+"favicon.ico", echo.WrapHandler(godest.HandleFS(basePath, em.StaticFS, week)),
	)
	er.GET(
		basePath+FontsURLPrefix+"*",
		echo.WrapHandler(godest.HandleFS(basePath+FontsURLPrefix, em.FontsFS, year)),
	)
	er.GET(
		basePath+StaticURLPrefix+"*",
		echo.WrapHandler(godest.HandleFSFileRevved(basePath+StaticURLPrefix, em.StaticHFS)),
	)
	er.GET(
		basePath+AppURLPrefix+"*",
		echo.WrapHandler(godest.HandleFSFileRevved(basePath+AppURLPrefix, em.AppHFS)),
	)
}
