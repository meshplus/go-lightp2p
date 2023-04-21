package network

import (
	"errors"
	"sync"

	"github.com/sirupsen/logrus"
)

type pool struct {
	logger    logrus.FieldLogger
	lock      sync.Mutex
	resources chan *stream
	factory   func(string) (*stream, error)
	closed    bool
}

func newPool(fn func(string) (*stream, error), logger logrus.FieldLogger, size int) (*pool, error) {
	if size <= 0 {
		return nil, errors.New("pool size too small")
	}

	return &pool{
		logger:    logger,
		resources: make(chan *stream, size),
		factory:   fn,
	}, nil
}

func (p *pool) acquire(peerID string) (*stream, error) {
	select {
	case r, ok := <-p.resources:
		if !ok {
			return nil, errors.New("pool already closed")
		}
		return r, nil
	default:
		return p.factory(peerID)
	}
}

func (p *pool) release(s *stream) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.closed {
		err := s.close()
		if err != nil {
			p.logger.Errorf("stream [%s] close err", s.pid)
		}
		return
	}

	select {
	case p.resources <- s:
	default:
		err := s.close()
		if err != nil {
			p.logger.Errorf("stream [%s] close err", s.pid)
		}
	}
}

func (p *pool) close() {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.closed {
		return
	}

	p.closed = true
	close(p.resources)
	for s := range p.resources {
		err := s.close()
		if err != nil {
			p.logger.Errorf("stream [%s] close err", s.pid)
		}
	}
}
