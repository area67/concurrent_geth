package correctness_tool

import (
	"fmt"
	"math/big"
	"math/rand"
	"sync/atomic"
	"testing"
)

type Data struct {
	sender string
	receiver string
	amount int
	tId int
}

var data [MAXTXNS]Data

func TestVerifierFunction(t *testing.T) {
	// Generating 50 random transactions
	var hexRunes = []rune("123456789abcdef")
	var transactionSenders = make([]rune,16)
	var transactionReceivers = make([]rune,16)
	var control string
	v := NewVerifier()
	//var transactions [50]TransactionData

	for i := 0; i < 32; i++ {
		for j := 0; j < 16; j++ {
			transactionSenders[j] = hexRunes[rand.Intn(len(hexRunes))]
			transactionReceivers[j] = hexRunes[rand.Intn(len(hexRunes))]
		}

		if i == 0 {
			data[i].sender = string(transactionSenders)
			data[i].receiver = string(transactionReceivers)
			data[i].amount = 1
			control = data[i].sender
		} else if i == 1 {
			data[i].sender = control
			data[i].receiver = string(transactionReceivers)
			data[i].amount = 51
		} else {
			data[i].sender = string(transactionSenders)
			data[i].receiver = string(transactionReceivers)
			data[i].amount = rand.Intn(50)
		}
		data[i].tId = int(atomic.LoadInt32(&v.numTxns))
		fmt.Printf("%v\n", data[i])
		newData := NewTxData(data[i].sender, data[i].receiver,  big.NewInt(50), int32(data[i].tId))
		v.LockFreeAddTxn(newData)
	}

	v.Verify()
	res := true
	for {
		for i := 0; i < len(v.done); i++ {
			if v.done[i].Load() != true {
				res = false
			}
		}
		if res == true {
			break
		}
	}
	v.Shutdown()
}
