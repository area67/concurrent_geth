package correctness_tool

type Status int
type Semantics int
type Types int
type State int32

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
const (
	WORKING = iota
	DONE
)

const MAXTXNS = 300