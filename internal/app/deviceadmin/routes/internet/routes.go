// Package internet contains the route handlers related to internet access.
package internet

import (
	"context"
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
	// device-access-points
	er.GET(h.r.BasePath+"internet/devices/:iface/access-points", h.HandleDeviceAPsGet())
	er.POST(h.r.BasePath+"internet/devices/:iface/access-points", h.HandleDeviceAPsPost())
	// conn-profiles
	er.GET(h.r.BasePath+"internet/conn-profiles/:uuid", h.HandleConnProfileGetByUUID())
	er.POST(h.r.BasePath+"internet/conn-profiles/:uuid", h.HandleConnProfilePostByUUID())
	er.POST(h.r.BasePath+"internet/conn-profiles", h.HandleConnProfilesPost())
}

func (h *Handlers) HandleInternetGet() echo.HandlerFunc {
	t := "internet/index.page.tmpl"
	h.r.MustHave(t)
	ta := "internet/index.advanced.page.tmpl"
	h.r.MustHave(ta)
	return func(c echo.Context) error {
		// Parse params
		mode := c.QueryParam("mode")

		// Run queries
		vd, err := getInternetViewData(c.Request().Context())
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

type InternetViewData struct {
	// TODO: figure out the current SSID, if it exists
	AvailableSSIDs []string

	WifiDevices     []networkmanager.Device
	EthernetDevices []networkmanager.Device
	OtherDevices    []networkmanager.Device

	WifiConnProfiles     []networkmanager.ConnProfileSettingsConnection
	EthernetConnProfiles []networkmanager.ConnProfileSettingsConnection
	OtherConnProfiles    []networkmanager.ConnProfileSettingsConnection
}

func getInternetViewData(ctx context.Context) (vd InternetViewData, err error) {
	const iface = "wlan0"
	availableAPs, err := networkmanager.ScanNetworks(ctx, iface)
	if err != nil {
		return vd, errors.Wrap(err, "couldn't scan for Wi-Fi networks")
	}
	for ssid, aps := range availableAPs {
		if len(aps) == 0 {
			continue
		}
		vd.AvailableSSIDs = append(vd.AvailableSSIDs, ssid)
	}
	slices.Sort(vd.AvailableSSIDs)

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
		switch connProfile.Settings.Connection.Type.Info().Short {
		case "wifi":
			vd.WifiConnProfiles = append(vd.WifiConnProfiles, connProfile.Settings.Connection)
		case "ethernet":
			vd.EthernetConnProfiles = append(vd.EthernetConnProfiles, connProfile.Settings.Connection)
		default:
			vd.OtherConnProfiles = append(vd.OtherConnProfiles, connProfile.Settings.Connection)
		}
	}

	return vd, nil
}
