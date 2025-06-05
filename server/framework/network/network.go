package network

import "net"

// GetLocalIP 获取本机内网IP
func GetLocalIP() string {
	// 获取所有网络接口
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	// 遍历所有网络接口
	for _, i := range interfaces {
		// 跳过未启用的接口
		if i.Flags&net.FlagUp == 0 {
			continue
		}
		// 跳过回环接口
		if i.Flags&net.FlagLoopback != 0 {
			continue
		}

		// 获取接口的所有地址
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}

		// 遍历接口的所有地址
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// 跳过无效IP
			if ip == nil || ip.IsLoopback() {
				continue
			}

			// 只获取IPv4地址
			ip = ip.To4()
			if ip == nil {
				continue
			}

			// 跳过私有IP
			// if ip.IsPrivate() {
			//     continue
			// }

			return ip.String()
		}
	}

	return ""
}
