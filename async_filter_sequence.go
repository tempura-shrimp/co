package co

type AsyncFilterSequence[R any] struct {
	previousIterator Iterator[R]

	predictorFn func(R, R) bool
}

func NewAsyncFilterSequence[R any](it Iterator[R]) *AsyncFilterSequence[R] {
	return &AsyncFilterSequence[R]{
		previousIterator: it,
		predictorFn:      func(_, _ R) bool { return true },
	}
}

func (c *AsyncFilterSequence[R]) SetPredicator(fn func(R, R) bool) *AsyncFilterSequence[R] {
	c.predictorFn = fn
	return c
}

func (c *AsyncFilterSequence[R]) Iterator() *asyncFilterSequenceIterator[R] {
	it := &asyncFilterSequenceIterator[R]{
		AsyncFilterSequence: c,
	}
	it.asyncSequenceIterator = NewAsyncSequenceIterator[R, R](it)
	return it
}

type asyncFilterSequenceIterator[R any] struct {
	*asyncSequenceIterator[R, R]

	*AsyncFilterSequence[R]

	preProcessed bool
	previousData *data[R]
}

func (it *asyncFilterSequenceIterator[R]) preflight() bool {
	defer func() { it.preProcessed = true }()

	if it.previousData == nil && !it.previousIterator.preflight() {
		return false
	}
	if it.previousData != nil && !it.previousIterator.preflight() {
		return true
	}
	if it.previousData == nil && it.previousIterator.preflight() {
		val, err := it.previousIterator.consume()
		it.previousData = NewDataWith(val, err)
		return true
	}
	return false
}

func (it *asyncFilterSequenceIterator[R]) consume() (R, error) {
	if !it.preProcessed {
		it.preflight()
	}
	defer func() { it.preProcessed = false }()

	for it.previousIterator.preflight() {
		val, err := it.previousIterator.consume()
		if err != nil {
			it.previousData = NewDataWith(val, err)
			return val, err
		}
		if it.predictorFn(it.previousData.value, val) {
			rData := it.previousData
			it.previousData = NewDataWith(val, err)
			return rData.value, rData.err
		}
	}
	rData := it.previousData
	it.previousData = nil
	return rData.value, rData.err
}

func (it *asyncFilterSequenceIterator[R]) next() (R, error) {
	it.preflight()
	return it.consume()
}
