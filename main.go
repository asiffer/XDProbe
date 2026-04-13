//go:generate go tool bpf2go -tags linux -target bpfel -output-stem program -output-suffix= -go-package kernel -output-dir kernel XDProbe kernel/program.c
package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"time"

	"github.com/asiffer/xdprobe/kernel"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/rlimit"
	"github.com/oschwald/maxminddb-golang/v2"
)

func main() {
	fmt.Println(BANNER)

	if err := ReadEnvAndFlags(); err != nil {
		log.Fatal().Err(err).Msg("Fail to load config")
	}

	if geoipdb == "" {
		log.Fatal().Msg("You must provide a GeoIP database (mmdb format)")
	}
	if _, err := os.Stat(geoipdb); os.IsNotExist(err) {
		log.Fatal().Str("file", geoipdb).Msg("GeoIP database file does not exist")
	}

	db, err := maxminddb.Open(geoipdb)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to open GeoIP database")
	}
	defer db.Close()
	log.Info().Str("file", geoipdb).Msg("Loaded GeoIP database")

	iface, err := net.InterfaceByName(nicName)
	if err != nil {
		log.Fatal().Err(err).Str("interface", nicName).Msg("Getting interface")
	}
	log.Info().Str("interface", nicName).Msg("Retrieved network interface")

	// Remove resource limits for kernels <5.11.
	if err := rlimit.RemoveMemlock(); err != nil {
		log.Fatal().Err(err).Msg("Removing memlock")
	}
	log.Info().Msg("Removed memlock resource limit")

	var objs kernel.XDProbeObjects
	if err := kernel.LoadXDProbeObjects(&objs, nil); err != nil {
		log.Fatal().Err(err).Msg("Loading objects")
	}
	defer func() { objs.Close(); log.Info().Msg("Unloaded eBPF objects") }()
	log.Info().Msg("Loaded eBPF program")

	// attach program to the network interface
	link, err := link.AttachXDP(link.XDPOptions{
		Program:   objs.Xdprobe,
		Interface: iface.Index,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Attaching XDP")
	}
	defer func() { link.Close(); log.Info().Msg("Detached XDP") }()

	// start the HTTP server
	channel := make(chan *Event, 10)
	errCh := make(chan error, 1)
	go func() { errCh <- serve(addr, channel) }()
	log.Info().Str("addr", "http://"+addr).Msg("Started HTTP server")

	// graceful shutdown
	stop := make(chan os.Signal, 5)
	signal.Notify(stop, os.Interrupt)

	// main loop
	ticker := time.Tick(tick)
	log.Info().Dur("tick", tick).Msg("Started main loop")

	for {
		select {
		case err := <-errCh:
			log.Fatal().Err(err).Msg("Server error")
		case <-ticker:
			// ports, err := updatePorts(&objs)
			// if err != nil {
			// 	log.Fatal("Map lookup:", err)
			// }
			ip4, err := updateIPs(&objs)
			if err != nil {
				log.Fatal().Err(err).Msg("Map lookup failed")
			}
			event := NewEvent(db, ip4)
			if len(event.Sources) == 0 {
				continue
			}
			channel <- event
		case <-stop:
			log.Warn().Msg("Received signal, exiting..")
			return
		}
	}

}
