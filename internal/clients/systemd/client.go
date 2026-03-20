// Package systemd provides an interface for systemd via its D-Bus API.
package systemd

import (
	"context"

	"github.com/godbus/dbus/v5"
	"github.com/pkg/errors"
	"github.com/sargassum-world/godest"
)

type Client struct {
	Config Config

	bus *dbus.Conn

	l godest.Logger
}

type Config struct{}

func NewClient(c Config, l godest.Logger) *Client {
	return &Client{
		Config: c,
		l:      l,
	}
}

func (c *Client) Open(ctx context.Context) (err error) {
	if c.bus, err = dbus.ConnectSystemBus(dbus.WithContext(ctx)); err != nil {
		return errors.Wrap(err, "couldn't connect to SystemBus bus to interact with systemd")
	}
	return nil
}

const sdName = "org.freedesktop.systemd1"

func (c *Client) getSystemdManager() dbus.BusObject {
	return c.bus.Object(sdName, "/org/freedesktop/systemd1")
}

// Boot

func (c *Client) Poweroff(ctx context.Context) error {
	sdm := c.getSystemdManager()
	if err := sdm.CallWithContext(ctx, sdName+".Manager.PowerOff", 0).Store(); err != nil {
		return errors.Wrap(err, "couldn't power-off")
	}
	return nil
}

func (c *Client) Reboot(ctx context.Context) error {
	sdm := c.getSystemdManager()
	if err := sdm.CallWithContext(ctx, sdName+".Manager.Reboot", 0).Store(); err != nil {
		return errors.Wrap(err, "couldn't reboot")
	}
	return nil
}

func (c *Client) SoftReboot(ctx context.Context) error {
	sd := c.getSystemdManager()
	if err := sd.CallWithContext(ctx, sdName+".Manager.SoftReboot", 0, "").Store(); err != nil {
		return errors.Wrap(err, "couldn't soft-reboot")
	}
	return nil
}
