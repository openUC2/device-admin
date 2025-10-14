package networkmanager

import (
	"context"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/pkg/errors"
)

// Wi-Fi devices

type WifiCaps uint32

func (c WifiCaps) HasNone() bool {
	return c == 0
}

func (c WifiCaps) SupportsCCMP() bool {
	return c&0x8 > 0
}

func (c WifiCaps) SupportsRSN() bool {
	return c&0x20 > 0
}

func (c WifiCaps) SupportsAP() bool {
	return c&0x40 > 0
}

func (c WifiCaps) SupportsAdhoc() bool {
	return c&0x80 > 0
}

func (c WifiCaps) Supports2GHz() bool {
	return (c&0x100 > 0) && (c&0x200 > 0)
}

func (c WifiCaps) Supports5GHz() bool {
	return (c&0x100 > 0) && (c&0x400 > 0)
}

func (c WifiCaps) Supports6GHz() bool {
	return (c&0x100 > 0) && (c&0x800 > 0)
}

func (c WifiCaps) SupportsMesh() bool {
	return c&0x1000 > 0
}

func (c WifiCaps) SupportsIBSSRSN() bool {
	return c&0x2000 > 0
}

type WifiMode string

const (
	WifiModeUnknown = "unknown"
	WifiModeAdhoc   = "ad-hoc"
	WifiModeInfra   = "access point"
	WifiModeAP      = "hotspot"
	WifiModeMesh    = "mesh"
)

type WifiDevice struct {
	Device
	Mode     WifiMode
	Caps     WifiCaps
	ActiveAP AccessPoint
	LastScan time.Duration
}

func GetWifiDevice(ctx context.Context, ipInterface string) (dev WifiDevice, err error) {
	devo, bus, err := findDevice(ctx, ipInterface)
	if err != nil {
		return WifiDevice{}, err
	}

	if dev, err = dumpWifiDevice(devo, bus); err != nil {
		return WifiDevice{}, errors.Wrapf(err, "couldn't inspect device %s", ipInterface)
	}
	return dev, nil
}

func dumpWifiDevice(devo dbus.BusObject, bus *dbus.Conn) (dev WifiDevice, err error) {
	if dev.Device, err = dumpDevice(devo); err != nil {
		return WifiDevice{}, errors.Wrap(
			err, "couldn't query for generic device properties",
		)
	}

	var rawMode uint32
	if err = devo.StoreProperty(nmName+".Device.Wireless.Mode", &rawMode); err != nil {
		return WifiDevice{}, errors.Wrap(err, "couldn't query for Wi-Fi mode")
	}
	if dev.Mode, err = parseWifiMode(rawMode); err != nil {
		return WifiDevice{}, errors.Wrap(err, "couldn't parse NetworkManager Wi-Fi mode")
	}

	if err = devo.StoreProperty(
		nmName+".Device.Wireless.WirelessCapabilities", &rawMode,
	); err != nil {
		return WifiDevice{}, errors.Wrap(err, "couldn't query for capabilities")
	}
	dev.Caps = WifiCaps(rawMode)

	var apPath dbus.ObjectPath
	if err = devo.StoreProperty(nmName+".Device.Wireless.ActiveAccessPoint", &apPath); err != nil {
		return WifiDevice{}, errors.Wrap(err, "couldn't query for active AP")
	}
	if dev.ActiveAP, err = dumpAccessPoint(bus.Object(nmName, apPath)); err != nil {
		return WifiDevice{}, errors.Wrapf(err, "couldn't query for access point %s", apPath)
	}

	if dev.LastScan, err = parseLastScan(devo); err != nil {
		return WifiDevice{}, errors.Wrap(err, "couldn't query for last scan")
	}

	return dev, nil
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

func parseLastScan(dev dbus.BusObject) (timestamp time.Duration, err error) {
	var rawTimestamp int64
	if err = dev.StoreProperty(nmName+".Device.Wireless.LastScan", &rawTimestamp); err != nil {
		return 0, errors.Wrapf(err, "couldn't query for time of last scan of access points")
	}

	if rawTimestamp == -1 {
		return -1, nil
	}
	return time.Duration(rawTimestamp) * time.Millisecond, nil
}
