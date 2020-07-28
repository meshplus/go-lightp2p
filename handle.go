package network

import (
	"context"
	"fmt"
	"io"
	"time"

	ggio "github.com/gogo/protobuf/io"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/meshplus/bitxhub-kit/network/pb"

	network_pb "github.com/meshplus/go-lightp2p/pb"
)

func (p2p *P2P) handleMessage(s network.Stream, reader ggio.ReadCloser) {
	msg := &network_pb.Message{}
	if err := reader.ReadMsg(msg); err != nil {
		if err != io.EOF {
			if err := s.Reset(); err != nil {
				p2p.logger.WithField("error", err).Error("Reset stream")
			}
		}

		return
	}

	if p2p.messageHandler != nil {
		p2p.messageHandler(s, msg.Data)
	}
}

// handle newly connected stream
func (p2p *P2P) handleNewStreamReusable(s network.Stream) {
	if err := s.SetReadDeadline(time.Time{}); err != nil {
		p2p.logger.WithField("error", err).Error("Set stream read deadline")
		return
	}

	reader := ggio.NewDelimitedReader(s, network.MessageSizeMax)
	for {
		p2p.handleMessage(s, reader)
	}
}

func (p2p *P2P) handleNewStream(s network.Stream) {
	if err := s.SetReadDeadline(time.Time{}); err != nil {
		p2p.logger.WithField("error", err).Error("Set stream read deadline")
		return
	}

	reader := ggio.NewDelimitedReader(s, network.MessageSizeMax)
	p2p.handleMessage(s, reader)
}

// waitMsg wait the incoming messages within time duration.
func waitMsg(stream io.Reader, timeout time.Duration) *network_pb.Message {
	reader := ggio.NewDelimitedReader(stream, network.MessageSizeMax)

	ch := make(chan *network_pb.Message)

	go func() {
		msg := &network_pb.Message{}
		if err := reader.ReadMsg(msg); err == nil {
			ch <- msg
		} else {
			ch <- nil
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	select {
	case r := <-ch:
		cancel()
		return r
	case <-ctx.Done():
		cancel()
		return nil
	}
}

func (p2p *P2P) send(s network.Stream, msg []byte) error {
	deadline := time.Now().Add(sendTimeout)

	if err := s.SetWriteDeadline(deadline); err != nil {
		return fmt.Errorf("set deadline: %w", err)
	}

	writer := ggio.NewDelimitedWriter(s)

	if err := writer.WriteMsg(&pb.Message{Data: msg}); err != nil {
		return fmt.Errorf("write msg: %w", err)
	}

	return nil
}
