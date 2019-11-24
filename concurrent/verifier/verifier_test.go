package correctness_tool

import (
	"math/big"
	"math/rand"
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
			data[i].amount = 50
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
		data[i].tId = i%4 //int(atomic.LoadInt32(&v.numTxns))
		v.LockFreeAddTxn(NewTxData(data[i].sender, data[i].receiver,  big.NewInt(int64(data[i].amount)), int32(data[i].tId)))
	}

	var result bool = v.Verify()

	// assert false
	if result{
		t.Errorf("Bad history passes verifier")
	}

	// TODO: make test case that we expect to pass
	v = NewVerifier()

	for i := 0; i < 32; i++ {
		for j := 0; j < 16; j++ {
			transactionSenders[j] = hexRunes[rand.Intn(len(hexRunes))]
			transactionReceivers[j] = hexRunes[rand.Intn(len(hexRunes))]
		}

		if i == 0 {
			data[i].sender = string(transactionSenders)
			data[i].receiver = string(transactionReceivers)
			data[i].amount = 52
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
		data[i].tId = i%4 //int(atomic.LoadInt32(&v.numTxns))
		v.LockFreeAddTxn(NewTxData(data[i].sender, data[i].receiver,  big.NewInt(int64(data[i].amount)), int32(data[i].tId)))
	}
	// TODO: transactions we expect to pass
	result = v.Verify()

	if ! result{
		t.Errorf("Good history failed verifier")
	}



}
