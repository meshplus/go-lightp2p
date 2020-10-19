package network

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p-core/host"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/pkg/errors"
)

type pubsubMgr struct {
	ps       *pubsub.PubSub
	pubTopic *pubsub.Topic
	subTopic *pubsub.Topic
	sub      *pubsub.Subscription
}

func initializePubSub(ctx context.Context, h host.Host, conf *Config) (*pubsubMgr, error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("initialize PubSub panic: %v\n", r)
		}
	}()

	ps, err := pubsub.NewGossipSub(ctx, h)
	psMng := &pubsubMgr{
		ps: ps,
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed on create GossipSub")
	}

	if conf.psMgr.enableSub {
		psMng.subTopic, err = ps.Join(conf.psMgr.subTopic)
		if err != nil {
			return nil, errors.Wrapf(err, "sub: failed on join topic: %v", conf.psMgr.subTopic)
		}
		psMng.sub, err = psMng.subTopic.Subscribe()
		if err != nil {
			return nil, errors.Wrapf(err, "sub: failed on subscribe topic: %v\n", conf.psMgr.subTopic)
		}
	}

	if conf.psMgr.enablePub {
		// cannot join the same topic twice
		if conf.psMgr.subTopic != conf.psMgr.pubTopic {
			psMng.pubTopic, err = ps.Join(conf.psMgr.pubTopic)
			if err != nil {
				return nil, errors.Wrapf(err, "pub: failed on join topic: %v\n", conf.psMgr.pubTopic)
			}
		} else {
			psMng.pubTopic = psMng.subTopic
		}
	}

	return psMng, nil
}
