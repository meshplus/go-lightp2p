package network

import (
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	crypto "github.com/libp2p/go-libp2p-crypto"
	network_pb "github.com/meshplus/go-lightp2p/pb"
	ma "github.com/multiformats/go-multiaddr"
)

type ConnectCallback func(*peer.AddrInfo) error

type MessageHandler func(network.Stream, *network_pb.Message)

type Network interface {
	// Start start the network service.
	Start(bootstrapAddrs map[string]ma.Multiaddr) error

	// Stop stop the network service.
	Stop() error

	// Connect connects peer by ID.
	Connect(*peer.AddrInfo) error

	// Disconnect peer with id
	Disconnect(*peer.AddrInfo) error

	// SetConnectionCallback sets the callback after connecting
	SetConnectCallback(ConnectCallback)

	// SetMessageHandler sets message handler
	SetMessageHandler(MessageHandler)

	// AsyncSend sends message to peer with peer info.
	AsyncSend(*peer.AddrInfo, *network_pb.Message) error

	// Send message using existed stream
	AsyncSendWithStream(network.Stream, *network_pb.Message) error

	// Send sends message waiting response
	Send(*peer.AddrInfo, *network_pb.Message) (*network_pb.Message, error)

	// Send message using existed stream
	SendWithStream(network.Stream, *network_pb.Message) (*network_pb.Message, error)

	// Broadcast message to all node
	Broadcast([]*peer.AddrInfo, *network_pb.Message) error

	PeerID() string

	PrivKey() crypto.PrivKey

	PeerStore() peerstore.Peerstore

	Peers() []peer.AddrInfo

	LocalAddr() string

	GetStream(pid peer.ID) (network.Stream, error)
//todo: storepeer
}
