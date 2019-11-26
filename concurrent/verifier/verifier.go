package correctness_tool

import (
	"C"
	"container/list"
	"github.com/ethereum/go-ethereum/concurrent"
	"github.com/golang-collections/collections/stack"
	"go.uber.org/atomic"
	"math"
	"math/big"
	"strings"
	"sync"
	Atomic "sync/atomic"
)


type Status int
type Semantics int
type Types int

type Method struct {
	id          int       // atomic var
	itemAddrS    string       // sender account address
	itemAddrR    string   // receiver account address
	semantics   Semantics // hardcode as FIFO per last email
	types       Types     // producing/consuming  adding/subtracting
	status      bool
	requestAmnt *big.Int
	// txnCtr      int32
}

type TransactionData struct {
	addrSender   string
	addrReceiver string
	balanceSender int
	balanceReceiver int
	amount       *big.Int
	tId          int32
}

type Verifier struct {
	allSenders      sync.Map // have we seen this address before //map[string]int
	txnCtr          int64				// counts up
	methodCount     int32
	finalOutcome    bool
	done            []atomic.Bool
	threadLists     []*list.List
	numTxns         int32				// what are you?
	isShuttingDown  bool
	isRunning       bool
	txnLock         sync.Mutex
	GlobalCount     int32
}

// method constructor
func  NewMethod(id int, itemAddrS string, itemAddrR string, semantics Semantics, types Types, status bool, requestAmnt *big.Int) *Method {
	return &Method{
		id: id,
		itemAddrS: itemAddrS,
		itemAddrR: itemAddrR,
		semantics: semantics,
		types: types,
		status: status,
		requestAmnt: requestAmnt,
	}
}


// constructor
func NewTxData(sender, receiver string, amount *big.Int, threadID int32) *TransactionData {
	return  &TransactionData{
		addrSender:   sender,
		addrReceiver: receiver,
		amount:       new(big.Int).Set(amount),
		tId:          threadID,
	}
}


// constructor
func NewVerifier() *Verifier {
	lists :=  make([]*list.List, concurrent.NumThreads)

	for i := 0 ; i < concurrent.NumThreads; i++ {
		lists[i] = list.New()
	}
	return  &Verifier{
		allSenders:      sync.Map{},//make(map[string]int),
		txnCtr:          0,
		methodCount:     0,
		finalOutcome:    true,
		done:            make([]atomic.Bool, concurrent.NumThreads, concurrent.NumThreads),
		threadLists:     lists,
		numTxns:         0,
		isRunning:       true,
		isShuttingDown:  false,
		GlobalCount:     0,
	}
}

func (v *Verifier) LockFreeAddTxn(txData *TransactionData) {
	index := Atomic.AddInt32(&v.numTxns, 1) - 1
	// could we get what we are looking for
	var res bool

	res = true
	method1 := NewMethod(int(index), txData.addrSender, txData.addrReceiver, FIFO, PRODUCER, res, txData.amount)

	v.threadLists[txData.tId].PushBack(*method1)
}

func (v *Verifier) handleFailedConsumer(methods map[int]*Method, items map[int]*Item, it int, itItem int, stackFailed *stack.Stack) {
	for it0 := 0; it0 != it; it0++ {
		// serializability
		//451
		if  v.correctnessCondition(it0,it,methods){

		 	//methods[it0].itemAddrS == methods[itItem].itemAddrS &&
			//methods[it0].requestAmnt.Cmp(methods[itItem].requestAmnt) == LESS &&
			//methods[it0].id < methods[itItem].id

			itemItr0 := items[it0].key

			//if methods[it0].types == PRODUCER &&
			if methods[it0].types == CONSUMER &&
				items[itItem].status == PRESENT &&
				methods[it0].semantics == FIFO ||
				strings.Compare(methods[itItem].itemAddrS,methods[it0].itemAddrS) == EQUAL{
				stackFailed.Push(itemItr0)
			}
		}
	}
}

