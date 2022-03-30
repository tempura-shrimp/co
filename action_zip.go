package co

import (
	"sync"
)

type actionZip[R any] struct {
	*actionCombineLatest[R]

	fn           func(*actionZip[R], []any, error, bool)
	updated      []int
	currentIndex int
}

func NewActionZip[R any](its []AnyExecutableIterator) *actionZip[R] {
	return &actionZip[R]{
		actionCombineLatest: NewActionCombineLatest[R](its),
		currentIndex:        1,
	}
}

func (a *actionZip[R]) setFn(fn func(*actionZip[R], []any, error, bool)) *actionZip[R] {
	a.fn = fn
	return a
}

func (z *actionZip[R]) ifReachesToIndexOrEnd(idx int) bool {
	for i := range z.its {
		if z.updated[i] != idx && !z.its[i].hasNext() {
			return false
		}
	}
	return true
}

func (z *actionZip[R]) run() {
	resultChan := make(chan actionAnyResult)

	wg := sync.WaitGroup{}
	wg.Add(len(z.its))

	cond := sync.Cond{}
	for i := range z.its {
		wg.Add(1)

		go func(idx int, seq AnyExecutableIterator) {
			defer wg.Done()

			for i := 0; seq.hasNext(); i++ {
				cond.Wait()

				data, err := seq.exeNextAsAny()
				SafeSend(resultChan, actionAnyResult{idx, data, err})
			}
		}(i, z.its[i])
	}

	latestResults := make([]any, len(z.its))
	go func() {
		for {
			select {
			case result := <-resultChan:
				latestResults[result.index] = result.data
				z.updated[result.index]++

				if !EvertGET(z.updated, 1) {
					continue
				}
				if !z.ifReachesToIndexOrEnd(z.currentIndex) {
					continue
				}
				z.currentIndex++

				rte := z.ifAllSequenceReachesToEnd()
				z.fn(z, latestResults, result.err, rte)

				cond.Broadcast()
				if rte {
					return
				}
			}
		}
	}()

	wg.Wait()
	z.done()
}

func Zip[T1, T2 any](fn func(T1, T2, error, bool), seq1 Concurrently[T1], seq2 Concurrently[T2]) *Action[ActionBulkResult[Type2[T1, T2]]] {
	action := NewActionZip[ActionBulkResult[Type2[T1, T2]]](castToAnyExecutableIterator(seq1.Iterator(), seq2.Iterator())).
		setFn(func(a *actionZip[ActionBulkResult[Type2[T1, T2]]], v []any, err error, b bool) {
			a.listenProgressive(ActionBulkResult[Type2[T1, T2]]{
				Value:        Type2[T1, T2]{CastOrNil[T1](v[0]), CastOrNil[T2](v[1])},
				Err:          err,
				ReachesToEnd: b,
			})
		})

	SafeGo(action.run)
	return action.Action
}

func Zip3[T1, T2, T3 any](fn func(T1, T2, T3, error, bool), seq1 Concurrently[T1], seq2 Concurrently[T2], seq3 Concurrently[T3]) *Action[ActionBulkResult[Type3[T1, T2, T3]]] {
	action := NewActionZip[ActionBulkResult[Type3[T1, T2, T3]]](castToAnyExecutableIterator(seq1.Iterator(), seq2.Iterator())).
		setFn(func(a *actionZip[ActionBulkResult[Type3[T1, T2, T3]]], v []any, err error, b bool) {
			a.listenProgressive(ActionBulkResult[Type3[T1, T2, T3]]{
				Value:        Type3[T1, T2, T3]{CastOrNil[T1](v[0]), CastOrNil[T2](v[1]), CastOrNil[T3](v[2])},
				Err:          err,
				ReachesToEnd: b,
			})
		})

	SafeGo(action.run)
	return action.Action
}
