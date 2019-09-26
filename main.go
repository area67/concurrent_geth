package main

import (
	"C"
	"container/list"
	"fmt"
	"github.com/golang-collections/collections/queue"
	"github.com/golang-collections/collections/stack"
	"go.uber.org/atomic"
	"math"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"sync"
	Atomic "sync/atomic"
	"syscall"
	"time"
)

const numThreads = 32

var testSize uint32

var tbbQueue, boostStack, tbbMap uint32

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
	PRESENT Status = iota
	ABSENT
)

type Semantics int

const (
	FIFO Semantics = iota
	LIFO
	SET
	MAPP
	PRIORITY
)

type Types int

const (
	PRODUCER Types = iota
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

func (m *Method) setMethod(id int, process int, itemKey int, itemVal int, semantics Semantics,
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

	demoteMethods []*Method

	producer int64 // map iterator


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

func (i *Item) setItem(key int) {
	i.key = key
	i.value = math.MinInt32
	i.sum = 0
	i.numerator = 0
	i.denominator = 1
	i.exponent = 0
	i.status = PRESENT
	i.sumF = 0
	i.numeratorF = 0
	i.denominatorF = 1
	i.exponentF = 0
	i.sumR = 0
	i.numeratorR = 0
	i.denominatorR = 1
	i.exponentR = 0
}

func (i *Item) setItemKV(key, value int) {
	i.key = key
	i.value = value
	i.sum = 0
	i.numerator = 0
	i.denominator = 1
	i.exponent = 0
	i.status = PRESENT
	i.sumF = 0
	i.numeratorF = 0
	i.denominatorF = 1
	i.exponentF = 0
	i.sumR = 0
	i.numeratorR = 0
	i.denominatorR = 1
	i.exponentR = 0
}

func (i *Item) addInt(x int64) {

	// C.printf("Test add function\n")
	addNum := x * i.denominator

	i.numerator = i.numerator + addNum

	// C.printf("addNum = %ld, numerator/denominator = %ld\n", add_num, numerator/denominator);
	i.sum = float64(i.numerator / i.denominator)

	// i.sum = i.sum + x
}

func (i *Item) subInt(x int64) {

	// C.printf("Test add function\n");
	subNum := x * i.denominator

	i.numerator = i.numerator - subNum

	// C.printf("subNum = %ld, i.numerator/i.denominator = %ld\n", subNum, i.numerator/i.denominator);
	i.sum = float64(i.numerator / i.denominator)

	// i.sum = i.sum + x
}

func (i *Item) addFrac(num int64, den int64) {

	// #if DEBUG_
	// if den == 0 {
	// 	 C.printf("WARNING: add_frac: den = 0\n")
	// }
	// if i.denominator == 0 {
	//	 C.printf("WARNING: add_frac: 1. denominator = 0\n")
	// }
	// #endif

	if i.denominator % den == 0 {
		i.numerator = i.numerator + num * i.denominator / den
	} else if den % i.denominator == 0 {
		i.numerator = i.numerator * den / i.denominator + num
		i.denominator = den
	} else {
		i.numerator = i.numerator * den + num * i.denominator
		i.denominator = i.denominator * den
	}

	// #if DEBUG_
	// if i.denominator == 0 {
	//   C.printf("WARNING: addFrac: 2. denominator = 0\n")
	// }
	// #endif

	i.sum = float64(i.numerator / i.denominator)
}

func (i *Item) subFrac(num, den int64) {

	// #if DEBUG_
	// if den == 0
	// 	 C.printf("WARNING: subFrac: den = 0\n")
	// if i.denominator == 0
	//	 C.printf("WARNING: subFrac: 1. denominator = 0\n");
	// #endif

	if i.denominator % den == 0 {
		i.numerator = i.numerator - num * i.denominator / den
	} else if den % i.denominator == 0 {
		i.numerator = i.numerator * den / i.denominator - num
		i.denominator = den
	} else {
		i.numerator = i.numerator * den - num * i.denominator
		i.denominator = i.denominator * den
	}

	// #if DEBUG_
	// if denominator == 0
	//	 C.printf("WARNING: subFrac: 2. denominator = 0\n")
	// #endif

	i.sum = float64(i.numerator / i.denominator)
}

func (i *Item) demote() {
	i.exponent = i.exponent + 1
	den := int64(math.Exp2(i.exponent))
	// C.printf("denominator = %ld\n", den);

	i.subFrac(1, den)
}

func (i *Item) promote() {
	den := int64(math.Exp2(i.exponent))

	// #if DEBUG_
	// if den == 0
	//   C.printf("2 ^ %f = %ld?\n", i.exponent, den);
	// #endif

	if i.exponent < 0 {
		den = 1
	}

	// C.printf("denominator = %ld\n", den);

	i.addFrac(1, den)
	i.exponent = i.exponent - 1
}

func (i *Item) addFracFailed(num int64, den int64) {

	// #if DEBUG_
	// if den == 0
	//   C.printf("WARNING: addFracFailed: den = 0\n");
	// if denominatorF == 0
	//   C.printf("WARNING: addFracFailed: 1. denominatorF = 0\n")
	// #endif

	if i.denominatorF % den == 0 {
		i.numeratorF = i.numeratorF + num * i.denominatorF / den
	} else if den % i.denominatorF == 0 {
		i.numeratorF = i.numeratorF * den / i.denominatorF + num
		i.denominatorF = den
	} else {
		i.numeratorF = i.numeratorF * den + num * i.denominatorF
		i.denominatorF = i.denominatorF * den
	}

	// #if DEBUG_
	// if denominatorF == 0
	//   C.printf("WARNING: addFracFailed: 2. denominatorF = 0\n");
	// #endif

	i.sumF = float64(i.numeratorF / i.denominatorF)
}

func (i *Item) subFracFailed(num int64, den int64) {
	// #if DEBUG_
	// if(den == 0)
	// C.printf("WARNING: sub_frac_f: den = 0\n");
	// if(denominator_f == 0)
	// C.printf("WARNING: sub_frac_f: 1. denominator_f = 0\n");
	// #endif
	if i.denominatorF % den == 0 {
		i.numeratorF = i.numeratorF - num * i.denominatorF / den
	} else if den % i.denominatorF == 0 {
		i.numeratorF = i.numeratorF * den / i.denominatorF - num
		i.denominatorF = den
	} else {
		i.numeratorF = i.numeratorF * den - num * i.denominatorF
		i.denominatorF = i.denominatorF * den
	}
	// #if DEBUG_
	// if(denominator_f == 0)
	// C.printf("WARNING: sub_frac_f: 2. denominator_f = 0\n");
	// #endif
	i.sumF = float64(i.numeratorF / i.denominatorF)
}

func (i *Item) demoteFailed() {
	i.exponentF = i.exponentF + 1
	var den = int64(math.Exp2(i.exponentF))
	// C.printf("denominator = %ld\n", den);
	i.subFracFailed(1, den)
}

func (i *Item) promoteFailed() {
	var den = int64(math.Exp2(i.exponentF))
	// C.printf("denominator = %ld\n", den);
	i.addFracFailed(1, den)
	i.exponentF = i.exponentF - 1
}

//Reader
func (i *Item) addFracReader(num int64, den int64) {
	if i.denominatorR % den == 0 {
		i.numeratorR = i.numeratorR + num * i.denominatorR / den
	} else if den % i.denominatorR == 0 {
		i.numeratorR = i.numeratorR * den / i.denominatorR + num
		i.denominatorR = den
	} else {
		i.numeratorR = i.numeratorR * den + num * i.denominatorR
		i.denominatorR = i.denominatorR * den
	}

	i.sumR = float64(i.numeratorR / i.denominatorR)
}

func (i *Item) subFracReader(num int64, den int64) {
	if i.denominatorR % den == 0 {
		i.numeratorR = i.numeratorR - num * i.denominatorR / den
	} else if den % i.denominatorR == 0 {
		i.denominatorR = i.numeratorR * den / i.denominatorR - num
		i.denominatorR = den
	} else {
		i.numeratorR = i.numeratorR * den - num * i.denominatorR
		i.denominatorR = i.denominatorR * den
	}

	i.sumR = float64(i.numeratorR / i.denominatorR)
}

func (i *Item) demoteReader() {
	i.exponentR = i.exponentR + 1
	var den = int64(math.Exp2(i.exponentR))
	// C.printf("denominator = %ld\n", den);
	i.subFracReader(1, den)
}

func (i *Item) promoteReader() {
	var den= int64(math.Exp2(i.exponentR))
	// C.printf("denominator = %ld\n", den);
	i.addFracReader(1, den)
	i.exponentR = i.exponentR - 1
}
// End of Item struct

type Block struct {
	start  int64
	finish int64
}

func (b *Block) setBlock(){
	b.start = 0
	b.finish = 0
}
// End of Block struct

var finalOutcome bool
var methodCount  uint32

func fncomp (lhs, rhs int64) bool{
	return lhs < rhs
}

var q queue.Queue
var s stack.Stack

var threadLists     = make([]list.List, 0, numThreads)// empty slice with capacity numThreads
var threadListsSize = make([]atomic.Int32, 0, numThreads)    // atomic ops only
var done            = make([]atomic.Bool, 0, numThreads)     // atomic ops only
var barrier int32                                            // atomic int


func wait(){
	Atomic.AddInt32(&barrier, 1)
	for Atomic.LoadInt32(&barrier) < numThreads {}
}

var methodTime   [numThreads]int64
var overheadTime [numThreads]int64

var start time.Time

var elapsedTimeVerify int64

func minOf(vars []int) int {
	min := vars[0]

	for _, i := range vars {
		if min > i {
			min = i
		}
	}

	return min
}

func maxOf(vars []int) int {
	max := vars[0]

	for _, i := range vars {
		if max < i {
			max = i
		}
	}

	return max
}

func findMethodKey(m map[int64]*Method, position string) (int64, error) {
	keys := make([]int, 0)
	for k := range m {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)

	begin := minOf(keys)
	end   := maxOf(keys)

	if position == "begin" {
		return int64(keys[begin]), nil
	} else if position == "end" {
		return int64(keys[end]), nil
	} else {
		return -1, fmt.Errorf("the map key could not be found")
	}
}

func reslice(s []*Method, index int) []*Method {
	return append(s[:index], s[index+1:]...)
}

func findItemKey(m map[int64]*Item, position string) (int64, error){
	keys := make([]int, 0)
	for k := range m {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)

	begin := minOf(keys)
	end   := maxOf(keys)

	if position == "begin" {
		return int64(keys[begin]), nil
	} else if position == "end" {
		return int64(keys[end]), nil
	} else {
		return -1, fmt.Errorf("the map key could not be found")
	}
}

func findBlockKey(m map[int64]Block, position string) (int64, error){
	keys := make([]int, 0)
	for k := range m {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)

	begin := minOf(keys)
	end   := maxOf(keys)

	if position == "begin" {
		return int64(keys[begin]), nil
	} else if position == "end" {
		return int64(keys[end]), nil
	} else {
		return -1, fmt.Errorf("the map key could not be found")
	}
}

// methodMapKey and itemMapKey are meant to serve in place of iterators
func handleFailedConsumer(methodMap map[int64]*Method, itemMap map[int64]*Item, mk int64, ik int64, stackFailed stack.Stack){

	for mk0, err := findMethodKey(methodMap, "begin"); mk0 != mk; mk0++{
		if err != nil {
			break
		}

		// linearizability
		// if methodMap[methItr0].response < methodMap[methodMapKey].invocation

		// sequential consistency
		// if methodMap[methItr0].response < methodMap[methodMapKey].invocation &&
		//     methodMap[methItr0].process == methodMap[methodMapKey].process

		// serializability
		if methodMap[mk0].response < methodMap[mk].invocation &&
			methodMap[mk0].txnID == methodMap[mk].txnID ||
			methodMap[mk0].txnID < methodMap[mk].txnID{

			itemItr0 := methodMap[mk0].itemKey

			if methodMap[mk0].types == PRODUCER &&
				itemMap[ik].status == PRESENT &&
				methodMap[mk0].semantics == FIFO ||
				methodMap[mk0].semantics == LIFO ||
				methodMap[mk].itemKey == methodMap[mk0].itemKey{

				stackFailed.Push(itemItr0)
			}
		}
	}
}

func handleFailedReader(methodMap map[int64]*Method, itemMap map[int64]*Item, mk int64, ik int64, stackFailed *stack.Stack){

	for mk0, err := findMethodKey(methodMap, "begin"); mk0 != mk; mk0++{
		if err != nil {
			break
		}

		// linearizability
		// #if methodMap[methItr0].response < methodMap[methodMapKey].invocation

		// sequential consistency
		// #elif methodMap[methItr0].response < methodMap[methodMapKey].invocation &&
		//     methodMap[methItr0].process == methodMap[methodMapKey].process

		// serializability
		if methodMap[mk0].response < methodMap[mk].invocation &&
			methodMap[mk0].txnID == methodMap[mk].txnID ||
			methodMap[mk0].txnID < methodMap[mk].txnID{

			itemItr0 := int64(methodMap[mk0].itemKey)

			if methodMap[mk0].types == PRODUCER &&
				itemMap[ik].status == PRESENT &&
				methodMap[mk].itemKey == methodMap[mk0].itemKey{

				stackFailed.Push(itemItr0)
			}
		}
	}
}

func verifyCheckpoint(mapMethods map[int64]*Method, mapItems map[int64]*Item, itStart int64, countIterated uint64, min int64, resetItStart bool, mapBlocks map[int64]Block){

	var stackConsumer  = stack.New()        // stack of map[int64]*Item
	var stackFinishedMethods stack.Stack  // stack of map[int64]*Method
	var stackFailed stack.Stack           // stack of map[int64]*Item

	if len(mapMethods) != 0 {

		it, err := findMethodKey(mapMethods, "begin")
		end, err2 := findMethodKey(mapMethods, "end")
		if err != nil && err2 != nil {
			return
		}
		if countIterated == 0 {
			resetItStart = false
		} else if it != end {
			itStart = itStart + 1
			it = itStart
		}

		for ; it != end; it++{
			if mapMethods[it].response > min{
				break
			}

			if methodCount % 5000 == 0 {
				fmt.Printf("methodCount = %d\n", methodCount)
			}
			methodCount = methodCount + 1

			itStart = it
			resetItStart = false
			countIterated = countIterated + 1

			itItems := int64(mapMethods[it].itemKey)

			// #if DEBUG_
			/// if mapItems[itItems].status != PRESENT{
			//  	fmt.Println("WARNING: Current item not present!")
			//} }

			// if mapMethods[it].types == PRODUCER{
			// fmt.Printf("PRODUCER invocation %ld, response %ld, item %d\n", mapMethods[it].invocation, mapMethods[it].response, mapMethods[it].itemKey)
			// }

			// else if mapMethods[it].types == CONSUMER {
			// fmt.Printf("CONSUMER invocation %ld, response %ld, item %d\n", mapMethods[it].invocation, mapMethods[it].response, mapMethods[it].itemKey)
			// }
			// #endif

			if mapMethods[it].types == PRODUCER {
				mapItems[itItems].producer = it

				if mapItems[itItems].status == ABSENT {

					// reset item parameters
					mapItems[itItems].status = PRESENT
					mapItems[itItems].demoteMethods = nil
				}

				mapItems[itItems].addInt(1)

				if mapMethods[it].semantics == FIFO || mapMethods[it].semantics == LIFO {
					it0, err := findMethodKey(mapMethods, "begin")
					if  err != nil{
						return
					}
					for ; it0 != it; it0++{
						// #if linearizability
						// if methodMap[methItr0].response < methodMap[methodMapKey].invocation

						// #elif sequential consistency
						// if methodMap[methItr0].response < methodMap[methodMapKey].invocation &&
						//     methodMap[methItr0].process == methodMap[methodMapKey].process

						// #elif serializability
						if mapMethods[it0].response < mapMethods[it].invocation &&
							mapMethods[it0].txnID == mapMethods[it].txnID ||
							mapMethods[it0].txnID < mapMethods[it].txnID{
							// #endif
							itItems0 := int64(mapMethods[it0].itemKey)

							// Demotion
							// FIFO Semantics
							if (mapMethods[it0].types == PRODUCER && mapItems[itItems0].status == PRESENT) &&
								(mapMethods[it].types == PRODUCER && mapMethods[it0].semantics == FIFO) {

								mapItems[itItems0].promoteItems.Push(mapItems[itItems].key)
								mapItems[itItems].demote()
								mapItems[itItems].demoteMethods = append(mapItems[itItems].demoteMethods, mapMethods[it0])
							}

							// LIFO Semantics
							if (mapMethods[it0].types == PRODUCER && mapItems[itItems0].status == PRESENT) &&
								(mapMethods[it].types == PRODUCER && mapMethods[it0].semantics == LIFO) {

								mapItems[itItems].promoteItems.Push(mapItems[itItems].key)
								mapItems[itItems0].demote()
								mapItems[itItems0].demoteMethods = append(mapItems[itItems0].demoteMethods, mapMethods[it])
							}
						}
					}
				}
			}

			if mapMethods[it].types == READER {
				if mapMethods[it].status == true {
					mapItems[itItems].demoteReader()
				} else{
					handleFailedReader(mapMethods, mapItems, it, itItems, &stackFailed)
				}
			}

			if mapMethods[it].types	== CONSUMER {

				/*std::unordered_map<int,std::unordered_map<int,Item>::iterator>::iterator it_consumer;
				it_consumer = map_consumer.find((it->second).key);
				if(it_consumer == map_consumer.end())
				{
					std::pair<int,std::unordered_map<int,Item>::iterator> entry ((it->second).key,it);
					//map_consumer.insert(std::make_pair<int,std::unordered_map<int,Item>::iterator>((it->second).key,it));
					map_consumer.insert(entry);
				} else {
					it_consumer->second = it_item_0;
				}*/

				if mapMethods[it].status == true {

					// promote reads
					if mapItems[itItems].sum > 0 {
						mapItems[itItems].sumR = 0
					}

					mapItems[itItems].subInt(1)
					mapItems[itItems].status = ABSENT

					if mapItems[itItems].sum < 0 {

						for idx := 0; idx != len(mapItems[itItems].demoteMethods) - 1; idx++ {

							if mapMethods[it].response < mapItems[itItems].demoteMethods[idx].invocation ||
								mapItems[itItems].demoteMethods[idx].response < mapMethods[it].invocation{
								// Methods do not overlap
								// fmt.Println("NOTE: Methods do not overlap")
							} else {
								mapItems[itItems].promote()

								// need to remove from promote list
								itMthdItem := int64(mapItems[itItems].demoteMethods[idx].itemKey)
								var temp stack.Stack

								for mapItems[itMthdItem].promoteItems.Peek() != nil{

									top := mapItems[itMthdItem].promoteItems.Peek()
									if top != mapMethods[it].itemKey {
										temp.Push(top)
									}
									mapItems[itMthdItem].promoteItems.Pop()
									fmt.Println("stuck here?")
								}
								// TODO: swap mapItems[itMthdItem].promoteItems with temp stack

								//
								mapItems[itItems].demoteMethods = reslice(mapItems[itItems].demoteMethods, idx)
							}
						}
					}
					stackConsumer.Push(itItems)
					stackFinishedMethods.Push(it)

					end, err = findMethodKey(mapMethods, "end")
					if mapItems[itItems].producer != end {
						stackFinishedMethods.Push(mapItems[itItems].producer)
					}
				} else {
					handleFailedConsumer(mapMethods, mapItems, it, itItems, stackFailed)
				}
			}
		}
		if resetItStart {
			itStart--
		}

		//NEED TO FLAG ITEMS ASSOCIATED WITH CONSUMER METHODS AS ABSENT
		for stackConsumer.Len() != 0 {

			itTop, ok := stackConsumer.Peek().(int64)
			if !ok {
				return
			}

			for mapItems[itTop].promoteItems.Len() != 0 {
				itemPromote := mapItems[itTop].promoteItems.Peek().(int64)
				itPromoteItem := itemPromote
				mapItems[itPromoteItem].promote()
				mapItems[itTop].promoteItems.Pop()
			}
			stackConsumer.Pop()
		}

		for stackFailed.Len() != 0 {
			itTop := stackFailed.Peek().(int64)

			if mapItems[itTop].status == PRESENT {
				mapItems[itTop].demoteFailed()
			}
			stackFailed.Pop()
		}

		// remove methods that are no longer active
		for stackFinishedMethods.Len() != 0 {
			itTop := stackFinishedMethods.Peek().(int64)
			delete(mapMethods, itTop)
			stackFinishedMethods.Pop()
		}

		// verify sums
		outcome := true
		itVerify, err := findItemKey(mapItems, "begin")
		end, endErr := findItemKey(mapItems, "end")

		if err != nil || endErr != nil {
			return
		}

		for ; itVerify != end; itVerify++ {

			if mapItems[itVerify].sum < 0 {
				outcome = false
				// #if DEBUG_
				// fmt.Printf("WARNING: Item %d, sum %.2lf\n", mapItems[itVerify].key, mapItems[itVerify].sum)
				// #endif
			}
			//printf("Item %d, sum %.2lf\n", it_verify->second.key, it_verify->second.sum);

			if (math.Ceil(mapItems[itVerify].sum) + mapItems[itVerify].sumR) < 0 {
				outcome = false

				// #if DEBUG_
				// fmt.Printf("WARNING: Item %d, sum_r %.2lf\n", mapItems[itVerify].key, mapItems[itVerify].sumR)
				// #endif
			}

			var n float64
			if mapItems[itVerify].sumF == 0 {
				n = 0
			} else {
				n = -1
			}

			if (math.Ceil(mapItems[itVerify].sum) + mapItems[itVerify].sumF) * n < 0 {
				outcome = false
				// #if DEBUG_
				// fmt.Printf("WARNING: Item %d, sum_f %.2lf\n", mapItems[itVerify].key, mapItems[itVerify].sumF)
				// #endif
			}

		}
		if outcome == true {
			finalOutcome = true
			// #if DEBUG_
			// fmt.Println("-------------Program Correct Up To This Point-------------")
			// #endif
		} else {
			finalOutcome = false

			// #if DEBUG_
			// fmt.Println("-------------Program Not Correct-------------")
			// #endif
		}
	}
}





func work(id int) {

	testSize := testSize
	wallTime := 0.0
	var tod syscall.Timeval
	if err := syscall.Gettimeofday(&tod); err != nil {
		return
	}
	wallTime += float64(tod.Sec)
	wallTime += float64(tod.Usec) * 1e-6

	var randomGenOp rand.Rand
	randomGenOp.Seed(int64(wallTime + float64(id) + 1000))
	s := rand.NewSource(time.Now().UnixNano())
	randDistOp := rand.New(s)

	// TODO: I'm 84% sure this is correct
	startTime := time.Unix(0, start.UnixNano())
	startTimeEpoch := time.Since(startTime)

	mId := id + 1

	var end time.Time

	wait()

	for i := uint32(0); i < testSize; i++ {

		var types Types
		itemKey := -1
		res := true
		opDist := uint32(1 + randDistOp.Intn(100))  // uniformly distributed pseudo-random number between 1 - 100 ??

		end = time.Now()

		preFunction := time.Unix(0, end.UnixNano())
		preFunctionEpoch := time.Since(preFunction)


		// Hmm, do we need .count()??
		//invocation := pre_function_epoch.count() - start_time_epoch.count()
		invocation := preFunctionEpoch.Nanoseconds() - startTimeEpoch.Nanoseconds()

		if invocation > (math.MaxInt64 - 10000000000) {
			//PREPROCESSOR DIRECTIVE lines 864 - 866:
			/*
			 * #if DEBUG_
			 *		printf("WARNING: TIME LIMIT REACHED! TERMINATING PROGRAM\n");
			 * #endif
			 */
			break
		}

		if opDist <= 50 {
			types = CONSUMER
			var itemPop int
			var itemPopPtr *uint64

			res := q.Peek()
			if res != 0 {
				q.Dequeue()  // try_pop(item_pop)
				itemKey = itemPop
			}else {
				itemKey = math.MaxInt32
			}
		} else {
			types = PRODUCER
			itemKey = mId
			q.Enqueue(itemKey)
		}

		// line 890
		// end = std::chrono::high_resolution_clock::now();
		end := time.Now().UnixNano()

		// auto post_function = std::chrono::time_point_cast<std::chrono::nanoseconds>(end);
		postFunction := end

		// Is this right??
		// auto post_function_epoch = post_function.time_since_epoch();
		postFunctionEpoch := time.Now().UnixNano() - postFunction

		//response := post_function_epoch.count() - start_time_epoch.count()
		response := postFunctionEpoch - startTimeEpoch.Nanoseconds()

		var m1 Method
		m1.setMethod(mId, id, itemKey, math.MinInt64, FIFO, types, invocation, response, res, mId)

		mId += numThreads

		threadLists[id].PushBack(m1)
		threadListsSize[id].Add(1)
		Atomic.AddInt64(&methodTime[id], 1)
	}

	done[id].Store(true)
}

func verify() {
	wait()

	startTime := time.Unix(0, start.UnixNano())
	startTimeEpoch := time.Since(startTime)

	end := time.Now()

	preVerify := time.Unix(0, end.UnixNano())
	preVerifyEpoch := time.Since(preVerify)


	verifyStart := preVerifyEpoch.Nanoseconds() - startTimeEpoch.Nanoseconds()

	fnPt       := fncomp
	mapMethods := make(map[int64]*Method, 0)
	mapBlock   := make(map[int64]Block, 0)
	mapItems    := make(map[int64]*Item, 0)
	it         := make([]int, 0, numThreads)
	var itStart int64


	// How to??? lines 1201 - 1209
	/*
		bool(*fn_pt)(long int,long int) = fncomp;
	  	std::map<long int,Method,bool(*)(long int,long int)> map_methods (fn_pt);
		std::map<long int,Block,bool(*)(long int,long int)> map_block (fn_pt);
		std::unordered_map<int,Item> map_items;
		std::map<long int,Method,bool(*)(long int,long int)>::iterator it_start;
		std::list<Method>::iterator it[NUM_THRDS];
		int it_count[NUM_THRDS];
	*/

	stop := false
	var countOverall uint32 = 0
	var countIterated uint32 = 0

	var min int64
	var oldMin int64
	var itCount[numThreads] int32

	// How to??? line 1225
	// std::map<long int,Method,bool(*)(long int,long int)>::iterator it_qstart;

	for {
		if stop {
			break
		}

		stop = true
		min = math.MaxInt64

		for i := 0; i < numThreads; i++ {
			if done[i].Load() == false {
				stop = false
			}

			var responseTime int64 = 0

			for {
				if itCount[i] >= threadListsSize[i].Load() {
					break
				} else if itCount[i] == 0 {
					it[i] = 0 //threadLists[i].Front()
				} else {
					//++it[i]
					it[i]++
				}

				m := threadLists[it[i]].Back().Value.(Method)

				mapMethodsEnd, err := findMethodKey(mapMethods, "end")
				if err != nil{
					return
				}

				itMethod := m.response
				for {
					if itMethod == mapMethodsEnd { //
						break
					}
					m.response++
					itMethod = m.response
				}
				responseTime = m.response

				mapMethods[m.response] = &m // map_methods.insert ( std::pair<long int,Method>(m.response,m) );

				itCount[i]++
				countOverall++

				itItem := m.itemKey // it_item = map_items.find(m.item_key);

				mapItemsEnd, err := findItemKey(mapItems, "end")
				if err != nil {
					return
				}

				if int64(itItem) == mapItemsEnd {
					var item Item
					item.key = m.itemKey



					item.producer, err = findMethodKey(mapMethods, "end")

					// How to??? line 1288
					// map_items.insert(std::pair<int,Item>(m.item_key,item) );
					mapItems[int64(m.itemKey)] = &item
					itItem = m.itemKey
				}
			}

			if responseTime < min {
				min = responseTime
			}
		}

		verifyCheckpoint(mapMethods, mapItems, itStart, uint64(countIterated), int64(min), true, mapBlock)

	}

	verifyCheckpoint(mapMethods, mapItems, itStart, uint64(countIterated), math.MaxInt64, false, mapBlock)

	/*
		#if DEBUG_
			printf("Count overall = %lu, count iterated = %lu, map_methods.size() = %lu\n", count_overall, count_iterated, map_methods.size());
		#endif

	#if DEBUG_
		printf("All threads finished!\n");


	itB, err := findBlockKey(mapBlock, "begin")
	itBEnd, err2 := findBlockKey(mapBlock, "end")
	if err != nil || err2 != nil {
		return
	}

	for ; itB != itBEnd; itB++ {
		fmt.Printf("Block start = %d, finish = %d\n", mapBlock[itB].start, mapBlock[itB].finish)
	}

	// How to??? line 1346
	// std::map<long int,Method,bool(*)(long int,long int)>::iterator it_;

	//for it = map_methods.begin(); it != map_methods.end(); ++it {
		// How to??? lines 1349 -1356
		/*
			std::unordered_map<int,Item>::iterator it_item;
			it_item = map_items.find(it_->second.item_key);
			if(it_->second.type == PRODUCER)
				printf("PRODUCER inv %ld, res %ld, item %d, sum %.2lf, sum_r = %.2lf, sum_f = %.2lf, tid = %d, qperiod = %d\n", it_->second.invocation, it_->second.response, it_->second.item_key, it_item->second.sum, it_item->second.sum_r, it_item->second.sum_f, it_->second.process, it_->second.quiescent_period);
			else if ((it_->second).type == CONSUMER)
				printf("CONSUMER inv %ld, res %ld, item %d, sum %.2lf, sum_r = %.2lf, sum_f = %.2lf, tid = %d, qperiod = %d\n", it_->second.invocation, it_->second.response, it_->second.item_key, it_item->second.sum, it_item->second.sum_r, it_item->second.sum_f, it_->second.process, it_->second.quiescent_period);
			else if ((it_->second).type == READER)
				printf("READER inv %ld, res %ld, item %d, sum %.2lf, sum_r = %.2lf, sum_f = %.2lf, tid = %d, qperiod = %d\n", it_->second.invocation, it_->second.response, it_->second.item_key, it_item->second.sum, it_item->second.sum_r, it_item->second.sum_f, it_->second.process, it_->second.quiescent_period);
		*/
	//}

	// #endif

	// How to??? lines 1360 - 1362
	// end = std::chrono::high_resolution_clock::now();
	// auto post_verify = std::chrono::time_point_cast<std::chrono::nanoseconds>(end);
	end = time.Now()
	postVerify := end.UnixNano()

	postVerifyEpoch := time.Now().UnixNano() - postVerify
	verifyFinish := postVerifyEpoch - startTimeEpoch.Nanoseconds()

	elapsedTimeVerify = verifyFinish - verifyStart
}

func main() {
	methodCount := 0

	finalOutcome := true

	// std::thread t[NUM_THRDS];

	start := time.Now()
	var wg sync.WaitGroup

	//TODO: thread/ channel stuff
	for i := 0; i < numThreads; i++ {
		go work(i) // ???
	}

	go verify() // v = std::thread(verify) ???

	if finalOutcome == true {
		fmt.Printf("-------------Program Correct Up To This Point-------------\n")
	} else {
		fmt.Printf("-------------Program Not Correct-------------\n")
	}

	finish      := time.Now()                            //auto finish = std::chrono::high_resolution_clock::now();
	elapsedTime := finish.UnixNano() - start.UnixNano()  //auto elapsed_time = std::chrono::duration_cast<std::chrono::nanoseconds>(finish - start);

	var elapsedTimeDouble float64 = float64(elapsedTime) * 0.000000001
	fmt.Printf("Total Time: %.15f seconds\n", elapsedTimeDouble)

	var elapsedTimeMethod int64 = 0
	var elapsedOverheadTime float64 = 0

	for i := 0; i < numThreads; i++ {
		if methodTime[i] > elapsedTimeMethod {
			elapsedTimeMethod = methodTime[i]
		}
		if float64(overheadTime[i]) > elapsedOverheadTime {
			elapsedOverheadTime = float64(overheadTime[i])
		}
	}

	var elapsedTimeMethodDouble float64 = float64(elapsedTimeMethod) * 0.000000001
	var elapsedOverheadTimeDouble float64= elapsedOverheadTime * 0.000000001
	var elapsedTimeVerifyDouble float64 = float64(elapsedTimeVerify) * 0.000000001

	fmt.Printf("Total Method Time: %.15f seconds\n", elapsedTimeMethodDouble)
	fmt.Printf("Total Overhead Time: %.15f seconds\n", elapsedOverheadTimeDouble)

	elapsed_time_verify_double = elapsed_time_verify_double - elapsed_time_method_double

	fmt.printf("Total Verification Time: %.15lf seconds\n", elapsed_time_verify_double)

	if TBB_QUEUE == 1 {
		fmt.printf("Final Queue Configuration: \n");
		// How to???
		//typedef tbb::concurrent_queue<int>::iterator iter;
		//for(iter i(queue.unsafe_begin()); i!=queue.unsafe_end(); i++)
		//printf("%d ", *i);
		//printf("\n");
	} else if BOOST_STACK == 1 {
		fmt.printf("Final Stack Configuration: \n")
		var stack_val int

		for
		{
			if stack.pop(stack_val)
			{
				fmt.printf("%d ", stack_val)
			}
			else
			{
				break
			}
			fmt.printf("\n")
		}
	} else if TBB_MAP == 1 {
		fmt.printf("Final Map Configuration: \n")
		// How to???
		/*
				tbb::concurrent_hash_map<int,int,MyHashCompare>::iterator it;
				for( it=map.begin(); it!=map.end(); ++it )
		    		printf("%d,%d ",it->first,it->second);
				printf("\n");
		*/
	}
}
