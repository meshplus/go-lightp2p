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
	timeout    *timeout
	pools      sync.Map
}

func newStreamMng(ctx context.Context, host host.Host, protocolID protocol.ID, logger logrus.FieldLogger, timeout *timeout) *streamMgr {
	return &streamMgr{
		ctx:        ctx,
		protocolID: protocolID,
		host:       host,
		logger:     logger,
		timeout:    timeout,
		pools:      sync.Map{},
	}
}

func (mng *streamMgr) get(peerID string) (*stream, error) {
	loadPool, ok := mng.pools.Load(peerID)
	if !ok {
		pool, err := newPool(mng.newStream, mng.logger, maxStreamNumPerConn)
		if err != nil {
			return nil, errors.Wrap(err, "failed on create new pool")
		}
		mng.pools.Store(peerID, pool)
		loadPool = pool
	}
	s, err := loadPool.(*pool).acquire(peerID)
	if err != nil {
		return nil, errors.Wrap(err, "failed on acquire stream")
	}

	return s, nil
}

func (mng *streamMgr) release(stream *stream) {
	peerID := stream.RemotePeerID()

	loadPool, ok := mng.pools.Load(peerID)
	if !ok {
		mng.logger.WithFields(logrus.Fields{"peer id": peerID, "err": "failed on get pool"}).Warn("failed on release stream")
		return
	}

	loadPool.(*pool).release(stream)
}

func (mng *streamMgr) newStream(peerID string) (*stream, error) {
	pid, err := peer.Decode(peerID)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(mng.ctx, newStreamTimeout)
	defer cancel()
	mng.logger.WithFields(logrus.Fields{"protocol id": mng.protocolID}).Debug("new stream")
	s, err := mng.host.NewStream(ctx, pid, mng.protocolID)
	if err != nil {
		return nil, errors.Wrap(err, "failed on creat new stream")
	}

	return newStream(s, mng.protocolID, DirOutbound, mng.timeout), nil
}

func (mng *streamMgr) stop() {
	mng.pools.Range(func(key, value interface{}) bool {
		pool := value.(*pool)
		pool.close()
		return true
	})
}
