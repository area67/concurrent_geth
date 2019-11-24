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

	v.threadLists.items[txData.tId] = append(v.threadLists.items[txData.tId].([]Method) , *method1)
	v.threadLists.items[txData.tId] = append(v.threadLists.items[txData.tId].([]Method), *method2)

	// TODO: This was v.threadListsSize[txData.tId].Add(1), it should be 2 right?
	v.threadListsSize[txData.tId].Add(2)

}

func (v *Verifier) handleFailedConsumer(methods []Method, items []Item, mk int, it int, stackFailed *stack.Stack) {
	for it0 := 0; it0 != it + 1; it0++ {
		// serializability
		//todo: > or <
		if methods[it0].itemAddrS == methods[it].itemAddrS &&
			methods[it0].requestAmnt.Cmp(methods[it].requestAmnt) == LESS &&
			methods[it0].id < methods[it].id {

			itemItr0 := methods[it0].itemAddrS

			//if methods[it0].types == PRODUCER &&
			if methods[it0].types == CONSUMER &&
				items[it].status == PRESENT &&
				methods[it0].semantics == FIFO ||
				methods[it0].semantics == LIFO ||
				methods[it].itemAddrS == methods[it0].itemAddrS {
				stackFailed.Push(itemItr0)
			}
		} else if methods[it0].itemAddrS == methods[it].itemAddrS &&
					methods[it0].requestAmnt.Cmp(methods[it].requestAmnt) == LESS &&
					methods[it0].id > methods[it].id {


			itemItr0 := methods[it0].itemAddrS

			if methods[it0].types == CONSUMER &&
				items[it].status == PRESENT &&
				methods[it0].semantics == FIFO ||
				methods[it0].semantics == LIFO ||
				methods[it].itemAddrS == methods[it0].itemAddrS {
				stackFailed.Push(itemItr0)
			}
		}
	}
}

func (v *Verifier) Shutdown() {
	v.isShuttingDown = true
}

