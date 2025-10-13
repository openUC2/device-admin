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
	"github.com/sargassum-world/godest"

	"github.com/openUC2/device-admin/internal/clients/networkmanager"
)

type Handlers struct {
	r godest.TemplateRenderer

	nmc *networkmanager.Client
}

func New(r godest.TemplateRenderer, nmc *networkmanager.Client) *Handlers {
	return &Handlers{
		r:   r,
		nmc: nmc,
	}
}

func (h *Handlers) Register(er godest.EchoRouter) {
	er.GET(h.r.BasePath+"internet", h.HandleInternetGet())
	er.POST(h.r.BasePath+"internet/wifi/networks", h.HandleWiFiNetworksPost())
}

type InternetViewData struct {
	// TODO: figure out the current SSID, if it exists
	AvailableSSIDs []string
	AvailableAPs   map[string][]networkmanager.AccessPoint
}

func (h *Handlers) HandleInternetGet() echo.HandlerFunc {
	t := "internet/main.page.tmpl"
	h.r.MustHave(t)
	return func(c echo.Context) error {
		// Run queries
		internetViewData, err := getInternetViewData(c.Request().Context(), h.nmc)
		if err != nil {
			return err
		}
		// Produce output
		return h.r.CacheablePage(c.Response(), c.Request(), t, internetViewData, struct{}{})
	}
}

func getInternetViewData(
	ctx context.Context, nmc *networkmanager.Client,
) (vd InternetViewData, err error) {
	if vd.AvailableAPs, err = nmc.ScanNetworks(ctx); err != nil {
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

func (h *Handlers) HandleWiFiNetworksPost() echo.HandlerFunc {
	return func(c echo.Context) error {
		// Parse params
		state := c.FormValue("state")

		// Run queries
		switch state {
		default:
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf(
				"invalid Wi-Fi networks state %s", state,
			))
		case "refreshed":
			// Note: this function call will block until the scan finishes:
			if err := h.nmc.RescanNetworks(c.Request().Context()); err != nil {
				return err
			}
			// Redirect user
			return c.Redirect(http.StatusSeeOther, h.r.BasePath+"internet")
		}
	}
}
