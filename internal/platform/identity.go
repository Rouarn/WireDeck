package platform

import (
	"net"
	"os"

	"woyuxiaji.cn/wiredeck/internal/model"
)

// GetHostIdentity 返回电脑的主机名与首选局域网 IPv4。
//
// 手机「无线调试 → 已配对设备」中显示的 `主机名@电脑IP` 即来自这两个值，
// 其中主机名由 adb 在配对时上报（gethostname），应用无法单独覆盖。
func GetHostIdentity() model.HostIdentity {
	hostname, _ := os.Hostname()
	return model.HostIdentity{
		Hostname: hostname,
		LANIP:    primaryLANIP(),
	}
}

func primaryLANIP() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			ip := ipnet.IP.To4()
			if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
				continue
			}

			return ip.String()
		}
	}

	return ""
}
