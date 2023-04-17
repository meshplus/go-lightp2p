package network

import (
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p-core/connmgr"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-libp2p-core/sec"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/sirupsen/logrus"
)

type connMgr struct {
	enabled bool
	lo      int
	hi      int
	grace   time.Duration
}

var (
	defaultConnectTimeout           = 10 * time.Second
	defaultSendTimeout              = 2 * time.Second
	defaultWaitTimeout              = 5 * time.Second
	defaultReusableProtocolIndex    = 0
	defaultNonReusableProtocolIndex = 1
)

type timeout struct {
	connectTimeout time.Duration
	sendTimeout    time.Duration
	waitTimeout    time.Duration
}

func defaultTimeout() *timeout {
	return &timeout{
		connectTimeout: defaultConnectTimeout,
		sendTimeout:    defaultSendTimeout,
		waitTimeout:    defaultWaitTimeout,
	}
}

type Config struct {
	localAddr                string
	privKey                  crypto.PrivKey
	protocolIDs              []protocol.ID
	logger                   logrus.FieldLogger
	bootstrap                []string
	connMgr                  *connMgr
	notify                   network.Notifiee
	gater                    connmgr.ConnectionGater
	transport                sec.SecureTransport
	transportID              string
	reusableProtocolIndex    int
	nonReusableProtocolIndex int
	timeout                  *timeout
}

type Option func(*Config)

func WithTransportId(tid string) Option {
	return func(config *Config) {
		config.transportID = tid
	}
}

func WithTransport(tpt sec.SecureTransport) Option {
	return func(config *Config) {
		config.transport = tpt
	}
}

func WithPrivateKey(privKey crypto.PrivKey) Option {
	return func(config *Config) {
		config.privKey = privKey
	}
}

func WithLocalAddr(addr string) Option {
	return func(config *Config) {
		config.localAddr = addr
	}
}

func WithProtocolIDs(ids []string) Option {
	return func(config *Config) {
		config.protocolIDs = []protocol.ID{}
		for _, id := range ids {
			config.protocolIDs = append(config.protocolIDs, protocol.ID(id))
		}
	}
}

func WithBootstrap(peers []string) Option {
	return func(config *Config) {
		config.bootstrap = peers
	}
}

func WithNotify(notify network.Notifiee) Option {
	return func(config *Config) {
		config.notify = notify
	}
}

func WithConnectionGater(gater connmgr.ConnectionGater) Option {
	return func(config *Config) {
		config.gater = gater
	}
}

// WithConnMgr * enable is the enable signal of the connection manager module.
// * lo and hi are watermarks governing the number of connections that'll be maintained.
//   When the peer count exceeds the 'high watermark', as many peers will be pruned (and
//   their connections terminated) until 'low watermark' peers remain.
// * grace is the amount of time a newly opened connection is given before it becomes
//   subject to pruning.
func WithConnMgr(enable bool, lo int, hi int, grace time.Duration) Option {
	return func(config *Config) {
		config.connMgr = &connMgr{
			enabled: enable,
			lo:      lo,
			hi:      hi,
			grace:   grace,
		}
	}
}

func WithLogger(logger logrus.FieldLogger) Option {
	return func(config *Config) {
		config.logger = logger
	}
}

func WithConnectTimeout(time time.Duration) Option {
	return func(config *Config) {
		config.timeout.connectTimeout = time
	}
}

func WithSendTimeout(time time.Duration) Option {
	return func(config *Config) {
		config.timeout.sendTimeout = time
	}
}
func WithWaitTimeout(time time.Duration) Option {
	return func(config *Config) {
		config.timeout.waitTimeout = time
	}
}

func WithReusableProtocolIndex(reusableProtocolIndex int) Option {
	return func(config *Config) {
		config.reusableProtocolIndex = reusableProtocolIndex
	}
}

func WithNonReusableProtocolIndex(nonReusableProtocolIndex int) Option {
	return func(config *Config) {
		config.nonReusableProtocolIndex = nonReusableProtocolIndex
	}
}

func checkConfig(config *Config) error {
	if config.logger == logrus.FieldLogger(nil) {
		config.logger = log.NewWithModule("p2p")
	}

	if config.localAddr == "" {
		return fmt.Errorf("empty local address")
	}

	return nil
}

func generateConfig(opts ...Option) (*Config, error) {
	conf := &Config{}
	timeout := defaultTimeout()
	conf.timeout = timeout
	conf.reusableProtocolIndex = defaultReusableProtocolIndex
	conf.nonReusableProtocolIndex = defaultNonReusableProtocolIndex
	for _, opt := range opts {
		opt(conf)
	}
	if err := checkConfig(conf); err != nil {
		return nil, fmt.Errorf("create p2p: %w", err)
	}
	return conf, nil
}
