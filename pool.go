package network

import (
	"errors"
	"log"
	"sync"

	"github.com/libp2p/go-libp2p-core/network"
)

type Pool struct {
	lock      sync.Mutex
	resources chan network.Stream
	factory   func(string) (network.Stream, error)
	closed    bool
}

func newPool(fn func(string) (network.Stream, error), size int) (*Pool, error) {
	if size <= 0 {
		return nil, errors.New("pool size too small")
	}

	return &Pool{
		resources: make(chan network.Stream, size),
		factory:   fn,
	}, nil
}

func (p *Pool) Acquire(peerID string) (network.Stream, error) {
	select {
	case r, ok := <-p.resources:
		log.Println("Acquire:", "Shared Resource")
		if !ok {
			return nil, errors.New("pool already closed")
		}
		return r, nil
	default:
		log.Println("Acquire:", "New Resource")
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
		log.Println("Release:", "In Queue")
	default:
		log.Println("Release:", "Closing")
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
