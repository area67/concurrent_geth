package correctness_tool

const (
	PRESENT Status = iota
	ABSENT
	LESS = -1
	EQUAL = 0
	MORE = 1
)

const (
	FIFO Semantics = iota
	LIFO
	SET
	MAPP
	PRIORITY
)

const (
	PRODUCER Types = iota
	CONSUMER
	READER
	WRITER
)
