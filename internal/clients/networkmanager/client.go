// Package networkmanager provides an interface for NetworkManager functionalities not exposed by
// nmstate.
package networkmanager

import (
	"context"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/pkg/errors"
	"github.com/sargassum-world/godest"
)

type Client struct {
	Config Config

	l godest.Logger
}

func NewClient(c Config, l godest.Logger) *Client {
	return &Client{
		Config: c,
		l:      l,
	}
}

type AccessPoint struct {
	SSID      string
	Frequency uint32 // MHz
	Strength  uint8  // %
	LastSeen  int32  // sec
	Mode      WifiMode
	RSN       RSNFlags // for WPA2 & WPA3
}

type WifiMode string

const (
	WifiModeUnknown = "unknown"
	WifiModeAdhoc   = "ad-hoc"
	WifiModeInfra   = "access point"
	WifiModeAP      = "hotspot"
	WifiModeMesh    = "mesh"
)

type RSNFlags uint32

func (f RSNFlags) IsNone() bool {
	return f == 0
}

func (f RSNFlags) SupportsPairCCMP() bool {
	return f&0x8 > 0
}

func (f RSNFlags) SupportsGroupCCMP() bool {
	return f&0x80 > 0
}

func (f RSNFlags) SupportsPSK() bool {
	return f&0x100 > 0
}

func (f RSNFlags) SupportsSAE() bool {
	return f&0x400 > 0
}

func (f RSNFlags) SupportsOWE() bool {
	return f&0x800 > 0
}

func (f RSNFlags) SupportsEAPSuiteB192() bool {
	return f&0x2000 > 0
}

func (c *Client) ScanNetworks(ctx context.Context) (networks map[string][]AccessPoint, err error) {
	const iface = "wlan0"
	dev, bus, err := c.getDevice(ctx, iface)
	if err != nil {
		return nil, err
	}

	var apPaths []dbus.ObjectPath
	if err = dev.StoreProperty(nmName+".Device.Wireless.AccessPoints", &apPaths); err != nil {
		return nil, errors.Wrapf(
			err, "couldn't query NetworkManager for access points found by %s", iface,
		)
	}

	networks = make(map[string][]AccessPoint)
	for _, apPath := range apPaths {
		ap, err := c.getAccessPoint(bus.Object(nmName, apPath))
		if err != nil {
			return nil, errors.Wrapf(
				err, "couldn't query NetworkManager for information about access point %s", apPath,
			)
		}
		networks[ap.SSID] = append(networks[ap.SSID], ap)
	}

	return networks, nil
}

const nmName = "org.freedesktop.NetworkManager"

func (c *Client) getDevice(
	ctx context.Context, iface string,
) (dev dbus.BusObject, bus *dbus.Conn, err error) {
	if bus, err = dbus.ConnectSystemBus(dbus.WithContext(ctx)); err != nil {
		return nil, nil, errors.Wrap(
			err, "couldn't connect to SystemBus bus to query NetworkManager",
		)
	}

	nm := bus.Object(nmName, "/org/freedesktop/NetworkManager")
	var devPath dbus.ObjectPath
	if err = nm.CallWithContext(
		ctx, nmName+".GetDeviceByIpIface", 0, iface,
	).Store(&devPath); err != nil {
		return nil, nil, errors.Wrapf(err, "couldn't query NetworkManager for %s device", iface)
	}

	return bus.Object(nmName, devPath), bus, nil
}

func (c *Client) getAccessPoint(apo dbus.BusObject) (ap AccessPoint, err error) {
	var rawSSID []uint8
	if err = apo.StoreProperty(nmName+".AccessPoint.Ssid", &rawSSID); err != nil {
		return AccessPoint{}, errors.Wrap(err, "couldn't query NetworkManager for SSID")
	}
	ap.SSID = string(rawSSID)

	if err = apo.StoreProperty(nmName+".AccessPoint.Frequency", &ap.Frequency); err != nil {
		return AccessPoint{}, errors.Wrap(err, "couldn't query NetworkManager for SSID")
	}

	if err = apo.StoreProperty(nmName+".AccessPoint.Strength", &ap.Strength); err != nil {
		return AccessPoint{}, errors.Wrap(err, "couldn't query NetworkManager for signal strength")
	}

	if err = apo.StoreProperty(nmName+".AccessPoint.LastSeen", &ap.LastSeen); err != nil {
		return AccessPoint{}, errors.Wrap(err, "couldn't query NetworkManager for signal strength")
	}

	var rawMode uint32
	if err = apo.StoreProperty(nmName+".AccessPoint.Mode", &rawMode); err != nil {
		return AccessPoint{}, errors.Wrap(err, "couldn't query NetworkManager for Wi-Fi mode")
	}
	if ap.Mode, err = parseWifiMode(rawMode); err != nil {
		return AccessPoint{}, errors.Wrap(err, "couldn't parse NetworkManager Wi-Fi mode")
	}

	if err = apo.StoreProperty(nmName+".AccessPoint.RsnFlags", &rawMode); err != nil {
		return AccessPoint{}, errors.Wrap(err, "couldn't query NetworkManager for WPA flags")
	}
	ap.RSN = RSNFlags(rawMode)

	return ap, err
}

func parseWifiMode(rawMode uint32) (WifiMode, error) {
	switch rawMode {
	default:
		return "", errors.Errorf("unknown 802.11 mode %d", rawMode)
	case 0:
		return WifiModeUnknown, nil
	case 1:
		return WifiModeAdhoc, nil
	case 2: //nolint:mnd // the relationship is clear on the next line
		return WifiModeInfra, nil
	case 3: //nolint:mnd // the relationship is clear on the next line
		return WifiModeAP, nil
	case 4: //nolint:mnd // the relationship is clear on the next line
		return WifiModeMesh, nil
	}
}

func (c *Client) RescanNetworks(ctx context.Context) (err error) {
	const iface = "wlan0"
	dev, _, err := c.getDevice(ctx, iface)
	if err != nil {
		return err
	}

	prevLastScan, err := c.lastScan(dev)
	if err != nil {
		return err
	}

	var ssids map[string]any
	if call := dev.CallWithContext(
		ctx, nmName+".Device.Wireless.RequestScan", 0, ssids,
	); call.Err != nil {
		return errors.Wrapf(call.Err, "couldn't request a re-scan of access points with %s", iface)
	}

	lastScan := prevLastScan
	for lastScan == prevLastScan {
		// Note: polling is much simpler than watching for a D-Bus signal, and we can just watch for a
		// different timestamp, so we might as well take the polling approach here. Caveat: here we are
		// assuming that a scan won't take so long that the last timestamp (after integer rollover) will
		// be the same as the next timestamp.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
		if err := ctx.Err(); err != nil {
			// Context was also canceled and it should have priority
			return err
		}

		if lastScan, err = c.lastScan(dev); err != nil {
			return errors.Wrapf(err, "couldn't check status of access point re-scan with %s", iface)
		}
	}

	return nil
}

func (c *Client) lastScan(dev dbus.BusObject) (timestamp time.Duration, err error) {
	prop, err := dev.GetProperty(nmName + ".Device.Wireless.LastScan")
	if err != nil {
		return 0, errors.Wrapf(
			err, "couldn't query NetworkManager for time of last scan of access points",
		)
	}
	var rawTimestamp int64
	if err = prop.Store(&rawTimestamp); err != nil {
		return 0, errors.Wrap(err, "couldn't parse access point scan timestamp")
	}

	if rawTimestamp == -1 {
		return -1, nil
	}
	return time.Duration(rawTimestamp) * time.Millisecond, nil
}
