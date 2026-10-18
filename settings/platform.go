package settings

import (
	"net"
	"os"
	"runtime"
	"strings"
)

// IsWSL reports whether this process runs inside Windows Subsystem for Linux.
// Only Linux can be WSL; detection reads /proc/version for the Microsoft
// kernel marker.
func IsWSL() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	b, err := os.ReadFile("/proc/version")
	return err == nil && strings.Contains(strings.ToLower(string(b)), "microsoft")
}

// LocalIPv4 returns the machine's primary non-loopback IPv4 address, or "" when
// none is found. In WSL this is the address the Windows host uses to reach the
// distro. Cross-platform (net.Interfaces), so it degrades gracefully.
func LocalIPv4() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok || ipnet.IP.IsLoopback() {
			continue
		}
		ip4 := ipnet.IP.To4()
		if ip4 == nil || ip4.IsLinkLocalUnicast() {
			continue
		}
		return ip4.String()
	}
	return ""
}
