package network

import (
	"time"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
)

type ConnectCallback func(string) error

type MessageHandler func(network.Stream, []byte)

type Network interface {
	// Start start the network service.
	Start() error

	// Stop stop the network service.
	Stop() error

	// Connect connects peer by ID.
	Connect(string) error

	// Disconnect peer with id
	Disconnect(string) error

	// SetConnectionCallback sets the callback after connecting
	SetConnectCallback(ConnectCallback)

	// SetMessageHandler sets message handler
	SetMessageHandler(MessageHandler)

	// AsyncSend sends message to peer with peer id.
	AsyncSend(string, []byte) error

	// Send sends message waiting response
	Send(string, []byte) ([]byte, error)

	// Broadcast message to all node
	Broadcast([]string, []byte) error

	// get peer new stream true:reusable stream false:non reusable stream
	GetStream(string, bool) (network.Stream, error)

	// Send message using existed stream
	AsyncSendWithStream(network.Stream, []byte) error

	// Send message using existed stream
	SendWithStream(network.Stream, []byte) ([]byte, error)

	// read message from stream
	ReadFromStream(network.Stream, time.Duration) ([]byte, error)

	// release stream
	ReleaseStream(network.Stream)

	// get local peer id
	PeerID() string

	// get peer private key
	PrivKey() crypto.PrivKey

	// get peer addr info by peer id
	PeerInfo(peerID string) ([]string, error)

	// get all network peers
	Peers() []peer.AddrInfo

	// get local peer addr
	LocalAddr() string

	// get peers num in peer store
	PeerNum() int

	// store peer to peer store
	StorePeer(peerID string, addr string) error

	// searches for a peer with peer id
	FindPeer(string) (string, error)
}
