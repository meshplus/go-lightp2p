package network

import (
	"errors"
	"sync"

	"github.com/sirupsen/logrus"
)

type Pool struct {
	logger    logrus.FieldLogger
	lock      sync.Mutex
	resources chan *stream
	factory   func(string) (*stream, error)
	closed    bool
}

func newPool(fn func(string) (*stream, error), logger logrus.FieldLogger, size int) (*Pool, error) {
	if size <= 0 {
		return nil, errors.New("pool size too small")
	}

	return &Pool{
		logger:    logger,
		resources: make(chan *stream, size),
		factory:   fn,
	}, nil
}

func (p *Pool) Acquire(peerID string) (*stream, error) {
	select {
	case r, ok := <-p.resources:
		p.logger.Debug("Acquire:", "Shared Resource")
		if !ok {
			return nil, errors.New("pool already closed")
		}
		return r, nil
	default:
		p.logger.Debug("Acquire:", "New Resource")
		return p.factory(peerID)
	}
}

func (p *Pool) Release(s *stream) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.closed {
		s.close()
		return
	}

	select {
	case p.resources <- s:
		p.logger.Debug("Release:", "In Queue")
	default:
		p.logger.Debug("Release:", "Closing")
		s.close()
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
		r.close()
	}
}
