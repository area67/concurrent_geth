package correctness_tool

import (
	"C"
	"fmt"
	"github.com/ethereum/go-ethereum/concurrent"
	"github.com/golang-collections/collections/stack"
	"go.uber.org/atomic"
	"math"
	"math/big"
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
	itemBalance int       // account balance
	semantics   Semantics // hardcode as FIFO per last email
	types       Types     // producing/consuming  adding/subtracting
	status      bool
	senderID    int       // same as itemAddr ??
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
	transactions    []TransactionData
	txnCtr          int64				// counts up
	methodCount     int32
	finalOutcome    bool
	done            []atomic.Bool
	threadListsSize []atomic.Int32
	threadLists     ConcurrentSlice
	numTxns         int32				// what are you?
	isShuttingDown  bool
	isRunning       bool
	txnLock         sync.Mutex
}

// method constructor
func  NewMethod(id int, itemAddrS string, itemAddrR string, itemBalance int, semantics Semantics,
	types Types, status bool, senderID int, requestAmnt *big.Int, txnCtr int32)*Method {
	return &Method{
		id: id,
		itemAddrS: itemAddrS,
		itemAddrR: itemAddrR,
		itemBalance: itemBalance,
		semantics: semantics,
		types: types,
		status: status,
		requestAmnt: requestAmnt,
	}

}
func (m *Method) setMethod(id int, itemAddrS string, itemAddrR string, itemBalance int, semantics Semantics,
	types Types, status bool, senderID int, requestAmnt *big.Int, txnCtr int32) {
	m.id = id
	m.itemAddrS = itemAddrS
	m.itemAddrR = itemAddrR
	m.itemBalance = itemBalance
	m.semantics = semantics
	m.types = types
	m.status = status
	//m.senderID = senderID
	m.requestAmnt = requestAmnt
	// m.txnCtr = txnCtr
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
	lists := ConcurrentSlice{items: make([]interface{}, concurrent.NumThreads),}

	for i := 0 ; i < concurrent.NumThreads; i++ {
		lists.items[i] = make([]Method, 0)
	}
	return  &Verifier{
		allSenders:      sync.Map{},//make(map[string]int),
		txnCtr:          0,
		transactions:    make([]TransactionData, 200),
		methodCount:     0,
		finalOutcome:    true,
		done:            make([]atomic.Bool, concurrent.NumThreads, concurrent.NumThreads),
		threadListsSize: make([]atomic.Int32, concurrent.NumThreads, concurrent.NumThreads),
		threadLists:     lists,
		numTxns:         0,
		isRunning:       true,
		isShuttingDown:  false,
	}
}

func (v *Verifier) LockFreeAddTxn(txData *TransactionData) {
	index := Atomic.AddInt32(&v.numTxns, 1) - 1
	v.transactions[index] = *txData
	methodIndex := index * 2
	// could we get what we are looking for
	var res bool
	// if seen address before
	// TODO: Ask Christina. Should this be a toggle every time we see a sender?
	if _,seen := v.allSenders.LoadOrStore(txData.addrSender,0); seen {

		v.allSenders.Store(txData.addrSender, 0)
		res = false
	} else {
		v.allSenders.Store(txData.addrSender,1)
		res = true
	}

	recAmount := big.NewInt(int64(0)).Mul(txData.amount, big.NewInt(int64(-1)))

	method1 := NewMethod(int(methodIndex),txData.addrSender,txData.addrReceiver,txData.balanceSender,FIFO,PRODUCER,res,int(methodIndex), txData.amount,txData.tId)
	method2 := NewMethod(int(methodIndex + 1),txData.addrSender,txData.addrReceiver,txData.balanceSender,FIFO,CONSUMER,res,int(methodIndex+1),recAmount,txData.tId)

	// println(v.threadLists.items[0])
	println(txData.tId)

	v.threadLists.items[txData.tId] = append(v.threadLists.items[txData.tId].([]Method) , *method1)
	v.threadLists.items[txData.tId] = append(v.threadLists.items[txData.tId].([]Method), *method2)
	v.threadListsSize[txData.tId].Add(1)

}

