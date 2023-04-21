package network

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	ma "github.com/multiformats/go-multiaddr"
)

type ConnectCallback func(string) error

type MessageHandler func(Stream, []byte)

type Stream interface {
	RemotePeerID() string

	RemotePeerAddr() ma.Multiaddr

	// AsyncSend async send message with stream
	AsyncSend([]byte) error

	// Send it send message with stream
	Send([]byte) ([]byte, error)

	// Read it read message from stream
	Read(time.Duration) ([]byte, error)
}

type StreamHandler interface {
	// GetStream opens a new stream to given peer.
	GetStream(peerID string) (Stream, error)

	// ReleaseStream release stream
	ReleaseStream(Stream)
}

type Network interface {
	StreamHandler

	PeerHandler

	DHTHandler

	// Start it start the network service.
	Start() error

	// Stop it stop the network service.
	Stop() error

	// Connect connects peer by addr.
	Connect(peer.AddrInfo) error

	// Disconnect peer with id
	Disconnect(string) error

	// SetConnectCallback SetConnectionCallback sets the callback after connecting
	SetConnectCallback(ConnectCallback)

	// SetMessageHandler sets message handler
	SetMessageHandler(MessageHandler)

	// AsyncSend sends message to peer with peer id.
	AsyncSend(string, []byte) error

	// Ping pings target peer.
	Ping(ctx context.Context, peerID string) (<-chan ping.Result, error)

	// Send sends message to peer with peer id waiting response
	Send(string, []byte) ([]byte, error)

	// Broadcast message to all node
	Broadcast([]string, []byte) error
}

type PeerHandler interface {
	// PeerID get local peer id
	PeerID() string

	// PrivKey get peer private key
	PrivKey() crypto.PrivKey

	// PeerInfo get peer addr info by peer id
	PeerInfo(string) (peer.AddrInfo, error)

	// GetPeers get all network peers
	GetPeers() []peer.AddrInfo

	// LocalAddr get local peer addr
	LocalAddr() string

	// PeersNum get peers num connected
	PeersNum() int

	// IsConnected check if it has an open connection to peer
	IsConnected(peerID string) bool

	// StorePeer store peer to peer store
	StorePeer(peer.AddrInfo) error

	// GetRemotePubKey gets remote public key
	GetRemotePubKey(id peer.ID) (crypto.PubKey, error)
}

type DHTHandler interface {
	// FindPeer searches for a peer with peer id
	FindPeer(string) (peer.AddrInfo, error)

	// FindProvidersAsync Search for peers who are able to provide a given key
	//
	// When count is 0, this method will return an unbounded number of
	// results.
	FindProvidersAsync(string, int) (<-chan peer.AddrInfo, error)

	// Provider Provide adds the given cid to the content routing system. If 'true' is
	// passed, it also announces it, otherwise it is just kept in the local
	// accounting of which objects are being provided.
	Provider(string, bool) error
}
