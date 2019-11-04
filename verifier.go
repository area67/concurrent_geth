package main

import (
	"C"
	"fmt"
	"github.com/golang-collections/collections/queue"
	"github.com/golang-collections/collections/stack"
	"go.uber.org/atomic"
	"math"
	"math/rand"
	"sort"
	"sync"
	Atomic "sync/atomic"
	"syscall"
	"time"
)

const numThreads = 32

var testSize uint32

type Status int

var txnCtr AtomicTxnCtr

var countIterated uint32 = 0

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
	id          int       // atomic var
	itemAddr    string       // account address
	itemBalance int       // account balance
	semantics   Semantics // hardcode as FIFO per last email
	types       Types     // producing/consuming  adding/subtracting
	status      bool
	senderID    int       // same as itemAddr ??
	requestAmnt int
	txnCtr      int32
}

type TransactionData struct {
	addrSender   string
	addrReceiver string
	balanceSender int
	balanceReceiver int
	amount       int
	tId          int32
}

type AtomicTxnCtr struct {
	val int32
	lock sync.Mutex
}

type ConcurrentSlice struct {
	sync.RWMutex
	items []interface{}
	//items []ConcurrentSliceItem
}

// Concurrent slice item
type ConcurrentSliceItem struct {
	Index int
	Value interface{}
}

func NewConcurrentSlice() *ConcurrentSlice {
	cs := &ConcurrentSlice{
		//items: make([]interface{}, 0),
		items: make([]interface{}, numThreads),
	}
	return cs
}

func (cs *ConcurrentSlice) Append(item interface{}) {
	cs.Lock()
	defer cs.Unlock()

	cs.items = append(cs.items, item)
}

func (cs *ConcurrentSlice) Iter() <-chan ConcurrentSliceItem {
	c := make(chan ConcurrentSliceItem)

	f := func() {
		cs.Lock()
		defer cs.Unlock()
		for index, value := range cs.items {
			c <- ConcurrentSliceItem{index, value}
		}
		close(c)
	}
	go f()

	return c
}

func (m *Method) setMethod(id int, itemAddr string, itemBalance int, semantics Semantics,
	types Types, status bool, senderID int, requestAmnt int, txnCtr int32) {
	m.id = id
	m.itemAddr = itemAddr
	m.itemBalance = itemBalance
	m.semantics = semantics
	m.types = types
	m.status = status
	//m.senderID = senderID
	m.requestAmnt = requestAmnt
	m.txnCtr = txnCtr
}

