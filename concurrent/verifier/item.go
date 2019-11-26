package correctness_tool

import (
	"container/list"
	"github.com/golang-collections/collections/stack"
	"math"
)

type Item struct {
	key           int // Account Hash ???
	value         int // Account Balance ???
	sum           float64
	numerator     int64
	denominator   int64
	exponent      float64
	status        Status
	promoteItems  stack.Stack
	demoteMethods list.List
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

func NewItem(key int) *Item {
	return &Item{
		key: key,
		value: math.MinInt32,
		sum: 0,
		numerator: 0,
		denominator: 1,
		exponent: 0,
		status: PRESENT,
		sumF: 0,
		numeratorF: 0,
		denominatorF: 1,
		exponentF: 0,
		sumR: 0,
		numeratorR: 0,
		denominatorR: 1,
		exponentR: 0,
		demoteMethods: list.List{},
	}
}

func (i *Item) SetItemKV(key int, value int) {
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

/*
modify sum
*/
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

	i.sumF = float64(i.numeratorF) / float64(i.denominatorF)
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
	i.sumF = float64(i.numeratorF) / float64(i.denominatorF)
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