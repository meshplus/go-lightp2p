package network

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/meshplus/bitxhub-kit/log"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
)

var (
	ErrMockP2PNotSupport = errors.New("mock p2p not support")

	ErrPeerNotExist = errors.New("peer id not exist")
)

const queueSize = 100

type MockP2P struct {
	host      *mockHost
	receiveCh chan *mockMsg

	messageHandler MessageHandler
	logger         logrus.FieldLogger
}

type mockMsg struct {
	stream *mockStream
	data   []byte
}

type MockHostManager struct {
	peers    []string
	connects map[string]chan *mockMsg
}

type mockHost struct {
	peerID   string
	connects map[string]chan *mockMsg
}

func NewMockP2P(peerID string, mockHostManager *MockHostManager, logger logrus.FieldLogger) (*MockP2P, error) {
	_, exist := mockHostManager.connects[peerID]
	if !exist {
		return nil, errors.New("the local address must be in MockHostManager")
	}
	if logger == nil {
		logger = log.NewWithModule("mock_p2p")
	}
	filteredConnects := make(map[string]chan *mockMsg, len(mockHostManager.connects)-1)
	for id := range mockHostManager.connects {
		if id != peerID {
			filteredConnects[id] = mockHostManager.connects[id]
		}
	}
	return &MockP2P{
		host: &mockHost{
			peerID:   peerID,
			connects: filteredConnects,
		},
		receiveCh:      mockHostManager.connects[peerID],
		messageHandler: nil,
		logger:         logger,
	}, nil
}

func GenMockHostManager(peers []string) *MockHostManager {
	filterMap := make(map[string]bool, len(peers))
	filteredPeers := make([]string, len(peers))
	for _, id := range peers {
		_, exist := filterMap[id]
		if !exist {
			filterMap[id] = true
			filteredPeers = append(filteredPeers, id)
		}
	}
	connects := make(map[string]chan *mockMsg, len(peers))
	for _, id := range filteredPeers {
		connects[id] = make(chan *mockMsg, queueSize)
	}
	return &MockHostManager{
		peers:    filteredPeers,
		connects: connects,
	}
}

type mockStream struct {
	host        *mockHost
	localPeer   string
	remotePeer  string
	sendCh      chan *mockMsg
	receiveCh   chan *mockMsg
	lock        *sync.RWMutex
	isConnected bool
}

func (s *mockStream) RemotePeerID() string {
	return s.remotePeer
}

func (s *mockStream) RemotePeerAddr() ma.Multiaddr {
	panic(ErrMockP2PNotSupport)
}

func (s *mockStream) AsyncSend(msg []byte) error {
	connect, exist := s.host.connects[s.remotePeer]
	if !exist {
		return errors.New(fmt.Sprintf("remote peer [%s] not exist", s.remotePeer))
	}
	msgCopy := make([]byte, len(msg))
	copy(msgCopy, msg)
	data := &mockMsg{
		stream: &mockStream{
			localPeer:   s.remotePeer,
			remotePeer:  s.localPeer,
			sendCh:      s.receiveCh,
			receiveCh:   s.sendCh,
			lock:        &sync.RWMutex{},
			isConnected: true,
		},
		data: msgCopy,
	}
	s.lock.Lock()
	if s.isConnected {
		s.sendCh <- data
	} else {
		connect <- data
	}
	s.isConnected = true
	s.lock.Unlock()
	return nil
}

func (s *mockStream) Send(msg []byte) ([]byte, error) {
	connect, exist := s.host.connects[s.remotePeer]
	if !exist {
		return nil, errors.New(fmt.Sprintf("remote peer [%s] not exist", s.remotePeer))
	}
	msgCopy := make([]byte, len(msg))
	copy(msgCopy, msg)
	data := &mockMsg{
		stream: &mockStream{
			localPeer:   s.remotePeer,
			remotePeer:  s.localPeer,
			sendCh:      s.receiveCh,
			receiveCh:   s.sendCh,
			lock:        &sync.RWMutex{},
			isConnected: true,
		},
		data: msgCopy,
	}
	s.lock.Lock()
	if s.isConnected {
		s.sendCh <- data
	} else {
		connect <- data
	}
	s.isConnected = true
	s.lock.Unlock()
	res := <-s.receiveCh
	return res.data, nil
}

func (s *mockStream) Read(timeout time.Duration) ([]byte, error) {
	_, exist := s.host.connects[s.remotePeer]
	if !exist {
		return nil, errors.New(fmt.Sprintf("remote peer [%s] not exist", s.remotePeer))
	}
	select {
	case res := <-s.receiveCh:
		return res.data, nil
	case <-time.After(timeout):
		return nil, errors.New("timeout")
	}
}

