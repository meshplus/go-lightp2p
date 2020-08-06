package network

import (
	"errors"
	"sync"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/sirupsen/logrus"
)

type Pool struct {
	logger    logrus.FieldLogger
	lock      sync.Mutex
	resources chan network.Stream
	factory   func(string) (network.Stream, error)
	closed    bool
}

func newPool(fn func(string) (network.Stream, error), logger logrus.FieldLogger, size int) (*Pool, error) {
	if size <= 0 {
		return nil, errors.New("pool size too small")
	}

	return &Pool{
		logger:    logger,
		resources: make(chan network.Stream, size),
		factory:   fn,
	}, nil
}

func (p *Pool) Acquire(peerID string) (network.Stream, error) {
	select {
	case r, ok := <-p.resources:
		p.logger.Info("Acquire:", "Shared Resource")
		if !ok {
			return nil, errors.New("pool already closed")
		}
		return r, nil
	default:
		p.logger.Info("Acquire:", "New Resource")
		return p.factory(peerID)
	}
}

func (p *Pool) Release(r network.Stream) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.closed {
		r.Close()
		return
	}

	select {
	case p.resources <- r:
		p.logger.Info("Release:", "In Queue")
	default:
		p.logger.Info("Release:", "Closing")
		r.Close()
	}
}

func (p *Pool) Close() {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.closed {
		return
	}

	p.closed = true

	close(p.resources)

	for r := range p.resources {
		r.Close()
	}
}
