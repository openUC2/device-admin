// Package handling provides utilities for handlers
package handling

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/sargassum-world/godest"
	"github.com/sargassum-world/godest/turbostreams"
)

// Handlers

func HandleTSMsg(r godest.TemplateRenderer) turbostreams.HandlerFunc {
	return func(c *turbostreams.Context) (err error) {
		return r.WriteTurboStream(c.MsgWriter(), c.Published()...)
	}
}

func AllowTSSub(_ godest.Logger) turbostreams.HandlerFunc {
	return func(c *turbostreams.Context) error {
		fmt.Println("SUB", c.Topic())
		return nil
	}
}

// Rendering

func PublishPageReload(
	c *turbostreams.Context, r godest.TemplateRenderer, templateName string, viewData any,
) error {
	rd, err := NewRenderData(c, r, viewData)
	if err != nil {
		return errors.Wrap(err, "couldn't make render data for turbostreams message")
	}
	c.Publish(turbostreams.Message{
		Action:   turbostreams.ActionReload,
		Data:     rd,
		Template: templateName,
	})
	return nil
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
