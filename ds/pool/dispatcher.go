package pool

import (
	"sync"
	"sync/atomic"

	"go.tempura.ink/co/ds/queue"
	syncx "go.tempura.ink/co/internal/syncx"
)

func NewDispatchPool[K any](maxWorkers int) *DispatcherPool[K] {
	p := &DispatcherPool[K]{
		poolBasic:      newPoolBasic[K](),
		idleDispatcher: int32(maxWorkers),

		workerCond: syncx.NewCondCh(&sync.Mutex{}),
		jobQueue:   queue.NewQueue[*job[K]](),
	}

	p.pool.New = func() any {
		return NewDispatcher(p)
	}

	p.startListening()
	return p
}

type DispatcherPool[K any] struct {
	*poolBasic[K]

	pool           sync.Pool
	workerCond     *syncx.CondCh
	idleDispatcher int32

	callbackFn func(id uint64, val K)
	quit       bool

	jobQueue *queue.Queue[*job[K]]
}

func (p *DispatcherPool[K]) startListening() {
	syncx.SafeGo(func() {
		for {
			p.workerCond.Waitify(func() bool {
				return !p.quit && (p.jobQueue.Len() == 0 || p.idleDispatcher == 0)
			}, func() {
				p.idleDispatcher--
			})

			if p.quit {
				return
			}

			w := p.pool.Get().(*Dispatcher[K])
			w.trigger(p.jobQueue.Dequeue())
		}
	})
}

func (p *DispatcherPool[K]) SetCallbackFn(fn func(uint64, K)) *DispatcherPool[K] {
	p.callbackFn = fn
	return p
}

func (p *DispatcherPool[K]) AddJobAt(seq uint64, fn func() K) uint64 {
	p.workerCond.Signalify(func() {
		load := p.jobCache.Get().(*job[K])
		load.fn, load.seq = fn, seq
		p.jobQueue.Enqueue(load)
	})

	p.doneWG.Add(1)
	return seq
}

func (p *DispatcherPool[K]) AddJob(fn func() K) uint64 {
	id := atomic.AddUint64(&p.seq, 1)
	return p.AddJobAt(id, fn)
}

func (p *DispatcherPool[K]) Wait() *DispatcherPool[K] {
	p.doneWG.Wait()
	return p
}

func (p *DispatcherPool[K]) Stop() {
	p.workerCond.Broadcastify(func() {
		p.quit = true
	})
}

type Dispatcher[K any] struct {
	pool *DispatcherPool[K]
}

func NewDispatcher[K any](p *DispatcherPool[K]) *Dispatcher[K] {
	w := Dispatcher[K]{
		pool: p,
	}
	return &w
}

func (w *Dispatcher[K]) trigger(load *job[K]) {
	go func() {
		defer func() {
			w.pool.workerCond.Signalify(func() {
				w.pool.pool.Put(w)
				w.pool.jobCache.Put(load)
				w.pool.idleDispatcher++
			})
			w.pool.doneWG.Done()
		}()

		val := load.fn()
		if w.pool.callbackFn != nil {
			syncx.SafeGo(func() {
				w.pool.callbackFn(load.seq, val)
			})
		}
	}()
}
