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

// IgnoredFsTypes lists filesystem types that should be excluded from disk
// usage collection. Docker overlay2 mounts (one per running container) and
// other pseudo-filesystems appear in /proc/mounts and would otherwise cause
// CollectSystemMetrics to call statfs() on every overlay mount every second.
var IgnoredFsTypes = map[string]bool{
	"overlay":         true,
	"overlayfs":       true,
	"tmpfs":           true,
	"squashfs":        true,
	"devtmpfs":        true,
	"devpts":          true,
	"cgroup":          true,
	"cgroup2":         true,
	"sysfs":           true,
	"proc":            true,
	"nsfs":            true,
	"autofs":          true,
	"fuse.lxcfs":      true,
	"fuse.gvfsd-fuse": true,
	"pstore":          true,
	"securityfs":      true,
	"debugfs":         true,
	"hugetlbfs":       true,
	"mqueue":          true,
	"configfs":        true,
	"binfmt_misc":     true,
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

func IsIgnoredFsType(fstype string) bool {
	return IgnoredFsTypes[fstype]
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
