package main

import "net"

func uint32ToIP(n uint32) net.IP {
	return net.IPv4(
		byte(n),
		byte(n>>8),
		byte(n>>16),
		byte(n>>24),
	)
}

func ipToUint32(ip net.IP) uint32 {
	ip = ip.To4()
	return uint32(ip[3])<<24 | uint32(ip[2])<<16 | uint32(ip[1])<<8 | uint32(ip[0])
}

func sum(array []uint64) uint64 {
	s := uint64(0)
	for _, v := range array {
		s += v
	}
	return s
}
