package config

var IgnoredMounts = map[string]bool{
	"/boot/efi": true,
	"/boot":     true,
	"/run":      true,
	"/run/lock": true,
	"/snap":     true,
	"/sys":      true,
	"/proc":     true,
	"/dev":      true,
	"/dev/shm":  true,
}

var IgnoredNetworkInterfaces = map[string]bool{
	"lo":         true,
	"virbr":      true,
	"docker0":    true,
	"br-":        true,
	"veth":       true,
	"tailscale0": true,
}

func IsIgnoredMount(mountpoint string) bool {
	return IgnoredMounts[mountpoint]
}

func IsIgnoredNetworkInterface(iface string) bool {
	if IgnoredNetworkInterfaces[iface] {
		return true
	}
	prefixes := []string{"br-", "veth", "virbr"}
	for _, p := range prefixes {
		if len(iface) >= len(p) && iface[:len(p)] == p {
			return true
		}
	}
	return false
}
