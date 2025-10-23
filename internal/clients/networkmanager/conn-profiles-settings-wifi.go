package networkmanager

import (
	"fmt"

	"github.com/godbus/dbus/v5"
)

type ConnProfileSettingsWifi struct {
	// APIsolation int32 // TODO: change this to an int32 enum
	// AssignedMACAddress  string
	Band ConnProfileSettingsWifiBand
	// BSSID               []string
	Channel uint32 // TODO
	// ChannelWidth int32
	Hidden bool // TODO
	// MACAddress          []byte
	// MACAddressBlacklist []string
	// MACAddressDenylist  []string
	Mode ConnProfileSettingsWifiMode // TODO
	// MTU                 uint32
	// Powersave uint32 // TODO: change this to a uint32 enum
	// SeenBSSIDs          []string
	SSID []byte // TODO
}

type ConnProfileSettingsWifiBand string

var connProfileSettingsWifiBand = map[ConnProfileSettingsWifiBand]EnumInfo{
	"a": {
		Short:   "a",
		Details: "802.11a (5 GHz)",
	},
	"bg": {
		Short:   "bg",
		Details: "802.11b/g (2.4 GHz)",
	},
	"": {
		Short:   "any",
		Details: "any available band",
	},
}

func (b ConnProfileSettingsWifiBand) Info() EnumInfo {
	info, ok := connProfileSettingsWifiBand[b]
	if !ok {
		return EnumInfo{
			Short:   "unknown",
			Details: fmt.Sprintf("unknown band (%s)", b),
			Level:   EnumInfoLevelError,
		}
	}
	return info
}

type ConnProfileSettingsWifiMode string

var connProfileSettingsWifiMode = map[ConnProfileSettingsWifiMode]EnumInfo{
	"": {
		Short:   "infrastructure",
		Details: "connect to an external Wi-Fi network",
	},
	"infrastructure": {
		Short:   "infrastructure",
		Details: "connect to an external Wi-Fi network",
	},
	"mesh": {
		Short:   "mesh",
		Details: "connect to a mesh Wi-Fi network",
	},
	"adhoc": {
		Short:   "ad-hoc",
		Details: "connect to an ad-hoc Wi-Fi network",
	},
	"ap": {
		Short:   "ap",
		Details: "create a Wi-Fi hotspot",
	},
}

func (m ConnProfileSettingsWifiMode) Info() EnumInfo {
	info, ok := connProfileSettingsWifiMode[m]
	if !ok {
		return EnumInfo{
			Short:   "unknown",
			Details: fmt.Sprintf("unknown mode (%s)", m),
			Level:   EnumInfoLevelError,
		}
	}
	return info
}

type ConnProfileSettingsWifiSec struct {
	Group    EnumSet[ConnProfileSettingsWifiSecGroup]
	KeyMgmt  ConnProfileSettingsWifiSecKeyMgmt
	Pairwise EnumSet[ConnProfileSettingsWifiSecPairwise]
	Proto    EnumSet[ConnProfileSettingsWifiSecProto]
	// Warning: NetworkManager only returns the real PSK if we're running as root; otherwise, it
	// returns an empty string!
	PSK      string
	PSKFlags ConnProfileSettingsWifiSecPSKFlags
}

type ConnProfileSettingsWifiSecGroup string

var ConnProfileSettingsWifiSecGroupInfo = map[ConnProfileSettingsWifiSecGroup]EnumInfo{
	"wep40": {
		Short:   "WEP-40",
		Details: "Wired Equivalent Privacy with 40-bit key (WEP, insecure)",
	},
	"wep104": {
		Short:   "WEP-104",
		Details: "Wired Equivalent Privacy with 104-bit key (WEP, insecure)",
	},
	"tkip": {
		Short:   "TKIP",
		Details: "Temporal Key Integrity Protocol (WPA, insecure)",
	},
	"ccmp": {
		Short:   "AES/CCMP",
		Details: "CCM mode Protocol (WPA2/3)",
	},
}

func (g ConnProfileSettingsWifiSecGroup) Info() EnumInfo {
	info, ok := ConnProfileSettingsWifiSecGroupInfo[g]
	if !ok {
		return EnumInfo{
			Short:   "unknown",
			Details: fmt.Sprintf("unknown group/broadcast encryption algorithm (%s)", g),
			Level:   EnumInfoLevelError,
		}
	}
	return info
}

type ConnProfileSettingsWifiSecKeyMgmt string

var connProfileSettingsWifiSecKeyMgmt = map[ConnProfileSettingsWifiSecKeyMgmt]EnumInfo{
	"": {
		Short:   "none",
		Details: "no password protection",
	},
	"none": {
		Short:   "WEP",
		Details: "WEP (insecure)",
	},
	"ieee8021x": {
		Short:   "IEEE 802.1x",
		Details: "Dynamic WEP (insecure)",
	},
	"owe": {
		Short:   "OWE",
		Details: "Opportunistic Wireless Encryption",
	},
	"wpa-psk": {
		Short:   "PSK",
		Details: "WPA2/3-Personal Pre-Shared Key",
	},
	"wpa-eap": {
		Short:   "EAP",
		Details: "WPA2/3-Enterprise Extensible Authentication Protocol",
	},
	"sae": {
		Short:   "SAE",
		Details: "WPA3-Personal Simultaneous Authentication of Equals",
	},
	"wpa-eap-suite-b-192": {
		Short:   "EAP SuiteB-192",
		Details: "WPA3-Enterprise Extensible Authentication Protocol with SuiteB-192 bit encryption",
	},
}

func (m ConnProfileSettingsWifiSecKeyMgmt) Info() EnumInfo {
	info, ok := connProfileSettingsWifiSecKeyMgmt[m]
	if !ok {
		return EnumInfo{
			Short:   "unknown",
			Details: fmt.Sprintf("unknown key management (%s)", m),
			Level:   EnumInfoLevelError,
		}
	}
	return info
}

