// Package internet contains the route handlers related to internet access.
package internet

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/sargassum-world/godest/handling"
	"github.com/sargassum-world/godest/turbostreams"

	dah "github.com/openUC2/device-admin/internal/app/deviceadmin/handling"
	nm "github.com/openUC2/device-admin/internal/clients/networkmanager"
)

func (h *Handlers) HandleDeviceAPsGetByIface() echo.HandlerFunc {
	t := "internet/devices/access-points/index.page.tmpl"
	h.r.MustHave(t)
	ta := "internet/devices/access-points/index.advanced.page.tmpl"
	h.r.MustHave(t)
	return func(c echo.Context) error {
		// Parse params
		iface := c.Param("iface")
		mode := c.QueryParam("mode")

		// Run queries
		vd, err := getDeviceAPsViewData(c.Request().Context(), iface, h.nmc)
		if err != nil {
			return err
		}
		// Produce output
		switch mode {
		default:
			return h.r.CacheablePage(c.Response(), c.Request(), t, vd, struct{}{})
		case dah.ViewModeAdvanced:
			return h.r.CacheablePage(c.Response(), c.Request(), ta, vd, struct{}{})
		}
	}
}

type DeviceAPsViewData struct {
	Interface      string
	AvailableSSIDs []string
	AvailableAPs   map[string][]nm.AccessPoint

	IsStreamPage bool
}

func getDeviceAPsViewData(
	ctx context.Context,
	iface string,
	nmc *nm.Client,
) (vd DeviceAPsViewData, err error) {
	vd.Interface = iface

	if vd.AvailableAPs, err = nmc.ScanNetworks(ctx, iface); err != nil {
		return vd, errors.Wrap(err, "couldn't scan for Wi-Fi networks")
	}
	for ssid, aps := range vd.AvailableAPs {
		if len(aps) == 0 {
			delete(vd.AvailableAPs, ssid)
			continue
		}
		slices.SortFunc(aps, func(a, b nm.AccessPoint) int {
			return cmp.Compare(b.Strength, a.Strength)
		})
		vd.AvailableAPs[ssid] = aps
	}

	vd.AvailableSSIDs = slices.SortedFunc(maps.Keys(vd.AvailableAPs), func(a, b string) int {
		return cmp.Compare(vd.AvailableAPs[b][0].Strength, vd.AvailableAPs[a][0].Strength)
	})

	return vd, nil
}

func (h *Handlers) HandleDeviceAPsSubByIface() turbostreams.HandlerFunc {
	return func(c *turbostreams.Context) error {
		// Parse params
		iface := c.Param("iface")

		// Run queries
		if _, err := h.nmc.GetDeviceByIface(c.Context(), iface); err != nil {
			return err
		}

		// Allow subscription
		return nil
	}
}

func (h *Handlers) HandleDeviceAPsPubByIface() turbostreams.HandlerFunc {
	t := "internet/devices/access-points/index.page.tmpl"
	h.r.MustHave(t)
	ta := "internet/devices/access-points/index.advanced.page.tmpl"
	h.r.MustHave(t)
	return func(c *turbostreams.Context) error {
		// Parse params
		iface := c.Param("iface")
		mode, err := c.QueryParam("mode")
		if err != nil {
			return errors.Wrap(err, "couldn't get query param 'mode'")
		}

		// Publish periodically
		const pubInterval = 4 * time.Second
		return handling.RepeatImmediate(c.Context(), pubInterval, func() (done bool, err error) {
			// Run queries
			if err := h.nmc.RescanNetworks(c.Context(), iface); err != nil {
				return false, err
			}
			vd, err := getDeviceAPsViewData(c.Context(), iface, h.nmc)
			if err != nil {
				return false, err
			}
			// Produce output
			vd.IsStreamPage = true
			template := t
			if mode == dah.ViewModeAdvanced {
				template = ta
			}
			return false, dah.PublishPageReload(c, h.r, template, vd)
		})
	}
}

func (h *Handlers) HandleDeviceAPsPostByIface() echo.HandlerFunc {
	return func(c echo.Context) error {
		// Parse params
		iface := c.Param("iface")
		state := c.FormValue("state")
		redirectTarget := c.FormValue("redirect-target")

		// Run queries
		switch state {
		default:
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf(
				"invalid Wi-Fi networks state %s", state,
			))
		case "refreshed":
			// Note: this function call will block until the scan finishes:
			if err := h.nmc.RescanNetworks(c.Request().Context(), iface); err != nil {
				return err
			}
			// Redirect user
			return c.Redirect(http.StatusSeeOther, redirectTarget)
		}
	}
}
