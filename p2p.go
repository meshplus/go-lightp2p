package network

import (
	"context"
	"fmt"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-connmgr"
	crypto2 "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p-core/routing"
	crypto "github.com/libp2p/go-libp2p-crypto"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	ddht "github.com/libp2p/go-libp2p-kad-dht/dual"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const module = "lightp2p"

var _ Network = (*P2P)(nil)

var (
	connectTimeout           = 10 * time.Second
	sendTimeout              = 5 * time.Second
	waitTimeout              = 5 * time.Second
	reusableProtocolIndex    = 0
	nonReusableProtocolIndex = 1
)

type P2P struct {
	config          *Config
	host            host.Host // manage all connections
	streamMng       *streamMgr
	connectCallback ConnectCallback
	messageHandler  MessageHandler
	logger          logrus.FieldLogger
	Routing         routing.Routing

	pingServer *ping.PingService
	ctx    context.Context
	cancel context.CancelFunc
}

func New(options ...Option) (*P2P, error) {
	conf, err := generateConfig(options...)
	if err != nil {
		return nil, fmt.Errorf("generate config: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	opts := []libp2p.Option{
		libp2p.Identity(conf.privKey),
		libp2p.ListenAddrStrings(conf.localAddr),
		//libp2p.Security(libp2ptls.ID, libp2ptls.New),
	}

	if conf.connMgr != nil && conf.connMgr.enabled {
		opts = append(opts, libp2p.ConnectionManager(newConnManager(conf.connMgr)))
	}

	h, err := libp2p.New(ctx, opts...)
	pingServer := ping.NewPingService(h)

	if err != nil {
		cancel()
		return nil, errors.Wrap(err, "failed on create p2p host")
	}

	addrInfos := make([]peer.AddrInfo, 0, len(conf.bootstrap))
	for i, pAddr := range conf.bootstrap {
		addr, err := ma.NewMultiaddr(pAddr)
		if err != nil {
			return nil, errors.Wrapf(err, "failed on create new multi addr %d", i)
		}

		addrInfo, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			return nil, errors.Wrapf(err, "failed on get addr info from multi addr %d", i)
		}

		addrInfos = append(addrInfos, *addrInfo)
	}

	routing, err := ddht.New(ctx, h, dht.BootstrapPeers(addrInfos...))
	if err != nil {
		cancel()
		return nil, errors.Wrap(err, "failed on create dht")
	}

	p2p := &P2P{
		config:    conf,
		host:      h,
		streamMng: newStreamMng(ctx, h, conf.protocolIDs[reusableProtocolIndex], conf.logger),
		logger:    conf.logger,
		Routing:   routing,
		pingServer: pingServer,
		ctx:       ctx,
		cancel:    cancel,
	}

	return p2p, nil
}

func newConnManager(cfg *connMgr) *connmgr.BasicConnMgr {
	if cfg == nil || !cfg.enabled {
		return nil
	}

	return connmgr.NewConnManager(cfg.lo, cfg.hi, cfg.grace)
}

func (p2p *P2P) Ping(peerID string) (<-chan ping.Result, error) {
	peerInfo, err := p2p.FindPeer(peerID)
	if err != nil {
		return nil, errors.Wrap(err, "failed on find peer")
	}

	ch := p2p.pingServer.Ping(p2p.ctx, peerInfo.ID)
	return ch, nil
}

// Start start the network service.
func (p2p *P2P) Start() error {
	p2p.host.SetStreamHandler(p2p.config.protocolIDs[reusableProtocolIndex], p2p.handleNewStreamReusable)
	if len(p2p.config.protocolIDs) > 1 {
		p2p.host.SetStreamHandler(p2p.config.protocolIDs[nonReusableProtocolIndex], p2p.handleNewStream)
	}
	//construct Bootstrap node's peer info
	var peers []peer.AddrInfo
	for _, maAddr := range p2p.config.bootstrap {
		pi, err := AddrToPeerInfo(maAddr)
		if err != nil {
			return err
		}
		peers = append(peers, *pi)
	}
	//if Bootstrap addr has config then connect it
	if len(peers) > 0 {
		err := p2p.BootstrapConnect(p2p.ctx, p2p.host, peers)
		if err != nil {
			p2p.logger.WithFields(logrus.Fields{"module": module, "error": err}).Warn("connect bootstrap peer error")
		}
	}

	if err := p2p.Routing.Bootstrap(p2p.ctx); err != nil {
		return errors.Wrap(err, "failed on bootstrap kad dht")
	}

	p2p.logger.WithFields(logrus.Fields{"module": module}).Info("start p2p success")
	return nil
}

//// BootstrapConnect refer to ipfs bootstrap
//// connect to bootstrap peers concurrently
func (p2p *P2P) BootstrapConnect(ctx context.Context, ph host.Host, peers []peer.AddrInfo) error {
	if len(peers) < 1 {
		return errors.New("not enough bootstrap peers")
	}

	errs := make(chan error, len(peers))
	var wg sync.WaitGroup
	for _, p := range peers {

		// performed asynchronously because when performed synchronously, if
		// one `Connect` call hangs, subsequent calls are more likely to
		// fail/abort due to an expiring context.
		// Also, performed asynchronously for dial speed.

		wg.Add(1)
		go func(p peer.AddrInfo) {
			defer wg.Done()
			fmt.Printf("%s bootstrapping to %s", ph.ID().Pretty(), p.ID.Pretty())

			ph.Peerstore().AddAddrs(p.ID, p.Addrs, peerstore.PermanentAddrTTL)
			if err := ph.Connect(ctx, p); err != nil {
				fmt.Printf("failed to bootstrap with %v: %s", p.ID, err)
				errs <- err
				return
			}
			fmt.Printf("bootstrapDialSuccess with %s", p.ID.Pretty())
			if p2p.connectCallback != nil {
				p2p.connectCallback(p.ID.String())
			}
		}(p)
	}
	wg.Wait()

	// our failure condition is when no connection attempt succeeded.
	// So drain the errs channel, counting the results.
	close(errs)
	count := 0
	var err error
	for err = range errs {
		if err != nil {
			count++
		}
	}
	if count == len(peers) {
		return fmt.Errorf("failed to bootstrap. %s", err)
	}

	return nil
}

// Connect peer.
func (p2p *P2P) Connect(addr peer.AddrInfo) error {
	ctx, cancel := context.WithTimeout(p2p.ctx, connectTimeout)
	defer cancel()

	if err := p2p.host.Connect(ctx, addr); err != nil {
		return err
	}

	p2p.host.Peerstore().AddAddrs(addr.ID, addr.Addrs, peerstore.PermanentAddrTTL)

	if p2p.connectCallback != nil {
		if err := p2p.connectCallback(addr.ID.String()); err != nil {
			return err
		}
	}

	return nil
}

func (p2p *P2P) SetConnectCallback(callback ConnectCallback) {
	p2p.connectCallback = callback
}

func (p2p *P2P) SetMessageHandler(handler MessageHandler) {
	p2p.messageHandler = handler
}

// AsyncSend message to peer with specific id.
func (p2p *P2P) AsyncSend(peerID string, msg []byte) error {
	if _, err := p2p.FindPeer(peerID); err != nil {
		return errors.Wrap(err, "failed on find peer")
	}

	s, err := p2p.streamMng.get(peerID)
	if err != nil {
		return errors.Wrap(err, "failed on get stream")
	}

	if err := p2p.send(s, msg); err != nil {
		return err
	}
	p2p.streamMng.release(s)
	return nil
}

func (p2p *P2P) AsyncSendWithStream(s Stream, msg []byte) error {
	return p2p.send(s.(*stream), msg)
}

func (p2p *P2P) Send(peerID string, msg []byte) ([]byte, error) {
	if _, err := p2p.FindPeer(peerID); err != nil {
		return nil, errors.Wrap(err, "failed on find peer")
	}

	s, err := p2p.streamMng.get(peerID)
	if err != nil {
		return nil, errors.Wrap(err, "failed on get stream")
	}

	if err := p2p.send(s, msg); err != nil {
		return nil, errors.Wrap(err, "failed on send msg")
	}

	defer p2p.streamMng.release(s)
	recvMsg, err := waitMsg(s.stream, waitTimeout)
	if err != nil {
		return nil, err
	}

	return recvMsg.Data, nil
}

func (p2p *P2P) Broadcast(ids []string, msg []byte) error {
	for _, id := range ids {
		if err := p2p.AsyncSend(id, msg); err != nil {
			p2p.logger.WithFields(logrus.Fields{
				"error": err,
				"id":    id,
			}).Error("Async Send message")
			continue
		}
	}

	return nil
}

// Stop stop the network service.
func (p2p *P2P) Stop() error {
	p2p.streamMng.stop()
	p2p.cancel()
	return p2p.host.Close()
}

// AddrToPeerInfo transfer addr to PeerInfo
// addr example: "/ip4/104.236.76.40/tcp/4001/ipfs/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64"
func AddrToPeerInfo(multiAddr string) (*peer.AddrInfo, error) {
	maddr, err := ma.NewMultiaddr(multiAddr)
	if err != nil {
		return nil, err
	}

	return peer.AddrInfoFromP2pAddr(maddr)
}

func (p2p *P2P) Disconnect(peerID string) error {
	pid, err := peer.Decode(peerID)
	if err != nil {
		return errors.Wrap(err, "failed on decode peerID")
	}

	return p2p.host.Network().ClosePeer(pid)
}

func (p2p *P2P) PeerID() string {
	return p2p.host.ID().String()
}

func (p2p *P2P) PrivKey() crypto.PrivKey {
	return p2p.config.privKey
}

func (p2p *P2P) GetPeers() []peer.AddrInfo {
	var peers []peer.AddrInfo

	peersID := p2p.host.Peerstore().Peers()
	for _, peerID := range peersID {
		addrs := p2p.host.Peerstore().Addrs(peerID)
		peers = append(peers, peer.AddrInfo{ID: peerID, Addrs: addrs})
	}

	return peers
}

func (p2p *P2P) LocalAddr() string {
	return p2p.config.localAddr
}

func (p2p *P2P) GetStream(peerID string) (Stream, error) {
	if _, err := p2p.FindPeer(peerID); err != nil {
		return nil, errors.Wrap(err, "failed on find peer")
	}

	pid, err := peer.Decode(peerID)
	if err != nil {
		return nil, errors.Wrap(err, "failed on decode peer id")
	}

	s, err := p2p.host.NewStream(p2p.ctx, pid, p2p.config.protocolIDs[nonReusableProtocolIndex])
	if err != nil {
		return nil, errors.Wrap(err, "failed on create new stream")
	}

	return newStream(s, p2p.config.protocolIDs[nonReusableProtocolIndex], DirOutbound), nil
}

func (p2p *P2P) ReleaseStream(s Stream) {
	stream, ok := s.(*stream)
	if !ok {
		p2p.logger.Error("stream type error")
		return
	}

	if stream.getProtocolID() == p2p.config.protocolIDs[nonReusableProtocolIndex] {
		stream.close()
		return
	}

	if stream.getProtocolID() == p2p.config.protocolIDs[reusableProtocolIndex] {
		if stream.getDirection() == DirOutbound {
			if stream.isValid() {
				p2p.streamMng.release(stream)
			}
		}
	}
}

func (p2p *P2P) StorePeer(addr peer.AddrInfo) error {
	p2p.host.Peerstore().AddAddrs(addr.ID, addr.Addrs, peerstore.AddressTTL)
	return nil
}

func (p2p *P2P) PeerInfo(peerID string) (peer.AddrInfo, error) {
	pid, err := peer.Decode(peerID)
	if err != nil {
		return peer.AddrInfo{}, errors.Wrap(err, "failed on get get peer id from string")
	}

	return p2p.host.Peerstore().PeerInfo(pid), nil
}

func (p2p *P2P) GetRemotePubKey(id peer.ID) (crypto2.PubKey, error) {
	conns := p2p.host.Network().ConnsToPeer(id)

	for _, conn := range conns {
		return conn.RemotePublicKey(), nil
	}

	return nil, fmt.Errorf("get remote pub key: not found")
}

func (p2p *P2P) PeersNum() int {
	return len(p2p.host.Network().Peers())
}

func (p2p *P2P) IsConnected(peerID string) bool {
	return p2p.host.Network().Connectedness(peer.ID(peerID)) == network.Connected
}

func (p2p *P2P) FindPeer(peerID string) (peer.AddrInfo, error) {
	id, err := peer.Decode(peerID)
	if err != nil {
		return peer.AddrInfo{}, fmt.Errorf("failed on decode peer id:%v", err)
	}
	return p2p.Routing.FindPeer(p2p.ctx, id)
}

func (p2p *P2P) Provider(peerID string, passed bool) error {
	_, err := peer.Decode(peerID)
	if err != nil {
		return errors.Wrap(err, "failed on decode peer id")
	}
	ccid, err := cid.Decode(peerID)
	if err != nil {
		return fmt.Errorf("failed on cast cid: %v", err)
	}
	return p2p.Routing.Provide(p2p.ctx, ccid, passed)
}

func (p2p *P2P) FindProvidersAsync(peerID string, i int) (<-chan peer.AddrInfo, error) {
	ccid, err := cid.Decode(peerID)
	if err != nil {
		return nil, fmt.Errorf("failed on cast cid: %v", err)
	}
	peerInfoC := p2p.Routing.FindProvidersAsync(p2p.ctx, ccid, i)
	return peerInfoC, nil
}
