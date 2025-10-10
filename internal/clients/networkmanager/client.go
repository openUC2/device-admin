// Package networkmanager provides an interface for NetworkManager functionalities not exposed by
// nmstate.
package networkmanager

import (
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

func (c *Client) ScanNetworks() (ssids []string, err error) {
	const iface = "wlan0"
	dev, bus, err := c.getDevice(iface)
	if err != nil {
		return nil, err
	}

	var apPaths []dbus.ObjectPath
	prop, err := dev.GetProperty(nmName + ".Device.Wireless.AccessPoints")
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't query NetworkManager for access points found by %s", iface,
		)
	}
	if err = prop.Store(&apPaths); err != nil {
		return nil, errors.Wrap(err, "couldn't parse list of access points")
	}

	for _, apPath := range apPaths {
		ap := bus.Object(nmName, apPath)
		prop, err := ap.GetProperty(nmName + ".AccessPoint.Ssid")
		if err != nil {
			return nil, errors.Wrap(err, "couldn't query NetworkManager for SSID of an access point")
		}
		var ssid []uint8
		if err = prop.Store(&ssid); err != nil {
			return nil, errors.Wrap(err, "couldn't parse access point SSID")
		}
		ssids = append(ssids, string(ssid))
	}

	return ssids, nil
}

func (c *Client) getDevice(iface string) (dev dbus.BusObject, bus *dbus.Conn, err error) {
	if bus, err = dbus.ConnectSystemBus(); err != nil {
		return nil, nil, errors.Wrap(
			err, "couldn't connect to SystemBus bus to query NetworkManager",
		)
	}

	nm := bus.Object(nmName, "/org/freedesktop/NetworkManager")
	var devPath dbus.ObjectPath
	if err = nm.Call(nmName+".GetDeviceByIpIface", 0, iface).Store(&devPath); err != nil {
		return nil, nil, errors.Wrapf(err, "couldn't query NetworkManager for %s device", iface)
	}

	return bus.Object(nmName, devPath), bus, nil
}

const nmName = "org.freedesktop.NetworkManager"

func (c *Client) RescanNetworks() (err error) {
	const iface = "wlan0"
	dev, _, err := c.getDevice(iface)
	if err != nil {
		return err
	}
	var ssids map[string]any
	if call := dev.Call(nmName+".Device.Wireless.RequestScan", 0, ssids); call.Err != nil {
		return errors.Wrapf(call.Err, "couldn't request a re-scan of access points with %s", iface)
	}

	return nil
}
