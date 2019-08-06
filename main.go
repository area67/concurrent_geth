package main

import (
	"C"
	"container/list"
	"github.com/golang-collections/collections/stack"
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
}

func main() {

}
