package networkmanager

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// ActiveConn

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
	StableID  string
	Timestamp time.Time
	Type      string // TODO: turn this into a string enum
	UUID      uuid.UUID
	// WaitActivationDelay time.Duration
	// WaitDeviceTimeout   time.Duration
	Zone string
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
	Addresses    []IPAddress
	Method       string // TODO: change this to a string enum // TODO
	NeverDefault bool   // TODO
}

type ConnProfileSettingsIPv6 struct {
	Addresses   []IPAddress
	AddrGenMode int32  // TODO: change this to an int32 enum // TODO
	Method      string // TODO: change this to a string enum // TODO
}

func GetConnProfileByUUID(ctx context.Context, uid uuid.UUID) (conn ConnProfile, err error) {
	conno, err := findConnProfileByUUID(ctx, uid)
	if err != nil {
		return ConnProfile{}, errors.Wrapf(err, "couldn't find connection profile with uuid %s", uid)
	}
	if conn, err = dumpConnProfile(ctx, conno); err != nil {
		return ConnProfile{}, errors.Wrapf(err, "couldn't dump connection profile with uuid %s", uid)
	}
	return conn, nil
}

func findConnProfileByUUID(ctx context.Context, uid uuid.UUID) (dev dbus.BusObject, err error) {
	nm, bus, err := getNetworkManagerSettings(ctx)
	if err != nil {
		return nil, err
	}

	var connPath dbus.ObjectPath
	if err = nm.CallWithContext(
		ctx, nmName+".Settings.GetConnectionByUuid", 0, uid.String(),
	).Store(&connPath); err != nil {
		return nil, errors.Wrapf(err, "couldn't query for connection profile with uuid %s", uid)
	}

	return bus.Object(nmName, connPath), nil
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
) (s ConnProfileSettings, err error) {
	var rawSettings map[string]map[string]dbus.Variant
	if err = conno.CallWithContext(
		ctx, nmName+".Settings.Connection.GetSettings", 0,
	).Store(&rawSettings); err != nil {
		return s, errors.Wrap(err, "couldn't get settings")
	}

	if s.Connection, err = dumpConnProfileSettingsConnection(
		rawSettings["connection"],
	); err != nil {
		return s, errors.Wrap(err, "couldn't parse 'connection' section")
	}

	if s.IPv4, err = dumpConnProfileSettingsIPv4(
		rawSettings["ipv4"],
	); err != nil {
		return s, errors.Wrap(err, "couldn't parse 'ipv4' section")
	}
	if s.IPv6, err = dumpConnProfileSettingsIPv6(
		rawSettings["ipv6"],
	); err != nil {
		return s, errors.Wrap(err, "couldn't parse 'ipv6' section")
	}

	return s, nil
}

func dumpConnProfileSettingsConnection(
	rawSettings map[string]dbus.Variant,
) (s ConnProfileSettingsConnection, err error) {
	if s.AuthRetries, err = ensureVar[int32](
		rawSettings, "auth-retries", "auth retries", false, -1,
	); err != nil {
		return s, err
	}
	if s.ID, err = ensureVar(rawSettings, "id", "ID", true, ""); err != nil {
		return s, err
	}

	if s.InterfaceName, err = ensureVar(rawSettings, "interface-name", "", true, ""); err != nil {
		return s, err
	}
	if s.Type, err = ensureVar(rawSettings, "type", "", true, ""); err != nil {
		return s, err
	}
	if s.StableID, err = ensureVar(rawSettings, "stable-id", "stable ID", false, ""); err != nil {
		return s, err
	}

	rawUint, err := ensureVar[uint64](rawSettings, "timestamp", "", false, 0)
	if err != nil {
		return s, err
	}
	if rawUint > math.MaxInt64 {
		// TODO: log a warning!
	} else {
		s.Timestamp = time.Unix(int64(rawUint), 0)
	}

	rawUUID, err := ensureVar(rawSettings, "uuid", "UUID", true, "")
	if err != nil {
		return s, err
	}
	if s.UUID, err = uuid.Parse(rawUUID); err != nil {
		return s, errors.Wrapf(err, "couldn't parse UUID %s", rawUUID)
	}

	if s.Zone, err = ensureVar(rawSettings, "zone", "", false, ""); err != nil {
		return s, err
	}

	if s, err = dumpConnProfileSettingsConnectionAutoconnect(rawSettings, s); err != nil {
		return s, err
	}

	return s, nil
}

