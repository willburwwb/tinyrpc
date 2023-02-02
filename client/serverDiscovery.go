package client

import (
	"errors"
	"hash/crc32"
	"math/rand"
	"sync"
	"time"
)

// 在client端实现服务发现，需要进行负载均衡

type Model int

const (
	RandomModel Model = iota
	RoundRobinModel
	IpHashModel
)

type Discovery interface {
	Update(servers []string) error
	Get(model Model) (string, error)
	GetAll() ([]string, error)
	Refresh() error
}

type ServerDiscovery struct {
	mu      sync.Mutex
	servers []string
	index   int
	ip      string
}

func NewServerDiscovery(servers []string, ip string) *ServerDiscovery {
	rand.Seed(time.Now().UnixNano())
	s := &ServerDiscovery{
		servers: servers,
		index:   rand.Int(),
		ip:      ip,
	}
	return s
}

func (sd *ServerDiscovery) Update(servers []string) error {
	sd.mu.Lock()
	defer sd.mu.Unlock()
	sd.servers = servers
	return nil
}
func (sd *ServerDiscovery) Get(model Model) (string, error) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	n := len(sd.servers)
	if n == 0 {
		return "", errors.New("client discovery error: no available server")
	}
	switch model {
	case RandomModel:
		return sd.servers[rand.Intn(n)], nil
	case RoundRobinModel:
		serverAddr := sd.servers[sd.index%n]
		sd.index = (sd.index + 1) % n
		return serverAddr, nil
	case IpHashModel:
		v := crc32.ChecksumIEEE([]byte(sd.ip)) % uint32(n)
		return sd.servers[v], nil
	default:
		return "", errors.New("client discovery: not supported model")
	}
}
