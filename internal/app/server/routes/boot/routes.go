// Package boot contains the route handlers related to the OS's boot state.
package boot

import (
	"context"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/sargassum-world/godest"
	"github.com/sargassum-world/godest/turbostreams"

	ipc "github.com/openUC2/machine-admin/internal/app/ipc/boot"
	sh "github.com/openUC2/machine-admin/internal/app/server/handling"
	sc "github.com/openUC2/machine-admin/internal/clients/sidecar"
)

type Handlers struct {
	r godest.TemplateRenderer

	scc *sc.Client

	l godest.Logger
}

func New(r godest.TemplateRenderer, scc *sc.Client, l godest.Logger) *Handlers {
	return &Handlers{
		r:   r,
		scc: scc,
		l:   l,
	}
}

func (h *Handlers) Register(er godest.EchoRouter) {
	er.GET(h.r.BasePath+"boot", h.HandleBootGet())
	er.POST(h.r.BasePath+"boot", h.HandleBootPost())
}

func (h *Handlers) HandleBootGet() echo.HandlerFunc {
	t := "boot/index.page.tmpl"
	h.r.MustHave(t)
	tm := "boot/index.minimal.page.tmpl"
	h.r.MustHave(tm)
	return func(c echo.Context) error {
		// Parse params
		mode := c.QueryParam("mode")

		// Produce output
		switch mode {
		default:
			return h.r.CacheablePage(c.Response(), c.Request(), t, struct{}{}, struct{}{})
		case sh.ViewModeMinimal:
			return h.r.CacheablePage(c.Response(), c.Request(), tm, struct{}{}, struct{}{})
		}
	}
}

func (h *Handlers) HandleBootPost() echo.HandlerFunc {
	st := "boot/shutdown-progress.partial.tmpl"
	h.r.MustHave(st)
	return func(c echo.Context) error {
		// Parse params
		state := c.FormValue("state")
		redirectTarget := c.FormValue("redirect-target")

		// Run queries
		ctx := c.Request().Context()
		if err := shutdown(ctx, state, h.scc, h.l); err != nil {
			return err
		}
		// Redirect user
		if turbostreams.Accepted(c.Request().Header) {
			return h.r.TurboStream(
				c.Response(),
				turbostreams.Message{
					Action:   turbostreams.ActionAppend,
					Target:   "boot_buttons",
					Template: st,
					Data: map[string]any{
						"state": state,
					},
				},
			)
		}
		return c.Redirect(http.StatusSeeOther, redirectTarget)
	}
}

func shutdown(
	ctx context.Context, state string, scc *sc.Client, l godest.Logger,
) error {
	switch state {
	default:
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf(
			"invalid boot state %s", state,
		))
	case "soft-rebooted":
		if err := shutdownViaSidecar(ctx, "SoftReboot", scc, l); err != nil {
			return errors.Wrapf(err, "couldn't soft-reboot through sidecar")
		}
	case "rebooted":
		if err := shutdownViaSidecar(ctx, "Reboot", scc, l); err != nil {
			return errors.Wrapf(err, "couldn't reboot through sidecar")
		}
	case "powered-off":
		if err := shutdownViaSidecar(ctx, "Poweroff", scc, l); err != nil {
			return errors.Wrapf(err, "couldn't power-off through sidecar")
		}
	}
	return nil
}

func shutdownViaSidecar(ctx context.Context, method string, scc *sc.Client, l godest.Logger) error {
	conn, err := scc.Open(ctx)
	if err != nil {
		return errors.Wrap(err, "couldn't open connection to sidecar")
	}
	defer sc.CloseConn(conn, l)

	switch method {
	default:
		return errors.Errorf("unknown sidecar method %s", method)
	case "SoftReboot":
		if err := ipc.SoftReboot().Call(ctx, conn); err != nil {
			return errors.Wrapf(err, "couldn't call sidecar's %s method", method)
		}
	case "Reboot":
		if err := ipc.Reboot().Call(ctx, conn); err != nil {
			return errors.Wrapf(err, "couldn't call sidecar's %s method", method)
		}
	case "Poweroff":
		if err := ipc.Poweroff().Call(ctx, conn); err != nil {
			return errors.Wrapf(err, "couldn't call sidecar's %s method", method)
		}
	}
	return nil
}
