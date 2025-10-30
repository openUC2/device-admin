// Package networkmanager provides an interface for NetworkManager via its D-Bus API.
package networkmanager

import (
	"context"

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

func getNetworkManager(ctx context.Context) (nm dbus.BusObject, bus *dbus.Conn, err error) {
	if bus, err = dbus.ConnectSystemBus(dbus.WithContext(ctx)); err != nil {
		return nil, bus, errors.Wrap(
			err, "couldn't connect to SystemBus bus to query NetworkManager",
		)
	}

	return bus.Object(nmName, "/org/freedesktop/NetworkManager"), bus, nil
}

func getNetworkManagerSettings(ctx context.Context) (nm dbus.BusObject, bus *dbus.Conn, err error) {
	if bus, err = dbus.ConnectSystemBus(dbus.WithContext(ctx)); err != nil {
		return nil, bus, errors.Wrap(
			err, "couldn't connect to SystemBus bus to query NetworkManager settings",
		)
	}

	return bus.Object(nmName, "/org/freedesktop/NetworkManager/Settings"), bus, nil
}

const nmName = "org.freedesktop.NetworkManager"