func (v *Verifier) Shutdown() {
	v.isShuttingDown = true
}

func (v *Verifier) verifyCheckpoint(methods map[int]*Method, items map[int]*Item, itStart *int, countIterated *uint64, resetItStart bool) {

	var stackConsumer = stack.New()      // stack of map[int64]*Item
	var stackFinishedMethods stack.Stack // stack of map[int64]*Method
	var stackFailed stack.Stack          // stack of map[int64]*Item

	// TODO: is there a reason for inconsistency between int, int32, and int64?
	//		minor grievance, but it makes reading harder to understand
	v.methodCount = int32(len(methods))
	if v.methodCount != 0 {

		it := 0
		end := len(methods) - 1

		//TODO: corner case
		if *countIterated == 0 {
			resetItStart = false
		} else if it != end {
			*itStart = *itStart + 1
			it = *itStart
		}

		for ; it < len(methods); it++ {

			v.methodCount++

			// 532
			*itStart = it
			resetItStart = false
			*countIterated++

			// itItems := it
			// TODO: I'm a little confused by this block, it maps to line 557 in Christina's code
			// do what if method is a producer?
			if methods[it].types == PRODUCER {
				// set the itItems producer to it. indexes used as keys
				items[it].producer = it

				if items[it].status == ABSENT {

					// reset item parameters
					items[it].status = PRESENT
					// clear item's demote methods
					items[it].demoteMethods.Init()
				}

				// What does addInt do in this context? something to do with promotion/demotion scheme?
				items[it].addInt(1)
				if methods[it].semantics == FIFO {
					// TODO: This starts on line 568, ill come back to it and review
					for it0 := 0; it0 != it; it0++ {
						// starts at first index and works up to current index
						// increment global count for big Oh calculation
						v.GlobalCount++
						// serializability, correctness condition. 576 TODO
						if v.correctnessCondition(it0,it,methods) {
							// itItems0 := it0

							// Demotion if meet correctness condition?
							// FIFO Semantics
							//if (methods.items[it0].(Method).types == PRODUCER && items.items[int(itItems0)].(Item).status == PRESENT) &&

							// promote/ demote based on amount that is being transfered
							//if (methods[it0].types == PRODUCER && items[it0].status == PRESENT) &&
							//	(methods[it].types == PRODUCER && methods[it0].semantics == FIFO) {
							//TODO: Ask christina why demoteMethods is never emptied
							if methods[it].requestAmnt.Cmp(methods[it0].requestAmnt) == LESS && items[it0].status == PRESENT && methods[it0].semantics == FIFO {
								items[it0].promoteItems.Push(items[it].key)
								items[it].demote()
								items[it].demoteMethods.PushBack(methods[it0])

							} 	else if methods[it0].requestAmnt.Cmp(methods[it].requestAmnt) == LESS && items[it0].status == PRESENT && methods[it0].semantics == FIFO {
								items[it].promoteItems.Push(items[it0].key)
								items[it0].demote()
								items[it0].demoteMethods.PushBack(methods[it])
							}

						}
					}
				}
			} else if methods[it].types == CONSUMER {
				// Do what if method is a consumer
				if methods[it].status == true {
					// promote reads
					if items[it].sum > 0 {
						items[it].sumR = 0
					}
					//println("251")
					// line 637
					// sub on consumer?
					items[it].subInt(1)
					items[it].status = ABSENT

					stackConsumer.Push(it)
					stackFinishedMethods.Push(it)

					if items[it].producer != end {
						stackFinishedMethods.Push(items[it].producer)
					}
				} else {
					v.handleFailedConsumer(methods, items, it, it, &stackFailed)
				}
			}
		}
		if resetItStart {
			*itStart--
		}

		for stackConsumer.Len() != 0 {

			// get next item index to whose demoted items need to be promoted
			// itTop not a good var name in this case
			itTop, _ := stackConsumer.Peek().(int)

			// 717

			for items[itTop].promoteItems.Len() != 0 {
				// TODO: Review this section in her code, it may diverge slightly in logic. Line 718
				// get the top item to promote from items's stack of items to promote
				itemPromote := items[itTop].promoteItems.Peek().(int)
				// promote item
				items[itemPromote].promote()
				// finished. pop and move to next item to promote
				items[itTop].promoteItems.Pop()
			}
			stackConsumer.Pop()
		}

		for stackFailed.Len() != 0 {

			itTop := stackFailed.Peek().(int)

				//if items.items[itTop].(*Item).status == PRESENT {
				if items[itTop].status == PRESENT {
					items[itTop].demoteFailed()
				}
				stackFailed.Pop()

		}

		// remove methods from
		for stackFinishedMethods.Len() != 0 {
			// TODO: may need to remove items from methods. if we want to delete need to convert methods and items to map
			//delete(methods, key) // problem: keys are indexes which would change if we remove items from slice
			stackFinishedMethods.Pop()
		}

		// verify sums
		outcome := true

		for itVerify := 0; itVerify < len(items); itVerify++ {
			if items[itVerify].sum < 0 {
				outcome = false
			}else if (math.Ceil(items[itVerify].sum) + items[itVerify].sumR) < 0 {
				outcome = false
			}

			var n int
			//if items.items[itVerify].(*Item).sumF == 0 {
			if items[itVerify].sumF == 0 {
				n = 0
			} else {
				n = -1
			}

			if (math.Ceil(items[itVerify].sum)+items[itVerify].sumF)*float64(n) < 0 {
				outcome = false
			}
		}
		if outcome == true {
			v.finalOutcome = true
		} else {
			v.finalOutcome = false
		}
	}
}

