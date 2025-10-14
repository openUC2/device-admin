package networkmanager

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/godbus/dbus/v5"
	"github.com/pkg/errors"
)

type DeviceCaps uint32

func (c DeviceCaps) HasNone() bool {
	return c == 0
}

func (c DeviceCaps) SupportedByNM() bool {
	return c&0x1 > 0
}

func (c DeviceCaps) CarrierDetectable() bool {
	return c&0x2 > 0
}

func (c DeviceCaps) IsSoftware() bool {
	return c&0x4 > 0
}

func (c DeviceCaps) SupportsSRIOV() bool {
	return c&0x8 > 0
}

type EnumInfo struct {
	Short   string
	Details string
	Level   string
}

type DeviceState uint32

var deviceStateInfo = map[DeviceState]EnumInfo{
	0: {
		Short: "unknown",
		Level: "error",
	},
	10: {
		Short:   "unmanaged",
		Details: "recognized, but not managed",
		Level:   "info",
	},
	20: {
		Short:   "unavailable",
		Details: "managed, but not available for use",
		Level:   "info",
	},
	30: {
		Short:   "disconnected",
		Details: "can be activated, but currently idle",
		Level:   "info",
	},
	40: {
		Short:   "prepare",
		Details: "preparing connection",
		Level:   "info",
	},
	50: {
		Short:   "config",
		Details: "connecting",
		Level:   "info",
	},
	60: {
		Short:   "need auth",
		Details: "more information needed to connect",
		Level:   "warning",
	},
	70: {
		Short:   "IP config",
		Details: "requesting IP addresses and routing information",
		Level:   "info",
	},
	80: {
		Short:   "IP check",
		Details: "checking whether more action is needed",
		Level:   "info",
	},
	90: {
		Short:   "secondaries",
		Details: "waiting for activation of secondary connection",
		Level:   "info",
	},
	100: {
		Short:   "activated",
		Details: "has a network connection",
		Level:   "success",
	},
	110: {
		Short:   "deactivating",
		Details: "cleaning up resources to disconnect from current connection",
		Level:   "info",
	},
	120: {
		Short:   "failed",
		Details: "failed to connect to requested network, cleaning up",
		Level:   "error",
	},
}

func (s DeviceState) Info() EnumInfo {
	info, ok := deviceStateInfo[s]
	if !ok {
		return EnumInfo{
			Short:   "unknown",
			Details: fmt.Sprintf("state (%d) was reported but could not be determined", s),
			Level:   "error",
		}
	}
	return info
}

type DeviceStateReason uint32

