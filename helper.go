package network

import (
	network_pb "github.com/meshplus/go-lightp2p/pb"
)

func Message(data []byte) *network_pb.Message {
	return &network_pb.Message{
		Data: data,
	}
}
