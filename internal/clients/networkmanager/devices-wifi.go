package networkmanager

import (
	"fmt"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/pkg/errors"
)

type DeviceWifiCaps uint32

func (c DeviceWifiCaps) HasNone() bool {
	return c == 0
}

func (c DeviceWifiCaps) SupportsCCMP() bool {
	return c&0x8 > 0
}

func (c DeviceWifiCaps) SupportsRSN() bool {
	return c&0x20 > 0
}

func (c DeviceWifiCaps) SupportsAP() bool {
	return c&0x40 > 0
}

func (c DeviceWifiCaps) SupportsAdhoc() bool {
	return c&0x80 > 0
}

func (c DeviceWifiCaps) Supports2GHz() bool {
	return (c&0x100 > 0) && (c&0x200 > 0)
}

func (c DeviceWifiCaps) Supports5GHz() bool {
	return (c&0x100 > 0) && (c&0x400 > 0)
}

func (c DeviceWifiCaps) Supports6GHz() bool {
	return (c&0x100 > 0) && (c&0x800 > 0)
}

func (c DeviceWifiCaps) SupportsMesh() bool {
	return c&0x1000 > 0
}

func (c DeviceWifiCaps) SupportsIBSSRSN() bool {
	return c&0x2000 > 0
}

type DeviceWifiMode uint32

var deviceWifiModeInfo = map[DeviceWifiMode]EnumInfo{
	0: {
		Short: "unknown",
		Level: "warning",
	},
	1: {
		Short:   "ad-hoc",
		Details: "part of an Ad-Hoc 802.11 network without a central coordinating access point",
		Level:   "info",
	},
	2: {
		Short: "infrastructure",
		Level: "success",
	},
	3: {
		Short:   "hotspot",
		Details: "this device is acting as an access point/hotspot",
		Level:   "info",
	},
	4: {
		Short:   "mesh",
		Details: "802.11s mesh point",
		Level:   "info",
	},
}

func (s DeviceWifiMode) Info() EnumInfo {
	info, ok := deviceWifiModeInfo[s]
	if !ok {
		return EnumInfo{
			Short:   "unknown",
			Details: fmt.Sprintf("mode (%d) was reported but could not be determined", s),
			Level:   "error",
		}
	}
	return info
}

type WifiDevice struct {
	Mode     DeviceWifiMode
	ActiveAP AccessPoint
	Caps     DeviceWifiCaps
	LastScan time.Duration
}

func (d WifiDevice) HasData() bool {
	return d != WifiDevice{}
}

func dumpWifiDevice(devo dbus.BusObject, bus *dbus.Conn) (dev WifiDevice, err error) {
	var rawMode uint32
	if err = devo.StoreProperty(nmName+".Device.Wireless.Mode", &rawMode); err != nil {
		return WifiDevice{}, errors.Wrap(err, "couldn't query for Wi-Fi mode")
	}
	dev.Mode = DeviceWifiMode(rawMode)

	var apPath dbus.ObjectPath
	if err = devo.StoreProperty(nmName+".Device.Wireless.ActiveAccessPoint", &apPath); err != nil {
		return WifiDevice{}, errors.Wrap(err, "couldn't query for active AP")
	}
	if dev.ActiveAP, err = dumpAccessPoint(bus.Object(nmName, apPath)); err != nil {
		return WifiDevice{}, errors.Wrapf(err, "couldn't query for access point %s", apPath)
	}

	if err = devo.StoreProperty(
		nmName+".Device.Wireless.WirelessCapabilities", &rawMode,
	); err != nil {
		return WifiDevice{}, errors.Wrap(err, "couldn't query for capabilities")
	}
	dev.Caps = DeviceWifiCaps(rawMode)

	if dev.LastScan, err = parseLastScan(devo); err != nil {
		return WifiDevice{}, errors.Wrap(err, "couldn't query for last scan")
	}

	return dev, nil
}

func parseLastScan(dev dbus.BusObject) (timestamp time.Duration, err error) {
	var rawTimestamp int32
	if err = dev.StoreProperty(nmName+".Device.Wireless.LastScan", &rawTimestamp); err != nil {
		return 0, errors.Wrapf(err, "couldn't query for time of last scan of access points")
	}

	if rawTimestamp == -1 {
		return -1, nil
	}
	return time.Duration(rawTimestamp) * time.Millisecond, nil
}
