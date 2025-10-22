package networkmanager

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/godbus/dbus/v5"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type ActiveConn struct {
	ID               string
	UUID             uuid.UUID
	Type             string
	DeviceInterfaces []string
	State            ActiveConnState
	StateFlags       ConnectionActivationStateFlags
	IsIPv4Default    bool
	IsIPv6Default    bool
	IsVPN            bool
}

func (c ActiveConn) HasData() bool {
	return c.ID != "" || c.UUID != uuid.UUID{} || c.Type != ""
}

type ActiveConnState uint32

var activeConnectionStateInfo = map[ActiveConnState]EnumInfo{
	0: {
		Short: "unknown",
		Level: "warning",
	},
	1: {
		Short:   "activating",
		Details: "network connection is being prepared",
		Level:   "info",
	},
	2: {
		Short:   "activated",
		Details: "there is a connection to the network",
		Level:   "success",
	},
	3: {
		Short:   "deactivating",
		Details: "network connection is being torn down and cleaned up",
		Level:   "info",
	},
	4: {
		Short:   "deactivating",
		Details: "network connection is disconnected and will be removed",
		Level:   "info",
	},
}

func (s ActiveConnState) Info() EnumInfo {
	info, ok := activeConnectionStateInfo[s]
	if !ok {
		return EnumInfo{
			Short:   "unknown",
			Details: fmt.Sprintf("state (%d) was reported but could not be determined", s),
			Level:   "error",
		}
	}
	return info
}

type ConnectionActivationStateFlags uint32

func (f ConnectionActivationStateFlags) HasNone() bool {
	return f == 0
}

func (f ConnectionActivationStateFlags) IsController() bool {
	return f&0x1 > 0
}

func (f ConnectionActivationStateFlags) IsPort() bool {
	return f&0x2 > 0
}

func (f ConnectionActivationStateFlags) Layer2Ready() bool {
	return f&0x4 > 0
}

func (f ConnectionActivationStateFlags) IPv4Ready() bool {
	return f&0x8 > 0
}

func (f ConnectionActivationStateFlags) IPv6Ready() bool {
	return f&0x10 > 0
}

func (f ConnectionActivationStateFlags) ControllerHasPorts() bool {
	return f&0x20 > 0
}

func (f ConnectionActivationStateFlags) LifetimeBoundToProfileVisibility() bool {
	return f&0x40 > 0
}

func (f ConnectionActivationStateFlags) External() bool {
	return f&0x80 > 0
}

func dumpActiveConn(conno dbus.BusObject, bus *dbus.Conn) (conn ActiveConn, err error) {
	const connName = nmName + ".Connection.Active"

	if err = conno.StoreProperty(connName+".Id", &conn.ID); err != nil {
		return ActiveConn{}, errors.Wrap(err, "couldn't query for connection ID")
	}

	var rawUUID string
	if err = conno.StoreProperty(connName+".Uuid", &rawUUID); err != nil {
		return ActiveConn{}, errors.Wrap(err, "couldn't query for connection UUID")
	}
	if conn.UUID, err = uuid.Parse(rawUUID); err != nil {
		return ActiveConn{}, errors.Wrapf(err, "couldn't parse connection UUID %s", rawUUID)
	}

	if err = conno.StoreProperty(connName+".Type", &conn.Type); err != nil {
		return ActiveConn{}, errors.Wrap(err, "couldn't query for connection type")
	}

	var rawEnum uint32
	if err = conno.StoreProperty(connName+".State", &rawEnum); err != nil {
		return ActiveConn{}, errors.Wrap(err, "couldn't query for connection state")
	}
	conn.State = ActiveConnState(rawEnum)

	var rawFlags uint32
	if err = conno.StoreProperty(connName+".StateFlags", &rawFlags); err != nil {
		return ActiveConn{}, errors.Wrap(
			err, "couldn't query for connection activation state flags",
		)
	}
	conn.StateFlags = ConnectionActivationStateFlags(rawFlags)

	if err = conno.StoreProperty(connName+".Default", &conn.IsIPv4Default); err != nil {
		return ActiveConn{}, errors.Wrap(
			err, "couldn't query for ownership of default IPv4 route",
		)
	}
	if err = conno.StoreProperty(connName+".Default6", &conn.IsIPv6Default); err != nil {
		return ActiveConn{}, errors.Wrap(
			err, "couldn't query for ownership of default IPv6 route",
		)
	}
	if err = conno.StoreProperty(connName+".Vpn", &conn.IsVPN); err != nil {
		return ActiveConn{}, errors.Wrap(err, "couldn't query for VPN")
	}

	if conn.DeviceInterfaces, err = dumpActiveConnDevices(conno, bus); err != nil {
		return ActiveConn{}, errors.Wrap(err, "couldn't query for interface names of devices")
	}

	return conn, nil
}

func dumpActiveConnDevices(
	conno dbus.BusObject, bus *dbus.Conn,
) (interfaces []string, err error) {
	const connName = nmName + ".Connection.Active"

	var devPaths []dbus.ObjectPath
	if err = conno.StoreProperty(connName+".Devices", &devPaths); err != nil {
		return nil, errors.Wrap(err, "couldn't query for devices")
	}

	for _, devPath := range devPaths {
		dev := Device{}
		devo := bus.Object(nmName, devPath)
		if err = devo.StoreProperty(nmName+".Device.Interface", &dev.ControlInterface); err != nil {
			return nil, errors.Wrapf(err, "couldn't query for control interface of %s", devo)
		}
		if err = devo.StoreProperty(nmName+".Device.IpInterface", &dev.IpInterface); err != nil {
			return nil, errors.Wrapf(err, "couldn't query for data interface of %s", devo)
		}
		interfaces = append(interfaces, cmp.Or(dev.IpInterface, dev.ControlInterface))
	}
	slices.Sort(interfaces)
	return interfaces, nil
}

func ListActiveConns(
	ctx context.Context,
) (conns map[string]ActiveConn, err error) { // keyed by UUID strings
	nm, bus, err := getNetworkManager(ctx)
	if err != nil {
		return nil, err
	}

	var connPaths []dbus.ObjectPath
	if err = nm.StoreProperty(nmName+".ActiveConnections", &connPaths); err != nil {
		return nil, errors.Wrap(err, "couldn't query for active connections")
	}

	conns = make(map[string]ActiveConn)
	for _, connPath := range connPaths {
		conn, err := dumpActiveConn(bus.Object(nmName, connPath), bus)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't dump active connection %s", connPath)
		}
		conns[conn.UUID.String()] = conn
	}

	return conns, nil
}
