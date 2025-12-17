// Package networkmanager provides an interface for NetworkManager via its D-Bus API.
package networkmanager

import (
	"context"
	"fmt"

	"github.com/godbus/dbus/v5"
	"github.com/pkg/errors"
	"github.com/sargassum-world/godest"
)

type Client struct {
	Config Config

	bus *dbus.Conn

	l godest.Logger
}

func NewClient(c Config, l godest.Logger) *Client {
	return &Client{
		Config: c,
		l:      l,
	}
}

func (c *Client) Open(ctx context.Context) (err error) {
	if c.bus, err = dbus.ConnectSystemBus(dbus.WithContext(ctx)); err != nil {
		return errors.Wrap(err, "couldn't connect to SystemBus bus to interact with NetworkManager")
	}
	return nil
}

func (c *Client) getNetworkManager() dbus.BusObject {
	return c.bus.Object(nmName, "/org/freedesktop/NetworkManager")
}

func (c *Client) getNetworkManagerSettings() dbus.BusObject {
	return c.bus.Object(nmName, "/org/freedesktop/NetworkManager/Settings")
}

const nmName = "org.freedesktop.NetworkManager"

type NetworkManager struct {
	NetworkingEnabled    bool
	WirelessEnabled      bool
	WirelessHWEnabled    bool
	PrimaryConnection    ActiveConn
	Version              string
	State                NetworkManagerState
	Connectivity         NetworkManagerConnectivity
	ConnectivityCheckURI string
}

type NetworkManagerState uint32

var networkManagerState = map[NetworkManagerState]EnumInfo{
	0: {
		Short:   "unknown",
		Details: "NetworkManager daemon error makes it unable to reasonably assess state",
		Level:   "error",
	},
	10: {
		Short:   "disabled",
		Details: "NetworkManager is disabled",
		Level:   "warning",
	},
	20: {
		Short:   "disconnected",
		Details: "no active network connection",
		Level:   "warning",
	},
	30: {
		Short:   "disconnecting",
		Details: "network connections are being cleaned up",
		Level:   "info",
	},
	40: {
		Short:   "connecting",
		Details: "network connection is being started",
		Level:   "info",
	},
	50: {
		Short:   "locally connected",
		Details: "no default route to access the internet",
		Level:   "info",
	},
	60: {
		Short:   "site-wide connected",
		Details: "default route is available, but internet connectivity check didn't succeed",
		Level:   "info",
	},
	70: {
		Short:   "globally connected",
		Details: "internet connectivity check succeeded",
		Level:   "success",
	},
}

func (s NetworkManagerState) Info() EnumInfo {
	info, ok := networkManagerState[s]
	if !ok {
		return EnumInfo{
			Short:   "unknown",
			Details: fmt.Sprintf("unknown state (%d)", s),
			Level:   EnumInfoLevelError,
		}
	}
	return info
}

type NetworkManagerConnectivity uint32

var networkManagerConnectivity = map[NetworkManagerConnectivity]EnumInfo{
	0: {
		Short:   "unknown",
		Details: "connectivity checks are disabled or have not yet been run",
		Level:   "info",
	},
	1: {
		Short:   "none",
		Details: "not connected to any network",
		Level:   "warning",
	},
	2: {
		Short:   "portal",
		Details: "internet connection is hijacked by a captive portal gateway",
		Level:   "info",
	},
	3: {
		Short:   "limited",
		Details: "connected to a network, unable to reach full internet; no captive portal detected",
		Level:   "info",
	},
	4: {
		Short:   "full",
		Details: "connected to network and able to reach full internet",
		Level:   "success",
	},
}

func (c NetworkManagerConnectivity) Info() EnumInfo {
	info, ok := networkManagerConnectivity[c]
	if !ok {
		return EnumInfo{
			Short:   "unknown",
			Details: fmt.Sprintf("unknown connectivity (%d)", c),
			Level:   EnumInfoLevelError,
		}
	}
	return info
}

func (c *Client) Get() (nm NetworkManager, err error) {
	nmo := c.getNetworkManager()

	if err = nmo.StoreProperty(nmName+".NetworkingEnabled", &nm.NetworkingEnabled); err != nil {
		return nm, errors.Wrap(err, "couldn't query for networking enablement")
	}
	if err = nmo.StoreProperty(nmName+".WirelessEnabled", &nm.WirelessEnabled); err != nil {
		return nm, errors.Wrap(err, "couldn't query for wireless enablement")
	}
	if err = nmo.StoreProperty(nmName+".WirelessHardwareEnabled", &nm.WirelessHWEnabled); err != nil {
		return nm, errors.Wrap(err, "couldn't query for wireless hardware enablement")
	}
	if err = nmo.StoreProperty(nmName+".Version", &nm.Version); err != nil {
		return nm, errors.Wrap(err, "couldn't query for version")
	}

	var rawUint uint32
	if err = nmo.StoreProperty(nmName+".State", &rawUint); err != nil {
		return nm, errors.Wrap(err, "couldn't query for state")
	}
	nm.State = NetworkManagerState(rawUint)

	if err = nmo.StoreProperty(nmName+".Connectivity", &rawUint); err != nil {
		return nm, errors.Wrap(err, "couldn't query for connectivity")
	}
	nm.Connectivity = NetworkManagerConnectivity(rawUint)

	if err = nmo.StoreProperty(nmName+".ConnectivityCheckUri", &nm.ConnectivityCheckURI); err != nil {
		return nm, errors.Wrap(err, "couldn't query for connectivity check URI")
	}

	var connPath dbus.ObjectPath
	if err = nmo.StoreProperty(nmName+".PrimaryConnection", &connPath); err != nil {
		return nm, errors.Wrap(err, "couldn't query for primary connection")
	}
	if connPath != "/" {
		if nm.PrimaryConnection, err = dumpActiveConn(
			c.bus.Object(nmName, connPath), c.bus,
		); err != nil {
			return nm, errors.Wrapf(err, "couldn't query for connection %s", connPath)
		}
	}

	return nm, nil
}
