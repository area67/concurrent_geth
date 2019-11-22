package correctness_tool

import (
	"C"
	"fmt"
	"github.com/ethereum/go-ethereum/concurrent"
	"github.com/golang-collections/collections/stack"
	"go.uber.org/atomic"
	"math"
	"math/big"
	"math/rand"
	"sync"
	Atomic "sync/atomic"
	"syscall"
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
	txnCtr      int32
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
	m.txnCtr = txnCtr
}

type TransactionData struct {
	addrSender   string
	addrReceiver string
	balanceSender int
	balanceReceiver int
	amount       *big.Int
	tId          int32
}

func NewTxData(sender, reciever string, amount *big.Int, threadID int32) *TransactionData {
	return  &TransactionData{
		addrSender: sender,
		addrReceiver: reciever,
		amount: new(big.Int).Set(amount),
		tId: threadID,
	}
}

type Verifier struct {
	allSenders      map[string]int
	transactions    [200]TransactionData
	txnCtr          int64
	methodCount     int32
	finalOutcome    bool
	done            []atomic.Bool
	threadListsSize []atomic.Int32
	threadLists     ConcurrentSlice
	numTxns         int32
	isRunning       bool
	txnLock 		sync.Mutex
}

func NewVerifier() *Verifier {
	return  &Verifier{
		allSenders: make(map[string]int),
		txnCtr: 0,
		methodCount: 0,
		finalOutcome: true,
		done: make([]atomic.Bool, concurrent.NumThreads, concurrent.NumThreads),
		threadListsSize: make([]atomic.Int32, concurrent.NumThreads, concurrent.NumThreads),
		threadLists: ConcurrentSlice{items: make([]interface{}, 0, concurrent.NumThreads),},
		numTxns: 0,
		isRunning: true,
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
	index := Atomic.AddInt64(&v.txnCtr, 1) - 1
	v.transactions[index] = *txData
}

func (v *Verifier) Shutdown() {
	v.isRunning = false
}


type Block struct {
	start  int64
	finish int64
}

func (b *Block) setBlock() {
	b.start = 0
	b.finish = 0
}

// methodMapKey and itemMapKey are meant to serve in place of iterators
func (v *Verifier) handleFailedConsumer(methods []Method, items []Item, mk int, it int, stackFailed *stack.Stack) {
	fmt.Printf("Handling failed consumer...it is %d\n", it)
	fmt.Println(methods)
	begin := 0
	for it0 := begin; it0 != it + 1; it0++ {
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

func (v *Verifier) verifyCheckpoint(methods []Method, items []Item, itStart *int, countIterated *uint64, min int64, resetItStart bool, mapBlocks []Block) {
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
		//fmt.Printf("it = %d\n", it)

		for ; it != len(methods); it++ {
			fmt.Printf("it is %d and len items is %d len methods is %d\n", it, len(items), len(methods))
			//fmt.Printf("!The sum at %s is %f\n", items[it].key, items[it].sum)
			/*if methods[it].response > min{
				break
			}
			*/

			if v.methodCount%5000 == 0 {
				fmt.Printf("methodCount = %d\n", v.methodCount)
			}
			v.methodCount = v.methodCount + 1

			*itStart = it
			resetItStart = false
			*countIterated++


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
				fmt.Printf("CONSUMER\n")
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
					fmt.Printf("methods[%d].status == true (ABSENT)\n", it)
					// promote reads
					//if items.items[itItems].(*Item).sum > 0 {
					if items[itItems].sum > 0 {
						//items.items[itItems].(*Item).sumR = 0
						items[itItems].sumR = 0
					}

					//items[itItems].subInt(1)
					fmt.Printf("subInt at items[%d] which has key %s, resulting in sum of %f\n", itItems, items[itItems].key, items[itItems].sum)
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
			fmt.Printf("stackFailed length non zero:\n")
			for i := 0; i < stackFailed.Len(); i++ {
				fmt.Println(stackFailed.Pop())
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
					fmt.Printf("Demoting item...\n")
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
			fmt.Printf("Negative sum!\n")
		}
		for ; itVerify != itEnd; itVerify++ {
			if items[itVerify].sum < 0 {
				outcome = false
				// #if DEBUG_
				//fmt.Printf("WARNING: Item %d, sum %.2f\n", items.items[itVerify].(*Item).key, items.items[itVerify].(*Item).sum)
				fmt.Printf("WARNING1: Item %s (items[%d]), sum %.2f\n", items[itVerify].key, itVerify, items[itVerify].sum)
				// #endif
			}
			//printf("Item %d, sum %.2lf\n", it_verify->second.key, it_verify->second.sum);

			//if (math.Ceil(items.items[itVerify].(*Item).sum) + items.items[itVerify].(*Item).sumR) < 0 {
			if (math.Ceil(items[itVerify].sum) + items[itVerify].sumR) < 0 {
				outcome = false

				// #if DEBUG_
				//fmt.Printf("WARNING: Item %d, sum_r %.2f\n", items.items[itVerify].(*Item).key, items.items[itVerify].(*Item).sumR)
				fmt.Printf("WARNING2: Item %s, sum_r %.2f\n", items[itVerify].key, items[itVerify].sumR)
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
				fmt.Printf("!!!!!!!!!!\n")
				outcome = false
				// #if DEBUG_
				//fmt.Printf("WARNING: Item %d, sum_f %.2f\n", items.items[itVerify].(*Item).key, items.items[itVerify].(*Item).sumF)
				//fmt.Printf("WARNING: Item %d, sum_f %.2f\n", items[itVerify].key, items[itVerify].sumF)
				// #endif
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

func (v *Verifier) work(id int, doneWG *sync.WaitGroup) {
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


	//TODO: could this be Atomic.LoadInt64(&txnCtr) * 2
	mId := Atomic.LoadInt64(&v.txnCtr) * 2
	//Atomic.AddInt64(&txnCtr.val, 1)
	//
	//var end time.Time

	//wait()
	//if(Atomic.LoadInt32(&numTxns) == 0) {
		//return;
	//}

	if v.numTxns == 0 {
		v.done[id].Store(true)
		doneWG.Done()
		return
	} else {
		Atomic.AddInt32(&v.numTxns, -1)
	}

	for i := int32(0); i < testSize; i++ {

		/*if Atomic.LoadInt32(&numTxns) == 0 {
			break;
		}*/
		var res bool
		index := Atomic.LoadInt64(&v.txnCtr)
		itemAddr1 := v.transactions[index].addrSender
		itemAddr2 := v.transactions[index].addrReceiver
		amount := v.transactions[index].amount
		Atomic.AddInt64(&v.txnCtr, 1)
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

		// auto post_function_epoch = post_function.time_since_epoch();
		//postFunctionEpoch := time.Now().UnixNano() - postFunction

		//response := post_function_epoch.count() - start_time_epoch.count()
		//response := postFunctionEpoch - startTimeEpoch.Nanoseconds()


		if v.allSenders[itemAddr1] == 1 {
			v.allSenders[itemAddr1] = 0
			res = false
		} else {
			v.allSenders[itemAddr1] = 1
			res = true
		}

		fmt.Printf("res for %s is %v\n", itemAddr1, res)
		var m1 Method
		m1.setMethod(int(mId), itemAddr1, itemAddr2, v.transactions[id].balanceSender, FIFO, PRODUCER, res, int(mId), amount, v.transactions[id].tId)

		Atomic.AddInt64(&mId, 1)
		var m2 Method
		m2.setMethod(int(mId),itemAddr1, itemAddr2, v.transactions[id].balanceReceiver, FIFO, CONSUMER, res, int(mId), amount.Mul(amount, big.NewInt(int64(-1))), v.transactions[id].tId)
		Atomic.AddInt64(&mId, 1)

		//Atomic.AddInt32(&numTxns, -1)
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
		//threadLists.Lock()
		//TODO: we want to append both...right?
		v.threadLists.items[id] = append(v.threadLists.items[id].([]Method), m1)
		v.threadLists.items[id] = append(v.threadLists.items[id].([]Method), m2)
		//fmt.Printf("threadlist %d: %v\n", id, threadLists.items[id])
		v.threadListsSize[id].Add(1)
		//Atomic.AddInt64(&methodTime[id], 1)
		//threadLists.Unlock()
	}

	v.done[id].Store(true)
	doneWG.Done()
}

func (v *Verifier) verify(doneWG *sync.WaitGroup) {
	//defer processTimer(time.Now(), &txnCtr.val)
	fmt.Println("Verifying...")
	//wait()
	var countIterated uint64 = 0

	// fnPt       := fncomp
	methods := make([]Method, 0)
	//methods := NewConcurrentSlice()
	blocks := make([]Block, 0)
	//items := make([]Item, 0, numTxns * 2)
	fmt.Printf("txnCtr is %v\n", v.txnCtr)
	items := make([]Item, 0, v.txnCtr * 2)
	it := make([]int, concurrent.NumThreads, concurrent.NumThreads)
	var itStart int

	stop := false
	var countOverall uint32 = 0

	var min int64
	//var oldMin int64
	itCount := make([]int32, concurrent.NumThreads)

	for {
		if stop {
			break
		}

		stop = true
		min = math.MaxInt64

		for i := 0; i < concurrent.NumThreads; i++ {
			if v.done[i].Load() == false {

				stop = false
			}

			tId := i

			// TODO: Correctness not based on time any more, so do we still need response field?
			//var responseTime int64 = 0

			for {
				//threadLists.Lock()
				fmt.Printf("itCount[%d]: %d\tthreadListsSize[%d]: %d\n", i, itCount[i], i, v.threadListsSize[i].Load())
				if v.threadListsSize[i].Load() > 0 {

				}
				if itCount[i] >= v.threadListsSize[i].Load() {
					break
				} else if itCount[i] == 0 {
					it[i] = 0 //threadLists[i].Front()
				} else {
					//++it[i]
					it[i]++
				}
				//fmt.Printf("it[i] = %v\n", it[i])

				var m Method
				var m2 Method

				//if it[i] < len(threadLists.items[tId].([]Method)) {
				if it[i] < int(v.threadListsSize[tId].Load()) {
					fmt.Printf("Address of methods txn sender at thread %d index %d: %s and at index %d: %s\n", i, it[i], v.threadLists.items[tId].([]Method)[it[i]].itemAddrS, it[i] + 1, v.threadLists.items[tId].([]Method)[it[i] + 1].itemAddrS)
					fmt.Printf("%v\n", v.threadLists.items[i].([]Method))
					m = v.threadLists.items[tId].([]Method)[it[i]]
					it[i]++
					m2 = v.threadLists.items[tId].([]Method)[it[i]]
				} else {
					fmt.Printf("Verifier threadlist error!\n")
					break;
				}
				fmt.Printf("m address = %s\nm2 address = %s\n", m.itemAddrS, m2.itemAddrS)
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
				methods = append(methods, m2)

				itCount[i]++
				countOverall++

				//itItem := m.itemKey // it_item = map_items.find(m.item_key);
				//itItem := findIndexForMethod(methods, m, "itemAddr")
				// itItem, _ := findMethodKey(mapMethods, m.itemAddr)

				itItem := 0
				//for i := range items {
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
				fmt.Printf("%d\t%d\n", itItem, mapItemsEnd)
				if itItem == mapItemsEnd {
					var item Item
					var item2 Item
					fmt.Printf("appending addresses to items: %v\t%v\n", m.itemAddrS, m2.itemAddrS)
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

		v.verifyCheckpoint(methods, items, &itStart, &countIterated, int64(min), true, blocks)

	}

	v.verifyCheckpoint(methods, items, &itStart, &countIterated, math.MaxInt64, false, blocks)

			//#if DEBUG_
				fmt.Printf("Count overall = %v, count iterated = %d, methods size = %d, items size = %d\n", fmt.Sprint(countOverall), countIterated, len(methods), len(items));
			//#endif

		//#if DEBUG_
			fmt.Printf("All threads finished!\n")


	//a := verifyStart / int64(time.Millisecond)
	//b := verifyFinish / int64(time.Millisecond)
	//fmt.Printf("verify start is %d verify finish is %d\n", a, b)

	doneWG.Done()
}

func (v *Verifier) Verify() {
	go v.mainLoop()
}

func (v *Verifier) mainLoop() {
	methodTime := make([]int64, concurrent.NumThreads)
	overheadTime := make([]int64, concurrent.NumThreads)
	currentSize := int64(0)

	for v.isRunning || currentSize != Atomic.LoadInt64(&v.txnCtr) {
		currentSize = Atomic.LoadInt64(&v.txnCtr)
		if currentSize == 0 {
			time.Sleep(time.Duration(rand.Intn(concurrent.MAX_DELAY) + concurrent.MIN_DELAY) * time.Nanosecond)
			continue
		}
		// will use for i:= range threadLists.iter() in place of findMethodKey.
		// Should we make methods, items, and blocks ConcurrentSliceItems or slap RWlocks around where we use them?
		// Whats the deal with the separate items slice?
		//allSenders := make(map[string]int)
		Atomic.StoreInt32(&v.numTxns, 0)

		for i := 0; i < len(v.transactions); i++ {
			Atomic.AddInt32(&v.numTxns, 1)
			v.allSenders[v.transactions[i].addrSender] = 0
		}

		//threadLists := NewConcurrentSlice()

		var doneWG sync.WaitGroup
		var control string

		for i := 0; i < concurrent.NumThreads; i++ {
			v.threadLists.Append(make([]Method, 0))
			v.threadListsSize[i].Store(0)
			doneWG.Add(1)
			go v.work(i, &doneWG)
			doneWG.Wait()
		}
		//doneWG.Wait()
		doneWG.Add(1)
		go v.verify(&doneWG)
		doneWG.Wait()
		fmt.Println("finished working and verifying!")

		fmt.Printf("Control was: %s\n", control)

		if v.finalOutcome == true {
			fmt.Printf("-------------Program Correct Up To This Point-------------\n")
		} else {
			fmt.Printf("-------------Program Not Correct-------------\n")
		}

		finish := time.Now() //auto finish = std::chrono::high_resolution_clock::now();
		elapsedSeconds := time.Since(finish).Seconds()
		fmt.Println("Total Time: ", elapsedSeconds, " seconds")

		var elapsedTimeMethod int64 = 0
		var elapsedOverheadTime float64 = 0

		for i := 0; i < concurrent.NumThreads; i++ {
			if methodTime[i] > elapsedTimeMethod {
				elapsedTimeMethod = methodTime[i]
			}
			//we don't change overheadTime[i] anywhere, neither did Christina by the looks of it
			if float64(overheadTime[i]) > elapsedOverheadTime {
				elapsedOverheadTime = float64(overheadTime[i])
			}
		}

	}
}