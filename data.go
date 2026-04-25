package main

import (
	"net"

	"github.com/asiffer/xdprobe/kernel"
	"github.com/cilium/ebpf"
)

var cpuCount = 0

type Hit struct {
	SrcIP   net.IP `json:"ip"`
	DstPort uint16 `json:"port"`
	Proto   uint8  `json:"proto"`
	Count   uint64 `json:"count"`
}

func init() {
	c, err := ebpf.PossibleCPU()
	if err != nil {
		panic(err)
	}
	cpuCount = c
}

func extractIPPorts(key uint64) (net.IP, uint16, uint8) {
	// srcIP := intToIP(uint32(key >> 32))
	srcIP := uint32ToIP(uint32(key >> 32))
	dstPort := uint16(key & 0xFFFF)
	proto := uint8((key >> 16) & 0xFF)
	return srcIP, dstPort, proto
}

func collectData(objs *kernel.XDProbeObjects) (map[string][]Hit, error) {
	// out := make(map[string]map[uint16]uint64)
	out := make(map[string][]Hit)
	toDelete := make([]uint64, 0, 100)

	it := objs.SourcesCount.Iterate()
	var key uint64
	c := make([]uint64, 0, cpuCount)
	for it.Next(&key, &c) {
		// dstPort is 0 for IP-level stats
		srcIP, dstPort, proto := extractIPPorts(key)
		if _, ok := out[srcIP.String()]; !ok {
			out[srcIP.String()] = make([]Hit, 0)
		}
		out[srcIP.String()] = append(out[srcIP.String()], Hit{
			SrcIP:   srcIP,
			DstPort: dstPort,
			Proto:   proto,
			Count:   sum(c),
		})
		toDelete = append(toDelete, key)
	}
	if err := it.Err(); err != nil {
		return nil, err
	}
	_, err := objs.SourcesCount.BatchDelete(toDelete, nil)
	return out, err
}
