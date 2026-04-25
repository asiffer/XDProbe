package main

import (
	"net/netip"

	"github.com/oschwald/maxminddb-golang/v2"
)

type GeoRecord struct {
	City struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"city"`
	Continent struct {
		Code      string            `maxminddb:"code"`
		GeonameID uint32            `maxminddb:"geoname_id"`
		Names     map[string]string `maxminddb:"names"`
	} `maxminddb:"continent"`
	Country struct {
		GeonameID         uint32            `maxminddb:"geoname_id"`
		ISOCode           string            `maxminddb:"iso_code"`
		IsInEuropeanUnion bool              `maxminddb:"is_in_european_union"`
		Names             map[string]string `maxminddb:"names"`
	} `maxminddb:"country"`
	Location struct {
		Latitude  float64 `maxminddb:"latitude"`
		Longitude float64 `maxminddb:"longitude"`
	} `maxminddb:"location"`
	Subdivisions []struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"subdivisions"`
}

func extractHits(hits []Hit) ([]Hit, uint64) {
	count := uint64(0)
	for i, hit := range hits {
		// Aggregate count for IP-level hits (proto=0, dstPort=0)
		if hit.Proto == 0 && hit.DstPort == 0 {
			return append(hits[:i], hits[i+1:]...), count
		}
	}
	return hits, count
}

func NewEvent(db *maxminddb.Reader, data map[string][]Hit) *Event {
	sources := make([]EventSource, 0, len(data))
	record := GeoRecord{}
	for ip, hits := range data {
		addr, err := netip.ParseAddr(ip)
		if err != nil {
			continue
		}
		remainingHits, count := extractHits(hits)

		if addr.IsPrivate() {
			sources = append(sources, EventSource{
				IP:        ip,
				City:      "LAN",
				Country:   "Private",
				Continent: "",
				Latitude:  1.0,
				Longitude: 1.0,
				Count:     count,
				Hits:      remainingHits,
			})
			continue
		}
		if err := db.Lookup(addr).Decode(&record); err != nil {
			continue
		}

		sources = append(sources, EventSource{
			IP:        ip,
			City:      record.City.Names["en"],
			Country:   record.Country.Names["en"],
			Continent: record.Continent.Names["en"],
			Latitude:  record.Location.Latitude,
			Longitude: record.Location.Longitude,
			Count:     count,
			Hits:      remainingHits,
		})
	}
	return &Event{Sources: sources}
}
