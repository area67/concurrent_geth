package correctness_tool

import (
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/concurrent"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
	"testing"
)

type Data struct {
	sender string
	receiver string
	amount int
	tId int
}

var data [MAXTXNS]Data

func TestSimpleVerifierFunction(t *testing.T){

}

func TestVerifierFunction(t *testing.T) {
	// Generating 50 random transactions
	//var hexRunes = []rune("123456789abcdef")
	//var transactionSenders = make([]rune,16)
	//var transactionReceivers = make([]rune,16)
	var control string
	v := NewVerifier()
	//var transactions [50]TransactionData
	var numAccounts = 50
	var senderKeys = make([]*ecdsa.PrivateKey,numAccounts)
	var receiverKeys = make([]*ecdsa.PrivateKey,numAccounts)
	var transactionSenders = make([]common.Address,numAccounts)
	var transactionReceivers = make([]common.Address,numAccounts)

	for i := 0; i<numAccounts; i++{
		senderKeys[i],_ = crypto.GenerateKey()
		receiverKeys[i],_ = crypto.GenerateKey()
		transactionSenders[i] = crypto.PubkeyToAddress(senderKeys[i].PublicKey)
		transactionReceivers[i] = crypto.PubkeyToAddress(receiverKeys[i].PublicKey)
	}

	for i := 0; i < concurrent.NumThreads; i++ {

		if i == 0 {
			data[i].sender = transactionSenders[i].String()
			data[i].receiver = transactionReceivers[i].String()
			data[i].amount = 50
			control = data[i].sender
		} else if i == 1 {
			data[i].sender = control
			data[i].receiver = transactionReceivers[i].String()
			data[i].amount = 51
		} else {
			data[i].sender = transactionSenders[i].String()
			data[i].receiver = transactionReceivers[i].String()
			data[i].amount = 49
		}
		data[i].tId = i%4 //int(atomic.LoadInt32(&v.numTxns))
		v.LockFreeAddTxn(NewTxData(data[i].sender, data[i].receiver,  big.NewInt(int64(data[i].amount)), int32(data[i].tId)))
	}


	var result bool = v.Verify()

	// assert false
	if result{
		t.Errorf("Bad history passes verifier")
	}


	// simple case we expect to pass
	v = NewVerifier()

	for i := 0; i < concurrent.NumThreads; i++ {

		if i == 0 {
			data[i].sender = transactionSenders[i].String()
			data[i].receiver = transactionReceivers[i].String()
			data[i].amount = 50
			control = data[i].sender
		} else if i == 1 {
			data[i].sender = control
			data[i].receiver = transactionReceivers[i].String()
			data[i].amount = 51
		} else {
			data[i].sender = transactionSenders[i].String()
			data[i].receiver = transactionReceivers[i].String()
			data[i].amount = 52
		}
		data[i].tId = i%4 //int(atomic.LoadInt32(&v.numTxns))
		v.LockFreeAddTxn(NewTxData(data[i].sender, data[i].receiver,  big.NewInt(int64(data[i].amount)), int32(data[i].tId)))
	}

	result = v.Verify()

	if ! result{
		t.Errorf("Good history failed verifier")
	}


	// TODO: make test case that we expect to pass
	v = NewVerifier()

	for i := 0; i < concurrent.NumThreads; i++ {

		if i == 0 {
			data[i].sender = transactionSenders[i].String()
			data[i].receiver = transactionReceivers[i].String()
			data[i].amount = 52
			control = data[i].sender
		} else if i == 1 {
			data[i].sender = control
			data[i].receiver = transactionReceivers[i].String()
			data[i].amount = 51
		} else {
			data[i].sender = transactionSenders[i].String()
			data[i].receiver = transactionReceivers[i].String()
			data[i].amount = 50
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
