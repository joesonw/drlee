package server

import "sync"

type replyBoxWatch struct {
	id string
	ch chan RPCResponse
}

type ReplyBox struct {
	deleteCh chan string
	insertCh chan RPCResponse
	watchCh  chan replyBoxWatch
	replies  map[string]replyBoxWatch
	watchMu  *sync.RWMutex
}

func newReplyBox() *ReplyBox {
	b := &ReplyBox{
		deleteCh: make(chan string, 64),
		insertCh: make(chan RPCResponse, 64),
		watchCh:  make(chan replyBoxWatch, 64),
		watchMu:  &sync.RWMutex{},
	}

	go func() {
		for {
			select {

			case id := <-b.deleteCh:
				{
					delete(b.replies, id)
				}
			case res := <-b.insertCh:
				{
					watch, ok := b.replies[res.ID]
					if !ok {
						b.watchMu.Lock()
						ch := make(chan RPCResponse, 1)
						b.replies[res.ID] = replyBoxWatch{
							id: res.ID,
							ch: ch,
						}
						ch <- res
						b.watchMu.Unlock()
					} else {
						delete(b.replies, res.ID)
						watch.ch <- res
					}
				}
			case watch := <-b.watchCh:
				{
					b.watchMu.Lock()
					b.replies[watch.id] = watch
					b.watchMu.Unlock()
				}
			}
		}

	}()

	return b
}

func (b *ReplyBox) Watch(id string) chan RPCResponse {
	b.watchMu.RLock()
	w, ok := b.replies[id]
	b.watchMu.RUnlock()
	if ok {
		return w.ch
	}
	ch := make(chan RPCResponse, 1)
	b.watchCh <- replyBoxWatch{
		id: id,
		ch: ch,
	}
	return ch
}

func (b *ReplyBox) Delete(id string) {
	b.deleteCh <- id
}

func (b *ReplyBox) Insert(res RPCResponse) {
	b.insertCh <- res
}