// methodMapKey and itemMapKey are meant to serve in place of iterators
func (v *Verifier) handleFailedConsumer(methods []Method, items []Item, mk int, it int, stackFailed *stack.Stack) {
	fmt.Printf("Handling failed consumer...it is %d\n", it)
	fmt.Println(methods)

	for it0 := 0; it0 != it + 1; it0++ {
		fmt.Printf("it0 address = %s and it address = %s\nit0 requestAmnt = %d and it requestAmnt = %d\n", methods[it0].itemAddrS, methods[it].itemAddrS, methods[it0].requestAmnt, methods[it].requestAmnt)
		// serializability
		//todo: > or <
		/*if (methods[it0].itemAddrS == methods[it].itemAddrS &&
			math.Abs(float64(methods[it0].requestAmnt)) < math.Abs(float64(methods[it].requestAmnt)) &&
			methods[it0].id < methods[it].id) {*/
		if methods[it0].itemAddrS == methods[it].itemAddrS &&
			methods[it0].requestAmnt.Cmp(methods[it].requestAmnt) == LESS &&
			methods[it0].id < methods[it].id {

			fmt.Printf("Handling failed consumer 2\n")

			itemItr0 := methods[it0].itemAddrS

			//if methods[it0].types == PRODUCER &&
			if methods[it0].types == CONSUMER &&
				items[it].status == PRESENT &&
				methods[it0].semantics == FIFO ||
				methods[it0].semantics == LIFO ||
				methods[it].itemAddrS == methods[it0].itemAddrS {
				fmt.Printf("Handling failed consumer 3\n")
				stackFailed.Push(itemItr0)
			}
		} else if methods[it0].itemAddrS == methods[it].itemAddrS &&
					methods[it0].requestAmnt.Cmp(methods[it].requestAmnt) == LESS &&
					methods[it0].id > methods[it].id {

			fmt.Printf("Handling failed consumer 4\n")

			itemItr0 := methods[it0].itemAddrS

			if methods[it0].types == CONSUMER &&
				items[it].status == PRESENT &&
				methods[it0].semantics == FIFO ||
				methods[it0].semantics == LIFO ||
				methods[it].itemAddrS == methods[it0].itemAddrS {
				fmt.Printf("Handling failed consumer 5\n")
				stackFailed.Push(itemItr0)
			}
		}
	}
}

func (v *Verifier) Shutdown() {
	v.isShuttingDown = true
}

