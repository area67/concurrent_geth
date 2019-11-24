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
	"time"
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

func (v *Verifier) AddTxn(txData *TransactionData) {
	v.txnLock.Lock()
	defer v.txnLock.Unlock()
	// TODO : I think we can do  index = Atomic.AddInt64(&v.txnCtr, 1) - 1
	//		the add function returns the new value, which whill always be the new size of the array, and we want to
	//		insert at size - 1
	//		see LockFreeAddTxn
	v.transactions[v.txnCtr] = *txData
	v.txnCtr++
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
	//println(txData.tId)

	v.threadLists.items[txData.tId] = append(v.threadLists.items[txData.tId].([]Method) , *method1)
	v.threadLists.items[txData.tId] = append(v.threadLists.items[txData.tId].([]Method), *method2)
	v.threadListsSize[txData.tId].Add(1)

}

func (v *Verifier) handleFailedConsumer(methods []Method, items []Item, mk int, it int, stackFailed *stack.Stack) {
	//fmt.Printf("Handling failed consumer...it is %d\n", it)
	//fmt.Println(methods)

	for it0 := 0; it0 != it + 1; it0++ {
		//fmt.Printf("it0 address = %s and it address = %s\nit0 requestAmnt = %d and it requestAmnt = %d\n", methods[it0].itemAddrS, methods[it].itemAddrS, methods[it0].requestAmnt, methods[it].requestAmnt)
		// serializability
		//todo: > or <
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
	//fmt.Println("Verifying Checkpoint...")

	var stackConsumer = stack.New()      // stack of map[int64]*Item
	var stackFinishedMethods stack.Stack // stack of map[int64]*Method
	var stackFailed stack.Stack          // stack of map[int64]*Item

	v.methodCount = int32(len(methods))
	//for i := range methods.Iter() {
		//fmt.Printf("methods.items[0].semantics = %v\n", methods.items[0].(Method).semantics)
	//}
	if v.methodCount != 0 {

		it := 0
		end := len(methods) - 1

		// fmt.Printf("end of methods = %d\n", end)

		//TODO: corner case
		if *countIterated == 0 {
			// fmt.Printf("setting resetItStart to false\n")
			resetItStart = false
		} else if it != end {
			// fmt.Printf("Incrementing itStart and it\n")
			*itStart = *itStart + 1
			it = *itStart
		}
		//fmt.Printf("it = %d\n", it)

		for ; it != len(methods); it++ {
			// fmt.Printf("it is %d and len items is %d len methods is %d\n", it, len(items), len(methods))
			//fmt.Printf("!The sum at %s is %f\n", items[it].key, items[it].sum)

			if v.methodCount%5000 == 0 {
				// fmt.Printf("methodCount = %d\n", v.methodCount)
			}
			v.methodCount = v.methodCount + 1

			*itStart = it
			resetItStart = false
			*countIterated++


			itItems := it //methods[it].itemAddr
			//itItems := int(methods[it].txnCtr)

			if methods[it].types == PRODUCER {
				// fmt.Printf("PRODUCER\n")
				items[it].producer = it

				if items[itItems].status == ABSENT {

					// reset item parameters
					items[it].status = PRESENT
					items[it].demoteMethods = nil
				}

				items[it].addInt(1)

				if methods[it].semantics == FIFO {
					// fmt.Println("MADE IT")
					for it0 := 0; it0 != it + 1; it0++ {
						// fmt.Println("MADE IT in 1")
						// serializability
						if methods[it0].itemAddrS == methods[it].itemAddrS &&
							methods[it0].requestAmnt.Cmp(methods[it].requestAmnt) == LESS {
							// fmt.Println("MADE IT in 2")
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

			/*if methods[it].semantics == FIFO {
				for it0 := 0; it0 != it; it0++{
					// serializability
					if methods[it0].itemAddr == methods[it].itemAddr &&
						methods[it0].requestAmnt > methods[it].requestAmnt{
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
			}*/
			//tempMethods = methods.items[it].(Method)
			//if methods.items[it].(*Method).types == CONSUMER {
			if methods[it].types == CONSUMER {
				//fmt.Printf("CONSUMER\n")
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
					//fmt.Printf("methods[%d].status == true (ABSENT)\n", it)
					// promote reads
					//if items.items[itItems].(*Item).sum > 0 {
					if items[itItems].sum > 0 {
						//items.items[itItems].(*Item).sumR = 0
						items[itItems].sumR = 0
					}

					//items[itItems].subInt(1)
					//fmt.Printf("subInt at items[%d] which has key %s, resulting in sum of %f\n", itItems, items[itItems].key, items[itItems].sum)
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
					v.handleFailedConsumer(methods, items, it + 1, itItems, &stackFailed)
				}
			}
		}
		if resetItStart {
			*itStart--
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
			// fmt.Printf("stackFailed length non zero:\n")
			for i := 0; i < stackFailed.Len(); i++ {
				//fmt.Println(stackFailed.Pop())
			}
			//itTop := stackFailed.Peek().(int)
			/*if items[itTop].status == PRESENT {
				//items.items[itTop].(*Item).demoteFailed()
				items[itTop].demoteFailed()
			}
			stackFailed.Pop()*/
			temp := stackFailed.Peek().(string)
			for itTop := range items {

				//if items.items[itTop].(*Item).status == PRESENT {
				if items[itTop].key == temp && items[itTop].status == PRESENT {
					//items.items[itTop].(*Item).demoteFailed()
					//fmt.Printf("Demoting item...\n")
					items[itTop].demoteFailed()
				}
				stackFailed.Pop()
			}
		}

		// remove methods that are no longer active
		//TODO: DANGER, this is the removal optimization that can cause segfaults, commented out the dangerous contents for now.
		for stackFinishedMethods.Len() != 0 {
			//itTop := stackFinishedMethods.Peek().(int64)
			//(methods, itTop)
			stackFinishedMethods.Pop()
		}

		// verify sums
		outcome := true
		itVerify := 0
		//itEnd := len(items.items) - 1
		itEnd := len(items)

		if items[itVerify].sum < 0 {
			//fmt.Printf("Negative sum!\n")
		}
		for ; itVerify != itEnd; itVerify++ {
			if items[itVerify].sum < 0 {
				outcome = false
				//fmt.Printf("WARNING: Item %d, sum %.2f\n", items.items[itVerify].(*Item).key, items.items[itVerify].(*Item).sum)
				//fmt.Printf("WARNING1: Item %s (items[%d]), sum %.2f\n", items[itVerify].key, itVerify, items[itVerify].sum)
			}
			//printf("Item %d, sum %.2lf\n", it_verify->second.key, it_verify->second.sum);

			//if (math.Ceil(items.items[itVerify].(*Item).sum) + items.items[itVerify].(*Item).sumR) < 0 {
			if (math.Ceil(items[itVerify].sum) + items[itVerify].sumR) < 0 {
				outcome = false

				// #if DEBUG_
				//fmt.Printf("WARNING: Item %d, sum_r %.2f\n", items.items[itVerify].(*Item).key, items.items[itVerify].(*Item).sumR)
				//fmt.Printf("WARNING2: Item %s, sum_r %.2f\n", items[itVerify].key, items[itVerify].sumR)
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
			//fmt.Printf("prior to outcome = false, at items[%d] key = %s, sum = %f and sumF = %f and sumR = %f and n = %v and outcome = %b\n",itVerify, items[itVerify].key, items[itVerify].sum, items[itVerify].sumF, items[itVerify].sumR, n, outcome)
			if (math.Ceil(items[itVerify].sum)+items[itVerify].sumF)*n < 0 {
				//fmt.Printf("!!!!!!!!!!\n")
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

			// #if DEBUG_
			 fmt.Println("-------------Program Not Correct-------------")
			// #endif
		}
	}
}



func (v *Verifier) verify() {

	// need to iterate on v.transactions

	fmt.Println("Verifying...")

	var countIterated uint64 = 0

	// fnPt       := fncomp
	methods := make([]Method, 0)
	//methods := NewConcurrentSlice()
	//blocks := make([]Block, 0)
	//items := make([]Item, 0, numTxns * 2)
	//fmt.Printf("txnCtr is %v\n", v.txnCtr)
	items := make([]Item, 0, v.txnCtr * 2)
	it := make([]int, concurrent.NumThreads, concurrent.NumThreads)
	var itStart int

	stop := false
	var countOverall uint32 = 0


	//var oldMin int64
	itCount := make([]int32, concurrent.NumThreads)

	// runs once when concurrent history generated for now
	for !stop{

		stop = true


		for i := 0; i < concurrent.NumThreads; i++ {
			// check if concurrent history is done generating in all threads
			// TODO: figure out
			//if v.done[i].Load() == false {
			//	stop = false
			//}

			for itCount[i] < v.threadListsSize[i].Load(){

				//fmt.Printf("itCount[%d]: %d\tthreadListsSize[%d]: %d\n", i, itCount[i], i, v.threadListsSize[i].Load())

				if itCount[i] == 0 {
					it[i] = 0 //threadLists[i].Front()
				} else {
					it[i]++
				}
				//fmt.Printf("it[i] = %v\n", it[i])

				var m Method
				var m2 Method

				//if it[i] < len(threadLists.items[i].([]Method)) {
				if it[i] < int(v.threadListsSize[i].Load()) {
					//fmt.Printf("Address of methods txn sender at thread %d index %d: %s and at index %d: %s\n", i, it[i], v.threadLists.items[i].([]Method)[it[i]].itemAddrS, it[i] + 1, v.threadLists.items[i].([]Method)[it[i] + 1].itemAddrS)
					//fmt.Printf("%v\n", v.threadLists.items[i].([]Method))
					// assign methods
					m = v.threadLists.items[i].([]Method)[it[i]]
					it[i]++
					m2 = v.threadLists.items[i].([]Method)[it[i]]
				} else {
					//fmt.Printf("Verifier threadlist error!\n")
					break
				}
				//fmt.Printf("m address = %s\nm2 address = %s\n", m.itemAddrS, m2.itemAddrS)

				methods = append(methods, m)
				methods = append(methods, m2)

				itCount[i]++
				countOverall++


				itItem := 0

				for range items {
					/*if items[i].key == m.itemAddrS {
						break
					}*/
					itItem++
				}


				mapItemsEnd := len(items)
				mapMethodsEnd := 0
				for range methods {
					mapMethodsEnd++
				}
				//fmt.Printf("%d\t%d\n", itItem, mapItemsEnd)
				if itItem == mapItemsEnd {
					var item Item
					var item2 Item
					//fmt.Printf("appending addresses to items: %v\t%v\n", m.itemAddrS, m2.itemAddrS)
					item.setItem(m.itemAddrS)
					item2.setItem(m2.itemAddrS)
					//item.key = m.itemAddr
					item.producer = mapMethodsEnd

					//items.items[item.key] = &item

					//items.Append(ConcurrentSliceItem{len(items.items), item})
					items = append(items, item)
					items = append(items, item2)
					//items.Append(item)

					//itItem, _ = findMethodKey(mapMethods, m.itemAddr)

					for i := range methods {
						if methods[i].itemAddrS == m.itemAddrS {
							itItem = i
						}
					}
				}/* else {
					fmt.Print(items)
				}*/
			}

			/*if responseTime < min {
				min = responseTime
			}
			*/
		}

		v.verifyCheckpoint(methods, items, &itStart, &countIterated, true)

	}

	v.verifyCheckpoint(methods, items, &itStart, &countIterated, false)

			//#if DEBUG_
				//fmt.Printf("Count overall = %v, count iterated = %d, methods size = %d, items size = %d\n", fmt.Sprint(countOverall), countIterated, len(methods), len(items));
			//#endif

		//#if DEBUG_
			fmt.Printf("All threads finished!\n")


	//a := verifyStart / int64(time.Millisecond)
	//b := verifyFinish / int64(time.Millisecond)
	//fmt.Printf("verify start is %d verify finish is %d\n", a, b)

	//doneWG.Done()
}

func (v *Verifier) Verify() {

	go v.mainLoop()
}

// what do you do?
//
//func (v *Verifier) work(id int, doneWG *sync.WaitGroup) {
//	//fmt.Printf("%d is working!!", id)
//	testSize := int32(1)
//
//	//var randomGenOp rand.Rand
//	//randomGenOp.Seed(int64(wallTime + float64(id) + 1000))
//	//s := rand.NewSource(time.Now().UnixNano())
//	//randDistOp := rand.New(s)
//	//
//	//// TODO: I'm 84% sure this is correct
//	//startTime := time.Unix(0, start.UnixNano())
//	//startTimeEpoch := time.Since(startTime)
//
//
//	//TODO: could this be Atomic.LoadInt64(&txnCtr) * 2
//	mId := Atomic.LoadInt64(&v.txnCtr) * 2
//	//Atomic.AddInt64(&txnCtr.val, 1)
//	//
//	//var end time.Time
//
//	//wait()
//	//if(Atomic.LoadInt32(&numTxns) == 0) {
//	//return;
//	//}
//
//	// check if we are done
//	if v.numTxns == 0 {
//		v.done[id].Store(true)
//		doneWG.Done()
//		return
//	} else {
//		Atomic.AddInt32(&v.numTxns, -1)
//	}
//
//	for i := int32(0); i < testSize; i++ {
//
//		/*if Atomic.LoadInt32(&numTxns) == 0 {
//			break;
//		}*/
//		var res bool
//		// gets
//		index := Atomic.LoadInt64(&v.txnCtr)
//		itemAddr1 := v.transactions[index].addrSender
//		itemAddr2 := v.transactions[index].addrReceiver
//		amount := v.transactions[index].amount
//		fmt.Println(v.transactions[index])
//		Atomic.AddInt64(&v.txnCtr, 1)
//
//
//		// if seen address before
//		if v.allSenders[itemAddr1] == 1 {
//
//			v.allSenders[itemAddr1] = 0
//			res = false
//		} else {
//			v.allSenders[itemAddr1] = 1
//			res = true
//		}
//
//		fmt.Printf("res for %s is %v\n", itemAddr1, res)
//		var m1 Method
//		m1.setMethod(int(mId), itemAddr1, itemAddr2, v.transactions[id].balanceSender, FIFO, PRODUCER, res, int(mId), amount, v.transactions[id].tId)
//
//		Atomic.AddInt64(&mId, 1)
//		var m2 Method
//		m2.setMethod(int(mId),itemAddr1, itemAddr2, v.transactions[id].balanceReceiver, FIFO, CONSUMER, res, int(mId), amount.Mul(amount, big.NewInt(int64(-1))), v.transactions[id].tId)
//		Atomic.AddInt64(&mId, 1)
//
//		//Atomic.AddInt32(&numTxns, -1)
//		// mId += numThreads
//
//		//threadLists[id] = append(threadLists[id], m1)
//		//csi := ConcurrentSliceItem{int(i), m1}
//		//threadLists.items[id].(*ConcurrentSliceItem).Value.(*ConcurrentSlice).Append(csi)
//		/*if(len(threadLists.items) == 0) {
//			threadLists.items[0].(*ConcurrentSlice).Append(m1)
//		} else {
//			for i := range threadLists.Iter() {
//				if i.Index == id {
//					i.Value.(*ConcurrentSlice).Append(csi)
//				}
//			}
//		}*/
//		//threadLists.Lock()
//		//TODO: we want to append both...right?
//		v.threadLists.items[id] = append(v.threadLists.items[id].([]Method), m1)
//		v.threadLists.items[id] = append(v.threadLists.items[id].([]Method), m2)
//		//fmt.Printf("threadlist %d: %v\n", id, threadLists.items[id])
//		v.threadListsSize[id].Add(1)
//		//Atomic.AddInt64(&methodTime[id], 1)
//		//threadLists.Unlock()
//	}
//
//	v.done[id].Store(true)
//	doneWG.Done()
//}

//
func (v *Verifier) mainLoop() {
	//fmt.Println("start main loop")
	//defer fmt.Println("end main loop")

	//methodTime := make([]int64, concurrent.NumThreads)

	currentSize := int64(0)
	for v.isRunning {
		//fmt.Println(v.transactions[0])
		currentSize = Atomic.LoadInt64(&v.txnCtr)

		// will use for i:= range threadLists.iter() in place of findMethodKey.
		// Should we make methods, items, and blocks ConcurrentSliceItems or slap RWlocks around where we use them?
		// Whats the deal with the separate items slice?
		//allSenders := make(map[string]int)



		Atomic.StoreInt32(&v.numTxns,int32( len(v.transactions)-1))


		var doneWG sync.WaitGroup
		//var control string


		// does this make the concurrent history?
		/*
		for i := 0; i < concurrent.NumThreads; i++ {
			// add empty method to thread list?
			v.threadLists.Append(make([]Method, 0))
			v.threadListsSize[i].Store(0)
			doneWG.Add(1)
			go v.work(i, &doneWG)
			doneWG.Wait()
		}
		*/

		//doneWG.Wait()
		doneWG.Add(1)
		go v.verify()
		doneWG.Wait()
		//fmt.Println("finished working and verifying!")

		//fmt.Printf("Control was: %s\n", control)

		if v.finalOutcome == true {
			fmt.Printf("-------------Program Correct Up To This Point-------------\n")
		} else {
			fmt.Printf("-------------Program Not Correct-------------\n")
		}

		finish := time.Now() //auto finish = std::chrono::high_resolution_clock::now();
		elapsedSeconds := time.Since(finish).Seconds()
		fmt.Println("Total Time: ", elapsedSeconds, " seconds")



		if currentSize == Atomic.LoadInt64(&v.txnCtr) && v.isShuttingDown {
			//fmt.Println("this happened")
			v.isRunning = false
		}
	}
}