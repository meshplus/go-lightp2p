package network

import (
	"fmt"
	"time"

	"github.com/pkg/errors"

	ggio "github.com/gogo/protobuf/io"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"
	network_pb "github.com/meshplus/go-lightp2p/pb"
	ma "github.com/multiformats/go-multiaddr"
)

type Direction int

const (
	// DirInbound is for when the remote peer initiated a stream.
	DirInbound = iota
	// DirOutbound is for when the local peer initiated a stream.
	DirOutbound
)

type stream struct {
	direction Direction
	stream    network.Stream
	pid       protocol.ID
	valid     bool
}

func newStream(s network.Stream, pid protocol.ID, dir Direction) *stream {
	return &stream{
		direction: dir,
		stream:    s,
		pid:       pid,
		valid:     true,
	}
}

func (s *stream) close() error {
	return s.stream.Close()
}

func (s *stream) getDirection() Direction {
	return s.direction
}

func (s *stream) getStream() network.Stream {
	return s.stream
}

func (s *stream) getProtocolID() protocol.ID {
	return s.pid
}

func (s *stream) isValid() bool {
	return s.valid
}

func (s *stream) reset() error {
	return s.stream.Reset()
}

func (s *stream) RemotePeerID() string {
	return s.stream.Conn().RemotePeer().String()
}

func (s *stream) RemotePeerAddr() ma.Multiaddr {
	return s.stream.Conn().RemoteMultiaddr()
}

func (s *stream) AsyncSend(msg []byte) error {
	deadline := time.Now().Add(sendTimeout)

	if err := s.getStream().SetWriteDeadline(deadline); err != nil {
		s.valid = false
		return fmt.Errorf("set deadline: %w", err)
	}

	writer := ggio.NewDelimitedWriter(s.getStream())

	if err := writer.WriteMsg(&network_pb.Message{Data: msg}); err != nil {
		s.valid = false
		return fmt.Errorf("write msg: %w", err)
	}

	return nil
}

func (s *stream) Send(msg []byte) ([]byte, error) {
	if err := s.AsyncSend(msg); err != nil {
		return nil, errors.Wrap(err, "failed on send msg")
	}

	recvMsg, err := waitMsg(s.getStream(), waitTimeout)
	if err != nil {
		s.valid = false
		return nil, err
	}

	return recvMsg.Data, nil
}

func (s *stream) Read(timeout time.Duration) ([]byte, error) {
	recvMsg, err := waitMsg(s.getStream(), timeout)
	if err != nil {
		s.valid = false
		return nil, err
	}

	return recvMsg.Data, nil
}
