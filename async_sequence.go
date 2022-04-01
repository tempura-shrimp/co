package co

import (
	"sync"
)

type asyncSequence[R any] struct {
	_defaultIterator Iterator[R]
}

func NewAsyncSequence[R any](it Iterator[R]) *asyncSequence[R] {
	return &asyncSequence[R]{_defaultIterator: it}
}

func (a *asyncSequence[R]) defaultIterator() Iterator[R] {
	return a._defaultIterator
}

func (a *asyncSequence[R]) Emitter() <-chan *data[R] {
	return a._defaultIterator.Emitter()
}

type asyncSequenceIterator[T any] struct {
	delegated Iterator[T]

	emitCh        []chan *data[T]
	isEmitRunning bool
	emitMux       sync.Mutex
}

func NewAsyncSequenceIterator[T any](it Iterator[T]) *asyncSequenceIterator[T] {
	return &asyncSequenceIterator[T]{delegated: it, emitCh: make([]chan *data[T], 0)}
}

func (it *asyncSequenceIterator[T]) consumeAny() (any, error) {
	return it.delegated.consume()
}

func (it *asyncSequenceIterator[T]) next() (T, error) {
	it.delegated.preflight()
	return it.delegated.consume()
}

func (it *asyncSequenceIterator[T]) emitData(d *data[T]) {
	it.emitMux.Lock()
	defer it.emitMux.Unlock()

	for _, ch := range it.emitCh {
		SafeSend(ch, d)
	}
}

func (it *asyncSequenceIterator[T]) runEmit() {
	if it.isEmitRunning {
		return
	}
	it.isEmitRunning = true

	SafeGo(func() {
		for it.delegated.preflight() {
			val, err := it.delegated.consume()
			it.emitData(NewDataWith(val, err))
		}
		for _, ch := range it.emitCh {
			SafeClose(ch)
		}
	})
}

func (it *asyncSequenceIterator[T]) Emitter() <-chan *data[T] {
	it.emitMux.Lock()
	defer it.emitMux.Unlock()

	eCh := make(chan *data[T])
	it.emitCh = append(it.emitCh, eCh)

	it.runEmit()
	return eCh
}
