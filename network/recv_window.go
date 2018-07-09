package network

import (
	"sync"

	"github.com/perlin-network/noise/protobuf"
	"github.com/pkg/errors"
)

type RecvWindow struct {
	sync.Mutex

	size int
	buffer *RingBuffer
	messageNonce uint64
}

func NewRecvWindow(size int) *RecvWindow {
	return &RecvWindow {
		size: size,
		buffer: NewRingBuffer(size),
		messageNonce: 1,
	}
}

func (w *RecvWindow) Update(n *Network) error {
	ready := make([]*protobuf.Message, 0)

	w.Lock()
	i := 0
	for ; i < w.size; i++ {
		cursor := w.buffer.Index(i)
		if *cursor == nil {
			break
		}
		ready = append(ready, (*cursor).(*protobuf.Message))
		*cursor = nil
	}
	if i > 0 && i < w.size {
		w.buffer.MoveForward(i)
	}
	w.messageNonce += uint64(i)
	w.Unlock()

	//glog.Infof("Sending %d messages", len(ready))

	for _, msg := range ready {
		select {
		case n.RecvQueue <- msg:
		default:
			return errors.New("recv queue is full")
			//glog.Errorf("recv queue full, dropping messages")
		}
	}

	return nil
}

func (w *RecvWindow) Input(msg *protobuf.Message) error {
	w.Lock()
	defer w.Unlock()

	offset := int(msg.MessageNonce - w.messageNonce)

	if offset < 0 || offset >= w.size {
		return errors.Errorf("Local message nonce is %d while received %d", w.messageNonce, msg.MessageNonce)
	}

	*w.buffer.Index(offset) = msg
	return nil
}
