package pool

import (
	"sync"
	"sync/atomic"

	co_sync "github.com/tempura-shrimp/co/sync"
)

func NewDispatchPool[K any](maxWorkers int) *DispatcherPool[K] {
	p := &DispatcherPool[K]{
		quitCh:         make(chan bool),
		doneCh:         make(chan *jobDone[K]),
		idleDispatcher: int32(maxWorkers),

		workerCond: sync.NewCond(&sync.Mutex{}),
		jobQueue:   NewQueue[*job[K]](),
	}

	p.pool.New = func() any {
		return NewDispatcher(&p.doneCh)
	}

	p.startListening()
	return p
}

type DispatcherPool[K any] struct {
	pool           sync.Pool
	workerCond     *sync.Cond
	idleDispatcher int32

	doneCh chan *jobDone[K]
	doneWG sync.WaitGroup

	callbackFn func(id uint64, val K)

	quit   bool
	quitCh chan bool

	jobQueue *Queue[*job[K]]

	seq uint64
}

func (p *DispatcherPool[K]) startListening() {
	co_sync.SafeGo(func() {
		for {
			select {
			case <-p.quitCh:
				return

			case data := <-p.doneCh:
				co_sync.CondSignal(p.workerCond, func() {
					p.pool.Put(data.workerRef)
					atomic.AddInt32(&p.idleDispatcher, 1)
				})

				if p.callbackFn != nil {
					co_sync.SafeGo(func() {
						p.callbackFn(data.seq, data.val)
					})
				}
				p.doneWG.Done()
			}
		}
	})

	co_sync.SafeGo(func() {
		for {
			co_sync.CondWait(p.workerCond, func() bool {
				return !p.quit && (p.jobQueue.Len() == 0 || p.idleDispatcher == 0)
			})
			if p.quit {
				return
			}

			atomic.AddInt32(&p.idleDispatcher, -1)

			w := p.pool.Get().(*Dispatcher[K])
			w.trigger(p.jobQueue.Dequeue())
		}
	})
}

func (p *DispatcherPool[K]) SetCallbackFn(fn func(uint64, K)) *DispatcherPool[K] {
	p.callbackFn = fn
	return p
}

func (p *DispatcherPool[K]) AddJob(fn func() K) uint64 {
	id := atomic.AddUint64(&p.seq, 1)

	co_sync.CondSignal(p.workerCond, func() {
		p.jobQueue.Enqueue(&job[K]{fn: fn, seq: id})
	})

	p.doneWG.Add(1)
	return id
}

func (p *DispatcherPool[K]) Wait() {
	p.doneWG.Wait()
}

func (p *DispatcherPool[K]) Stop() {
	p.quitCh <- true
	p.quit = true
	p.workerCond.Broadcast()
}

type Dispatcher[K any] struct {
	doneCh *chan *jobDone[K]
}

func NewDispatcher[K any](doneCh *chan *jobDone[K]) *Dispatcher[K] {
	w := Dispatcher[K]{
		doneCh: doneCh,
	}
	return &w
}

func (w *Dispatcher[K]) trigger(load *job[K]) {
	co_sync.SafeGo(func() {
		*(w.doneCh) <- &jobDone[K]{val: load.fn(), seq: load.seq, workerRef: w}
	})
}
