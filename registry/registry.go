package registry

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

var DefaultRegistry = NewRegistry(time.Second)

type Registry struct {
	mu      sync.Mutex
	timeout time.Duration
	servers map[string]time.Time
}

func NewRegistry(timeout time.Duration) *Registry {
	return &Registry{
		servers: make(map[string]time.Time),
		timeout: timeout,
	}
}
func (r *Registry) addServer(addr string) {
	r.mu.Lock()
	defer r.mu.Lock()

	r.servers[addr] = time.Now()
}
func (r *Registry) getActiveServers() []string {
	r.mu.Lock()
	defer r.mu.Lock()
	var servers []string
	for addr, t := range r.servers {
		if t.Add(r.timeout).After(time.Now()) {
			servers = append(servers, addr)
		} else {
			delete(r.servers, addr)
		}
	}
	return servers
}
func (r *Registry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	method := req.Method
	if method == "GET" {
		w.Header().Set("servers", strings.Join(r.getActiveServers(), ","))
	} else if method == "POST" {
		addr := req.Header.Get("server")
		r.addServer(addr)
	}
}

func HandleHTTP() {
	http.Handle("/registry", DefaultRegistry)
}
