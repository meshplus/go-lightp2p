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
	p.logger.Debugf("//////start get resource:%s", peerID)
	select {
	case r, ok := <-p.resources:
		if !ok {
			return nil, errors.New("pool already closed")
		}
		p.logger.Debugf("/////end get resource:%s", peerID)
		return r, nil
	default:
		p.logger.Debugf("/////default====end get resource:%s", peerID)
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
	default:
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
