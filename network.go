package network

import (
	"time"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

type ConnectCallback func(string) error

type MessageHandler func(Stream, []byte)

type Stream interface {
	RemotePeerID() string

	RemotePeerAddr() ma.Multiaddr

	// async send message with stream
	AsyncSend([]byte) error

	// send message with stream
	Send([]byte) ([]byte, error)

	// read message from stream
	Read(time.Duration) ([]byte, error)
}

type StreamHandler interface {
	// opens a new stream to given peer.
	GetStream(peerID string) (Stream, error)

	// release stream
	ReleaseStream(Stream)
}

type Network interface {
	StreamHandler

	PeerHandler

	DHTHandler

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

	// GetRemotePubKey gets remote public key
	GetRemotePubKey(id peer.ID) (crypto.PubKey, error)
}

type DHTHandler interface {
	// searches for a peer with peer id
	FindPeer(string) (peer.AddrInfo, error)

	// Search for peers who are able to provide a given key
	//
	// When count is 0, this method will return an unbounded number of
	// results.
	FindProvidersAsync(string, int) (<-chan peer.AddrInfo, error)

	// Provide adds the given cid to the content routing system. If 'true' is
	// passed, it also announces it, otherwise it is just kept in the local
	// accounting of which objects are being provided.
	Provider(string, bool) error
}
