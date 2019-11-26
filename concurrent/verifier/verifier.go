package correctness_tool

import (
	"C"
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
	threadLists     [][]Method
	numTxns         int32				// what are you?
	isShuttingDown  bool
	isRunning       bool
	txnLock         sync.Mutex
	globalCount 	int32
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
	lists :=  make([][]Method, concurrent.NumThreads)

	for i := 0 ; i < concurrent.NumThreads; i++ {
		lists[i] = make([]Method, 0)
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
		globalCount:     0,
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
	//if _,seen := v.allSenders.LoadOrStore(txData.addrSender,1); seen {
	//	//
	//	v.allSenders.Store(txData.addrSender, 0)
	//	res = false
	//} else {
	//	v.allSenders.Store(txData.addrSender,1)
	//	res = true
	//}
	res = true
	recAmount := big.NewInt(int64(1))
	recAmount.Mul(txData.amount, big.NewInt(int64(-1)))

	method1 := NewMethod(int(methodIndex),txData.addrSender,txData.addrReceiver,txData.balanceSender,FIFO,PRODUCER,res,int(methodIndex), txData.amount,txData.tId)
	method2 := NewMethod(int(methodIndex + 1),txData.addrSender,txData.addrReceiver,txData.balanceSender,FIFO,CONSUMER,res,int(methodIndex+1),recAmount,txData.tId)

	//println(txData.amount.String(), ", ", recAmount.String())
	v.threadLists[txData.tId] = append(v.threadLists[txData.tId] , *method1)
	v.threadLists[txData.tId] = append(v.threadLists[txData.tId], *method2)

	// TODO: This was v.threadListsSize[txData.tId].Add(1), it should be 2 right?
	// yes - Ross
	v.threadListsSize[txData.tId].Add(2)

}

func (v *Verifier) handleFailedConsumer(methods []Method, items []Item, it int, itItem int, stackFailed *stack.Stack) {
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
		for ; it < len(methods); it++ {


			// TODO: Minor, but v.methodCount++
			v.methodCount = v.methodCount + 1

			// TODO in her code itStart is an iterator, because this is a sequential program, does itStart need to be a pointer?
			// 532
			*itStart = it
			resetItStart = false
			*countIterated++

			// TODO: This diverges from her code i believe. it_item = map_items.find(it->second.item_key);
			//			it could contain this value, but if it does im a little confused by it
			itItems := it
			// TODO: I'm a little confused by this block, it maps to line 557 in Christina's code
			if methods[it].types == PRODUCER {

				items[itItems].producer = it

				if items[itItems].status == ABSENT {

					// reset item parameters
					items[itItems].status = PRESENT
					// TODO: Minor, but this should probably be a go List object to better mirror her code, go slices are very expensive
					items[itItems].demoteMethods = nil
				}

				items[itItems].addInt(1)
				// TODO: If our semantics are constant can we remove this?
				if methods[it].semantics == FIFO {
					// TODO: This starts on line 568, ill come back to it and review
					for it0 := 0; it0 != it; it0++ {
						v.globalCount++
						// serializability, correctness condition. 576 TODO
						if v.correctnessCondition(it0,it,methods) {
							itItems0 := it0

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

				if methods[it].status == true {
					if items[itItems].sum > 0 {
						items[itItems].sumR = 0
					}
					//println("251")
					// line 637
					items[itItems].subInt(1)
					items[itItems].status = ABSENT

					stackConsumer.Push(itItems)
					stackFinishedMethods.Push(it)

					if items[itItems].producer != end {
						stackFinishedMethods.Push(items[itItems].producer)
					}
				} else {
					v.handleFailedConsumer(methods, items, it, itItems, &stackFailed)
				}
			}
		}
		if resetItStart {
			*itStart--
		}

		for stackConsumer.Len() != 0 {

			itTop, _ := stackConsumer.Peek().(int)

			// 717
			for items[itTop].promoteItems.Len() != 0 {
				// TODO: Review this section in her code, it may diverge slightly in logic. Line 718
				itemPromote := items[itTop].promoteItems.Peek().(int)
				items[itemPromote].promote()
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
			// TODO: may need to remove items from methods
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
			 //fmt.Println("-------------Program Correct Up To This Point-------------")
		} else {
			v.finalOutcome = false

			 //fmt.Println("-------------Program Not Correct-------------")
		}
	}
}



func (v *Verifier) verify() {

	var countIterated uint64 = 0
	Atomic.StoreInt64(&v.txnCtr, int64(Atomic.LoadInt32(&v.numTxns)))
	methods := make([]Method, 0)
	items := make([]Item, 0, v.txnCtr * 2)
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
			// iterate thorough each thread's methods
			//for itCount[i] < v.threadListsSize[i].Load() {
			for j := range v.threadLists[tId]{

				var m Method
				// line 1259
				if j < int(v.threadListsSize[tId].Load()) {
					// get thread i's it[i]th method
					m = v.threadLists[tId][j]
				}

				m.id = len(methods)

				methods = append(methods, m)

				itCount[tId]++
				countOverall++

				// look up item with given method, parallel arrays, use index

				itItemIndex :=  m.id


				itemsEndIndex := len(items)
				methodsEndIndex := len(methods)-1

				// line 1277
				// do something if current item is last item
				if itItemIndex == itemsEndIndex {
					// TODO: look at the logic of this section, in her code she uses a map(k,v)
					// 		to map item keys to items, why are we not using a map? there also is no
					// 		key stored here. is that not important?
					var item Item

					item.setItem(m.id)

					item.producer = methodsEndIndex

					items = append(items, item)

				}
			}
		}

		v.verifyCheckpoint(methods, items, &itStart, &countIterated, true)
	}

	v.verifyCheckpoint(methods, items, &itStart, &countIterated, false)
	//println(v.globalCount)
	//println()
}

func (v *Verifier) Verify() bool{
	v.verify()
	return v.finalOutcome
}

func (v *Verifier) ConcurrentVerify() {
	go v.verify()
}

func (v *Verifier) correctnessCondition(index0, index1 int, methods []Method) bool {
	//return methods[index0].itemAddrS == methods[index1].itemAddrS && methods[index0].requestAmnt.Cmp(methods[index1].requestAmnt) == LESS
	return true
}