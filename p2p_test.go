package network

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	protocolID1 string = "/test/1.0.0" // magic protocol
	protocolID2 string = "/test/2.0.0" // magic protocol
)

func TestP2P_Connect(t *testing.T) {
	p1, addr1 := generateNetwork(t, 6001)
	p2, addr2 := generateNetwork(t, 6002)

	err := p1.Connect(addr2)
	assert.Nil(t, err)

	err = p2.Connect(addr1)
	assert.Nil(t, err)
}

func TestP2p_ConnectWithNullIDStore(t *testing.T) {
	p1, addr1 := generateNetwork(t, 6003)
	p2, addr2 := generateNetwork(t, 6004)

	err := p1.Connect(addr2)
	assert.Nil(t, err)
	err = p2.Connect(addr1)
	assert.Nil(t, err)
}

func TestP2P_MultiStreamSend(t *testing.T) {
	p1, addr1 := generateNetwork(t, 6005)
	p2, addr2 := generateNetwork(t, 6006)
	fmt.Println(addr1)
	msg := []byte("hello world")
	ack := []byte("ack")
	p2.SetMessageHandler(func(s Stream, data []byte) {
		fmt.Println("p2 received:", string(data))
		err := s.AsyncSend(ack)
		assert.Nil(t, err)
	})

	err := p1.Start()
	assert.Nil(t, err)
	err = p2.Start()
	assert.Nil(t, err)

	err = p1.Connect(addr2)
	assert.Nil(t, err)
	err = p2.Connect(addr1)
	assert.Nil(t, err)
	testStreamNum := 100
	var wg sync.WaitGroup

	send := func(wg *sync.WaitGroup) {
		defer wg.Done()
		resp, err := p1.Send(p2.PeerID(), msg)
		assert.Nil(t, err)
		assert.EqualValues(t, resp, ack)
	}

	for i := 0; i < testStreamNum; i++ {
		wg.Add(1)
		go send(&wg)
	}

	wg.Wait()
}

func TestP2P_MultiStreamAsyncSend(t *testing.T) {
	p1, addr1 := generateNetwork(t, 6005)
	p2, addr2 := generateNetwork(t, 6006)
	fmt.Println(addr1)
	msg := []byte("hello world")
	p2.SetMessageHandler(func(s Stream, data []byte) {
		fmt.Println("p2 received:", string(data))
	})

	err := p1.Start()
	assert.Nil(t, err)
	err = p2.Start()
	assert.Nil(t, err)

	err = p1.Connect(addr2)
	assert.Nil(t, err)
	err = p2.Connect(addr1)
	assert.Nil(t, err)
	testStreamNum := 100
	var wg sync.WaitGroup

	send := func(wg *sync.WaitGroup) {
		defer wg.Done()
		err := p1.AsyncSend(p2.PeerID(), msg)
		assert.Nil(t, err)
	}

	for i := 0; i < testStreamNum; i++ {
		wg.Add(1)
		go send(&wg)
	}

	wg.Wait()
}

func TestP2P_MultiStreamSendWithStream(t *testing.T) {
	p1, addr1 := generateNetwork(t, 6005)
	p2, addr2 := generateNetwork(t, 6006)
	msg := []byte("hello world")
	ack := []byte("ack")
	p2.SetMessageHandler(func(s Stream, data []byte) {
		fmt.Println("p2 received:", string(data))
		err := s.AsyncSend(ack)
		assert.Nil(t, err)
	})

	err := p1.Start()
	assert.Nil(t, err)
	err = p2.Start()
	assert.Nil(t, err)

	err = p1.Connect(addr2)
	assert.Nil(t, err)
	err = p2.Connect(addr1)
	assert.Nil(t, err)
	testStreamNum := 100
	var wg sync.WaitGroup

	send := func(wg *sync.WaitGroup) {
		defer wg.Done()
		s, err := p1.GetStream(p2.PeerID(), true)
		assert.Nil(t, err)
		defer p1.ReleaseStream(s)
		resp, err := s.Send(msg)
		assert.Nil(t, err)
		assert.EqualValues(t, resp, ack)
	}

	for i := 0; i < testStreamNum; i++ {
		wg.Add(1)
		go send(&wg)
	}

	wg.Wait()
}

