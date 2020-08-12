package network

import (
	"time"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
)

type ConnectCallback func(string) error

type MessageHandler func(network.Stream, []byte)

type StreamHandler interface {
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
}

type Network interface {
	// Start start the network service.
	Start() error

	// Stop stop the network service.
	Stop() error

	// Connect connects peer by addr.
	Connect(peer.AddrInfo) error

	// Disconnect peer with id
	Disconnect(string) error

	// SetConnectionCallback sets the callback after connecting
	SetConnectCallback(ConnectCallback)

	// SetMessageHandler sets message handler
	SetMessageHandler(MessageHandler)

	// AsyncSend sends message to peer with peer id.
	AsyncSend(string, []byte) error

	// Send sends message to peer with peer id waiting response
	Send(string, []byte) ([]byte, error)

	// Broadcast message to all node
	Broadcast([]string, []byte) error
}

type PeerHandler interface {
	// get local peer id
	PeerID() string

	// get peer private key
	PrivKey() crypto.PrivKey

	// get peer addr info by peer id
	PeerInfo(string) (peer.AddrInfo, error)

	// get all network peers
	GetPeers() []peer.AddrInfo

	// get local peer addr
	LocalAddr() string

	// get peers num connected
	PeersNum() int

	// check if have an open connection to peer
	IsConnected(peerID string) bool

	// store peer to peer store
	StorePeer(peer.AddrInfo) error

	// searches for a peer with peer id
	FindPeer(string) (peer.AddrInfo, error)
}
