package main

import (
	"embed"
	"encoding/hex"
	"html/template"
	"mime"
	"net"
	"strings"

	"net/http"

	"github.com/asiffer/xdprobe/kernel"
)

//go:embed templates/*
var templatesFS embed.FS

//go:embed assets/*
var assetsFS embed.FS

const (
	LOGIN_ROUTE  = "/auth/login/"
	LOGOUT_ROUTE = "/auth/logout/"
)

type EventSource struct {
	IP        string  `json:"ip"`
	City      string  `json:"city,omitempty"`
	Count     uint64  `json:"count,omitempty"`
	Country   string  `json:"country,omitempty"`
	Continent string  `json:"continent,omitempty"`
	Latitude  float64 `json:"lat,omitempty"`
	Longitude float64 `json:"lon,omitempty"`
	Hits      []Hit   `json:"hits,omitempty"`
}

type Event struct {
	Sources []EventSource `json:"sources"`
}

func index(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	t, err := template.ParseFS(templatesFS,
		"templates/base.html",
		"templates/dark.html",
		"templates/sse.html",
		"templates/index.html",
		"templates/header.html",
		"templates/logs.html",
		"templates/policy.html",
	)
	if err != nil {
		log.Error().Err(err).Msg("parsing error")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if err := t.ExecuteTemplate(w, "base", nil); err != nil {
		log.Error().Err(err).Msg("execution error")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func serve(addr string, channel <-chan *Event, objs *kernel.XDProbeObjects) error {
	broker := NewBroker()
	go func() {
		for event := range channel {
			broker.Publish(event)
		}
	}()

	hashPassword, err := hex.DecodeString(password)
	if err != nil {
		log.Error().Err(err).Msg("Fail to decode password hash")
		return err
	}
	auth := NewAuth(username, hashPassword)

	mime.AddExtensionType(".gl", "text/javascript")
	mux := http.NewServeMux()

	mux.Handle("GET "+LOGIN_ROUTE, http.HandlerFunc(auth.loginGet))
	mux.Handle("POST "+LOGIN_ROUTE, http.HandlerFunc(auth.loginPost))
	mux.Handle("POST "+LOGOUT_ROUTE, http.HandlerFunc(auth.Logout))

	mux.Handle("GET /policy", getPolicyHandler(objs.IpPolicies))
	mux.Handle("POST /policy", postPolicyHandler(objs.IpPolicies))
	mux.Handle("DELETE /policy/{ip}", deletePolicyHandler(objs.IpPolicies))
	// keep the prefix
	mux.Handle("GET /assets/", http.FileServer(http.FS(assetsFS)))
	mux.Handle("GET /live", broker)
	mux.Handle("/", http.HandlerFunc(index))

	if insecure {
		sv := &http.Server{Addr: addr, Handler: http.NewCrossOriginProtection().Handler(mux)}
		return sv.ListenAndServe()
	}

	sv := &http.Server{Handler: http.NewCrossOriginProtection().Handler(auth.RequireAuth(mux))}

	if strings.Contains(addr, ":") {
		// tcp socket
		sv.Addr = addr
		return sv.ListenAndServe()
	} else {
		// unix socket
		conn, err := net.Listen("unix", addr)
		if err != nil {
			return err
		}
		return sv.Serve(conn)
	}

}
