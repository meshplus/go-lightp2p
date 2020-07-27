package network

import (
	"time"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	network_pb "github.com/meshplus/go-lightp2p/pb"
	ma "github.com/multiformats/go-multiaddr"
)

type ConnectCallback func(string) error

type MessageHandler func(network.Stream, []byte)

type Network interface {
	// Start start the network service.
	Start() error

	// Stop stop the network service.
	Stop() error

	// Connect connects peer by ID.
	Connect(*peer.AddrInfo) error

	// Disconnect peer with id
	Disconnect(string) error

	// SetConnectionCallback sets the callback after connecting
	SetConnectCallback(ConnectCallback)

	// SetMessageHandler sets message handler
	SetMessageHandler(MessageHandler)

	// AsyncSend sends message to peer with peer info.
	AsyncSend(string, []byte) error

	// Send message using existed stream
	AsyncSendWithStream(network.Stream, *network_pb.Message) error

	// Send sends message waiting response
	Send(string, *network_pb.Message) (*network_pb.Message, error)

	// Send message using existed stream
	SendWithStream(network.Stream, *network_pb.Message) (*network_pb.Message, error)

	// read message from stream
	ReadFromStream(network.Stream, time.Duration) (*network_pb.Message, error)

	// Broadcast message to all node
	Broadcast([]string, *network_pb.Message) error

	// get local peer id
	PeerID() string

	// get peer private key
	PrivKey() crypto.PrivKey

	// get peer addr info by peer id
	PeerInfo(peerID string) (*peer.AddrInfo, error)

	// get all network peers
	Peers() []peer.AddrInfo

	// get local peer addr
	LocalAddr() string

	// get peer new stream
	GetStream(pid peer.ID) (network.Stream, error)

	// get peers num in peer store
	PeerNum() int

	// store peer to peer store
	StorePeer(peerID string, addr ma.Multiaddr) error

	FindPeer(string) (string, error)
}
