package main

import (
	"fmt"
	"net"
	"strings"
)

func redactIP(url string) string {
	host, _, err := net.SplitHostPort(url)
	if err != nil {
		return url
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return host
	}

	ip4 := ip.To4()
	if ip4 != nil {
		return fmt.Sprintf("%d.%d.%d.x", ip4[0], ip4[1], ip4[2])
	}

	if lastColon := strings.LastIndex(host, ":"); lastColon != -1 {
		return host[:lastColon+1] + "x"
	}

	return host
}
