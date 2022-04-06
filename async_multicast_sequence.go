package co

import (
	"sync"

	"github.com/tempura-shrimp/co/ds/queue"
	co_sync "github.com/tempura-shrimp/co/sync"
)

type AsyncMulticastSequence[R any] struct {
	previousIterator Iterator[R]
	sourceEnded      bool
	runOnce          sync.Once

	bufferedQueue *queue.MultiReceiverQueue[R]
	bufferWait    *sync.Cond
}

func NewAsyncMulticastSequence[R any](it AsyncSequenceable[R]) *AsyncMulticastSequence[R] {
	a := &AsyncMulticastSequence[R]{
		previousIterator: it.iterator(),
		bufferedQueue:    queue.NewMultiReceiverQueue[R](),
		bufferWait:       sync.NewCond(&sync.Mutex{}),
	}
	a.startListening()
	return a
}

func (a *AsyncMulticastSequence[T]) startListening() {
	a.runOnce.Do(func() {
		co_sync.SafeGo(func() {
			for op, err := a.previousIterator.next(); op.valid; op, err = a.previousIterator.next() {
				if err != nil {
					continue
				}
				co_sync.CondBoardcast(a.bufferWait, func() {
					a.bufferedQueue.Enqueue(op.data)
				})
			}
			co_sync.CondBoardcast(a.bufferWait, func() { a.sourceEnded = true })
		})
	})
}

func (a *AsyncMulticastSequence[R]) Connect() *AsyncMulticastConnector[R] {
	c := &AsyncMulticastConnector[R]{
		AsyncMulticastSequence: a,
	}
	c.asyncSequence = NewAsyncSequence[R](c)
	return c
}

type AsyncMulticastConnector[R any] struct {
	*asyncSequence[R]

	*AsyncMulticastSequence[R]
}

func (a *AsyncMulticastConnector[R]) iterator() Iterator[R] {
	it := &asyncMulticastSequenceIterator[R]{
		AsyncMulticastSequence: a.AsyncMulticastSequence,
		receiver:               a.bufferedQueue.Receiver(),
	}
	it.asyncSequenceIterator = NewAsyncSequenceIterator[R](it)
	return it
}

type asyncMulticastSequenceIterator[R any] struct {
	*asyncSequenceIterator[R]

	*AsyncMulticastSequence[R]
	receiver *queue.QueueReceiver[R]
}

func (it *asyncMulticastSequenceIterator[R]) next() (*Optional[R], error) {
	co_sync.CondWait(it.bufferWait, func() bool {
		return !it.sourceEnded && it.receiver.IsEmpty()
	})

	if it.sourceEnded && it.receiver.IsEmpty() {
		return NewOptionalEmpty[R](), nil
	}

	return OptionalOf(it.receiver.Dequeue()), nil
}
