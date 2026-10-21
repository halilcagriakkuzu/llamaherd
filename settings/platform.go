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

// Frameless reports whether the main window should draw its own title bar
// instead of using the native one. The default is on in WSL, where the frame is
// WSLg's Windows-side chrome wrapped around a Linux window; everything else
// keeps the native frame. LLAMAHERD_FRAMELESS (1/0) and the "frameless" config
// key override it.
func Frameless() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(EnvFrameless))) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	}
	if f := Config().Frameless; f != nil {
		return *f
	}
	return IsWSL()
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
