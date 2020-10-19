package network

import (
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/sirupsen/logrus"
)

type connMgr struct {
	enabled bool
	lo      int
	hi      int
	grace   time.Duration
}

type psMgr struct {
	enablePub bool
	enableSub bool
	pubTopic  string
	subTopic  string
}

type Config struct {
	localAddr   string
	privKey     crypto.PrivKey
	protocolIDs []protocol.ID
	logger      logrus.FieldLogger
	bootstrap   []string
	connMgr     *connMgr
	psMgr       psMgr
}

type Option func(*Config)

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

// * enable is the enable signal of the connection manager module.
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

// * enable indicates whether current peer wants to use pubsub.
//   If enable is false, then this function is no-op.
//   If enable is true, then libp2p will initialize a GossipPubSub instance,
//   and ensure current peer will join in this topic.
//   After that, peer can publish messages to this topic.
// * topicID is topic name.
func WithPublish(enable bool, topicID string) Option {
	if enable {
		return func(config *Config) {
			config.psMgr.pubTopic = topicID
			config.psMgr.enablePub = true
		}
	} else {
		return func(config *Config) {
		}
	}
}

// * enable indicates whether current peer wants to use pubsub.
//   If enable is false, then this function is no-op.
//   If enable is true, then libp2p will initialize a GossipPubSub instance,
//   and ensure current peer will subscribe to this topic.
//   After that, peer can receive messages from other peers that published to the same topic.
// * topicID is topic name.
func WithSubscribe(enable bool, topicID string) Option {
	if enable {
		return func(config *Config) {
			config.psMgr.subTopic = topicID
			config.psMgr.enableSub = true
		}
	} else {
		return func(config *Config) {
		}
	}
}

func WithLogger(logger logrus.FieldLogger) Option {
	return func(config *Config) {
		config.logger = logger
	}
}

func checkConfig(config *Config) error {
	if config.logger == nil {
		config.logger = log.NewWithModule("p2p")
	}

	if config.localAddr == "" {
		return fmt.Errorf("empty local address")
	}

	return nil
}

func generateConfig(opts ...Option) (*Config, error) {
	conf := &Config{}
	for _, opt := range opts {
		opt(conf)
	}

	if err := checkConfig(conf); err != nil {
		return nil, fmt.Errorf("create p2p: %w", err)
	}

	return conf, nil
}
