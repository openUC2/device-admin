// Package boot contains the route handlers related to the OS's boot state.
package boot

import (
	"context"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/sargassum-world/godest"

	ipc "github.com/openUC2/device-admin/internal/app/ipc/boot"
	sh "github.com/openUC2/device-admin/internal/app/server/handling"
	sc "github.com/openUC2/device-admin/internal/clients/sidecar"
	sd "github.com/openUC2/device-admin/internal/clients/systemd"
)

type Handlers struct {
	r godest.TemplateRenderer

	sdc *sd.Client
	scc *sc.Client

	l godest.Logger
}

func New(r godest.TemplateRenderer, sdc *sd.Client, scc *sc.Client, l godest.Logger) *Handlers {
	return &Handlers{
		r:   r,
		sdc: sdc,
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
	return func(c echo.Context) error {
		// Parse params
		state := c.FormValue("state")
		redirectTarget := c.FormValue("redirect-target")

		// Run queries
		ctx := c.Request().Context()
		switch state {
		default:
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf(
				"invalid boot state %s", state,
			))
		case "soft-rebooted":
			if err := shutdownViaSidecar(ctx, "SoftReboot", h.scc, h.l); err != nil {
				if sdErr := h.sdc.SoftReboot(ctx); err != nil {
					return errors.Wrapf(
						sdErr, "couldn't soft-reboot through sidecar (%s) or directly", err.Error(),
					)
				}
				h.l.Warnf("soft-rebooted directly after failure to soft-reboot through sidecar", err)
			}
			// Redirect user
			return c.Redirect(http.StatusSeeOther, redirectTarget)
		case "rebooted":
			if err := shutdownViaSidecar(ctx, "Reboot", h.scc, h.l); err != nil {
				if sdErr := h.sdc.Reboot(ctx); err != nil {
					return errors.Wrapf(
						sdErr, "couldn't reboot through sidecar (%s) or directly", err.Error(),
					)
				}
				h.l.Warnf("rebooted directly after failure to reboot through sidecar", err)
			}
			// Redirect user
			return c.Redirect(http.StatusSeeOther, redirectTarget)
		case "powered-off":
			if err := shutdownViaSidecar(ctx, "Poweroff", h.scc, h.l); err != nil {
				if sdErr := h.sdc.Poweroff(ctx); err != nil {
					return errors.Wrapf(
						sdErr, "couldn't power-off through sidecar (%s) or directly", err.Error(),
					)
				}
				h.l.Warnf("powered-off directly after failure to power-off through sidecar", err)
			}
			// Redirect user
			return c.Redirect(http.StatusSeeOther, redirectTarget)
		}
	}
}

func shutdownViaSidecar(ctx context.Context, method string, scc *sc.Client, l godest.Logger) error {
	conn, err := scc.Open(ctx)
	if err != nil {
		return errors.Wrap(err, "couldn't open connection to sidecar")
	}
	defer func() {
		if conn == nil {
			return
		}
		if err := conn.Close(); err != nil {
			l.Error(errors.New("couldn't close connection to sidecar"))
		}
	}()
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
