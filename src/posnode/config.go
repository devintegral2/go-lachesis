package posnode

import (
	"net"
	"path/filepath"
	"strconv"
	"time"

	"github.com/Fantom-foundation/go-lachesis/src/utils"
)

// Config is a set of nodes params.
type Config struct {
	EventParentsCount int // max count of event's parents (includes self-parent)
	Port              int // default service port

	GossipThreads    int           // count of gossiping goroutines
	EmitInterval     time.Duration // event emission interval
	DiscoveryTimeout time.Duration // how often discovery should try to request

	ConnectTimeout time.Duration // how long dialer will for connection to be established
	ClientTimeout  time.Duration // how long will gRPC client will wait for response

	CertPath string // directory to store pem keys & certs
}

// DefaultConfig returns default config.
func DefaultConfig() *Config {
	dataDir := utils.DefaultDataDir()

	certPath := filepath.Join(dataDir, "certs")

	return &Config{
		EventParentsCount: 3,
		Port:              55555,

		GossipThreads:    4,
		EmitInterval:     10 * time.Second,
		DiscoveryTimeout: 5 * time.Minute,

		ConnectTimeout: 15 * time.Second,
		ClientTimeout:  15 * time.Second,

		CertPath: certPath,
	}
}

// NetAddrOf makes listen address from host and configured port.
func (n *Node) NetAddrOf(host string) string {
	port := strconv.Itoa(n.conf.Port)
	return net.JoinHostPort(host, port)
}