func (v *Verifier) verify() {

	var countIterated uint64 = 0
	Atomic.StoreInt64(&v.txnCtr, int64(Atomic.LoadInt32(&v.numTxns)))
	//methods := make([]Method, 0)
	methods := make(map[int]*Method)
	//items := make([]Item, 0, v.txnCtr * 2)
	items := make(map[int]*Item)
	//it := make([]int, concurrent.NumThreads, concurrent.NumThreads)
	var itStart int

	stop := false
	var countOverall uint32 = 0

	//var oldMin int64
	itCount := make([]int32, concurrent.NumThreads)

	for !stop{
		stop = true

		// for each thread
		for tId := 0; tId < concurrent.NumThreads; tId++ {

			// TODO: This is where Christina has her thread.done() checking
			// can maybe use the shutdown function here, this object may need a wait group to signal when verify has finished

			// iterate thorough each thread's methods
			//for itCount[i] < v.threadListsSize[i].Load() {
			for e := v.threadLists[tId].Front(); e != nil; e = e.Next() {
				// TODO: For concurrent execution can potentially store the list, atomic swap in a new list then evaluate.
				var m Method
				// line 1259
				m = e.Value.(Method)

				m.id = len(methods)

				// Add the method and items to the map, both use m.id as the key

				methods[m.id] = &m
				itCount[tId]++
				countOverall++

				itItemIndex :=  m.id


				itemsEndIndex := len(items)

				// line 1277
				// do something if current item is last item
				if itItemIndex == itemsEndIndex {
					item := NewItem(m.id)
					item.producer = m.id
					items[item.key] = item
				}
			}
		}
		v.verifyCheckpoint(methods, items, &itStart, &countIterated, true)
	}
	v.verifyCheckpoint(methods, items, &itStart, &countIterated, false)
}

func (v *Verifier) Verify() bool{
	v.verify()
	return v.finalOutcome
}

func (v *Verifier) ConcurrentVerify() {
	go v.verify()
}

// transfer ampount < balance
func (v *Verifier) correctnessCondition(index0, index1 int, methods map[int]*Method) bool {
	return methods[index0].itemAddrS == methods[index1].itemAddrS && methods[index0].requestAmnt.Cmp(methods[index1].requestAmnt) == LESS
}