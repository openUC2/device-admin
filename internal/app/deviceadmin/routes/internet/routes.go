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
	er.GET(h.r.BasePath+"internet/connection-profiles/:uuid", h.HandleConnProfilesGetByUUID())
	er.POST(h.r.BasePath+"internet/connection-profiles/:uuid", h.HandleConnProfilesPostByUUID())
	er.POST(h.r.BasePath+"internet/connection-profiles", h.HandleConnProfilesPost())
}

func (h *Handlers) HandleInternetGet() echo.HandlerFunc {
	t := "internet/main.page.tmpl"
	h.r.MustHave(t)
	return func(c echo.Context) error {
		// Run queries
		vd, err := getInternetViewData(c.Request().Context())
		if err != nil {
			return err
		}
		// Produce output
		return h.r.CacheablePage(c.Response(), c.Request(), t, vd, struct{}{})
	}
}

type InternetViewData struct {
	// TODO: figure out the current SSID, if it exists
	AvailableSSIDs []string
	AvailableAPs   map[string][]networkmanager.AccessPoint

	WifiDevices     []networkmanager.Device
	EthernetDevices []networkmanager.Device
	OtherDevices    []networkmanager.Device

	ConnProfiles []networkmanager.ConnProfileSettingsConnection
}

const iface = "wlan0"

func getInternetViewData(ctx context.Context) (vd InternetViewData, err error) {
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

	allDevices, err := networkmanager.GetDevices(ctx)
	if err != nil {
		return vd, errors.Wrap(err, "couldn't list network devices")
	}
	for _, device := range allDevices {
		switch device.Type.Info().Short {
		default:
			vd.OtherDevices = append(vd.OtherDevices, device)
		case "wifi":
			vd.WifiDevices = append(vd.WifiDevices, device)
		case "ethernet":
			vd.EthernetDevices = append(vd.EthernetDevices, device)
		}
	}

	connProfiles, err := networkmanager.ListConnProfiles(ctx)
	if err != nil {
		return vd, errors.Wrap(err, "couldn't list connection profiles")
	}
	for _, connProfile := range connProfiles {
		vd.ConnProfiles = append(vd.ConnProfiles, connProfile.Settings.Connection)
	}

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
			if err := networkmanager.RescanNetworks(c.Request().Context(), iface); err != nil {
				return err
			}
			// Redirect user
			return c.Redirect(http.StatusSeeOther, h.r.BasePath+"internet")
		}
	}
}
