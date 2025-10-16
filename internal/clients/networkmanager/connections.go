package networkmanager

import (
	"cmp"
	"context"
	"fmt"
	"net/netip"
	"slices"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// ActiveConn

type ActiveConn struct {
	ID               string
	UUID             string
	Type             string
	DeviceInterfaces []string
	State            ActiveConnState
	StateFlags       ConnectionActivationStateFlags
	IsIPv4Default    bool
	IsIPv6Default    bool
	IsVPN            bool
}

func (c ActiveConn) HasData() bool {
	return c.ID != "" || c.UUID != "" || c.Type != ""
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
	if err = conno.StoreProperty(connName+".Uuid", &conn.UUID); err != nil {
		return ActiveConn{}, errors.Wrap(err, "couldn't query for connection UUID")
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

	// var rawString string
	// if err = conno.StoreProperty(nmName+".Connection.Active.Connection", &rawString); err != nil {
	// 	return ActiveConn{}, errors.Wrap(err, "couldn't query for connection path")
	// }
	// fmt.Println("Connection", rawString)

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

// ConnProfile

type ConnProfile struct {
	Unsaved  bool
	Flags    ConnProfileFlags
	Filename string
	Settings ConnProfileSettings
}

type ConnProfileFlags uint32

func (f ConnProfileFlags) HasNone() bool {
	return f == 0
}

func (f ConnProfileFlags) Unsaved() bool {
	return f&0x1 > 0
}

func (f ConnProfileFlags) GeneratedByNM() bool {
	return f&0x2 > 0
}

func (f ConnProfileFlags) Volatile() bool {
	return f&0x4 > 0
}

func (f ConnProfileFlags) External() bool {
	return f&0x8 > 0
}

type ConnProfileSettings struct {
	Connection ConnProfileSettingsConnection
	Wifi       ConnProfileSettings80211Wireless
	WifiSec    ConnProfileSettings80211WirelessSecurity
	// WifiAuthn  ConnProfileSettings8021x
	// Ethernet ConnProfileSettings8023Ethernet
	IPv4 ConnProfileSettingsIPv4
	IPv6 ConnProfileSettingsIPv6
}

type ConnProfileSettingsConnection struct {
	AuthRetries         int32
	Autoconnect         bool
	AutoconnectPriority int32
	AutoconnectRetries  int32
	ID                  string
	InterfaceName       string
	// IPPingAddresses     []netip.Addr
	// IPPingTimeout       time.Duration
	StableID            string
	Timestamp           time.Time
	Type                string
	UUID                uuid.UUID
	WaitActivationDelay time.Duration
	WaitDeviceTimeout   time.Duration
	Zone                string
}

type ConnProfileSettings80211Wireless struct {
	// APIsolation int32 // TODO: change this to an int32 enum
	// AssignedMACAddress  string
	Band string // TODO: change this to a string enum
	// BSSID               []string
	Channel uint32
	// ChannelWidth int32
	Hidden bool
	// MACAddress          []byte
	// MACAddressBlacklist []string
	// MACAddressDenylist  []string
	Mode string // TODO: change this to a string enum
	// MTU                 uint32
	// Powersave uint32 // TODO: change this to a uint32 enum
	// SeenBSSIDs          []string
	SSID string
}

type ConnProfileSettings80211WirelessSecurity struct {
	AuthAlg  string   // TODO: change this to a string enum
	Group    []string // TODO: change this to an array of string enums
	KeyMgmt  string   // TODO: change this to a string enum
	Pairwise []string // TODO: change this to an array of string enums
	Proto    []string // TODO: change this to an array of string enums
	PSK      string
	PSKFlags uint32 // TODO: change this to a uint32 flags type
}

type ConnProfileSettingsIPv4 struct {
	AddressData  []netip.Prefix
	Method       string // TODO: change this to a string enum
	NeverDefault bool
}

type ConnProfileSettingsIPv6 struct {
	AddrGenMode int32  // TODO: change this to an int32 enum
	Method      string // TODO: change this to a string enum
}

func dumpConnProfile(ctx context.Context, conno dbus.BusObject) (conn ConnProfile, err error) {
	const connName = nmName + ".Settings.Connection"

	if err = conno.StoreProperty(connName+".Unsaved", &conn.Unsaved); err != nil {
		return ConnProfile{}, errors.Wrap(err, "couldn't query for unsaved")
	}

	var rawFlags uint32
	if err = conno.StoreProperty(connName+".Flags", &rawFlags); err != nil {
		return ConnProfile{}, errors.Wrap(err, "couldn't query for flags")
	}
	conn.Flags = ConnProfileFlags(rawFlags)

	if err = conno.StoreProperty(connName+".Filename", &conn.Filename); err != nil {
		return ConnProfile{}, errors.Wrap(err, "couldn't query for filename")
	}

	if conn.Settings, err = dumpConnProfileSettings(ctx, conno); err != nil {
		return ConnProfile{}, errors.Wrap(err, "couldn't query for connection settings")
	}

	return conn, nil
}

func dumpConnProfileSettings(
	ctx context.Context, conno dbus.BusObject,
) (settings ConnProfileSettings, err error) {
	var rawSettings map[string]map[string]dbus.Variant
	if err = conno.CallWithContext(
		ctx, nmName+".Settings.Connection.GetSettings", 0,
	).Store(&rawSettings); err != nil {
		return ConnProfileSettings{}, errors.Wrap(err, "couldn't get settings")
	}
	// fmt.Printf("%+v\n", rawSettings)

	if settings.Connection, err = dumpConnProfileSettingsConnection(
		rawSettings["connection"],
	); err != nil {
		return ConnProfileSettings{}, errors.Wrap(err, "couldn't query for connection ID")
	}

	return settings, nil
}

func dumpConnProfileSettingsConnection(
	rawSettings map[string]dbus.Variant,
) (s ConnProfileSettingsConnection, err error) {
	if err = rawSettings["id"].Store(&s.ID); err != nil {
		return ConnProfileSettingsConnection{}, errors.Wrap(err, "couldn't query for connection ID")
	}
	return s, nil
}
