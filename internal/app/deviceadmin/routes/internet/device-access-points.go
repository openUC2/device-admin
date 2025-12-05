// Package internet contains the route handlers related to internet access.
package internet

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"net/http"
	"slices"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/openUC2/device-admin/internal/clients/networkmanager"
)

func (h *Handlers) HandleDeviceAPsGet() echo.HandlerFunc {
	t := "internet/devices/access-points/index.page.tmpl"
	h.r.MustHave(t)
	ta := "internet/devices/access-points/index.advanced.page.tmpl"
	h.r.MustHave(t)
	return func(c echo.Context) error {
		// Parse params
		iface := c.Param("iface")
		mode := c.QueryParam("mode")

		// Run queries
		vd, err := getDeviceAPsViewData(c.Request().Context(), iface)
		if err != nil {
			return err
		}
		// Produce output
		switch mode {
		default:
			return h.r.CacheablePage(c.Response(), c.Request(), t, vd, struct{}{})
		case "advanced":
			return h.r.CacheablePage(c.Response(), c.Request(), ta, vd, struct{}{})
		}
	}
}

type DeviceAPsViewData struct {
	Interface      string
	AvailableSSIDs []string
	AvailableAPs   map[string][]networkmanager.AccessPoint
}

func getDeviceAPsViewData(
	ctx context.Context,
	iface string,
) (vd DeviceAPsViewData, err error) {
	vd.Interface = iface

	if vd.AvailableAPs, err = networkmanager.ScanNetworks(ctx, iface); err != nil {
		return vd, errors.Wrap(err, "couldn't scan for Wi-Fi networks")
	}
	for ssid, aps := range vd.AvailableAPs {
		if len(aps) == 0 {
			delete(vd.AvailableAPs, ssid)
			continue
		}
		slices.SortFunc(aps, func(a, b networkmanager.AccessPoint) int {
			return cmp.Compare(b.Strength, a.Strength)
		})
		vd.AvailableAPs[ssid] = aps
	}

	vd.AvailableSSIDs = slices.SortedFunc(maps.Keys(vd.AvailableAPs), func(a, b string) int {
		return cmp.Compare(vd.AvailableAPs[b][0].Strength, vd.AvailableAPs[a][0].Strength)
	})

	return vd, nil
}

func (h *Handlers) HandleDeviceAPsPost() echo.HandlerFunc {
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
			if err := networkmanager.RescanNetworks(c.Request().Context(), iface); err != nil {
				return err
			}
			// Redirect user
			return c.Redirect(http.StatusSeeOther, redirectTarget)
		}
	}
}