var deviceStateReasons = map[DeviceStateReason]EnumInfo{
	0: {},
	1: {
		Short: "unknown error",
	},
	2: {
		Short: "now managed",
	},
	3: {
		Short: "now unmanaged",
	},
	4: {
		Short:   "config failed",
		Details: "device couldn't be readied for configuration",
	},
	5: {
		Short:   "IP config unavailable",
		Details: "IP configuration couldn't be reserved",
	},
	6: {
		Short:   "IP config expired",
		Details: "IP configuration no longer valid",
	},
	7: {
		Short:   "no secrets",
		Details: "required secrets weren't provided",
	},
	8: {
		Short:   "supplicant disconnect",
		Details: "802.1x supplicant disconnected",
	},
	9: {
		Short:   "supplicant config failed",
		Details: "802.1x supplicant configuration failed",
	},
	10: {
		Short:   "supplicant failed",
		Details: "802.1x supplicant failed",
	},
	11: {
		Short:   "supplicant timeout",
		Details: "802.1x supplicant took too long to authenticate",
	},
	12: {
		Short:   "PPP start failed",
		Details: "PPP service failed to start",
	},
	13: {
		Short:   "PPP disconnect",
		Details: "PPP service disconnected",
	},
	14: {
		Short:   "PPP failed",
		Details: "PPP service failed",
	},
	15: {
		Short:   "DHCP start failed",
		Details: "DHCP client failed to start",
	},
	16: {
		Short:   "DHCP error",
		Details: "DHCP client error",
	},
	17: {
		Short:   "DHCP failed",
		Details: "DHCP client failed",
	},
	18: {
		Short:   "shared start failed",
		Details: "shared connection service failed to start",
	},
	19: {
		Short:   "shared failed",
		Details: "shared connection service failed",
	},
	20: {
		Short:   "AutoIP start failed",
		Details: "AutoIP service failed to start",
	},
	21: {
		Short:   "AutoIP error",
		Details: "AutoIP service error",
	},
	22: {
		Short:   "AutoIP failed",
		Details: "AutoIP service failed",
	},
	23: {
		Short:   "modem busy",
		Details: "line is busy",
	},
	24: {
		Short: "modem no dial tone",
	},
	25: {
		Short:   "modem no carrier",
		Details: "carrier couldn't be established",
	},
	26: {
		Short:   "modem dial timeout",
		Details: "dialing request timed out",
	},
	27: {
		Short:   "modem dial failed",
		Details: "dialing attempt failed",
	},
	28: {
		Short:   "modem init failed",
		Details: "modem initialization failed",
	},
	29: {
		Short:   "GSM APN failed",
		Details: "failed to select specified APN",
	},
	30: {
		Short:   "GSM registration not searching",
		Details: "not searching for networks",
	},
	31: {
		Short:   "GSM registration denied",
		Details: "network registration denied",
	},
	32: {
		Short:   "GSM registration timeout",
		Details: "network registration timed out",
	},
	33: {
		Short:   "GSM registration failed",
		Details: "failed to register with requested network",
	},
	34: {
		Short: "GSM PIN check failed",
	},
	35: {
		Short: "firmware missing",
	},
	36: {
		Short:   "removed",
		Details: "device was removed",
	},
	37: {
		Short:   "sleeping",
		Details: "NetworkManager went to sleep",
	},
	38: {
		Short:   "connection removed",
		Details: "active connection disappeared",
	},
	39: {
		Short:   "user requested",
		Details: "disconnected by user or client",
	},
	40: {
		Short:   "carrier",
		Details: "carrier/link changed",
	},
	41: {
		Short:   "connection assumed",
		Details: "existing connection was assumed",
	},
	42: {
		Short:   "supplicant available",
		Details: "supplicant is now available",
	},
	43: {
		Short: "modem not found",
	},
	44: {
		Short:   "BT failed",
		Details: "Bluetooth connection failed or timed out",
	},
	45: {
		Short:   "GSM SIM not inserted",
		Details: "GSM modem's SIM card not inserted",
	},
	46: {
		Short: "GSM PIN required",
	},
	47: {
		Short: "GSM PUK required",
	},
	48: {
		Short: "GSM SIM wrong",
	},
	49: {
		Short:   "InfiniBand mode",
		Details: "InfiniBand device does not support connected mode",
	},
	50: {
		Short:   "dependency failed",
		Details: "dependency of the connection failed",
	},
	51: {
		Short:   "BR2684 failed",
		Details: "problem with the RFC 2684 Ethernet over ADSL bridge",
	},
	52: {
		Short:   "ModemManager unavailable",
		Details: "not running",
	},
	53: {
		Short:   "SSID not found",
		Details: "Wi-Fi network couldn't be found",
	},
	54: {
		Short:   "secondary connection failed",
		Details: "secondary connection of base connection failed",
	},
	55: {
		Short:   "DCB FCoE failed",
		Details: "DCB or FCoE setup failed",
	},
	56: {
		Short: "teamd control failed",
	},
	57: {
		Short:   "modem failed",
		Details: "modem failed or no longer available",
	},
	58: {
		Short:   "modem available",
		Details: "modem now ready and available",
	},
	59: {
		Short: "SIM PIN incorrect",
	},
	60: {
		Short:   "new activation",
		Details: "new connection activation was enqueued",
	},
	61: {
		Short:   "parent changed",
		Details: "device's parent changed",
	},
	62: {
		Short:   "parent managed changed",
		Details: "device's parent's management changed",
	},
	63: {
		Short:   "OVSDB failed",
		Details: "problem communicating with Open vSwitch database",
	},
	64: {
		Short:   "IP address duplicate",
		Details: "duplicate IP address detected",
	},
	65: {
		Short:   "IP method unsupported",
		Details: "selected IP method not supported",
	},
	66: {
		Short:   "SR-IOV configuration failed",
		Details: "configuration of SR-IOV parameters failed",
	},
	67: {
		Short:   "peer not found",
		Details: "Wi-Fi P2P peer not found",
	},
	68: {
		Short:   "device handler failed",
		Details: "device handler dispatcher returned error",
	},
	69: {
		Short:   "unmanaged by default",
		Details: "because device type is unmanaged by default",
	},
	70: {
		Short:   "unmanaged external down",
		Details: "it's an external device and is unconfigured (down or no addresses)",
	},
	71: {
		Short:   "unmanaged link not init",
		Details: "the link is not initialized by udev",
	},
	72: {
		Short:   "unmanaged quitting",
		Details: "NetworkManager is quitting",
	},
	73: {
		Short:   "unmanaged sleeping",
		Details: "networking id siabled or the system is suspended",
	},
	74: {
		Short:   "unmanaged user conf",
		Details: "unmanaged by user decision in NetworkManager.conf ('unmanaged' in a device section)",
	},
	75: {
		Short:   "unmanaged user explicit",
		Details: "unmanaged by explicit user decision",
	},
	76: {
		Short:   "unmanaged user settings",
		Details: "unmanaged by user decision via settings plugin",
	},
	77: {
		Short:   "unmanaged user udev",
		Details: "unmanaged via udev rule",
	},
}

