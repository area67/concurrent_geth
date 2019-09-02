package main

import (
	"C"
	"container/list"
	"github.com/golang-collections/collections/stack"
	"go/types"
	"math"
	"os"
	"reflect"
	"sort"
	"strconv"
	"sync/atomic"
	"time"
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

	demoteItems list.List

	producer map[int64]reflect.MapIter // TODO: not sure if this is equivalent


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

var threadLists     = make([]Method, 0, numThreads)// empty slice with capacity numThreads
var threadListsSize [numThreads]int32              // atomic ops only
var done            [numThreads]bool               // atomic ops only
var barrier int32 = 0

func wait(){
	atomic.AddInt32(&barrier, 1)
	for atomic.LoadInt32(&barrier) < numThreads {}
}

var methodTime   [numThreads]int64
var overheadTime [numThreads]int64

var start time.Time

var elapsedTimeVerify int64


func findFirstMethodKey(m map[int64]Method) int64{
	keys := make([]int, 0)
	for k := range m {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	return int64(keys[0])
}

func findFirstItemKey(m map[int64]Item) int64{
	keys := make([]int, 0)
	for k := range m {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	return int64(keys[0])
}

// methodMapKey and itemMapKey are meant to serve in place of iterators
func handleFailedConsumer(methodMap map[int64]Method, itemMap map[int64]Item, methodMapKey int64, itemMapKey int64, stackFailed stack.Stack){

	for methItr0 := findFirstMethodKey(methodMap); methItr0 != methodMapKey; methItr0++{

		// linearizability
		// if methodMap[methItr0].response < methodMap[methodMapKey].invocation

		// sequential consistency
		// if methodMap[methItr0].response < methodMap[methodMapKey].invocation &&
		//     methodMap[methItr0].process == methodMap[methodMapKey].process

		// serializability
		if methodMap[methItr0].response < methodMap[methodMapKey].invocation &&
			methodMap[methItr0].txnID == methodMap[methodMapKey].txnID ||
			methodMap[methItr0].txnID < methodMap[methodMapKey].txnID{

			itemItr0 := methodMap[methItr0].itemKey

			if methodMap[methItr0].types == PRODUCER &&
				itemMap[itemMapKey].status == PRESENT &&
				methodMap[methItr0].semantics == FIFO ||
				methodMap[methItr0].semantics == LIFO ||
				methodMap[methodMapKey].itemKey == methodMap[methItr0].itemKey{

				stackFailed.Push(itemItr0)
			}
		}
	}


}





func work_queue(id int) {
	testSize := TEST_SIZE
	wallTime := 0.0
	var tod timeval
	gettimeofday(&tod, 0)
	wallTime += tod.tv_sec
	wallTime += tod.tv_usec * 1e-6

	// How to??? lines 824 - 828
	/*
	 *boost::mt19937 randomGenOp
     *randomGenOp.seed(wallTime + id + 1000)
     *boost::uniform_int<unsigned int> randomDistOp(1, 100)
	*/

	// Is this right? lines 831 and 832
    //auto start_time = std::chrono::time_point_cast<std::chrono::nanoseconds>(start);
	//auto start_time_epoch = start_time.time_since_epoch();
	var start_time int64 = time.Now().UnixNano()
	var start_time_epoch int64 = start_time - time.Now().UnixNano()

	m_id := id + 1

	// line 839
	//std::chrono::time_point<std::chrono::high_resolution_clock> end;
	var end int64

	wait()

	for var i uint32 = 0; i < testSize; i++
	{
	 	item_key := -1
	 	res := true

	 	// Keep this? rendomGenOP is from boost library which we supposedly don't need
	 	var op_dist uint32 = randomDistOp(randomGenOp)

	 	// end = std::chrono::high_resolution_clock::now();
	 	end = time.Now().UnixNano()

	 	// How to??? lines 852 and 853
	 	/*
	 	 *auto pre_function = std::chrono::time_point_cast<std::chrono::nanoseconds>(end);
		 *auto pre_function_epoch = pre_function.time_since_epoch();
		*/


		// Hmm, do we need .count()??
		//invocation := pre_function_epoch.count() - start_time_epoch.count()
		invocation := pre_function_epoch - start_time_epoch

		if invocation > (LONG_MAX - 10000000000)
		{
			//PREPROCESSOR DIRECTIVE lines 864 - 866:
			/*
			 * #if DEBUG_
			 *		printf("WARNING: TIME LIMIT REACHED! TERMINATING PROGRAM\n");
			 * #endif
			 */
			break
		}

		if op_dist <= 50
		{
			type := CONSUMER
			var item_pop int 
			var item_pop_ptr uint32*

			res := queue.try_pop(item_pop)
			if res
			{
				item_key := item_pop
			}
			else
			{
				item_key := INT_MIN
			}
		}
		else
		{
			type := PRODUCER
			item_key := m_id
			queue.push(item_key)
		}

		// line 890
		// end = std::chrono::high_resolution_clock::now();
		end = time.Now().UnixNano()

		// auto post_function = std::chrono::time_point_cast<std::chrono::nanoseconds>(end);
		post_function = end

		// Is this right??
		// auto post_function_epoch = post_function.time_since_epoch();
		post_function_epoch := time.Now().UnixNano() - post_function

		//response := post_function_epoch.count() - start_time_epoch.count()
		response := post_function_epoch - start_time_epoch

		// How to??? line 915
		// Method m1(m_id, id, item_key, INT_MIN, FIFO, type, invocation, response, res, m_id);

		m_id += NUM_THRDS
		thrd_lists[id].push_back(m1)
		thrd_lists_size[id].fetch_add(1)
		method_time[id] = method_time[id] + (response - invocation)
	}

	done[id].store(true)
}

func work_stack(id int) {
	testSize := TEST_SIZE
	wallTime := 0.0
	var tod timeval
	gettimeofday(&tod, 0)
	wallTime += tod.tv_sec
	wallTime += tod.tv_usec * 1e-6

	// How to??? lines 945 - 953
	/*
	 *boost::mt19937 randomGenOp
     *randomGenOp.seed(wallTime + id + 1000)
     *boost::uniform_int<unsigned int> randomDistOp(1, 100)

	 *auto start_time = std::chrono::time_point_cast<std::chrono::nanoseconds>(start);
	 *auto start_time_epoch = start_time.time_since_epoch();
	*/

	m_id := id + 1

	// How to??? line 957
	//std::chrono::time_point<std::chrono::high_resolution_clock> end;

	wait()

	for var i uint32 = 0; i < testSize; i++ {
	 	item_key := -1
	 	res := true
	 	var op_dist uint32 = randomDistOp(randomGenOp)

	 	// How to??? line 970 - 973
	 	/*
	 	 *end = std::chrono::high_resolution_clock::now();
	 	 *auto pre_function = std::chrono::time_point_cast<std::chrono::nanoseconds>(end);
		 *auto pre_function_epoch = pre_function.time_since_epoch();
		*/

		invocation := pre_function_epoch.count() - start_time_epoch.count()

		if invocation > (LONG_MAX - 10000000000)
		{
			// How to??? PREPROCESSOR DIRECTIVE lines 984 - 986:
			/*
			 * #if DEBUG_
			 *		printf("WARNING: TIME LIMIT REACHED! TERMINATING PROGRAM\n");
			 * #endif
			 */
			break
		}

		if op_dist <= 50
		{
			type := CONSUMER
			var item_pop int
			res = stack.pop(item_pop)

			if res
			{
				item_key := item_pop;
			}
			else
			{
				item_key := INT_MIN;
			}
		}
		else
		{
			type := PRODUCER
			item_key := m_id
			stack.push(item_key)
		}

		// How to??? lines 1006 - 1009
		// end = std::chrono::high_resolution_clock::now();
		// auto post_function = std::chrono::time_point_cast<std::chrono::nanoseconds>(end);
		// auto post_function_epoch = post_function.time_since_epoch();

		response := post_function_epoch.count() - start_time_epoch.count()

		// How to??? line 1034
		// Method m1(m_id, id, item_key, INT_MIN, LIFO, type, invocation, response, res, m_id);

		m_id = m_id + NUM_THRDS

		thrd_lists[id].push_back(m1)
		
		thrd_lists_size[id].fetch_add(1)

		method_time[id] = method_time[id] + (response - invocation)

	}

	done[id].store(true)
}

func work_map(id int) {
	testSize := TEST_SIZE
	wallTime := 0.0
	var tod timeval
	gettimeofday(&tod, 0)
	wallTime += tod.tv_sec
	wallTime += tod.tv_usec * 1e-6

	// How to??? lines 1064 - 1071
	/*
	 *boost::mt19937 randomGenOp
     *randomGenOp.seed(wallTime + id + 1000)
     *boost::uniform_int<unsigned int> randomDistOp(1, 100)

	 *auto start_time = std::chrono::time_point_cast<std::chrono::nanoseconds>(start);
	 *auto start_time_epoch = start_time.time_since_epoch();
	*/

	m_id := id + 1

	// How to??? line 1075
	//std::chrono::time_point<std::chrono::high_resolution_clock> end;

	wait()

	for var i uint32 = 0; i < testSize; i++ {
	 	item_key := -1
	 	item_val := -1

	 	res := true
	 	var op_dist uint32 = randomDistOp(randomGenOp)

	 	// How to??? line 1090 - 1093
	 	/*
	 	 *end = std::chrono::high_resolution_clock::now();
	 	 *auto pre_function = std::chrono::time_point_cast<std::chrono::nanoseconds>(end);
		 *auto pre_function_epoch = pre_function.time_since_epoch();
		*/

		invocation := pre_function_epoch.count() - start_time_epoch.count()

		if invocation > (LONG_MAX - 10000000000)
		{
			// How to??? PREPROCESSOR DIRECTIVE lines 1104 - 1106:
			/*
			 * #if DEBUG_
			 *		printf("WARNING: TIME LIMIT REACHED! TERMINATING PROGRAM\n");
			 * #endif
			 */
			break
		}

		// How to??? line 1111
		// tbb::concurrent_hash_map<int,int,MyHashCompare>::accessor a

		if op_dist <= 33
		{
			type := CONSUMER
			item_erase := m_id - 2*NUM_THRDS
			res := map.erase(item_erase)

			if res
			{
				item_key := item_erase
			}
			else
			{
				item_key := INT_MIN
			}
		}
		else if op dist <= 66
		{
			type := CONSUMER
			item_key := m_id
			item_val := m_id
			map.insert(a, item_key)
			// How to??? line 1130
			// a->second = item_val;
		}
		else
		{
			type := READER
			item_key := m_id - NUM_THRDS
			res := map.find(a, item_key)

			if res
			{
				// How to??? line 1138
				// item_val = a->second
			}
			else
			{
				item_key := INT_MIN
				item_val := INT_MIN
			}
		}

		// How to??? lines 1145 - 1148
		//end = std::chrono::high_resolution_clock::now();
		//auto post_function = std::chrono::time_point_cast<std::chrono::nanoseconds>(end);
		//auto post_function_epoch = post_function.time_since_epoch();

		response := post_function_epoch.count() - start_time_epoch.count()

		// How to??? line 1169
		//Method m1(m_id, id, item_key, item_val, MAP, type, invocation, response, res, m_id);

		m_id := m_id + NUM_THRDS
		
		thrd_lists[id].push_back(m1)
		
		thrd_lists_size[id].fetch_add(1)

		method_time[id] = method_time[id] + (response - invocation)
	}

	done[id].store(true)
}

func verify() {
	wait()

	// How to??? lines 1188 - 1196
	/*
	auto start_time = std::chrono::time_point_cast<std::chrono::nanoseconds>(start);
	auto start_time_epoch = start_time.time_since_epoch();

	std::chrono::time_point<std::chrono::high_resolution_clock> end;

	end = std::chrono::high_resolution_clock::now();

	auto pre_verify = std::chrono::time_point_cast<std::chrono::nanoseconds>(end);
	auto pre_verify_epoch = pre_verify.time_since_epoch();
	*/

	verify_start := pre_verify_epoch.count() - start_time_epoch.count()

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
	var count_overall uint32 := 0
	var count_iterated uint32 := 0

	var min int32
	var old_min int32
	var it_count int[NUM_THRDS]

	// How to??? line 1225
	// std::map<long int,Method,bool(*)(long int,long int)>::iterator it_qstart;

	for {
		if stop {
			break
		}

		stop = true
		min = LONG_MAX

		for i := 0; i < NUM_THRDS; i++ {
			if done[i].load() == false {
				stop = false
			}

			var response_time int32 = 0

			for {
				if it_count[i] >= thrd_lists_size[i].load() {
					break
				}
				else if it_count[i] == 0 {
					it[i] = thrd_lists[i].begin()
				}
				else {
					++it[i]
				}

				var m Method = *it[i]

				// How to??? line 1258
				// std::map<long int,Method,bool(*)(long int,long int)>::iterator it_method;

				it_method = map_methods.find(m.response)

				for {
					if it_method == map_methods.end() {
						break
					}
					m.response++
					it_method = map_methods.find(m.response)
				}
				response_time = m.response
				// How to???
				// map_methods.insert ( std::pair<long int,Method>(m.response,m) );

				it_count[i]++
				count_overall++

				// How to??? line 1275
				// std::unordered_map<int,Item>::iterator it_item;

				it_item = map_items.find(m.item_key)

				if it_item == map_items.end()
				{
					var item(m.item_key) Item
					item.producer = map_methods.end()

					// How to??? line 1288
					// map_items.insert(std::pair<int,Item>(m.item_key,item) );

					it_item = map_items.find(m.item_key)
				}
			}

			if response_time < min
			{
				min = response_time
			}
		}

		verify_checkpoint(map_methods, map_items, it_start, count_iterated, min, true, map_block)

	}

	verify_checkpoint(map_methods, map_items, it_start, count_iterated, LONG_MAX, false, map_block)

	// How to??? lines 1326 - 1331
	/*
	#if DEBUG_
		printf("Count overall = %lu, count iterated = %lu, map_methods.size() = %lu\n", count_overall, count_iterated, map_methods.size());
	#endif

	#if DEBUG_
		printf("All threads finished!\n");
	*/

	// How to??? line 1339
	// std::map<long int,Block,bool(*)(long int,long int)>::iterator it_b;

	for it_b = map_block.begin(); it_b != map_block.end(); ++it_b {
		fmt.printf("Block start = %ld, finish = %ld\n", it_b->second.start, it_b->second.finish)
	}

	// How to??? line 1346
	// std::map<long int,Method,bool(*)(long int,long int)>::iterator it_;

	for it_ = map_methods.begin(); it != map_methods.end(); ++it_ {
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
	}

	// How to??? line 1358
	// #endif

	// How to??? lines 1360 - 1362
	// end = std::chrono::high_resolution_clock::now();
	// auto post_verify = std::chrono::time_point_cast<std::chrono::nanoseconds>(end);

	post_verify_epoch := post_verify.time_since_epoch()
	verify_finish := post_verify_epoch.count() - start_time_epoch.count()

	elapsed_time_verify = verify_finish - verify_start
}

func main() {
	method_count := 0
	TBB_QUEUE := 0
	BOOST_STACK := 0
	TBB_MAP := 0

	if len(os.Args) == 2
	{
		fmt.printf("Test size = %d\n", strconv.Atoi(os.Args[1]))
		TEST_SIZE = strconv.Atoi(os.Args[1])
		TBB_QUEUE = 1
		fmt.printf("Testing TBB_QUEUE")
	}
	else if len(os.Args) == 3
	{
		fmt.printf("Test size = %d\n", strconv.Atoi(os.Args[1]))
		TEST_SIZE = strconv.Atoi(os.Args[1])

		if strconv.Atoi(os.Args[2]) == 0
		{
			TBB_QUEUE = 1
			fmt.printf("Testing TBB_QUEUE\n")
		}
		else if strconv.Atoi(os.Args[2]) == 1
		{
			BOOST_STACK = 1
			fmt.printf("Testing BOOST_STACK\n")
		}
		else if strconv.Atoi(os.Args[2]) == 2
		{
			TBB_MAP = 1
			fmt.printf("Testing TBB_MAP\n")
		}
	}
	else
	{
		fmt.printf("Test size = 10\n")
		TEST_SIZE = 10
		TBB_QUEUE = 1
		fmt.printf("Testing TBB_QUEUE\n")
	}

	final_outcome := true

	// How to???
	/*
	std::thread t[NUM_THRDS];
	
	std::thread v;

	start = std::chrono::high_resolution_clock::now();
	*/

	for i := 0; i < NUM_THRDS; i++
	{
		if TBB_QUEUE
		{
			//t[i] = std::thread(work_queue,i);
		}
		else if BOOST_STACK
		{
			//t[i] = std::thread(work_stack,i);
		}
		else if TBB_MAP
		{
			//t[i] = std::thread(work_map,i);
		}
	}

	//v = std::thread(verify);
	/*
	for(int i = 0; i < NUM_THRDS; i++)
	{
		t[i].join();
	}
	//TODO: Uncomment line 1426
	v.join();
	*/

	if final_outcome == true
	{
		fmt.printf("-------------Program Correct Up To This Point-------------\n")
	}
	else
	{
		fmt.printf("-------------Program Not Correct-------------\n")
	}

	//auto finish = std::chrono::high_resolution_clock::now();
	//auto elapsed_time = std::chrono::duration_cast<std::chrono::nanoseconds>(finish - start);

	var elapsed_time_double float64 = elapsed_time.count() * 0.000000001
	fmt.printf("Total Time: %.15lf seconds\n", elapsed_time_double);

	var elapsed_time_method int32 = 0
	var elapsed_overhead_time_double int32 = 0

	for i = 0; i < NUM_THRDS; i++
	{
		if method_time[i] > elapsed_time_method
		{
			elapsed_time_method = method_time[i]
		}
		if overhead_time[i] > elapsed_overhead_time
		{
			elapsed_overhead_time = overhead_time[i]
		}
	}

	var elapsed_time_method_double float64 = elapsed_time_method * 0.000000001
	var elapsed_overhead_time_double float64 = elapsed_overhead_time * 0.000000001
	var elapsed_time_verify_double float64 = elapsed_time_verify * 0.000000001

	fmt.printf("Total Method Time: %.15lf seconds\n", elapsed_time_method_double)
	fmt.printf("Total Overhead Time: %.15lf seconds\n", elapsed_overhead_time_double)

	elapsed_time_verify_double = elapsed_time_verify_double - elapsed_time_method_double

	fmt.printf("Total Verification Time: %.15lf seconds\n", elapsed_time_verify_double)

	if TBB_QUEUE
	{
		fmt.printf("Final Queue Configuration: \n");
		// How to???
		//typedef tbb::concurrent_queue<int>::iterator iter;
		//for(iter i(queue.unsafe_begin()); i!=queue.unsafe_end(); i++)
			//printf("%d ", *i);
		//printf("\n");
	}
	else if BOOST_STACK
	{
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
	}
	else if TBB_MAP
	{
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
