package network

import (
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMockP2P_Send(t *testing.T) {
	peerIDs := []string{"1", "2"}
	mockHostManager := GenMockHostManager(peerIDs)
	peer1, err := NewMockP2P(peerIDs[0], mockHostManager, nil)
	assert.Nil(t, err)
	peer2, err := NewMockP2P(peerIDs[1], mockHostManager, nil)
	assert.Nil(t, err)
	err = peer1.Start()
	assert.Nil(t, err)
	err = peer2.Start()
	assert.Nil(t, err)

	peer1.SetMessageHandler(func(s Stream, msg []byte) {
		err = peer1.AsyncSendWithStream(s, msg)
		assert.Nil(t, err)
	})

	msg := []byte("test")
	res, err := peer2.Send(peer1.PeerID(), msg)
	assert.Nil(t, err)
	assert.Equal(t, msg, res)
}

func TestMockP2P_AsyncSend(t *testing.T) {
	peerIDs := []string{"1", "2"}
	mockHostManager := GenMockHostManager(peerIDs)
	peer1, err := NewMockP2P(peerIDs[0], mockHostManager, nil)
	assert.Nil(t, err)
	peer2, err := NewMockP2P(peerIDs[1], mockHostManager, nil)
	assert.Nil(t, err)
	err = peer1.Start()
	assert.Nil(t, err)
	err = peer2.Start()
	assert.Nil(t, err)

	data := []byte("test")
	peer1.SetMessageHandler(func(s Stream, msg []byte) {
		assert.Equal(t, string(data), string(msg))
	})
	peer2.SetMessageHandler(func(s Stream, msg []byte) {
		assert.Equal(t, string(data), string(msg))
	})

	err = peer2.AsyncSend(peer1.PeerID(), data)
	assert.Nil(t, err)

	err = peer1.AsyncSend(peer2.PeerID(), data)
	assert.Nil(t, err)
}

func TestMockP2P_AsyncSendWithStream(t *testing.T) {
	peerIDs := []string{"1", "2"}
	mockHostManager := GenMockHostManager(peerIDs)
	peer1, err := NewMockP2P(peerIDs[0], mockHostManager, nil)
	assert.Nil(t, err)
	peer2, err := NewMockP2P(peerIDs[1], mockHostManager, nil)
	assert.Nil(t, err)
	err = peer1.Start()
	assert.Nil(t, err)
	err = peer2.Start()
	assert.Nil(t, err)

	data := genTestMessages(100)

	peer1.SetMessageHandler(func(s Stream, msg []byte) {
		err = peer1.AsyncSendWithStream(s, msg)
		assert.Nil(t, err)
		for _, v := range data {
			err = peer1.AsyncSendWithStream(s, v)
			assert.Nil(t, err)
		}
	})

	stream, err := peer2.GetStream(peer1.PeerID(), false)
	assert.Nil(t, err)

	reqMsg := []byte("test")
	res, err := peer2.SendWithStream(stream, reqMsg)
	assert.Nil(t, err)
	assert.Equal(t, reqMsg, res)

	for _, v := range data {
		d, err := peer2.ReadFromStream(stream, 5*time.Second)
		assert.Nil(t, err)
		assert.Equal(t, string(v), string(d))
	}
}

func TestMockP2P_Broadcast(t *testing.T) {
	receiveNum := 100

	sendID := "0"
	peerIDs := []string{sendID}
	for i := 1; i <= receiveNum; i++ {
		peerIDs = append(peerIDs, strconv.Itoa(i))
	}

	mockHostManager := GenMockHostManager(peerIDs)
	sender, err := NewMockP2P(peerIDs[0], mockHostManager, nil)
	assert.Nil(t, err)
	err = sender.Start()
	assert.Nil(t, err)

	data := []byte("test")

	var receivers []*MockP2P
	for i := 1; i <= receiveNum; i++ {
		peer, err := NewMockP2P(peerIDs[i], mockHostManager, nil)
		assert.Nil(t, err)
		err = peer.Start()
		assert.Nil(t, err)

		peer.SetMessageHandler(func(s Stream, msg []byte) {
			assert.Equal(t, string(data), string(msg))
		})

		receivers = append(receivers, peer)
	}

	err = sender.Broadcast(peerIDs[1:], data)
	assert.Nil(t, err)

	err = sender.Broadcast(peerIDs, data)
	assert.Nil(t, err)
}

func TestMockP2P_ParallelSend(t *testing.T) {
	receiveNum := 100

	sendID := "0"
	peerIDs := []string{sendID}
	for i := 1; i <= receiveNum; i++ {
		peerIDs = append(peerIDs, strconv.Itoa(i))
	}

	mockHostManager := GenMockHostManager(peerIDs)
	sender, err := NewMockP2P(peerIDs[0], mockHostManager, nil)
	assert.Nil(t, err)
	err = sender.Start()
	assert.Nil(t, err)

	data := genTestMessages(1000)

	var receivers []*MockP2P
	for i := 1; i <= receiveNum; i++ {
		peer, err := NewMockP2P(peerIDs[i], mockHostManager, nil)
		assert.Nil(t, err)
		err = peer.Start()
		assert.Nil(t, err)
		receivers = append(receivers, peer)

		peer.SetMessageHandler(func(s Stream, msg []byte) {
			err = sender.AsyncSendWithStream(s, msg)
			assert.Nil(t, err)
			for _, v := range data {
				err = sender.AsyncSendWithStream(s, v)
				assert.Nil(t, err)
			}
		})
	}

	wg := sync.WaitGroup{}
	wg.Add(receiveNum)
	send := func(p *MockP2P) {
		stream, err := sender.GetStream(p.PeerID(), false)
		assert.Nil(t, err)

		reqMsg := []byte("test")
		res, err := sender.SendWithStream(stream, reqMsg)
		assert.Nil(t, err)
		assert.Equal(t, reqMsg, res)

		for _, v := range data {
			d, err := sender.ReadFromStream(stream, 5*time.Second)
			assert.Nil(t, err)
			assert.Equal(t, string(v), string(d))
		}
		wg.Done()
	}
	for _, v := range receivers {
		go send(v)
	}
	wg.Wait()
}

func TestMockP2P_ParallelReceive(t *testing.T) {
	senderNum := 100

	receiverID := "0"
	peerIDs := []string{receiverID}
	for i := 1; i <= senderNum; i++ {
		peerIDs = append(peerIDs, strconv.Itoa(i))
	}

	mockHostManager := GenMockHostManager(peerIDs)
	receiver, err := NewMockP2P(peerIDs[0], mockHostManager, nil)
	assert.Nil(t, err)
	err = receiver.Start()
	assert.Nil(t, err)

	var senders []*MockP2P
	for i := 1; i <= senderNum; i++ {
		peer, err := NewMockP2P(peerIDs[i], mockHostManager, nil)
		assert.Nil(t, err)
		err = peer.Start()
		assert.Nil(t, err)
		senders = append(senders, peer)
	}

	data := genTestMessages(1000)

	receiver.SetMessageHandler(func(s Stream, msg []byte) {
		err = receiver.AsyncSendWithStream(s, msg)
		assert.Nil(t, err)
		for _, v := range data {
			err = receiver.AsyncSendWithStream(s, v)
			assert.Nil(t, err)
		}
	})

	wg := sync.WaitGroup{}
	wg.Add(senderNum)
	send := func(p *MockP2P) {
		stream, err := p.GetStream(receiver.PeerID(), false)
		assert.Nil(t, err)

		reqMsg := []byte("test")
		res, err := p.SendWithStream(stream, reqMsg)
		assert.Nil(t, err)
		assert.Equal(t, reqMsg, res)

		for _, v := range data {
			d, err := p.ReadFromStream(stream, 5*time.Second)
			assert.Nil(t, err)
			assert.Equal(t, string(v), string(d))
		}
		wg.Done()
	}
	for _, v := range senders {
		go send(v)
	}
	wg.Wait()
}

func genTestMessages(size int) [][]byte {
	msgList := make([][]byte, size)
	for i := 1; i <= size; i++ {

		msgList = append(msgList, []byte(strconv.Itoa(i)))
	}
	return msgList
}