func ensureVar[T any](
	rawSettings map[string]dbus.Variant, key, errorName string, required bool, defaultResult T,
) (result T, err error) {
	if errorName == "" {
		errorName = strings.ReplaceAll(key, "-", " ")
	}

	variant, ok := rawSettings[key]
	if !ok {
		if !required {
			return defaultResult, nil
		}
		return result, errors.Errorf("no %s", errorName)
	}
	if err = variant.Store(&result); err != nil {
		return result, errors.Wrapf(err, "%s has unexpected type %T", errorName, variant.Value())
	}
	return result, nil
}

func dumpConnProfileSettingsConnectionAutoconnect(
	rawSettings map[string]dbus.Variant, s ConnProfileSettingsConnection,
) (ConnProfileSettingsConnection, error) {
	var err error

	if s.Autoconnect, err = ensureVar(rawSettings, "autoconnect", "", false, true); err != nil {
		return s, err
	}
	if s.AutoconnectPriority, err = ensureVar[int32](
		rawSettings, "autoconnect-priority", "", false, 0,
	); err != nil {
		return s, err
	}
	if s.AutoconnectRetries, err = ensureVar[int32](
		rawSettings, "autoconnect-retries", "", false, -1,
	); err != nil {
		return s, err
	}

	return s, nil
}

func dumpConnProfileSettingsIPv4(
	rawSettings map[string]dbus.Variant,
) (s ConnProfileSettingsIPv4, err error) {
	rawObjs, err := ensureVar[[]map[string]dbus.Variant](rawSettings, "address-data", "", false, nil)
	if err != nil {
		return s, nil
	}
	for _, obj := range rawObjs {
		address, err := parseIPAddress(obj)
		if err != nil {
			return s, errors.Wrapf(err, "couldn't parse IP address %+v", address)
		}
		s.Addresses = append(s.Addresses, address)
	}

	return s, nil
}

func dumpConnProfileSettingsIPv6(
	rawSettings map[string]dbus.Variant,
) (s ConnProfileSettingsIPv6, err error) {
	rawObjs, err := ensureVar[[]map[string]dbus.Variant](rawSettings, "address-data", "", false, nil)
	if err != nil {
		return s, nil
	}
	for _, obj := range rawObjs {
		address, err := parseIPAddress(obj)
		if err != nil {
			return s, errors.Wrapf(err, "couldn't parse IP address %+v", address)
		}
		s.Addresses = append(s.Addresses, address)
	}

	return s, nil
}

func ListConnProfiles(ctx context.Context) (conns []ConnProfile, err error) {
	nm, bus, err := getNetworkManagerSettings(ctx)
	if err != nil {
		return nil, err
	}

	var connPaths []dbus.ObjectPath
	if err = nm.CallWithContext(
		ctx, nmName+".Settings.ListConnections", 0,
	).Store(&connPaths); err != nil {
		return nil, errors.Wrap(err, "couldn't list connection profiles")
	}

	for _, connPath := range connPaths {
		conn, err := dumpConnProfile(ctx, bus.Object(nmName, connPath))
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't dump connection profile %s", connPath)
		}
		conns = append(conns, conn)
	}

	slices.SortFunc(conns, func(a, b ConnProfile) int {
		return cmp.Compare(a.Settings.Connection.ID, b.Settings.Connection.ID)
	})

	return conns, nil
}

func ReloadConnProfiles(ctx context.Context) error {
	nm, _, err := getNetworkManagerSettings(ctx)
	if err != nil {
		return err
	}

	var status bool
	if err = nm.CallWithContext(
		ctx, nmName+".Settings.ReloadConnections", 0,
	).Store(&status); err != nil {
		return errors.Wrap(err, "couldn't reload connection profiles")
	}
	if !status {
		return errors.New("reload of connection profiles encountered an unexpected failure")
	}

	return nil
}

func ActivateConnProfile(ctx context.Context, uid uuid.UUID) error {
	nm, _, err := getNetworkManager(ctx)
	if err != nil {
		return err
	}
	conno, err := findConnProfileByUUID(ctx, uid)
	if err != nil {
		return errors.Wrap(err, "couldn't find connection profile to activate")
	}

	var activeConn dbus.ObjectPath
	if err = nm.CallWithContext(
		ctx, nmName+".ActivateConnection", 0, conno.Path(), dbus.ObjectPath("/"), dbus.ObjectPath("/"),
	).Store(&activeConn); err != nil {
		return errors.Wrapf(err, "couldn't activate connection profile with UUID %s", uid)
	}

	return nil
}
