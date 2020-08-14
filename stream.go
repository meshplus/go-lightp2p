package network

import (
	"fmt"
	"time"

	"github.com/pkg/errors"

	ggio "github.com/gogo/protobuf/io"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/protocol"
	network_pb "github.com/meshplus/go-lightp2p/pb"
	ma "github.com/multiformats/go-multiaddr"
)

type stream struct {
	stream network.Stream
	pid    protocol.ID
}

func newStream(s network.Stream, pid protocol.ID) *stream {
	return &stream{
		stream: s,
		pid:    pid,
	}
}

func (s *stream) close() error {
	return s.stream.Close()
}

func (s *stream) getStream() network.Stream {
	return s.stream
}

func (s *stream) getProtocolID() protocol.ID {
	return s.pid
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
		return fmt.Errorf("set deadline: %w", err)
	}

	writer := ggio.NewDelimitedWriter(s.getStream())

	if err := writer.WriteMsg(&network_pb.Message{Data: msg}); err != nil {
		return fmt.Errorf("write msg: %w", err)
	}

	return nil
}

func (s *stream) Send(msg []byte) ([]byte, error) {
	if err := s.AsyncSend(msg); err != nil {
		return nil, errors.Wrap(err, "failed on send msg")
	}

	recvMsg := waitMsg(s.getStream(), waitTimeout)
	if recvMsg == nil {
		return nil, errors.New("send msg to stream timeout")
	}

	return recvMsg.Data, nil
}

func (s *stream) Read(timeout time.Duration) ([]byte, error) {
	recvMsg := waitMsg(s.getStream(), timeout)
	if recvMsg == nil {
		return nil, errors.New("read msg from stream timeout")
	}

	return recvMsg.Data, nil
}
