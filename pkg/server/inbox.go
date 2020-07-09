package server

import (
	"sync"
	"time"

	coreRPC "github.com/joesonw/drlee/pkg/core/rpc"
	"github.com/joesonw/drlee/pkg/utils"
	"github.com/nsqio/go-diskqueue"
	uuid "github.com/satori/go.uuid"
)

type Inbox struct {
	diskqueue.Interface
	*sync.Mutex
	consumers map[int]chan *coreRPC.Request
}

func newInbox(queue diskqueue.Interface) *Inbox {
	return &Inbox{
		Interface: queue,
		Mutex:     &sync.Mutex{},
		consumers: map[int]chan *coreRPC.Request{},
	}
}

func (inbox *Inbox) Reset() {
	inbox.Lock()
	defer inbox.Unlock()
	for _, ch := range inbox.consumers {
		close(ch)
	}
	inbox.consumers = map[int]chan *coreRPC.Request{}
}

func (inbox *Inbox) Put(req *RPCRequest) error {
	var b []byte
	b, err := utils.MarshalGOB(req)
	if err != nil {
		return err
	}
	return inbox.Interface.Put(b)
}

func (inbox *Inbox) Broadcast(req *RPCRequest) []string {
	var ids []string
	for _, consumer := range inbox.consumers {
		id := uuid.NewV4().String()
		ids = append(ids, id)
		consumer <- &coreRPC.Request{
			ID:         id,
			Name:       req.Name,
			Body:       req.Body,
			NodeName:   req.NodeName,
			IsLoopBack: req.IsLoopBack,
			ExpiresAt:  req.Timestamp.Add(req.Timeout),
		}
	}
	return ids
}

func (inbox *Inbox) NewConsumer(id int) <-chan *coreRPC.Request {
	ch := make(chan *coreRPC.Request, 1)
	consumer := make(chan *coreRPC.Request, 64)
	inbox.Lock()
	inbox.consumers[id] = consumer
	inbox.Unlock()
	read := inbox.ReadChan()
	go func() {
		for {
			select {
			case data := <-read:
				{
					req := &RPCRequest{}
					if err := utils.UnmarshalGOB(data, req); err != nil {
						continue
					}
					var expiresAt time.Time
					if req.Timeout != 0 {
						expiresAt = req.Timestamp.Add(req.Timeout)
						if expiresAt.Before(time.Now()) {
							continue
						}
					}
					ch <- &coreRPC.Request{
						ID:         req.ID,
						Name:       req.Name,
						Body:       req.Body,
						NodeName:   req.NodeName,
						IsLoopBack: req.IsLoopBack,
						ExpiresAt:  expiresAt,
					}
					continue
				}
			case req := <-consumer:
				ch <- req
			}
		}
	}()
	return ch
}