func TestP2P_MultiStreamSendWithAsyncStream(t *testing.T) {
	p1, addr1 := generateNetwork(t, 6005)
	p2, addr2 := generateNetwork(t, 6006)
	fmt.Println(addr1)
	msg := []byte("hello world")
	p2.SetMessageHandler(func(s Stream, data []byte) {
		fmt.Println("p2 received:", string(data))
	})

	err := p1.Start()
	assert.Nil(t, err)
	err = p2.Start()
	assert.Nil(t, err)

	err = p1.Connect(addr2)
	assert.Nil(t, err)
	err = p2.Connect(addr1)
	assert.Nil(t, err)
	testStreamNum := 100
	var wg sync.WaitGroup

	send := func(wg *sync.WaitGroup) {
		defer wg.Done()
		s, err := p1.GetStream(p2.PeerID(), true)
		assert.Nil(t, err)
		defer p1.ReleaseStream(s)
		err = s.AsyncSend(msg)
		assert.Nil(t, err)
	}

	for i := 0; i < testStreamNum; i++ {
		wg.Add(1)
		go send(&wg)
	}

	wg.Wait()
}

func TestP2P_AsyncSend(t *testing.T) {
	p1, addr1 := generateNetwork(t, 6005)
	p2, addr2 := generateNetwork(t, 6006)

	msg := []byte("hello")

	ch := make(chan struct{})

	p2.SetMessageHandler(func(s Stream, data []byte) {
		fmt.Println("receive:", string(data))
		assert.EqualValues(t, msg, data)
		close(ch)
	})

	err := p1.Start()
	assert.Nil(t, err)
	err = p2.Start()
	assert.Nil(t, err)

	err = p1.Connect(addr2)
	assert.Nil(t, err)
	err = p2.Connect(addr1)
	assert.Nil(t, err)

	err = p1.AsyncSend(p2.PeerID(), msg)
	assert.Nil(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	select {
	case <-ch:
		return
	case <-ctx.Done():
		assert.Error(t, fmt.Errorf("timeout"))
		return
	}
}

func TestSendWithNonReusableStream(t *testing.T) {
	p1, addr1 := generateNetwork(t, 6005)
	p2, addr2 := generateNetwork(t, 6006)
	msg := []byte("hello world")
	ack := []byte("ack")
	p2.SetMessageHandler(func(s Stream, data []byte) {
		fmt.Println("p2 received:", string(data))
		err := s.AsyncSend(ack)
		assert.Nil(t, err)
		err = s.AsyncSend(ack)
		assert.Nil(t, err)
		p2.ReleaseStream(s)
	})

	err := p1.Start()
	assert.Nil(t, err)
	err = p2.Start()
	assert.Nil(t, err)

	err = p1.Connect(addr2)
	assert.Nil(t, err)
	err = p2.Connect(addr1)
	assert.Nil(t, err)

	s, err := p1.GetStream(p2.PeerID(), false)
	assert.Nil(t, err)
	defer p1.ReleaseStream(s)
	err = s.AsyncSend(msg)
	assert.Nil(t, err)
	for {
		rawMsg, err := s.Read(2 * time.Second)
		if err != nil {
			if err != io.EOF {
				t.Fatal(err)
				return
			}
			return
		}
		assert.Equal(t,ack,rawMsg)
	}
}

func TestSendWithReusableStream(t *testing.T) {
	p1, addr1 := generateNetwork(t, 6005)
	p2, addr2 := generateNetwork(t, 6006)
	msg := []byte("hello world")
	ack := []byte("ack")
	p2.SetMessageHandler(func(s Stream, data []byte) {
		fmt.Println("p2 received:", string(data))
		assert.Equal(t,msg,data)
		err := s.AsyncSend(ack)
		assert.Nil(t, err)
		p2.ReleaseStream(s)
	})

	err := p1.Start()
	assert.Nil(t, err)
	err = p2.Start()
	assert.Nil(t, err)

	err = p1.Connect(addr2)
	assert.Nil(t, err)
	err = p2.Connect(addr1)
	assert.Nil(t, err)

	s, err := p1.GetStream(p2.PeerID(), true)
	assert.Nil(t, err)
	defer p1.ReleaseStream(s)
	err = s.AsyncSend(msg)
	assert.Nil(t, err)
	rawMsg, err := s.Read(2 * time.Second)
	assert.Nil(t, err)
	assert.Equal(t,ack,rawMsg)
}

func TestP2p_MultiSend(t *testing.T) {
	p1, addr1 := generateNetwork(t, 6007)
	p2, addr2 := generateNetwork(t, 6008)

	err := p1.Start()
	assert.Nil(t, err)
	err = p2.Start()
	assert.Nil(t, err)

	err = p1.Connect(addr2)
	assert.Nil(t, err)
	err = p2.Connect(addr1)
	assert.Nil(t, err)

	N := 50
	msg := []byte("hello")
	count := 0
	ch := make(chan struct{})

	p2.SetMessageHandler(func(s Stream, data []byte) {
		assert.EqualValues(t, msg, data)
		count++
		if count == N {
			close(ch)
			return
		}
	})

	go func() {
		for i := 0; i < N; i++ {
			time.Sleep(200 * time.Microsecond)
			err = p1.AsyncSend(p2.PeerID(), msg)
			assert.Nil(t, err)
		}

	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	select {
	case <-ch:
		return
	case <-ctx.Done():
		assert.Error(t, fmt.Errorf("timeout"))
	}
}

func TestP2P_FindPeer(t *testing.T) {
	bootstrap1, bsAddr1 := generateNetwork(t, 6007)
	bootstrap2, bsAddr2 := generateNetwork(t, 6008)
	bootstrap3, bsAddr3 := generateNetwork(t, 6011)
	_, bsAddr4 := generateNetwork(t, 6014)

	err := bootstrap1.Start()
	require.Nil(t, err)
	err = bootstrap2.Start()
	require.Nil(t, err)
	err = bootstrap3.Start()
	require.Nil(t, err)

	err = bootstrap1.Connect(bsAddr2)
	require.Nil(t, err)

	err = bootstrap2.Connect(bsAddr3)
	require.Nil(t, err)

	addrs1, err := peer.AddrInfoToP2pAddrs(&bsAddr1)
	require.Nil(t, err)
	addrs2, err := peer.AddrInfoToP2pAddrs(&bsAddr2)
	require.Nil(t, err)
	addrs3, err := peer.AddrInfoToP2pAddrs(&bsAddr3)
	require.Nil(t, err)

	var bs1 = []string{addrs1[0].String()}
	var bs2 = []string{addrs2[0].String()}
	var bs3 = []string{addrs3[0].String()}
	dht1, s1, id1 := generateNetworkWithDHT(t, 6009, bs1)
	dht2, s2, id2 := generateNetworkWithDHT(t, 6010, bs2)
	dht3, s3, id3 := generateNetworkWithDHT(t, 6012, bs3)

	err = dht1.Start()
	require.Nil(t, err)
	err = dht2.Start()
	require.Nil(t, err)
	err = dht3.Start()
	require.Nil(t, err)

	err = dht3.Provider(bsAddr4.ID.String(), true)
	require.Nil(t, err)

	findPeer1, err := dht1.FindPeer(id2.String())
	require.Nil(t, err)
	require.Equal(t, s2.String(), findPeer1.String())

	findPeer2, err := dht2.FindPeer(id1.String())
	require.Nil(t, err)
	require.Equal(t, s1.String(), findPeer2.String())

	findPeer3, err := dht1.FindPeer(id3.String())
	require.Nil(t, err)
	require.Equal(t, s3.String(), findPeer3.String())

	findPeer4, err := dht3.FindPeer(id1.String())
	require.Nil(t, err)
	require.Equal(t, s1.String(), findPeer4.String())

	findPeerC, err := dht1.FindProvidersAsync(bsAddr4.ID.String(), 1)
	require.Nil(t, err)
	findPeer5 := <-findPeerC
	require.Equal(t, s3.String(), findPeer5.String())
}

func generateNetwork(t *testing.T, port int) (Network, peer.AddrInfo) {
	privKey, pubKey, err := crypto.GenerateECDSAKeyPair(rand.Reader)
	assert.Nil(t, err)

	pid1, err := peer.IDFromPublicKey(pubKey)
	assert.Nil(t, err)
	addr := fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port)
	maddr := fmt.Sprintf("%s/p2p/%s", addr, pid1)
	p2p, err := New(
		WithLocalAddr(addr),
		WithPrivateKey(privKey),
		WithProtocolIDs([]string{protocolID1, protocolID2}),
	)
	assert.Nil(t, err)

	multiaddr, err := ma.NewMultiaddr(maddr)
	require.Nil(t, err)
	addrInfo, err := peer.AddrInfoFromP2pAddr(multiaddr)
	require.Nil(t, err)

	return p2p, *addrInfo
}

func generateNetworkWithDHT(t *testing.T, port int, bootstrap []string) (Network, *peer.AddrInfo, peer.ID) {
	privKey, pubKey, err := crypto.GenerateECDSAKeyPair(rand.Reader)
	assert.Nil(t, err)

	pid1, err := peer.IDFromPublicKey(pubKey)
	assert.Nil(t, err)
	addr := fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port)
	maddr := fmt.Sprintf("%s/p2p/%s", addr, pid1)
	p2p, err := New(
		WithLocalAddr(addr),
		WithPrivateKey(privKey),
		WithBootstrap(bootstrap),
		WithProtocolIDs([]string{protocolID1, protocolID2}),
	)
	assert.Nil(t, err)

	multiaddr, err := ma.NewMultiaddr(maddr)
	require.Nil(t, err)
	addrInfo, err := peer.AddrInfoFromP2pAddr(multiaddr)
	require.Nil(t, err)

	return p2p, addrInfo, pid1
}
