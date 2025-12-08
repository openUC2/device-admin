// Package internet contains the route handlers related to internet access.
package internet

import (
	"context"
	"slices"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/sargassum-world/godest"

	nm "github.com/openUC2/device-admin/internal/clients/networkmanager"
)

type Handlers struct {
	r godest.TemplateRenderer

	nmc *nm.Client
}

func New(r godest.TemplateRenderer, nmc *nm.Client) *Handlers {
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
	NM nm.NetworkManager

	AvailableSSIDs           []string
	Wlan0InternetConnProfile nm.ConnProfile
	Wlan0HotspotConnProfile  nm.ConnProfile

	WifiDevices     []nm.Device
	EthernetDevices []nm.Device
	OtherDevices    []nm.Device

	WifiConnProfiles     []nm.ConnProfileSettingsConn
	EthernetConnProfiles []nm.ConnProfileSettingsConn
	OtherConnProfiles    []nm.ConnProfileSettingsConn
}

func getInternetViewData(ctx context.Context) (vd InternetViewData, err error) {
	if vd.NM, err = nm.Get(ctx); err != nil {
		return vd, errors.Wrap(err, "couldn't get overall information about NetworkManager")
	}

	const iface = "wlan0"
	// Note(ethanjli): the list of APs is just for autocompletion in the simplified wifi management
	// view, and it can be missing just after activating wlan0-hotspot; so it's fine if we don't
	// provide any data about available APs on this page:
	availableAPs, _ := nm.ScanNetworks(ctx, iface)
	for ssid, aps := range availableAPs {
		if len(aps) == 0 {
			continue
		}
		vd.AvailableSSIDs = append(vd.AvailableSSIDs, ssid)
	}
	slices.Sort(vd.AvailableSSIDs)

	allDevices, err := nm.GetDevices(ctx)
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

	connProfiles, err := nm.ListConnProfiles(ctx)
	if err != nil {
		return vd, errors.Wrap(err, "couldn't list connection profiles")
	}
	for _, connProfile := range connProfiles {
		switch connProfile.Settings.Conn.Type.Info().Short {
		case "wifi":
			vd.WifiConnProfiles = append(vd.WifiConnProfiles, connProfile.Settings.Conn)
			switch conn := connProfile.Settings.Conn; conn.ID {
			case "wlan0-internet":
				vd.Wlan0InternetConnProfile = connProfile
			case "wlan0-hotspot":
				vd.Wlan0HotspotConnProfile = connProfile
			}
		case "ethernet":
			vd.EthernetConnProfiles = append(vd.EthernetConnProfiles, connProfile.Settings.Conn)
		default:
			vd.OtherConnProfiles = append(vd.OtherConnProfiles, connProfile.Settings.Conn)
		}
	}

	return vd, nil
}