func (m *MockP2P) Start() error {
	go func() {
		for {
			select {
			case msg := <-m.receiveCh:
				msg.stream.host = m.host
				go m.messageHandler(msg.stream, msg.data)
			}
		}
	}()
	return nil
}

func (m *MockP2P) Stop() error {
	panic(ErrMockP2PNotSupport)
}

func (m *MockP2P) Connect(addr peer.AddrInfo) error {
	panic(ErrMockP2PNotSupport)
}

func (m *MockP2P) Disconnect(string) error {
	panic(ErrMockP2PNotSupport)
}

func (m *MockP2P) SetConnectCallback(callback ConnectCallback) {}

func (m *MockP2P) SetMessageHandler(handler MessageHandler) {
	m.messageHandler = handler
}

func (m *MockP2P) AsyncSend(peerID string, msg []byte) error {
	connect, exist := m.host.connects[peerID]
	if !exist {
		return ErrPeerNotExist
	}
	msgCopy := make([]byte, len(msg))
	copy(msgCopy, msg)
	connect <- &mockMsg{
		stream: &mockStream{
			localPeer:   peerID,
			remotePeer:  m.PeerID(),
			sendCh:      m.receiveCh,
			receiveCh:   connect,
			lock:        &sync.RWMutex{},
			isConnected: true,
		},
		data: msgCopy,
	}
	return nil
}

func (m *MockP2P) Send(peerID string, msg []byte) ([]byte, error) {
	connect, exist := m.host.connects[peerID]
	if !exist {
		return nil, ErrPeerNotExist
	}
	msgCopy := make([]byte, len(msg))
	copy(msgCopy, msg)
	sendCh := make(chan *mockMsg)
	receiveCh := make(chan *mockMsg)
	connect <- &mockMsg{
		stream: &mockStream{
			localPeer:   peerID,
			remotePeer:  m.PeerID(),
			sendCh:      receiveCh,
			receiveCh:   sendCh,
			lock:        &sync.RWMutex{},
			isConnected: true,
		},
		data: msgCopy,
	}
	res := <-receiveCh
	return res.data, nil
}

func (m *MockP2P) Broadcast(peerIDs []string, msg []byte) error {
	for _, id := range peerIDs {
		err := m.AsyncSend(id, msg)
		if err != nil {
			m.logger.WithFields(logrus.Fields{
				"error": err,
				"id":    id,
			}).Error("Async Send message")
			continue
		}
	}
	return nil
}

func (m *MockP2P) GetStream(peerID string, reusable bool) (Stream, error) {
	_, exist := m.host.connects[peerID]
	if !exist {
		return nil, ErrPeerNotExist
	}
	if reusable {
		return nil, errors.New("not support reusable stream")
	}
	sendCh := make(chan *mockMsg)
	receiveCh := make(chan *mockMsg)
	stream := &mockStream{
		host:        m.host,
		localPeer:   m.PeerID(),
		remotePeer:  peerID,
		sendCh:      sendCh,
		receiveCh:   receiveCh,
		lock:        &sync.RWMutex{},
		isConnected: false,
	}
	return stream, nil
}

func (m *MockP2P) ReleaseStream(s Stream) {
	stream := s.(*mockStream)
	close(stream.sendCh)
	close(stream.receiveCh)
}

func (m *MockP2P) PeerID() string {
	return m.host.peerID
}

func (m *MockP2P) PrivKey() crypto.PrivKey {
	panic(ErrMockP2PNotSupport)
}

func (m *MockP2P) PeerInfo(peerID string) (peer.AddrInfo, error) {
	panic(ErrMockP2PNotSupport)
}

func (m *MockP2P) GetPeers() []peer.AddrInfo {
	var peers []peer.AddrInfo
	for id := range m.host.connects {
		peers = append(peers, peer.AddrInfo{
			ID: peer.ID(id),
		})
	}
	return peers
}

func (m *MockP2P) LocalAddr() string {
	panic(ErrMockP2PNotSupport)
}

func (m *MockP2P) PeersNum() int {
	return len(m.host.connects)
}

func (m *MockP2P) IsConnected(peerID string) bool {
	_, exist := m.host.connects[peerID]
	return exist
}

func (m *MockP2P) StorePeer(peer.AddrInfo) error {
	panic(ErrMockP2PNotSupport)
}

func (m *MockP2P) GetRemotePubKey(id peer.ID) (crypto.PubKey, error) {
	panic(ErrMockP2PNotSupport)
}

func (m *MockP2P) FindPeer(peerID string) (peer.AddrInfo, error) {
	panic(ErrMockP2PNotSupport)
}

func (m *MockP2P) FindProvidersAsync(peerID string, i int) (<-chan peer.AddrInfo, error) {
	panic(ErrMockP2PNotSupport)
}

func (m *MockP2P) Provider(peerID string, passed bool) error {
	panic(ErrMockP2PNotSupport)
}
