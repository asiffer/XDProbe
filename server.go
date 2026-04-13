package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"mime"
	"net"
	"strings"

	"net/http"
	"sync"
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

type Broker struct {
	clients map[chan []byte]struct{}
	mu      sync.RWMutex
}

func NewBroker() *Broker {
	return &Broker{clients: make(map[chan []byte]struct{})}
}

func (b *Broker) Subscribe() chan []byte {
	ch := make(chan []byte, 64)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

func (b *Broker) Unsubscribe(ch chan []byte) {
	b.mu.Lock()
	delete(b.clients, ch)
	close(ch)
	b.mu.Unlock()
}

func (b *Broker) Publish(event *Event) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range b.clients {
		select {
		case ch <- data:
		default:
			// slow client, drop message
		}
	}
}

func (b *Broker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rc := http.NewResponseController(w)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	for {
		select {
		case <-r.Context().Done():
			return
		case data := <-ch:
			fmt.Fprintf(w, "event: probe\ndata: %s\n\n", data)
			if err := rc.Flush(); err != nil {
				return // client gone
			}
		}
	}
}

func serve(addr string, channel <-chan *Event) error {
	broker := NewBroker()
	go func() {
		for event := range channel {
			broker.Publish(event)
		}
	}()

	auth := NewAuth(username, password)

	mime.AddExtensionType(".gl", "text/javascript")
	mux := http.NewServeMux()

	mux.Handle("/live", broker)
	// keep the prefix
	mux.Handle("/assets/", http.FileServer(http.FS(assetsFS)))
	mux.Handle("/", http.HandlerFunc(index))

	if insecure {
		sv := &http.Server{Addr: addr, Handler: http.NewCrossOriginProtection().Handler(mux)}
		return sv.ListenAndServe()
	}

	mux.Handle(LOGIN_ROUTE, http.HandlerFunc(auth.Login))
	mux.Handle(LOGOUT_ROUTE, http.HandlerFunc(auth.Logout))

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
