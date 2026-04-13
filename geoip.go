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

func NewEvent(db *maxminddb.Reader, ips map[string]uint64) *Event {
	sources := make([]EventSource, 0, len(ips))
	record := GeoRecord{}
	for ip, count := range ips {
		addr, err := netip.ParseAddr(ip)
		if err != nil {
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
		})
	}
	return &Event{Sources: sources}
}
