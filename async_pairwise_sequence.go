package co

type AsyncPairwiseSequence[R any, T []R] struct {
	*asyncSequence[T]

	previousIterator Iterator[R]
}

func NewAsyncPairwiseSequence[R any, T []R](it AsyncSequenceable[R]) *AsyncPairwiseSequence[R, T] {
	a := &AsyncPairwiseSequence[R, T]{
		previousIterator: it.iterator(),
	}
	a.asyncSequence = NewAsyncSequence[T](a)
	return a
}

func (c *AsyncPairwiseSequence[R, T]) iterator() Iterator[T] {
	it := &asyncPairwiseSequenceIterator[R, T]{
		AsyncPairwiseSequence: c,
	}
	it.asyncSequenceIterator = NewAsyncSequenceIterator[T](it)
	return it
}

type asyncPairwiseSequenceIterator[R any, T []R] struct {
	*asyncSequenceIterator[T]

	*AsyncPairwiseSequence[R, T]

	previousData Optional[R]
}

func (it *asyncPairwiseSequenceIterator[R, T]) preflight() {
	if it.previousData.valid {
		return
	}

	for op := it.previousIterator.next(); op.valid; op = it.previousIterator.next() {
		it.previousData = *op
		break
	}
}

func (it *asyncPairwiseSequenceIterator[R, T]) next() *Optional[T] {
	it.preflight()

	previousData := it.previousData.data
	for op := it.previousIterator.next(); op.valid; op = it.previousIterator.next() {
		it.previousData = *op
		return OptionalOf(T{previousData, op.data})
	}

	return NewOptionalEmpty[T]()
}