func (v *Verifier) verifyCheckpoint(methods []Method, items []Item, itStart *int, countIterated *uint64, resetItStart bool) {

	var stackConsumer = stack.New()      // stack of map[int64]*Item
	var stackFinishedMethods stack.Stack // stack of map[int64]*Method
	var stackFailed stack.Stack          // stack of map[int64]*Item

	v.methodCount = int32(len(methods))
	if v.methodCount != 0 {

		it := 0
		end := len(methods) - 1

		fmt.Printf("end of methods = %d\n", end)

		//TODO: corner case
		if *countIterated == 0 {
			fmt.Printf("setting resetItStart to false\n")
			resetItStart = false
		} else if it != end {
			fmt.Printf("Incrementing itStart and it\n")
			*itStart = *itStart + 1
			it = *itStart
		}

		for ; it != len(methods); it++ {
			fmt.Printf("it is %d and len items is %d len methods is %d\n", it, len(items), len(methods))

			if v.methodCount%5000 == 0 {
				fmt.Printf("methodCount = %d\n", v.methodCount)
			}
			v.methodCount = v.methodCount + 1

			*itStart = it
			resetItStart = false
			*countIterated++


			itItems := it //methods[it].itemAddr

			if methods[it].types == PRODUCER {
				fmt.Printf("PRODUCER\n")
				items[it].producer = it

				if items[itItems].status == ABSENT {

					// reset item parameters
					items[it].status = PRESENT
					items[it].demoteMethods = nil
				}

				items[it].addInt(1)

				if methods[it].semantics == FIFO {
					fmt.Println("MADE IT")
					for it0 := 0; it0 != it + 1; it0++ {
						fmt.Println("MADE IT in 1")
						// serializability
						if methods[it0].itemAddrS == methods[it].itemAddrS &&
							methods[it0].requestAmnt.Cmp(methods[it].requestAmnt) == LESS {
							fmt.Println("MADE IT in 2")
							// #endif
							itItems0 := 0

							// Demotion
							// FIFO Semantics
							//if (methods.items[it0].(Method).types == PRODUCER && items.items[int(itItems0)].(Item).status == PRESENT) &&
							if (methods[it0].types == PRODUCER && items[itItems0].status == PRESENT) &&
								(methods[it].types == PRODUCER && methods[it0].semantics == FIFO) {

								items[itItems0].promoteItems.Push(items[itItems].key)
								items[itItems].demote()
								items[itItems].demoteMethods = append(items[itItems].demoteMethods, &methods[it0])
							}
						}
					}
				}
			}


			if methods[it].types == CONSUMER {
				fmt.Printf("CONSUMER\n")

				if methods[it].status == true {
					fmt.Printf("methods[%d].status == true (ABSENT)\n", it)

					if items[itItems].sum > 0 {
						items[itItems].sumR = 0
					}

					fmt.Printf("subInt at items[%d] which has key %s, resulting in sum of %f\n", itItems, items[itItems].key, items[itItems].sum)
					items[itItems].status = ABSENT

					stackConsumer.Push(itItems)
					stackFinishedMethods.Push(it)

					end = len(methods) - 1
					if items[itItems].producer != end {
						stackFinishedMethods.Push(items[itItems].producer)
					}
				} else {
					v.handleFailedConsumer(methods, items, it + 1, itItems, &stackFailed)
				}
			}
		}
		if resetItStart {
			*itStart--
		}

		for stackConsumer.Len() != 0 {

			itTop, ok := stackConsumer.Peek().(int)
			if !ok {
				return
			}

			for items[itTop].promoteItems.Len() != 0 {
				itemPromote := items[itTop].promoteItems.Peek().(int)
				itPromoteItem := itemPromote
				items[itPromoteItem].promote()
				items[itTop].promoteItems.Pop()
			}
			stackConsumer.Pop()
		}

		for stackFailed.Len() != 0 {
			fmt.Printf("stackFailed length non zero:\n")
			for i := 0; i < stackFailed.Len(); i++ {
				fmt.Println(stackFailed.Pop())
			}

			temp := stackFailed.Peek().(string)
			for itTop := range items {

				//if items.items[itTop].(*Item).status == PRESENT {
				if items[itTop].key == temp && items[itTop].status == PRESENT {
					//items.items[itTop].(*Item).demoteFailed()
					fmt.Printf("Demoting item...\n")
					items[itTop].demoteFailed()
				}
				stackFailed.Pop()
			}
		}

		//TODO: DANGER, this is the removal optimization that can cause segfaults, commented out the dangerous contents for now.
		for stackFinishedMethods.Len() != 0 {
			stackFinishedMethods.Pop()
		}

		// verify sums
		outcome := true
		itVerify := 0
		itEnd := len(items)

		if items[itVerify].sum < 0 {
			fmt.Printf("Negative sum!\n")
		}
		for ; itVerify != itEnd; itVerify++ {
			if items[itVerify].sum < 0 {
				outcome = false
				fmt.Printf("WARNING1: Item %s (items[%d]), sum %.2f\n", items[itVerify].key, itVerify, items[itVerify].sum)
			}
			if (math.Ceil(items[itVerify].sum) + items[itVerify].sumR) < 0 {
				outcome = false

				fmt.Printf("WARNING2: Item %s, sum_r %.2f\n", items[itVerify].key, items[itVerify].sumR)
			}

			var n float64
			//if items.items[itVerify].(*Item).sumF == 0 {
			if items[itVerify].sumF == 0 {
				n = 0
			} else {
				n = -1
			}

			if (math.Ceil(items[itVerify].sum)+items[itVerify].sumF)*n < 0 {
				fmt.Printf("!!!!!!!!!!\n")
				outcome = false
			}

		}
		if outcome == true {
			v.finalOutcome = true
			// #if DEBUG_
			 fmt.Println("-------------Program Correct Up To This Point-------------")
			// #endif
		} else {
			v.finalOutcome = false

			 fmt.Println("-------------Program Not Correct-------------")
		}
	}
}