func (r DeviceStateReason) Info() EnumInfo {
	info, ok := deviceStateReasons[r]
	if !ok {
		return EnumInfo{
			Short:   "unknown",
			Details: fmt.Sprintf("reason (%d) was reported but could not be determined", r),
			Level:   "error",
		}
	}
	return info
}

type DeviceType uint32

var deviceTypeInfo = map[DeviceType]EnumInfo{
	0: {
		Short:   "unknown",
		Details: "unknown device",
		Level:   "warning",
	},
	14: {
		Short:   "generic",
		Details: "generic support for unrecognized device types",
		Level:   "warning",
	},
	1: {
		Short:   "ethernet",
		Details: "wired ethernet device",
		Level:   "success",
	},
	2: {
		Short:   "wifi",
		Details: "802.11 Wi-Fi device",
		Level:   "success",
	},
	3: {
		Short:   "unused 1",
		Details: "not used",
		Level:   "error",
	},
	4: {
		Short:   "unused 2",
		Details: "not used",
		Level:   "error",
	},
	5: {
		Short:   "bluetooth",
		Details: "Bluetooth device supporting PAN or DUN access protocols",
		Level:   "info",
	},
	6: {
		Short:   "olpc mesh",
		Details: "OLPC XO mesh networking device",
		Level:   "info",
	},
	7: {
		Short:   "wimax",
		Details: "802.16e Mobile WiMAX broadband device",
		Level:   "info",
	},
	8: {
		Short:   "modem",
		Details: "modem supporting analog telephone, CDMA/EVDO, GSM/UMTS, or LTE protocols",
		Level:   "info",
	},
	9: {
		Short:   "infiniband",
		Details: "IP-over-InfiniBand device",
		Level:   "info",
	},
	10: {
		Short:   "bond",
		Details: "bond controller interface",
		Level:   "info",
	},
	11: {
		Short:   "vlan",
		Details: "802.1Q VLAN interfface",
		Level:   "info",
	},
	12: {
		Short:   "adsl",
		Details: "ADSL modem",
		Level:   "info",
	},
	13: {
		Short:   "bridge",
		Details: "bridge controller interface",
		Level:   "info",
	},
	15: {
		Short:   "team",
		Details: "802.1Q VLAN interface",
		Level:   "info",
	},
	16: {
		Short:   "tun",
		Details: "TUN or TAP interface",
		Level:   "info",
	},
	17: {
		Short:   "ip tunnel",
		Details: "IP tunnel interface",
		Level:   "info",
	},
	18: {
		Short:   "macvlan",
		Details: "MACVLAN interface",
		Level:   "info",
	},
	19: {
		Short:   "vxlan",
		Details: "VXLAN interface",
		Level:   "info",
	},
	20: {
		Short:   "veth",
		Details: "VETH interface",
		Level:   "info",
	},
	21: {
		Short:   "macsec",
		Details: "MACsec interface",
		Level:   "info",
	},
	22: {
		Short:   "dummy",
		Details: "dummy interface",
		Level:   "info",
	},
	23: {
		Short:   "ppp",
		Details: "PPP interface",
		Level:   "info",
	},
	24: {
		Short:   "ovs",
		Details: "Open vSwitch interface",
		Level:   "info",
	},
	25: {
		Short:   "ovs port",
		Details: "Open vSwitch port",
		Level:   "info",
	},
	26: {
		Short:   "ovs bridge",
		Details: "Open vSwitch bridge",
		Level:   "info",
	},
	27: {
		Short:   "wpan",
		Details: "IEEE 802.15.4 (WPAN) MAC Layer device",
		Level:   "info",
	},
	28: {
		Short:   "6lowpan",
		Details: "6LoWPAN interface",
		Level:   "info",
	},
	29: {
		Short:   "wireguard",
		Details: "WireGuard interface",
		Level:   "info",
	},
	30: {
		Short:   "wifi p2p",
		Details: "802.11 Wi-Fi P2P device",
		Level:   "info",
	},
	31: {
		Short:   "vrf",
		Details: "VRF (Virtual Routing and Forwarding) interface",
		Level:   "info",
	},
	32: {
		Short:   "loopback",
		Details: "loopback interface",
		Level:   "info",
	},
	33: {
		Short:   "hsr",
		Details: "HSR/PRP device",
		Level:   "info",
	},
	34: {
		Short:   "ipvlan",
		Details: "IPVLAN device",
		Level:   "info",
	},
}

