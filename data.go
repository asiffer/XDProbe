package main

import (
	"encoding/binary"
	"net"

	"github.com/asiffer/xdprobe/kernel"
	"github.com/cilium/ebpf"
)

var cpuCount = 0

func init() {
	c, err := ebpf.PossibleCPU()
	if err != nil {
		panic(err)
	}
	cpuCount = c
}

func sum(array []uint64) uint64 {
	s := uint64(0)
	for _, v := range array {
		s += v
	}
	return s
}

func intToIP(n uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, n)
	return ip
}

func updatePorts(objs *kernel.XDProbeObjects) (map[uint16]uint64, error) {
	out := make(map[uint16]uint64)
	toDelete := make([]uint16, 0, 32)
	it := objs.TcpDstPortCount.Iterate()
	var p uint16
	c := make([]uint64, 0, cpuCount)
	for it.Next(&p, &c) {
		out[p] = sum(c)
		toDelete = append(toDelete, p)
	}
	if err := it.Err(); err != nil {
		return nil, err
	}
	_, err := objs.TcpDstPortCount.BatchDelete(toDelete, nil)
	return out, err
}

func updateIPs(objs *kernel.XDProbeObjects) (map[string]uint64, error) {
	out := make(map[string]uint64)
	toDelete := make([]uint32, 0, 32)

	it := objs.SrcIpCount.Iterate()
	var ip uint32
	c := make([]uint64, 0, cpuCount)
	for it.Next(&ip, &c) {
		out[intToIP(ip).String()] = sum(c)
		toDelete = append(toDelete, ip)
	}
	if err := it.Err(); err != nil {
		return nil, err
	}
	_, err := objs.SrcIpCount.BatchDelete(toDelete, nil)
	return out, err
}
