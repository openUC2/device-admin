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
	tr.SUB(h.r.BasePath+"internet", dah.AllowTSSub(h.l))
	tr.PUB(h.r.BasePath+"internet", h.HandleInternetPub())
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
		initialized := false

		// Parse params
		ctx := c.Context()
		queryParams, err := c.QueryParams()
		if err != nil {
			return errors.Wrap(err, "couldn't parse query params")
		}
		mode := ""
		if rawMode, ok := queryParams["mode"]; ok {
			mode = rawMode[0]
		}

		// Publish periodically
		const pubInterval = 4 * time.Second
		return handling.RepeatImmediate(ctx, pubInterval, func() (done bool, err error) {
			if !initialized {
				// We just started publishing because a page added a subscription, so there's no need to
				// send the devices list again - that page already has the latest version
				initialized = true
				return false, nil
			}

			// Run queries
			vd, err := getInternetViewData(ctx, h.nmc)
			if err != nil {
				return false, err
			}
			vd.IsStreamPage = true
			template := t
			if mode == dah.ViewModeAdvanced {
				template = ta
			}
			// Produce output
			rd, err := dah.NewRenderData(c, h.r, vd)
			if err != nil {
				return false, errors.Wrap(err, "couldn't make render data for turbostreams message")
			}
			c.Publish(turbostreams.Message{
				Action:   turbostreams.ActionReload,
				Data:     rd,
				Template: template,
			})
			return false, nil
		})
	}
}