func (s DeviceType) Info() EnumInfo {
	info, ok := deviceTypeInfo[s]
	if !ok {
		return EnumInfo{
			Short:   "unknown",
			Details: fmt.Sprintf("type (%d) was reported but could not be determined", s),
			Level:   "error",
		}
	}
	return info
}

type DeviceConnectivityState uint32

var deviceConnectivityStateInfo = map[DeviceConnectivityState]EnumInfo{
	0: {
		Short:   "unknown",
		Details: "connectivity checks disabled or not run yet; internet might be available",
		Level:   "info",
	},
	1: {
		Short:   "none",
		Details: "network connection unavailable, no default route to the internet",
		Level:   "info",
	},
	2: {
		Short:   "captive portal",
		Details: "blocked by a captive portal",
		Level:   "info",
	},
	3: {
		Short:   "limited",
		Details: "connected to network without full internet access",
		Level:   "info",
	},
	4: {
		Short:   "full",
		Details: "connected to network with full internet access",
		Level:   "success",
	},
}

func (s DeviceConnectivityState) Info() EnumInfo {
	info, ok := deviceConnectivityStateInfo[s]
	if !ok {
		return EnumInfo{
			Short:   "unknown",
			Details: fmt.Sprintf("state (%d) was reported but could not be determined", s),
			Level:   "error",
		}
	}
	return info
}

type DeviceInterfaceFlags uint32

func (c DeviceInterfaceFlags) HasNone() bool {
	return c == 0
}

func (c DeviceInterfaceFlags) IsUp() bool {
	return c&0x1 > 0
}

func (c DeviceInterfaceFlags) IsLowerUp() bool {
	return c&0x2 > 0
}

func (c DeviceInterfaceFlags) IsPromiscuous() bool {
	return c&0x4 > 0
}

func (c DeviceInterfaceFlags) HasCarrier() bool {
	return c&0x10000 > 0
}

func (c DeviceInterfaceFlags) LLDPClient() bool {
	return c&0x20000 > 0
}

type Device struct {
	ControlInterface string
	IpInterface      string
	Driver           string
	DriverVersion    string
	FirmwareVersion  string
	Caps             DeviceCaps
	State            DeviceState
	StateReason      DeviceStateReason
	// TODO: describe the active connection
	// TODO: describe IPv4 & IPv6 configs; these have IP addresses
	// TODO: describe IPv4 & IPv6 DHCP configs
	Managed         bool
	Autoconnect     bool
	FirmwareMissing bool
	NMPluginMissing bool
	Type            DeviceType
	// TODO: describe available connections
	IPv4Connectivity DeviceConnectivityState
	IPv6Connectivity DeviceConnectivityState
	InterfaceFlags   DeviceInterfaceFlags
	HardwareAddress  string
}

func GetDevices(ctx context.Context) (devs []Device, err error) {
	nm, bus, err := getNetworkManager(ctx)
	if err != nil {
		return nil, err
	}
	devPaths := make([]dbus.ObjectPath, 0)
	if err = nm.CallWithContext(ctx, nmName+".GetDevices", 0).Store(&devPaths); err != nil {
		return nil, errors.Wrap(err, "couldn't query for devices")
	}
	for _, devPath := range devPaths {
		dev, err := dumpDevice(bus.Object(nmName, devPath))
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't dump device %s", devPath)
		}
		devs = append(devs, dev)
	}
	slices.SortFunc(devs, func(a, b Device) int {
		return cmp.Compare(
			cmp.Or(a.IpInterface, a.ControlInterface),
			cmp.Or(b.IpInterface, b.ControlInterface),
		)
	})
	return devs, nil
}

func findDevice(
	ctx context.Context, ipInterface string,
) (dev dbus.BusObject, bus *dbus.Conn, err error) {
	nm, bus, err := getNetworkManager(ctx)
	if err != nil {
		return nil, nil, err
	}

	var devPath dbus.ObjectPath
	if err = nm.CallWithContext(
		ctx, nmName+".GetDeviceByIpIface", 0, ipInterface,
	).Store(&devPath); err != nil {
		return nil, nil, errors.Wrapf(err, "couldn't query for %s device", ipInterface)
	}

	return bus.Object(nmName, devPath), bus, nil
}

