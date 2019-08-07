package main

import (
	"C"
	"container/list"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/golang-collections/collections/stack"
	"go/types"
)

const numThreads = 32

var testSize uint32

const LINEARIZABILITY = 0
const SEQUENTIAL_CONSISTENCY = 0
const SERIALIZABILITY = 1
const DEBUG_ = 0

var queue, stackk, mapp uint32

// MyHashCompare are blah, blah, blah
type MyHashCompare struct{}

func (mhc MyHashCompare) hash(x int) C.size_t {
	return x
}

func (mhc MyHashCompare) equal(x int, y int) bool {
	return x == y
}

type Status int

const (
	PRESENT Status = 1 + iota
	ABSENT
)

type Semantics int

const (
	FIFO Semantics = 1 + iota
	LIFO
	SET
	MAPP
	PRIORITY
)

type Types int

const (
	PRODUCER Types = 1 + iota
	CONSUMER
	READER
	WRITER
)

type Method struct {
	id              int
	process         int
	itemKey         int
	itemVal         int
	semantics       Semantics
	types           Types
	invocation      int64
	response        int64
	quiescentPeriod int
	status          bool
	txnID           int
}

func (m *Method) SetMethod(id int, process int, itemKey int, itemVal int, semantics Semantics,
	types Types, invocation int64, response int64, status bool, txnID int) {
	m.id = id
	m.process = process
	m.itemKey = itemKey
	m.itemVal = itemVal
	m.semantics = semantics
	m.types = types
	m.invocation = invocation
	m.response = response
	m.status = status
	m.txnID = txnID
}

type Item struct {
	key   int
	value int
	//sum   int
	sum float64

	numerator   int64
	denominator int64

	exponent float64

	status Status

	promoteItems stack.Stack

	demoteItems list.List

	producer types.Map // TODO: not sure if this is equivalent

	// Failed Consumer
	sumF         float64
	numeratorF   int64
	denominatorF int64
	exponentF    float64

	// Reader
	sumR         float64
	numeratorR   int64
	denominatorR int64
	exponentR    float64
}

func (i *Item) SetItem(key int){
	i.key          = key
	i.value        = math.MinInt32
	i.sum          = 0
	i.numerator    = 0
	i.denominator  = 1
	i.exponent     = 0
	i.status       = PRESENT
	i.sumF         = 0
	i.numeratorF   = 0
	i.denominatorF = 1
	i.exponentF    = 0
	i.sumR         = 0
	i.numeratorR   = 0
	i.denominatorR = 1
	i.exponentR    = 0
}

func (i *Item) SetItemKV(key, value int){
	i.key          = key
	i.value        = value
	i.sum          = 0
	i.numerator    = 0
	i.denominator  = 1
	i.exponent     = 0
	i.status       = PRESENT
	i.sumF         = 0
	i.numeratorF   = 0
	i.denominatorF = 1
	i.exponentF    = 0
	i.sumR         = 0
	i.numeratorR   = 0
	i.denominatorR = 1
	i.exponentR    = 0
}

func (i *Item) addInt(x int64) {

	// C.printf("Test add function\n")
	addNum := x * i.denominator

	i.numerator = i.numerator + addNum

	// C.printf("addNum = %ld, numerator/denominator = %ld\n", add_num, numerator/denominator);
	i.sum = float64(i.numerator / i.denominator)

	// i.sum = i.sum + x;
}

func main() {

}
