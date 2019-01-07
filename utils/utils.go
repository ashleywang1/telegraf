package utils

import (
	"log"
	"net"
	"os"
)

// GetIP returns the outbound IP of this machine, or an empty string on failure
func GetIP() string {
	IP := os.Getenv("MYPRVIP")
	if IP != "" {
		return IP
	}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Printf("error in GetIP - %v\n", err)
		return ""
	}
	for _, address := range addrs {
		// return the first address that is not a loopback
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}
