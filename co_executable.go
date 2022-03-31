package co

type CoExecutable[R any] struct {
	executables *executablesList[R]
	data        *determinedDataList[R]

	_defaultIterator Iterator[R]
}

func NewCoExecutable[R any]() *CoExecutable[R] {
	return &CoExecutable[R]{
		executables: NewExecutablesList[R](),
		data:        NewDeterminedDataList[R](),
	}
}

func (c *CoExecutable[R]) len() int {
	return c.executables.len()
}

func (c *CoExecutable[R]) exe(i int) (R, error) {
	if c.executables.getAt(i).isExecuted() {
		return c.data.getAt(i)
	}

	return c.forceExeAt(i)
}

func (c *CoExecutable[R]) forceExeAt(i int) (R, error) {
	val, err := c.executables.getAt(i).exe()
	c.data.setAt(i, val, err)

	return val, err
}

func (c *CoExecutable[R]) AddFn(fns ...func() (R, error)) *CoExecutable[R] {
	for i := range fns {
		c.data.List.add(NewData[R]())

		e := NewExecutor[R]()
		e.fn = fns[i]
		c.executables.add(e)
	}
	return c
}

func (c *CoExecutable[R]) defaultIterator() Iterator[R] {
	if c._defaultIterator != nil {
		return c._defaultIterator
	}
	c._defaultIterator = c.Iterator()
	return c._defaultIterator
}

func (c *CoExecutable[R]) Iterator() Iterator[R] {
	return &coExecutableSequenceIterator[R]{
		CoExecutable:          c,
		iterativeListIterator: c.executables.iterativeList.Iterator(),
	}
}

type coExecutableSequenceIterator[R any] struct {
	*CoExecutable[R]
	iterativeListIterator[*executable[R]]
}

func (it *coExecutableSequenceIterator[R]) next() (R, error) {
	defer func() { it.currentIndex++ }()
	return it.exe(it.currentIndex)
}