type Item struct {
	key           string // Account Hash ???
	value         int // Account Balance ???
	sum           float64
	numerator     int64
	denominator   int64
	exponent      float64
	status        Status
	promoteItems  stack.Stack
	demoteMethods []*Method
	producer      int // map iterator

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

func (i *Item) setItem(key string) {
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

func (i *Item) setItemKV(key string, value int) {
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

	if i.denominator%den == 0 {
		i.numerator = i.numerator + num*i.denominator/den
	} else if den%i.denominator == 0 {
		i.numerator = i.numerator*den/i.denominator + num
		i.denominator = den
	} else {
		i.numerator = i.numerator*den + num*i.denominator
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

	if i.denominator%den == 0 {
		i.numerator = i.numerator - num*i.denominator/den
	} else if den%i.denominator == 0 {
		i.numerator = i.numerator*den/i.denominator - num
		i.denominator = den
	} else {
		i.numerator = i.numerator*den - num*i.denominator
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

	if i.denominatorF%den == 0 {
		i.numeratorF = i.numeratorF + num*i.denominatorF/den
	} else if den%i.denominatorF == 0 {
		i.numeratorF = i.numeratorF*den/i.denominatorF + num
		i.denominatorF = den
	} else {
		i.numeratorF = i.numeratorF*den + num*i.denominatorF
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
	if i.denominatorF%den == 0 {
		i.numeratorF = i.numeratorF - num*i.denominatorF/den
	} else if den%i.denominatorF == 0 {
		i.numeratorF = i.numeratorF*den/i.denominatorF - num
		i.denominatorF = den
	} else {
		i.numeratorF = i.numeratorF*den - num*i.denominatorF
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
	if i.denominatorR%den == 0 {
		i.numeratorR = i.numeratorR + num*i.denominatorR/den
	} else if den%i.denominatorR == 0 {
		i.numeratorR = i.numeratorR*den/i.denominatorR + num
		i.denominatorR = den
	} else {
		i.numeratorR = i.numeratorR*den + num*i.denominatorR
		i.denominatorR = i.denominatorR * den
	}

	i.sumR = float64(i.numeratorR / i.denominatorR)
}

func (i *Item) subFracReader(num int64, den int64) {
	if i.denominatorR%den == 0 {
		i.numeratorR = i.numeratorR - num*i.denominatorR/den
	} else if den%i.denominatorR == 0 {
		i.denominatorR = i.numeratorR*den/i.denominatorR - num
		i.denominatorR = den
	} else {
		i.numeratorR = i.numeratorR*den - num*i.denominatorR
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
	var den = int64(math.Exp2(i.exponentR))
	// C.printf("denominator = %ld\n", den);
	i.addFracReader(1, den)
	i.exponentR = i.exponentR - 1
}

// End of Item struct

type Block struct {
	start  int64
	finish int64
}

func (b *Block) setBlock() {
	b.start = 0
	b.finish = 0
}

// End of Block struct

var finalOutcome bool
var methodCount int32

func fncomp(lhs, rhs int64) bool {
	return lhs < rhs
}

var q queue.Queue
var s stack.Stack

var threadLists ConcurrentSlice // empty slice with capacity numThreads
var threadListsSize= make([]atomic.Int32, numThreads, numThreads) // atomic ops only
var done = make([]atomic.Bool, 32, numThreads)             // atomic ops only
var barrier int32                                         // atomic int

func wait() {
	Atomic.AddInt32(&barrier, 1)
	for Atomic.LoadInt32(&barrier) < numThreads {
	}
}

var methodTime [numThreads]int64
var overheadTime [numThreads]int64

var start time.Time

var elapsedTimeVerify int64

func minOf(vars []int) int {
	if len(vars) == 0 {
		return -1
	}
	sort.Ints(vars)
	return vars[0]
}

func maxOf(vars []int) int {
	if len(vars) == 0 {
		return -1
	}
	sort.Ints(vars)
	return vars[len(vars) - 1]
}

func findIndexForMethod(methods []*Method, method Method, field string) int {
	if field == "itemAddr" {
		for i, m := range methods {
			if m.itemAddr == method.itemAddr{
				return i
			}
		}
	}
	return -1
}

//
//func reslice(s []*Method, index int) []*Method {
//	return append(s[:index], s[index+1:]...)
//}
//


// methodMapKey and itemMapKey are meant to serve in place of iterators
func handleFailedConsumer(methods []Method, items []Item, mk int, it int, stackFailed stack.Stack) {
	fmt.Println("Handling failed consumer...")
	begin := 0
	for it0 := begin; it0 != it; it0++ {
		// serializability
		if methods[it0].itemAddr == methods[it].itemAddr &&
			methods[it0].requestAmnt > methods[mk].requestAmnt {

			itemItr0 := methods[it0].itemAddr

			if methods[it0].types == PRODUCER &&
				//items.items[it].(*Item).status == PRESENT &&
				items[it].status == PRESENT &&
				methods[it0].semantics == FIFO ||
				methods[it0].semantics == LIFO ||
				methods[mk].itemAddr == methods[it0].itemAddr {

				stackFailed.Push(itemItr0)
			}
		}
	}
}

/*func handleFailedReader(methods []*Method, items []*Item, mk int, ik int, stackFailed *stack.Stack){

	for it0 := 0; it0 != ik; it0++ {
		// serializability
		if methods[it0].senderID == methods[mk].senderID &&
			methods[it0].requestAmnt > methods[mk].requestAmnt{

			itemItr0 := methods[it0].itemAddr

			if methods[it0].types == PRODUCER &&
				items[ik].status == PRESENT &&
				methods[it0].itemAddr == methods[it0].itemAddr{

				stackFailed.Push(items[itemItr0])
			}
		}
	}
}*/

func verifyCheckpoint(methods []Method, items []Item, itStart int, countIterated uint64, min int64, resetItStart bool, mapBlocks []Block) {
	//fmt.Println("Verifying Checkpoint...")

	var stackConsumer = stack.New()      // stack of map[int64]*Item
	var stackFinishedMethods stack.Stack // stack of map[int64]*Method
	var stackFailed stack.Stack          // stack of map[int64]*Item

	methodCount = int32(len(methods))
	//for i := range methods.Iter() {
		//fmt.Printf("methods.items[0].semantics = %v\n", methods.items[0].(Method).semantics)
	//}
	if methodCount != 0 {

		it := 0
		end := len(methods) - 1

		//fmt.Printf("end = %d\n", end)

		//TODO: corner case
		if countIterated == 0 {
			resetItStart = false
		} else if it != end {
			itStart = itStart + 1
			it = itStart
		}
		fmt.Printf("it = %d\n", it)

		/*for ; it != len(methods) - 1; it++{
		if methods[it].response > min{
			break
		}
		*/

		if methodCount%5000 == 0 {
			fmt.Printf("methodCount = %d\n", methodCount)
		}
		methodCount = methodCount + 1

		itStart = it
		resetItStart = false
		countIterated++

		//fmt.Printf("Do we even get here? %d", countIterated)

		itItems := it //methods[it].itemAddr
		//itItems := int(methods[it].txnCtr)

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


		if methods[it].types == PRODUCER {

			items[it].producer = it

			if items[itItems].status == ABSENT {

				// reset item parameters
				items[it].status = PRESENT
				items[it].demoteMethods = nil
			}

			items[it].addInt(1)

			if methods[it].semantics == FIFO {
				for it0 := 0; it0 != it; it0++ {
					// serializability
					//fmt.Println("MADE IT")
					if methods[it0].itemAddr == methods[it].itemAddr &&
						methods[it0].requestAmnt > methods[it].requestAmnt {
						fmt.Println("MADE IT in 1")
						// #endif
						itItems0 := 0

						// Demotion
						// FIFO Semantics
						//if (methods.items[it0].(Method).types == PRODUCER && items.items[int(itItems0)].(Item).status == PRESENT) &&
						if (methods[it0].types == PRODUCER && items[itItems0].status == PRESENT) &&
							(methods[it].types == PRODUCER && methods[it0].semantics == FIFO) {


							//items.items[itItems].(*Item).promoteItems.Push(items.items[itItems].(*Item).key)
							//items.items[itItems].(*Item).demote()
							//items.items[itItems].(*Item).demoteMethods = append(items.items[itItems].(*Item).demoteMethods, methods.items[it0].(*Method))
							items[itItems0].promoteItems.Push(items[itItems].key)
							items[itItems].demote()
							items[itItems].demoteMethods = append(items[itItems].demoteMethods, &methods[it0])
						}
					}
				}
			}
		}
		if methods[it].semantics == FIFO {
			//fmt.Printf("it is %d\n", it)
			for it0 := 0; it0 != it; it0++{
				fmt.Printf("MADE it to FIFO loop\n")
				// serializability
				fmt.Printf("CHECKING CORRECTNESS 1\n")
				if methods[it0].itemAddr == methods[it].itemAddr &&
					methods[it0].requestAmnt > methods[it].requestAmnt{
					fmt.Println("MADE IT in 2")
					// #endif
					itItems0 := it0 //methods[it0].itemAddr

					// Demotion
					// FIFO Semantics
					//if (methods.items[it0].(*Method).types == PRODUCER && items.items[itItems0].(*Item).status == PRESENT) &&
					if (methods[it0].types == PRODUCER && items[itItems0].status == PRESENT) &&
						(methods[it].types == PRODUCER && methods[it0].semantics == FIFO) {

						//items.items[itItems0].(*Item).promoteItems.Push(items.items[itItems].(*Item).key)
						//items.items[itItems].(*Item).demote()
						//items.items[itItems].(*Item).demoteMethods = append(items.items[itItems].(*Item).demoteMethods, methods.items[it0].(*Method))
					}
				}
			}
		}
		//tempMethods = methods.items[it].(Method)
		//if methods.items[it].(*Method).types == CONSUMER {
		if methods[it].types == CONSUMER {

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

			if methods[it].status == true {

				// promote reads
				//if items.items[itItems].(*Item).sum > 0 {
				if items[itItems].sum > 0 {
					//items.items[itItems].(*Item).sumR = 0
					items[itItems].sumR = 0
				}

				//items.items[itItems].(*Item).subInt(1)
				items[itItems].subInt(1)
				//items.items[itItems].(*Item).status = ABSENT
				items[itItems].status = ABSENT

				//if mapItems[itItems].sum < 0 {
				//
				//	for idx := 0; idx != len(mapItems[itItems].demoteMethods) - 1; idx++ {
				//
				//		if mapMethods[it].response < mapItems[itItems].demoteMethods[idx].invocation ||
				//			mapItems[itItems].demoteMethods[idx].response < mapMethods[it].invocation{
				//			// Methods do not overlap
				//			// fmt.Println("NOTE: Methods do not overlap")
				//		} else {
				//			mapItems[itItems].promote()
				//
				//			// need to remove from promote list
				//			itMthdItem := int64(mapItems[itItems].demoteMethods[idx].itemKey)
				//			var temp stack.Stack
				//
				//			for mapItems[itMthdItem].promoteItems.Peek() != nil{
				//
				//				top := mapItems[itMthdItem].promoteItems.Peek()
				//				if top != mapMethods[it].itemKey {
				//					temp.Push(top)
				//				}
				//				mapItems[itMthdItem].promoteItems.Pop()
				//				fmt.Println("stuck here?")
				//			}
				//			// TODO: swap mapItems[itMthdItem].promoteItems with temp stack
				//
				//			//
				//			mapItems[itItems].demoteMethods = reslice(mapItems[itItems].demoteMethods, idx)
				//		}
				//	}
				//}
				stackConsumer.Push(itItems)
				stackFinishedMethods.Push(it)

				end = len(methods) - 1
				//if items.items[itItems].(*Item).producer != end {
				if items[itItems].producer != end {
					//stackFinishedMethods.Push(items.items[itItems].(*Item).producer)
					stackFinishedMethods.Push(items[itItems].producer)
				}
			} else {
				handleFailedConsumer(methods, items, it, itItems, stackFailed)
			}
		}
		//}
		if resetItStart {
			itStart--
		}

		//NEED TO FLAG ITEMS ASSOCIATED WITH CONSUMER METHODS AS ABSENT
		for stackConsumer.Len() != 0 {

			itTop, ok := stackConsumer.Peek().(int)
			if !ok {
				return
			}

			//for items.items[itTop].(*Item).promoteItems.Len() != 0 {
			for items[itTop].promoteItems.Len() != 0 {
				/*itemPromote := items.items[itTop].(*Item).promoteItems.Peek().(int)
				itPromoteItem := itemPromote
				items.items[itPromoteItem].(*Item).promote()
				items.items[itTop].(*Item).promoteItems.Pop()*/
				itemPromote := items[itTop].promoteItems.Peek().(int)
				itPromoteItem := itemPromote
				items[itPromoteItem].promote()
				items[itTop].promoteItems.Pop()
			}
			stackConsumer.Pop()
		}

		for stackFailed.Len() != 0 {
			itTop := stackFailed.Peek().(int)

			//if items.items[itTop].(*Item).status == PRESENT {
			if items[itTop].status == PRESENT {
				//items.items[itTop].(*Item).demoteFailed()
				items[itTop].demoteFailed()
			}
			stackFailed.Pop()
		}

		// remove methods that are no longer active
		//TODO: DANGER, this is the removal optimization that can cause segfaults, commented out the dangerous contents for now.
		for stackFinishedMethods.Len() != 0 {
			//itTop := stackFinishedMethods.Peek().(int64)
			//delete(methods, itTop)
			stackFinishedMethods.Pop()
		}

		// verify sums
		outcome := true
		itVerify := 0
		//itEnd := len(items.items) - 1
		itEnd := len(items) - 1

		for ; itVerify != itEnd; itVerify++ {
			if items[itVerify].sum < 0 {
				outcome = false
				// #if DEBUG_
				//fmt.Printf("WARNING: Item %d, sum %.2f\n", items.items[itVerify].(*Item).key, items.items[itVerify].(*Item).sum)
				fmt.Printf("WARNING: Item %d, sum %.2f\n", items[itVerify].key, items[itVerify].sum)
				// #endif
			}
			//printf("Item %d, sum %.2lf\n", it_verify->second.key, it_verify->second.sum);

			//if (math.Ceil(items.items[itVerify].(*Item).sum) + items.items[itVerify].(*Item).sumR) < 0 {
			if (math.Ceil(items[itVerify].sum) + items[itVerify].sumR) < 0 {
				outcome = false

				// #if DEBUG_
				//fmt.Printf("WARNING: Item %d, sum_r %.2f\n", items.items[itVerify].(*Item).key, items.items[itVerify].(*Item).sumR)
				fmt.Printf("WARNING: Item %d, sum_r %.2f\n", items[itVerify].key, items[itVerify].sumR)
				// #endif
			}

			var n float64
			//if items.items[itVerify].(*Item).sumF == 0 {
			if items[itVerify].sumF == 0 {
				n = 0
			} else {
				n = -1
			}

			//if (math.Ceil(items.items[itVerify].(*Item).sum)+items.items[itVerify].(*Item).sumF)*n < 0 {
			if (math.Ceil(items[itVerify].sum)+items[itVerify].sumF)*n < 0 {
				outcome = false
				// #if DEBUG_
				//fmt.Printf("WARNING: Item %d, sum_f %.2f\n", items.items[itVerify].(*Item).key, items.items[itVerify].(*Item).sumF)
				fmt.Printf("WARNING: Item %d, sum_f %.2f\n", items[itVerify].key, items[itVerify].sumF)
				// #endif
			}

		}
		if outcome == true {
			finalOutcome = true
			// #if DEBUG_
			 fmt.Println("-------------Program Correct Up To This Point-------------")
			// #endif
		} else {
			finalOutcome = false

			// #if DEBUG_
			 fmt.Println("-------------Program Not Correct-------------")
			// #endif
		}
	}
}

func work(id int, doneWG *sync.WaitGroup) {
	//fmt.Printf("%d is working!!", id)
	testSize := int32(1)
	wallTime := 0.0
	var tod syscall.Timeval
	if err := syscall.Gettimeofday(&tod); err != nil {
		fmt.Println("Error: get time of day")
		return
	}
	wallTime += float64(tod.Sec)
	wallTime += float64(tod.Usec) * 1e-6

	//var randomGenOp rand.Rand
	//randomGenOp.Seed(int64(wallTime + float64(id) + 1000))
	//s := rand.NewSource(time.Now().UnixNano())
	//randDistOp := rand.New(s)
	//
	//// TODO: I'm 84% sure this is correct
	//startTime := time.Unix(0, start.UnixNano())
	//startTimeEpoch := time.Since(startTime)
	//
	txnCtr.lock.Lock()
	mId := txnCtr.val
	Atomic.AddInt32(&txnCtr.val, 1)
	txnCtr.lock.Unlock()
	//
	//var end time.Time

	//wait()

	for i := int32(0); i < testSize; i++ {

		var res bool
		itemAddr1 := transactions[id].addrSender
		itemAddr2 := transactions[id].addrReceiver
		//opDist := uint32(1 + randDistOp.Intn(100))  // uniformly distributed pseudo-random number between 1 - 100 ??

		//end = time.Now()

		//preFunction := time.Unix(0, end.UnixNano())
		//preFunctionEpoch := time.Since(preFunction)

		// Hmm, do we need .count()??
		//invocation := pre_function_epoch.count() - start_time_epoch.count()
		// invocation := preFunctionEpoch.Nanoseconds() - startTimeEpoch.Nanoseconds()

		// if invocation > (math.MaxInt64 - 10000000000) {
		//PREPROCESSOR DIRECTIVE lines 864 - 866:
		/*
		 * #if DEBUG_
		 *		printf("WARNING: TIME LIMIT REACHED! TERMINATING PROGRAM\n");
		 * #endif
		 */
		// break
		// }

		//if opDist <= 50 {
		//	types = CONSUMER
		//	var itemPop int
		//	// var itemPopPtr *uint64
		//
		//	val := q.Dequeue()
		//	if val != nil{
		//		res = true
		//	} else {
		//		res = false
		//	}
		//	if res {
		//		q.Dequeue()  // try_pop(item_pop)
		//		itemKey = itemPop
		//	}else {
		//		itemKey = math.MaxInt32
		//	}
		//} else {
		//	types = PRODUCER
		//	itemKey = mId
		//	q.Enqueue(itemKey)
		//}

		// line 890
		// end = std::chrono::high_resolution_clock::now();
		//end := time.Now().UnixNano()

		// auto post_function = std::chrono::time_point_cast<std::chrono::nanoseconds>(end);
		//postFunction := end

		// Is this right??
		// auto post_function_epoch = post_function.time_since_epoch();
		//postFunctionEpoch := time.Now().UnixNano() - postFunction

		//response := post_function_epoch.count() - start_time_epoch.count()
		//response := postFunctionEpoch - startTimeEpoch.Nanoseconds()

		// account being added to
		var m1 Method
		m1.setMethod(int(mId), itemAddr1, transactions[id].balanceSender, FIFO, PRODUCER, res, int(mId), transactions[id].amount, transactions[id].tId)

		// account being subtracted from
		Atomic.AddInt32(&mId, 1)
		var m2 Method
		m2.setMethod(int(mId), itemAddr2, transactions[id].balanceReceiver, FIFO, CONSUMER, res, int(mId), -(transactions[id].amount), transactions[id].tId)

		//fmt.Printf("%v\n", m1.itemAddr)

		// mId += numThreads

		//threadLists[id] = append(threadLists[id], m1)
		//csi := ConcurrentSliceItem{int(i), m1}
		//threadLists.items[id].(*ConcurrentSliceItem).Value.(*ConcurrentSlice).Append(csi)
		/*if(len(threadLists.items) == 0) {
			threadLists.items[0].(*ConcurrentSlice).Append(m1)
		} else {
			for i := range threadLists.Iter() {
				if i.Index == id {
					i.Value.(*ConcurrentSlice).Append(csi)
				}
			}
		}*/
		threadLists.Lock()
		//fmt.Printf("threadList len = %d and id = %d\n", len(threadLists.items), id)
		//temp := threadLists.items[id].(ConcurrentSliceItem).Value.([]Method)
		//threadLists.items[id].(ConcurrentSliceItem).Value.([]Method).Append(m1)
		//temp := threadLists.items[id].([]Method)
		threadLists.items[id] = append(threadLists.items[id].([]Method), m1)
		//fmt.Printf("appended: %v\n", threadLists.items[id].([]Method)[0].itemAddr)
		threadListsSize[id].Add(1)
		Atomic.AddInt64(&methodTime[id], 1)
		threadLists.Unlock()
	}

	done[id].Store(true)
	doneWG.Done()
}

func verify(doneWG *sync.WaitGroup) {
	fmt.Println("Verifying...")
	//wait()
	//countIterated = 0

	startTime := time.Unix(0, start.UnixNano())
	startTimeEpoch := time.Since(startTime)

	end := time.Now()

	preVerify := time.Unix(0, end.UnixNano())
	preVerifyEpoch := time.Since(preVerify)

	verifyStart := preVerifyEpoch.Nanoseconds() - startTimeEpoch.Nanoseconds()

	// fnPt       := fncomp
	//TODO: Either need to make these concurrent slices, or if easier just start slapping RWlocks around the use of them.
	methods := make([]Method, 0)
	//methods := NewConcurrentSlice()
	blocks := make([]Block, 0)
	items := make([]Item, 0)
	//items := NewConcurrentSlice()
	it := make([]int, numThreads, numThreads)
	var itStart int

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

	var min int64
	//var oldMin int64
	var itCount [numThreads]int32

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

			tId := i

			// TODO: Correctness not based on time any more, so do we still need response field?
			//var responseTime int64 = 0

			for {
				//threadLists.Lock()
				if itCount[i] >= threadListsSize[i].Load() {
					break
				} else if itCount[i] == 0 {
					it[i] = 0 //threadLists[i].Front()
				} else {
					//++it[i]
					it[i]++
				}
				//fmt.Printf("it[i] = %v\n", it[i])

				var m Method

				m = threadLists.items[tId].([]Method)[it[i]]
				//threadLists.Unlock()

				/*mapMethodsEnd, err := findMethodKey(mapMethods, "end")
				if err != nil{
					return
				}
				*/

				// TODO: Correctness not based on time any more, so do we still need response field?
				/*itMethod := m.response
				for {
					if itMethod == mapMethodsEnd {
						break
					}
					m.response++
					itMethod = m.response
				}
				responseTime = m.response

				mapMethods[m.response] = &m // map_methods.insert ( std::pair<long int,Method>(m.response,m) );
				*/
				//methods.Append(ConcurrentSliceItem{int(m.txnCtr), m})

				//methods.Append(m)
				methods = append(methods, m)

				itCount[i]++
				countOverall++

				//itItem := m.itemKey // it_item = map_items.find(m.item_key);
				//itItem := findIndexForMethod(methods, m, "itemAddr")
				// itItem, _ := findMethodKey(mapMethods, m.itemAddr)

				itItem := 0
				for i := range items {
					if items[i].key == m.itemAddr {
						break
					}
					itItem++
				}
				/*for i := range items.Iter() {
					j := i.Value.(Item)
					if j.key == m.itemAddr {
						break
					}
					itItem++
				}*/

				//mapItemsEnd := len(items) - 1
				/*mapItemsEnd := 0
				for range items.Iter() {
					mapItemsEnd++
				}*/
				mapItemsEnd := len(items)
				mapMethodsEnd := 0
				for range methods {
					mapMethodsEnd++
				}

				if itItem == mapItemsEnd {
					var item Item
					item.setItem(m.itemAddr)
					//item.key = m.itemAddr
					item.producer = mapMethodsEnd

					//items.items[item.key] = &item

					//items.Append(ConcurrentSliceItem{len(items.items), item})
					items = append(items, item)
					//items.Append(item)

					/*for i := range items.Iter() {
						//if i.Index == item.key {
							items.Append(ConcurrentSliceItem{item.key, item})
					}*/
					//itItem, _ = findMethodKey(mapMethods, m.itemAddr)
					/*for i := range methods.Iter() {
						//k := i.Value.(ConcurrentSliceItem)
						//if k.Value.(Method).itemAddr == m.itemAddr {
						if i.Value.(Method).itemAddr == m.itemAddr {
							itItem = i.Index
						}
					}*/
					for i := range methods {
						if methods[i].itemAddr == m.itemAddr {
							itItem = i
						}
					}
				}
			}

			/*if responseTime < min {
				min = responseTime
			}
			*/
		}

		verifyCheckpoint(methods, items, itStart, uint64(countIterated), int64(min), true, blocks)

	}

	verifyCheckpoint(methods, items, itStart, uint64(countIterated), math.MaxInt64, false, blocks)

			//#if DEBUG_
				fmt.Printf("Count overall = %v, count iterated = %d, map_methods.size(1) = %v\n", fmt.Sprint(countOverall), countIterated, fmt.Sprint(len(methods)));
			//#endif

		//#if DEBUG_
			fmt.Printf("All threads finished!\n")

/*
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
	//a := verifyStart / int64(time.Millisecond)
	//b := verifyFinish / int64(time.Millisecond)
	//fmt.Printf("verify start is %d verify finish is %d\n", a, b)

	elapsedTimeVerify = verifyFinish - verifyStart

	doneWG.Done()
}

var transactions [100]TransactionData

func main() {
	// Notes: Need to USE THE CHANNELS.
	// will use for i:= range threadLists.iter() in place of findMethodKey.
	// Should we make methods, items, and blocks ConcurrentSliceItems or slap RWlocks around where we use them?
	// Whats the deal with the separate items slice?
	methodCount = 0

	finalOutcome = true

	//threadLists := NewConcurrentSlice()
	threadLists = ConcurrentSlice{items: make([]interface{}, 0, numThreads),}

	var doneWG sync.WaitGroup

// Generating 50 random transactions
	var hexRunes = []rune("0123456789abcdef")
	var transactionSenders = make([]rune,16)
	var transactionReceivers = make([]rune,16)

	for i := 0; i < 100; i++ {
		for j := 0; j < 16; j++ {
			transactionSenders[j] = hexRunes[rand.Intn(len(hexRunes))]
			transactionReceivers[j] = hexRunes[rand.Intn(len(hexRunes))]
		}
		transactions[i].addrSender = string(transactionSenders)
		transactions[i].addrReceiver = string(transactionReceivers)
		transactions[i].amount = rand.Intn(50)
		transactions[i].balanceSender = rand.Intn(50)
		transactions[i].balanceReceiver = rand.Intn(50)
		transactions[i].tId = txnCtr.val
		Atomic.AddInt32(&txnCtr.val, 1)
	}
	txnCtr.val = 0
	start := time.Now()

	//TODO: thread/ channel stuff

	for i := 0; i < numThreads; i++ {
		//csi := ConcurrentSliceItem{i, NewConcurrentSlice()}
		//cs := &ConcurrentSlice{items: make([], numThreads),}
		//threadLists.items[i] = cs
		//threadLists.RLock()
		//threadLists.Append(ConcurrentSliceItem{i, make([]Method, 0)})
		//threadLists.Append(NewConcurrentSlice())
		threadLists.Append(make([]Method, 0))
		threadListsSize[i].Store(0)
		//threadLists.RUnlock()
		//threadLists.items[i].(*ConcurrentSlice).Append()
		doneWG.Add(1)
		go work(i, &doneWG)
	}
	doneWG.Wait()
	doneWG.Add(1)
	go verify(&doneWG)
	doneWG.Wait()
	fmt.Println("finished working and verifying!")

	if finalOutcome == true {
		fmt.Printf("-------------Program Correct Up To This Point-------------\n")
	} else {
		fmt.Printf("-------------Program Not Correct-------------\n")
	}

	finish := time.Now()                                //auto finish = std::chrono::high_resolution_clock::now();
	elapsedTime := finish.UnixNano() - start.UnixNano() //auto elapsed_time = std::chrono::duration_cast<std::chrono::nanoseconds>(finish - start);

	var elapsedTimeDouble float64 = float64(elapsedTime) * 0.000000001
	fmt.Printf("Total Time: %.15f seconds\n", elapsedTimeDouble)

	var elapsedTimeMethod int64 = 0
	var elapsedOverheadTime float64 = 0

	for i := 0; i < numThreads; i++ {
		if methodTime[i] > elapsedTimeMethod {
			elapsedTimeMethod = methodTime[i]
		}
		//we don't change overheadTime[i] anywhere, neither did Christina by the looks of it
		if float64(overheadTime[i]) > elapsedOverheadTime {
			elapsedOverheadTime = float64(overheadTime[i])
		}
	}

	var elapsedTimeMethodDouble float64 = float64(elapsedTimeMethod) * 0.000000001
	//var elapsedOverheadTimeDouble float64 = elapsedOverheadTime * 0.000000001
	var elapsedTimeVerifyDouble float64 = float64(elapsedTimeVerify) * 0.000000001

	fmt.Printf("Total Method Time: %.15f seconds\n", elapsedTimeMethodDouble)
	//fmt.Printf("Total Overhead Time: %.15f seconds\n", elapsedOverheadTimeDouble)

	elapsedTimeVerifyDouble = elapsedTimeVerifyDouble - elapsedTimeMethodDouble

	fmt.Printf("Total Verification Time: %.15f seconds\n", elapsedTimeVerifyDouble)
}