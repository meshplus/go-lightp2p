package network

import (
	"context"
	"fmt"
	"io"
	"time"

	ggio "github.com/gogo/protobuf/io"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/pkg/errors"

	network_pb "github.com/meshplus/go-lightp2p/pb"
)

func (p2p *P2P) handleMessage(s *stream, reader ggio.ReadCloser) error {
	msg := &network_pb.Message{}
	if err := reader.ReadMsg(msg); err != nil {
		if err != io.EOF {
			if err := s.reset(); err != nil {
				p2p.logger.WithField("error", err).Error("Reset stream")
			}

			return errors.Wrap(err, "failed on read msg")
		}

		return nil
	}

	if p2p.messageHandler != nil {
		p2p.messageHandler(s, msg.Data)
	}

	return nil
}

// handle newly connected stream
func (p2p *P2P) handleNewStreamReusable(s network.Stream) {
	if err := s.SetReadDeadline(time.Time{}); err != nil {
		p2p.logger.WithField("error", err).Error("Set stream read deadline")
		return
	}

	reader := ggio.NewDelimitedReader(s, network.MessageSizeMax)
	for {
		stream := newStream(s, p2p.config.protocolIDs[reusableProtocolIndex], DirInbound)
		msg := &network_pb.Message{}
		if err := reader.ReadMsg(msg); err != nil {
			if err != io.EOF {
				if err := stream.reset(); err != nil {
					p2p.logger.WithField("error", err).Error("Reset stream")
				}

			}

			return
		}

		if p2p.messageHandler != nil {
			p2p.messageHandler(stream, msg.Data)
		}

	}
}

func (p2p *P2P) handleNewStream(s network.Stream) {
	if err := s.SetReadDeadline(time.Time{}); err != nil {
		p2p.logger.WithField("error", err).Error("Set stream read deadline")
		return
	}

	reader := ggio.NewDelimitedReader(s, network.MessageSizeMax)

	p2p.handleMessage(newStream(s, p2p.config.protocolIDs[nonReusableProtocolIndex], DirInbound), reader)
}

// waitMsg wait the incoming messages within time duration.
func waitMsg(stream io.Reader, timeout time.Duration) (*network_pb.Message, error) {
	reader := ggio.NewDelimitedReader(stream, network.MessageSizeMax)

	ch := make(chan error)
	msg := &network_pb.Message{}
	go func() {
		if err := reader.ReadMsg(msg); err != nil {
			ch <- err
		} else {
			ch <- nil
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	select {
	case r := <-ch:
		cancel()
		return msg, r
	case <-ctx.Done():
		cancel()
		return nil, errors.New("wait msg timeout")
	}
}

func (p2p *P2P) send(s *stream, msg []byte) error {
	deadline := time.Now().Add(sendTimeout)

	if err := s.getStream().SetWriteDeadline(deadline); err != nil {
		return fmt.Errorf("set deadline: %w", err)
	}

	writer := ggio.NewDelimitedWriter(s.getStream())

	if err := writer.WriteMsg(&network_pb.Message{Data: msg}); err != nil {
		return fmt.Errorf("write msg: %w", err)
	}

	return nil
}
