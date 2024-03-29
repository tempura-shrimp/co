package co

import (
	"sync"
	"sync/atomic"

	"go.tempura.ink/co/ds/pool"
	syncx "go.tempura.ink/co/internal/syncx"
)

type seqItem[R any] struct {
	seq uint64
	val R
}

type asyncRoundTrip[R any, E any, T seqItem[R]] struct {
	async AsyncSequenceable[T]
	it    Iterator[T]

	globalSeq uint64
	sourceCh  chan T

	interminCh   chan T
	executorPool *pool.DispatcherPool[*data[E]]
	workerMap    sync.Map
	callbackMap  sync.Map
	runOnce      sync.Once
}

func NewAsyncRoundTrip[R any, E any, T seqItem[R]]() *asyncRoundTrip[R, E, T] {
	ch := make(chan T)
	a := &asyncRoundTrip[R, E, T]{
		async:        FromChanBuffered(ch),
		sourceCh:     ch,
		interminCh:   make(chan T),
		executorPool: pool.NewDispatchPool[*data[E]](512),
	}
	a.it = a.async.iterator()
	a.executorPool.SetCallbackFn(a.receiveCallback)
	a.startListening()
	return a
}

func (a *asyncRoundTrip[R, E, T]) SendItem(val R, callbackFn func(E, error)) *asyncRoundTrip[R, E, T] {
	seq := atomic.AddUint64(&a.globalSeq, 1)
	a.sourceCh <- T(seqItem[R]{seq, val})
	a.callbackMap.Store(seq, callbackFn)
	return a
}

func (a *asyncRoundTrip[R, E, T]) Transform(fn func(AsyncSequenceable[T]) AsyncSequenceable[T]) *asyncRoundTrip[R, E, T] {
	a.async = fn(a.async)
	a.it = a.async.iterator()
	return a
}

func (a *asyncRoundTrip[R, E, T]) SetAsyncSequenceable(async AsyncSequenceable[T]) *asyncRoundTrip[R, E, T] {
	a.async = async
	a.it = a.async.iterator()
	return a
}

func (a *asyncRoundTrip[R, E, T]) startListening() {
	a.runOnce.Do(func() {
		syncx.SafeGo(func() {
			for op := a.it.next(); op.valid; op = a.it.next() {
				a.interminCh <- op.data
			}
		})
	})
}

func (a *asyncRoundTrip[R, E, T]) Complete() *asyncRoundTrip[R, E, T] {
	syncx.SafeClose(a.sourceCh)
	a.executorPool.Wait().Stop()
	syncx.SafeClose(a.interminCh)
	return a
}

func (a *asyncRoundTrip[R, E, T]) Handle(fn func(R) (E, error)) {
	for item := range a.interminCh {
		func(item seqItem[R]) {
			pSeq := a.executorPool.ReserveSeq()
			a.workerMap.Store(pSeq, item.seq)
			a.executorPool.AddJobAt(pSeq, func() *data[E] {
				val, err := fn(item.val)
				return NewDataWith(val, err)
			})
		}(seqItem[R](item))
	}
}

func (a *asyncRoundTrip[R, E, T]) receiveCallback(pSeq uint64, val *data[E]) {
	seq, ok := a.workerMap.Load(pSeq)
	if !ok {
		panic("co / roundTrip: pSeq to seq map not found")
	}
	callbackFn, ok := a.callbackMap.Load(seq.(uint64))
	if !ok {
		panic("co / roundTrip: seq to callback fn map not found")
	}

	func(val *data[E]) {
		syncx.SafeGo(func() {
			callbackFn.(func(E, error))(val.GetValue(), val.GetError())
		})
	}(val)
}
