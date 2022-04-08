package co

import (
	"time"

	co_sync "github.com/tempura-shrimp/co/sync"
)

type AsyncInterval[R any] struct {
	*asyncSequence[R]

	period int
	ended  co_sync.AtomicBool
}

func Interval(period int) *AsyncInterval[int] {
	a := &AsyncInterval[int]{
		period: period,
	}
	a.asyncSequence = NewAsyncSequence[int](a)
	return a
}

func (a *AsyncInterval[R]) Done() *AsyncInterval[R] {
	a.ended.Set(true)
	return a
}

func (a *AsyncInterval[R]) iterator() Iterator[R] {
	it := &asyncIntervalIterator[R]{
		AsyncInterval: a,
		timer:         time.NewTicker(time.Duration(a.period) * time.Millisecond),
	}
	it.asyncSequenceIterator = NewAsyncSequenceIterator[R](it)
	return it
}

type asyncIntervalIterator[R any] struct {
	*asyncSequenceIterator[R]

	*AsyncInterval[R]

	timer *time.Ticker
}

func (it *asyncIntervalIterator[R]) cleanUp() (*Optional[R], error) {
	it.timer.Stop()
	return NewOptionalEmpty[R](), nil
}

func (it *asyncIntervalIterator[R]) next() (*Optional[R], error) {
	if it.ended.Get() {
		return it.cleanUp()
	}

	<-it.timer.C
	return OptionalOf(*new(R)), nil
}
