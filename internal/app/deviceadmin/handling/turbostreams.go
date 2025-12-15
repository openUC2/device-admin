// Package handling provides utilities for handlers
package handling

import (
	"github.com/pkg/errors"
	"github.com/sargassum-world/godest"
	"github.com/sargassum-world/godest/turbostreams"
)

// Rendering

func HandleTSMsg(r godest.TemplateRenderer) turbostreams.HandlerFunc {
	return func(c *turbostreams.Context) (err error) {
		return r.WriteTurboStream(c.MsgWriter(), c.Published()...)
	}
}

func AllowTSSub(l godest.Logger) turbostreams.HandlerFunc {
	return func(c *turbostreams.Context) error {
		l.Info("SUB " + c.Topic())
		return nil
	}
}

func NewRenderData(
	c *turbostreams.Context, tr godest.TemplateRenderer, data any,
) (godest.RenderData, error) {
	d := tr.NewRenderData(nil, data, struct{}{})
	m, err := makeMeta(c, tr)
	if err != nil {
		return godest.RenderData{}, errors.Wrap(err, "couldn't determine render metadata from context")
	}
	return godest.RenderData{
		Meta:    m,
		Inlines: d.Inlines,
		Data:    data,
	}, nil
}

func makeMeta(c *turbostreams.Context, tr godest.TemplateRenderer) (godest.RenderDataMeta, error) {
	formValues, err := c.QueryParams()
	if err != nil {
		return godest.RenderDataMeta{}, errors.Wrap(err, "couldn't parse query params")
	}
	return godest.RenderDataMeta{
		BasePath: tr.BasePath,
		Form:     godest.FormValues{Values: formValues},
	}, nil
}