type ConnProfileSettingsWifiSecPairwise string

var ConnProfileSettingsWifiSecPairwiseInfo = map[ConnProfileSettingsWifiSecPairwise]EnumInfo{
	"tkip": {
		Short:   "TKIP",
		Details: "Temporal Key Integrity Protocol (WPA, insecure)",
	},
	"ccmp": {
		Short:   "AES/CCMP",
		Details: "CCM mode Protocol (WPA2/3)",
	},
}

func (p ConnProfileSettingsWifiSecPairwise) Info() EnumInfo {
	info, ok := ConnProfileSettingsWifiSecPairwiseInfo[p]
	if !ok {
		return EnumInfo{
			Short:   "unknown",
			Details: fmt.Sprintf("unknown pairwise encryption algorithm (%s)", p),
			Level:   EnumInfoLevelError,
		}
	}
	return info
}

type ConnProfileSettingsWifiSecProto string

var ConnProfileSettingsWifiSecProtoInfo = map[ConnProfileSettingsWifiSecProto]EnumInfo{
	"wpa": {
		Short:   "WPA",
		Details: "Wi-Fi Protected Access (insecure)",
	},
	"rsn": {
		Short:   "WPA2/RSN",
		Details: "Wi-Fi Protected Access 2/3 (Robust Security Network)",
	},
}

func (p ConnProfileSettingsWifiSecProto) Info() EnumInfo {
	info, ok := ConnProfileSettingsWifiSecProtoInfo[p]
	if !ok {
		return EnumInfo{
			Short:   "unknown",
			Details: fmt.Sprintf("unknown WPA protocol version (%s)", p),
			Level:   EnumInfoLevelError,
		}
	}
	return info
}

type ConnProfileSettingsWifiSecPSKFlags uint32

func (f ConnProfileSettingsWifiSecPSKFlags) None() bool {
	return f == 0
}

func (f ConnProfileSettingsWifiSecPSKFlags) WithNone() ConnProfileSettingsWifiSecPSKFlags {
	return 0
}

func (f ConnProfileSettingsWifiSecPSKFlags) AgentOwned() bool {
	return f&0x1 > 0
}

func (f ConnProfileSettingsWifiSecPSKFlags) WithAgentOwned() ConnProfileSettingsWifiSecPSKFlags {
	f |= 0x1
	return f
}

func (f ConnProfileSettingsWifiSecPSKFlags) NotSaved() bool {
	return f&0x2 > 0
}

func (f ConnProfileSettingsWifiSecPSKFlags) WithNotSaved() ConnProfileSettingsWifiSecPSKFlags {
	f |= 0x2
	return f
}

func (f ConnProfileSettingsWifiSecPSKFlags) NotRequired() bool {
	return f&0x4 > 0
}

func (f ConnProfileSettingsWifiSecPSKFlags) WithNotRequired() ConnProfileSettingsWifiSecPSKFlags {
	f |= 0x4
	return f
}

func dumpConnProfileSettingsWifi(
	rawSettings map[string]dbus.Variant,
) (s ConnProfileSettingsWifi, err error) {
	rawBand, err := ensureVar(rawSettings, "band", "", false, "")
	if err != nil {
		return s, err
	}
	s.Band = ConnProfileSettingsWifiBand(rawBand)

	if s.Channel, err = ensureVar[uint32](rawSettings, "channel", "", false, 0); err != nil {
		return s, err
	}

	if s.Hidden, err = ensureVar(rawSettings, "hidden", "", false, false); err != nil {
		return s, err
	}

	rawMode, err := ensureVar(rawSettings, "mode", "", false, "")
	if err != nil {
		return s, err
	}
	if rawMode == "" {
		rawMode = "infrastructure"
	}
	s.Mode = ConnProfileSettingsWifiMode(rawMode)

	if s.SSID, err = ensureVar(rawSettings, "ssid", "SSID", false, []byte{}); err != nil {
		return s, err
	}

	return s, nil
}

func dumpConnProfileSettingsWifiSec(
	rawSettings map[string]dbus.Variant, rawSecrets map[string]dbus.Variant,
) (s ConnProfileSettingsWifiSec, err error) {
	rawGroup, err := ensureVar(
		rawSettings, "group", "group/broadcast encryption algorithms whitelist", false, []string{},
	)
	if err != nil {
		return s, err
	}
	for _, pairwise := range rawGroup {
		s.Group = append(s.Group, ConnProfileSettingsWifiSecGroup(pairwise))
	}

	// Note(ethanjli): if there's no key-mgmt, that means there should be no password either
	rawKeyMgmt, err := ensureVar(rawSettings, "key-mgmt", "key management method", false, "")
	if err != nil {
		return s, err
	}
	s.KeyMgmt = ConnProfileSettingsWifiSecKeyMgmt(rawKeyMgmt)

	rawPairwise, err := ensureVar(
		rawSettings, "pairwise", "pairwise encryption algorithms whitelist", false, []string{},
	)
	if err != nil {
		return s, err
	}
	for _, pairwise := range rawPairwise {
		s.Pairwise = append(s.Pairwise, ConnProfileSettingsWifiSecPairwise(pairwise))
	}

	rawProto, err := ensureVar(
		rawSettings, "proto", "WPA protocol versions whitelist", false, []string{},
	)
	if err != nil {
		return s, err
	}
	for _, pairwise := range rawProto {
		s.Proto = append(s.Proto, ConnProfileSettingsWifiSecProto(pairwise))
	}

	if s.PSK, err = ensureVar(rawSecrets, "psk", "PSK", false, ""); err != nil {
		return s, err
	}

	return s, nil
}
