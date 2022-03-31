package co

type ActionMode int

const (
	ActionModeData ActionMode = iota
	ActionModeChan
	ActionModeFn
)

type Action[E any] struct {
	externalFn        func(E)
	externalCh        chan E
	externalCloseChan chan bool
	externalData      []E

	actionMode ActionMode
	closeChan  chan bool
}

func NewAction[E any]() *Action[E] {
	return &Action[E]{
		externalCh:        make(chan E),
		externalData:      make([]E, 0),
		externalCloseChan: make(chan bool),
		closeChan:         make(chan bool),
	}
}

func (a *Action[E]) AsChan() (chan E, chan bool) {
	a.actionMode = ActionModeChan
	return a.externalCh, a.externalCloseChan
}

func (a *Action[E]) AsFn(fn func(E)) *Action[E] {
	a.actionMode = ActionModeFn
	a.externalFn = fn
	return a
}

func (a *Action[E]) AsData() *Action[E] {
	a.actionMode = ActionModeData
	return a
}
func (a *Action[E]) GetData() []E {
	a.wait()
	return a.externalData
}

func (a *Action[E]) PeakData() E {
	a.wait()
	if len(a.externalData) == 0 {
		return *new(E)
	}
	return a.externalData[0]
}

func (a *Action[E]) wait() {
	<-a.closeChan
}

func (a *Action[E]) listenProgressive(e E) {
	switch a.actionMode {
	case ActionModeFn:
		a.externalFn(e)
	case ActionModeChan:
		a.externalCh <- e
	case ActionModeData:
		a.externalData = append(a.externalData, e)
	}
}

func (a *Action[E]) listenBulk(e []E) {
	switch a.actionMode {
	case ActionModeFn:
		for i := range e {
			a.externalFn(e[i])
		}
	case ActionModeChan:
		for i := range e {
			a.externalCh <- e[i]
		}
	case ActionModeData:
		a.externalData = append(a.externalData, e...)
	}
}

func (a *Action[E]) done() {
	switch a.actionMode {
	case ActionModeChan, ActionModeData:
		a.closeChan <- true
	}
	SafeClose(a.externalCh)
	SafeClose(a.externalCloseChan)
	SafeClose(a.closeChan)
}

func MapAction[T1, T2 any](a1 *Action[T1], fn func(T1) T2) *Action[T2] {
	a2 := NewAction[T2]()
	a2.actionMode = a1.actionMode
	a2.closeChan = a1.closeChan

	a2.externalData = make([]T2, len(a1.externalData))
	for i := range a1.externalData {
		a2.externalData[i] = fn(a1.externalData[i])
	}
	return a2
}