func (v *Verifier) verify() {

	fmt.Println("Verifying...")

	var countIterated uint64 = 0

	methods := make([]Method, 0)
	fmt.Printf("txnCtr is %v\n", v.txnCtr)
	items := make([]Item, 0, v.txnCtr * 2)
	it := make([]int, concurrent.NumThreads, concurrent.NumThreads)
	var itStart int

	stop := false
	var countOverall uint32 = 0


	//var oldMin int64
	itCount := make([]int32, concurrent.NumThreads)

	for !stop{

		stop = true


		for i := 0; i < concurrent.NumThreads; i++ {
			for itCount[i] < v.threadListsSize[i].Load(){

				fmt.Printf("itCount[%d]: %d\tthreadListsSize[%d]: %d\n", i, itCount[i], i, v.threadListsSize[i].Load())

				if itCount[i] == 0 {
					it[i] = 0 //threadLists[i].Front()
				} else {
					it[i]++
				}

				var m Method
				var m2 Method

				//if it[i] < len(threadLists.items[i].([]Method)) {
				if it[i] < int(v.threadListsSize[i].Load()) {
					fmt.Printf("Address of methods txn sender at thread %d index %d: %s and at index %d: %s\n", i, it[i], v.threadLists.items[i].([]Method)[it[i]].itemAddrS, it[i] + 1, v.threadLists.items[i].([]Method)[it[i] + 1].itemAddrS)
					fmt.Printf("%v\n", v.threadLists.items[i].([]Method))
					// assign methods
					m = v.threadLists.items[i].([]Method)[it[i]]
					it[i]++
					m2 = v.threadLists.items[i].([]Method)[it[i]]
				} else {
					fmt.Printf("Verifier threadlist error!\n")
					break
				}
				fmt.Printf("m address = %s\nm2 address = %s\n", m.itemAddrS, m2.itemAddrS)

				methods = append(methods, m)
				methods = append(methods, m2)

				itCount[i]++
				countOverall++


				itItem := 0

				for range items {
					itItem++
				}


				mapItemsEnd := len(items)
				mapMethodsEnd := 0
				for range methods {
					mapMethodsEnd++
				}
				fmt.Printf("%d\t%d\n", itItem, mapItemsEnd)
				if itItem == mapItemsEnd {
					var item Item
					var item2 Item
					fmt.Printf("appending addresses to items: %v\t%v\n", m.itemAddrS, m2.itemAddrS)
					item.setItem(m.itemAddrS)
					item2.setItem(m2.itemAddrS)
					item.producer = mapMethodsEnd

					items = append(items, item)
					items = append(items, item2)

					for i := range methods {
						if methods[i].itemAddrS == m.itemAddrS {
							itItem = i
						}
					}
				}
			}
		}

		v.verifyCheckpoint(methods, items, &itStart, &countIterated, true)

	}

	v.verifyCheckpoint(methods, items, &itStart, &countIterated, false)
	fmt.Printf("All threads finished!\n")


	//fmt.Printf("verify start is %d verify finish is %d\n", a, b)
}

func (v *Verifier) Verify() {

	v.verify()
}