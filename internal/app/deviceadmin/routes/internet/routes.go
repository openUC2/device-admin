// Package internet contains the route handlers related to internet access.
package internet

import (
	"context"
	"slices"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/sargassum-world/godest"
	"github.com/sargassum-world/godest/handling"
	"github.com/sargassum-world/godest/turbostreams"

	dah "github.com/openUC2/device-admin/internal/app/deviceadmin/handling"
	nm "github.com/openUC2/device-admin/internal/clients/networkmanager"
)

type Handlers struct {
	r godest.TemplateRenderer

	tsh *turbostreams.Hub

	nmc *nm.Client

	l godest.Logger
}

func New(
	r godest.TemplateRenderer, tsh *turbostreams.Hub, nmc *nm.Client, l godest.Logger,
) *Handlers {
	return &Handlers{
		r:   r,
		tsh: tsh,
		nmc: nmc,
		l:   l,
	}
}

func (h *Handlers) Register(er godest.EchoRouter, tr turbostreams.Router) {
	er.GET(h.r.BasePath+"internet", h.HandleInternetGet())
	tr.SUB(h.r.BasePath+"internet", dah.AllowTSSub())
	tr.PUB(h.r.BasePath+"internet", h.HandleInternetPub())
	// device-access-points
	er.GET(h.r.BasePath+"internet/devices/:iface/access-points", h.HandleDeviceAPsGetByIface())
	tr.SUB(h.r.BasePath+"internet/devices/:iface/access-points", h.HandleDeviceAPsSubByIface())
	tr.PUB(h.r.BasePath+"internet/devices/:iface/access-points", h.HandleDeviceAPsPubByIface())
	er.POST(h.r.BasePath+"internet/devices/:iface/access-points", h.HandleDeviceAPsPostByIface())
	// conn-profiles
	er.POST(h.r.BasePath+"internet/conn-profiles", h.HandleConnProfilesPost())
	er.GET(h.r.BasePath+"internet/conn-profiles/:uuid", h.HandleConnProfileGetByUUID())
	tr.SUB(h.r.BasePath+"internet/conn-profiles/:uuid", h.HandleConnProfileSubByUUID())
	tr.PUB(h.r.BasePath+"internet/conn-profiles/:uuid", h.HandleConnProfilePubByUUID())
	er.POST(h.r.BasePath+"internet/conn-profiles/:uuid", h.HandleConnProfilePostByUUID())
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
		vd, err := getInternetViewData(c.Request().Context(), h.nmc)
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

	IsStreamPage bool
}

func getInternetViewData(ctx context.Context, nmc *nm.Client) (vd InternetViewData, err error) {
	if vd.NM, err = nmc.Get(); err != nil {
		return vd, errors.Wrap(err, "couldn't get overall information about NetworkManager")
	}

	const iface = "wlan0"
	// Note(ethanjli): the list of APs is just for autocompletion in the simplified wifi management
	// view, and it can be missing just after activating wlan0-hotspot; so it's fine if we don't
	// provide any data about available APs on this page:
	availableAPs, _ := nmc.ScanNetworks(ctx, iface)
	for ssid, aps := range availableAPs {
		if len(aps) == 0 {
			continue
		}
		vd.AvailableSSIDs = append(vd.AvailableSSIDs, ssid)
	}
	slices.Sort(vd.AvailableSSIDs)

	allDevices, err := nmc.GetDevices(ctx)
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

	connProfiles, err := nmc.ListConnProfiles(ctx)
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

func (h *Handlers) HandleInternetPub() turbostreams.HandlerFunc {
	t := "internet/index.page.tmpl"
	h.r.MustHave(t)
	ta := "internet/index.advanced.page.tmpl"
	h.r.MustHave(ta)
	return func(c *turbostreams.Context) error {
		// Parse params
		mode, err := c.QueryParam("mode")
		if err != nil {
			return errors.Wrap(err, "couldn't get query param 'mode'")
		}

		// Publish periodically
		const pubInterval = 4 * time.Second
		return handling.RepeatImmediate(c.Context(), pubInterval, func() (done bool, err error) {
			// Run queries
			if mode != dah.ViewModeAdvanced {
				_ = h.nmc.RescanNetworks(c.Context(), "wlan0")
			}
			vd, err := getInternetViewData(c.Context(), h.nmc)
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