func dumpDevice(devo dbus.BusObject) (dev Device, err error) {
	if err = devo.StoreProperty(nmName+".Device.Interface", &dev.ControlInterface); err != nil {
		return Device{}, errors.Wrap(err, "couldn't query for control interface")
	}
	if err = devo.StoreProperty(nmName+".Device.IpInterface", &dev.IpInterface); err != nil {
		return Device{}, errors.Wrap(err, "couldn't query for data interface")
	}
	if err = devo.StoreProperty(nmName+".Device.Driver", &dev.Driver); err != nil {
		return Device{}, errors.Wrap(err, "couldn't query for driver")
	}
	if err = devo.StoreProperty(nmName+".Device.DriverVersion", &dev.DriverVersion); err != nil {
		return Device{}, errors.Wrap(err, "couldn't query for driver version")
	}
	if err = devo.StoreProperty(nmName+".Device.FirmwareVersion", &dev.FirmwareVersion); err != nil {
		return Device{}, errors.Wrap(err, "couldn't query for firmware version")
	}

	var rawEnum uint32
	if err = devo.StoreProperty(nmName+".Device.DeviceType", &rawEnum); err != nil {
		return Device{}, errors.Wrap(err, "couldn't query for device type")
	}
	dev.Type = DeviceType(rawEnum)

	if err = devo.StoreProperty(nmName+".Device.HwAddress", &dev.HardwareAddress); err != nil {
		return Device{}, errors.Wrap(err, "couldn't query for hardware address")
	}

	if dev, err = dumpDeviceStateInfo(devo, dev); err != nil {
		return Device{}, err
	}
	if dev, err = dumpDeviceFlags(devo, dev); err != nil {
		return Device{}, err
	}

	return dev, nil
}

func dumpDeviceStateInfo(devo dbus.BusObject, dev Device) (Device, error) {
	var err error

	rawStateAndReason := make([]any, 2) //nolint:mnd // this is always a 2-tuple
	if err = devo.StoreProperty(nmName+".Device.StateReason", &rawStateAndReason); err != nil {
		return Device{}, errors.Wrap(err, "couldn't query for state")
	}
	var ok bool
	rawState, ok := rawStateAndReason[0].(uint32)
	if !ok {
		return Device{}, errors.Wrap(err, "couldn't parse device state")
	}
	dev.State = DeviceState(rawState)
	rawStateReason, ok := rawStateAndReason[1].(uint32)
	if !ok {
		return Device{}, errors.Wrap(err, "couldn't parse device state reason")
	}
	dev.StateReason = DeviceStateReason(rawStateReason)

	var rawEnum uint32
	if err = devo.StoreProperty(nmName+".Device.Ip4Connectivity", &rawEnum); err != nil {
		return Device{}, errors.Wrap(err, "couldn't query for IPv4 connectivity")
	}
	dev.IPv4Connectivity = DeviceConnectivityState(rawEnum)
	if err = devo.StoreProperty(nmName+".Device.Ip6Connectivity", &rawEnum); err != nil {
		return Device{}, errors.Wrap(err, "couldn't query for IPv6 connectivity")
	}
	dev.IPv6Connectivity = DeviceConnectivityState(rawEnum)

	return dev, nil
}

func dumpDeviceFlags(devo dbus.BusObject, dev Device) (Device, error) {
	var err error

	var rawFlags uint32
	if err = devo.StoreProperty(nmName+".Device.Capabilities", &rawFlags); err != nil {
		return Device{}, errors.Wrap(err, "couldn't query for capabilities")
	}
	dev.Caps = DeviceCaps(rawFlags)

	if err = devo.StoreProperty(nmName+".Device.Managed", &dev.Managed); err != nil {
		return Device{}, errors.Wrap(err, "couldn't query flag for managed")
	}
	if err = devo.StoreProperty(nmName+".Device.Autoconnect", &dev.Autoconnect); err != nil {
		return Device{}, errors.Wrap(err, "couldn't query flag for autoconnect")
	}
	if err = devo.StoreProperty(nmName+".Device.FirmwareMissing", &dev.FirmwareMissing); err != nil {
		return Device{}, errors.Wrap(err, "couldn't query flag for missing firmware")
	}
	if err = devo.StoreProperty(nmName+".Device.NmPluginMissing", &dev.NMPluginMissing); err != nil {
		return Device{}, errors.Wrap(err, "couldn't query flag for missing NetworkManager plugin")
	}

	if err = devo.StoreProperty(nmName+".Device.InterfaceFlags", &rawFlags); err != nil {
		return Device{}, errors.Wrap(err, "couldn't query for interface flags")
	}
	dev.InterfaceFlags = DeviceInterfaceFlags(rawFlags)

	return dev, nil
}
