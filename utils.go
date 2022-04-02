package co

import (
	"fmt"
	"sync"

	"golang.org/x/exp/constraints"
)

func NewLockedMutex() *sync.Mutex {
	mux := &sync.Mutex{}
	mux.Lock()
	return mux
}

func ReadBoolChan(ch chan bool) (bool, bool) {
	select {
	case x, ok := <-ch:
		if ok {
			return x, ok
		} else {
			return false, false
		}
	default:
		return false, true
	}
}

func SafeSend[T any](ch chan T, value T) (closed bool) {
	defer func() {
		if recover() != nil {
			closed = true
			fmt.Printf("channel %+v send out %+v failed\n", ch, value)
		}
	}()

	ch <- value
	return false
}

func SafeClose[T any](ch chan T) (closed bool) {
	defer func() {
		if recover() != nil {
			closed = false
		}
	}()

	close(ch)
	return true
}

func SafeGo(fn func()) {
	go func() {
		defer func() {
			recover()
		}()
		fn()
	}()
}

func Copy[T any](v *T) *T {
	v2 := *v
	return &v2
}

func CastOrNil[T any](el any) T {
	if el == nil {
		return *new(T)
	}
	return el.(T)
}

func EvertGET[T constraints.Ordered](ele []T, target T) bool {
	for _, e := range ele {
		if e <= target {
			return false
		}
	}
	return true
}

func EvertET[T comparable](ele []T, target T) bool {
	for _, e := range ele {
		if e == target {
			return false
		}
	}
	return true
}
