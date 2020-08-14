package network

import (
	"context"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	newStreamTimeout    = 5 * time.Second
	maxStreamNumPerConn = 16
)

type streamMgr struct {
	ctx        context.Context
	protocolID protocol.ID
	host       host.Host
	logger     logrus.FieldLogger

	pools map[string]*Pool
	sync.RWMutex
}

func newStreamMng(ctx context.Context, host host.Host, protocolID protocol.ID, logger logrus.FieldLogger) *streamMgr {
	return &streamMgr{
		ctx:        ctx,
		protocolID: protocolID,
		host:       host,
		logger:     logger,
		pools:      make(map[string]*Pool),
	}
}

func (mng *streamMgr) get(peerID string) (*stream, error) {
	mng.Lock()
	defer mng.Unlock()

	_, ok := mng.pools[peerID]
	if !ok {
		pool, err := newPool(mng.newStream, mng.logger, maxStreamNumPerConn)
		if err != nil {
			return nil, errors.Wrap(err, "failed on create new pool")
		}

		mng.pools[peerID] = pool
	}

	s, err := mng.pools[peerID].Acquire(peerID)
	if err != nil {
		return nil, errors.Wrap(err, "failed on acquire stream")
	}

	return s, nil
}

func (mng *streamMgr) release(stream *stream) {
	peerID := stream.RemotePeerID()
	mng.Lock()
	defer mng.Unlock()

	pool, ok := mng.pools[peerID]
	if !ok {
		mng.logger.WithFields(logrus.Fields{"peer id": peerID, "err": "failed on get pool"}).Warn("failed on release stream")
		return
	}

	pool.Release(stream)
}

func (mng *streamMgr) remove(peerID string) {
	mng.Lock()
	defer mng.Unlock()
	pool, ok := mng.pools[peerID]
	if !ok {
		return
	}

	pool.Close()
	delete(mng.pools, peerID)
}

func (mng *streamMgr) newStream(peerID string) (*stream, error) {
	pid, err := peer.Decode(peerID)
	ctx, cancel := context.WithTimeout(mng.ctx, newStreamTimeout)
	defer cancel()
	mng.logger.WithFields(logrus.Fields{"protocol id": mng.protocolID}).Info("new stream")
	s, err := mng.host.NewStream(ctx, pid, mng.protocolID)
	if err != nil {
		return nil, errors.Wrap(err, "failed on creat new stream")
	}

	return newStream(s, mng.protocolID, DirOutbound), nil
}

func (mng *streamMgr) stop() {
	mng.Lock()
	defer mng.Unlock()

	for _, pool := range mng.pools {
		pool.Close()
	}
}