func (v *Verifier) verifyCheckpoint(methods []Method, items []Item, itStart *int, countIterated *uint64, resetItStart bool) {

	// TODO: This looks like a divergence from her code
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
			// TODO: in her code: it=map_methods.begin();
			//		shouldn't this be it = 0?
			resetItStart = false
		} else if it != end {
			*itStart = *itStart + 1
			it = *itStart
		}

		// TODO: isnt len(methods) equal to "end" in this code?
		for ; it != len(methods); it++ {

			// TODO: What is this?
			if v.methodCount%5000 == 0 {
			}
			// TODO: Minor, but v.methodCount++
			v.methodCount = v.methodCount + 1

			// TODO in her code itStart is an iterator, because this is a sequential program, does itStart need to be a pointer?
			*itStart = it
			resetItStart = false
			*countIterated++

			// TODO: This diverges from her code i believe. it_item = map_items.find(it->second.item_key);
			//			it could contain this value, but if it does im a little confused by it
			itItems := it
			// TODO: I'm a little confused by this block, it maps to line 557 in Christina's code
			if methods[it].types == PRODUCER {
				items[it].producer = it

				if items[itItems].status == ABSENT {

					// reset item parameters
					items[it].status = PRESENT
					// TODO: Minor, but this should probably be a go List object to better mirror her code, go slices are very expensive
					items[it].demoteMethods = nil
				}

				items[it].addInt(1)
				// TODO: If our semantics are constant can we remove this?
				if methods[it].semantics == FIFO {
					// TODO: This starts on line 575, ill come back to it and review
					for it0 := 0; it0 != it + 1; it0++ {
						// serializability
						if methods[it0].itemAddrS == methods[it].itemAddrS &&
							methods[it0].requestAmnt.Cmp(methods[it].requestAmnt) == LESS {
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

			// TODO: we are missing a section about a READER line 603, are we not supporting this?
			//			I'd guess our semnatics only support PRODUCER CONSUMER

			if methods[it].types == CONSUMER {

				if methods[it].status == true {
					if items[itItems].sum > 0 {
						items[itItems].sumR = 0
					}
					// TODO: this is missing: it_item->second.sub_int(1); // line 637
					items[itItems].status = ABSENT

					stackConsumer.Push(itItems)
					stackFinishedMethods.Push(it)

					// TODO: Isnt end already set to this?
					end = len(methods) - 1
					if items[itItems].producer != end {
						stackFinishedMethods.Push(items[itItems].producer)
					}
				} else {
					// TODO she passes it, why are we pssing it + 1?
					v.handleFailedConsumer(methods, items, it + 1, itItems, &stackFailed)
				}
			}
		}
		if resetItStart {
			*itStart--
		}

		for stackConsumer.Len() != 0 {

			itTop, ok := stackConsumer.Peek().(int)
			// TODO: what is this for?
			if !ok {
				return
			}

			for items[itTop].promoteItems.Len() != 0 {
				// TODO: Review this section in her code, it may diverge slightly in logic. Line 718
				itemPromote := items[itTop].promoteItems.Peek().(int)
				itPromoteItem := itemPromote
				items[itPromoteItem].promote()
				items[itTop].promoteItems.Pop()
			}
			stackConsumer.Pop()
		}

		for stackFailed.Len() != 0 {
			for i := 0; i < stackFailed.Len(); i++ {
				// TODO: What is this for?
				// 	line 750 in her code, missing some items here
			}

			// TODO this diverges some,
			temp := stackFailed.Peek().(string)
			for itTop := range items {

				//if items.items[itTop].(*Item).status == PRESENT {
				if items[itTop].key == temp && items[itTop].status == PRESENT {
					items[itTop].demoteFailed()
				}
				stackFailed.Pop()
			}
		}

		//TODO: DANGER, this is the removal optimization that can cause segfaults, commented out the dangerous contents for now.
		for stackFinishedMethods.Len() != 0 {
			stackFinishedMethods.Pop()
		}

		// TODO: ---------------------------------------------------
		// TODO: 				RESUME HERE
		// TODO: ---------------------------------------------------

		// verify sums
		outcome := true
		itVerify := 0
		itEnd := len(items)

		if items[itVerify].sum < 0 {
		}
		for ; itVerify != itEnd; itVerify++ {
			if items[itVerify].sum < 0 {
				outcome = false
			}
			if (math.Ceil(items[itVerify].sum) + items[itVerify].sumR) < 0 {
				outcome = false
			}

			var n float64
			//if items.items[itVerify].(*Item).sumF == 0 {
			if items[itVerify].sumF == 0 {
				n = 0
			} else {
				n = -1
			}

			if (math.Ceil(items[itVerify].sum)+items[itVerify].sumF)*n < 0 {
				outcome = false
			}

		}
		if outcome == true {
			v.finalOutcome = true
			 fmt.Println("-------------Program Correct Up To This Point-------------")
		} else {
			v.finalOutcome = false

			 fmt.Println("-------------Program Not Correct-------------")
		}
	}
}



func (v *Verifier) verify() {
	fmt.Println("Total Transactions: ", v.numTxns)
	fmt.Println("Methods = 2*Txs: ", v.numTxns * 2)
	defer fmt.Println("txnCtr: ", v.txnCtr)


	fmt.Println("Verifying...")

	var countIterated uint64 = 0
	Atomic.StoreInt64(&v.txnCtr, int64(Atomic.LoadInt32(&v.numTxns)))
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

		// for each thread
		for i := 0; i < concurrent.NumThreads; i++ {

			// TODO: This is where Christina has her thread.done() checking
			// iterate thorough each thread's methods
			for itCount[i] < v.threadListsSize[i].Load() {

				// TODO: This can be cleaned up some
				if itCount[i] == 0 {
					it[i] = 0 //threadLists[i].Front()
				} else {
					it[i]++
				}

				var m Method
				var m2 Method
				// TODO: ????? line 1259
				if it[i] < int(v.threadListsSize[i].Load()) {
					// read methods
					// TODO can we do this one at a time b/c we iterate through every method, maybe not #lol
					m = v.threadLists.items[i].([]Method)[it[i]]
					it[i]++
					m2 = v.threadLists.items[i].([]Method)[it[i]]
				}

				// TODO: There is a block of code missing on line 1258 of the tool here

				methods = append(methods, m)
				methods = append(methods, m2)

				itCount[i]++
				countOverall++

				// look up item with given method, parallel arrays, use index

				itItemIndex :=  m.id //len(items) // items at m.item_key


				itemsEndIndex := len(items)-1
				methodsEndIndex := len(methods)-1

				// line 1277
				// do something if current item is last item
				if itItemIndex == itemsEndIndex {
					// TODO: look at the logic of this section, in her code she uses a map(k,v)
					// 		to map item keys to items, why are we not using a map? there also is no
					// 		key stored here. is that not important?
					var item Item
					var item2 Item
					item.setItem(m.itemAddrS)
					item2.setItem(m2.itemAddrS)
					item.producer = methodsEndIndex

					items = append(items, item)
					items = append(items, item2)

					// TODO: This is not in her code at all, what is it for?
					for i := range methods {
						if methods[i].itemAddrS == m.itemAddrS {
							itItemIndex = i
						}
					}
				}
			}
		}

		v.verifyCheckpoint(methods, items, &itStart, &countIterated, true)
	}

	v.verifyCheckpoint(methods, items, &itStart, &countIterated, false)
	fmt.Printf("All threads finished!\n")
}

func (v *Verifier) Verify() {
	v.verify()
}

func (v *Verifier) ConcurrentVerify() {
	go v.verify()
}