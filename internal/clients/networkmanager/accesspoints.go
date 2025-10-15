package networkmanager

import (
	"context"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/pkg/errors"
)

type AccessPoint struct {
	SSID      string
	Frequency uint32 // MHz
	Strength  uint8  // %
	LastSeen  time.Duration
	Mode      DeviceWifiMode
	RSN       RSNFlags // for WPA2 & WPA3
}

func (ap AccessPoint) HasData() bool {
	return ap != AccessPoint{}
}

// RSNFlags

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

// Client

func ScanNetworks(
	ctx context.Context, iface string,
) (networks map[string][]AccessPoint, err error) {
	dev, bus, err := findDevice(ctx, iface)
	if err != nil {
		return nil, err
	}

	var apPaths []dbus.ObjectPath
	if err = dev.StoreProperty(nmName+".Device.Wireless.AccessPoints", &apPaths); err != nil {
		return nil, errors.Wrapf(err, "couldn't query for access points found by %s", iface)
	}

	networks = make(map[string][]AccessPoint)
	for _, apPath := range apPaths {
		ap, err := dumpAccessPoint(bus.Object(nmName, apPath))
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't query for access point %s", apPath)
		}
		networks[ap.SSID] = append(networks[ap.SSID], ap)
	}

	return networks, nil
}

func dumpAccessPoint(apo dbus.BusObject) (ap AccessPoint, err error) {
	var rawSSID []uint8
	if err = apo.StoreProperty(nmName+".AccessPoint.Ssid", &rawSSID); err != nil {
		return AccessPoint{}, errors.Wrap(err, "couldn't query for SSID")
	}
	ap.SSID = string(rawSSID)

	if err = apo.StoreProperty(nmName+".AccessPoint.Frequency", &ap.Frequency); err != nil {
		return AccessPoint{}, errors.Wrap(err, "couldn't query for SSID")
	}

	if err = apo.StoreProperty(nmName+".AccessPoint.Strength", &ap.Strength); err != nil {
		return AccessPoint{}, errors.Wrap(err, "couldn't query for signal strength")
	}

	var rawLastSeen int32
	if err = apo.StoreProperty(nmName+".AccessPoint.LastSeen", &rawLastSeen); err != nil {
		return AccessPoint{}, errors.Wrap(err, "couldn't query for signal strength")
	}
	ap.LastSeen = time.Duration(rawLastSeen) * time.Second
	if rawLastSeen == -1 {
		ap.LastSeen = -1
	}

	var rawMode uint32
	if err = apo.StoreProperty(nmName+".AccessPoint.Mode", &rawMode); err != nil {
		return AccessPoint{}, errors.Wrap(err, "couldn't query for Wi-Fi mode")
	}
	ap.Mode = DeviceWifiMode(rawMode)

	if err = apo.StoreProperty(nmName+".AccessPoint.RsnFlags", &rawMode); err != nil {
		return AccessPoint{}, errors.Wrap(err, "couldn't query for WPA flags")
	}
	ap.RSN = RSNFlags(rawMode)

	return ap, err
}

func RescanNetworks(ctx context.Context, iface string) (err error) {
	dev, _, err := findDevice(ctx, iface)
	if err != nil {
		return err
	}

	prevLastScan, err := parseLastScan(dev)
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

		if lastScan, err = parseLastScan(dev); err != nil {
			return errors.Wrapf(err, "couldn't check status of access point re-scan with %s", iface)
		}
	}

	return nil
}
