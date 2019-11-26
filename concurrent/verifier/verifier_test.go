package correctness_tool

import (
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/concurrent"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
	"testing"
)

type TxnTestData struct {
	sender string
	receiver string
	amount int
	tId int
}

var transactionData [MAXTXNS]TxnTestData

/*
called before each test case?
*/
func init(){

}

/*
The dumbest, smallest test case possible
*/
func TestSimpleVerifierFunction(t *testing.T){

	var numAccounts = 2
	var senderKeys = make([]*ecdsa.PrivateKey,numAccounts)
	var receiverKeys = make([]*ecdsa.PrivateKey,numAccounts)
	var transactionSenders = make([]common.Address,numAccounts)
	var transactionReceivers = make([]common.Address,numAccounts)
	var txnDatum TxnTestData
	//var numTestThreads = 2
	var result bool
	v := NewVerifier()

	for i := 0; i<numAccounts; i++{
		senderKeys[i],_ = crypto.GenerateKey()
		receiverKeys[i],_ = crypto.GenerateKey()
		transactionSenders[i] = crypto.PubkeyToAddress(senderKeys[i].PublicKey)
		transactionReceivers[i] = crypto.PubkeyToAddress(receiverKeys[i].PublicKey)
	}

	// single thread, single txn
	txnDatum.sender = transactionSenders[0].String()
	txnDatum.receiver =  transactionReceivers[0].String()
	txnDatum.tId = 0
	txnDatum.amount = 50
	v.LockFreeAddTxn(NewTxData(txnDatum.sender,txnDatum.receiver,big.NewInt(int64(txnDatum.amount)),int32(txnDatum.tId)))

	result = v.Verify()
	// expect true
	if !result{
		t.Errorf("Single transaction on sigle thread failed verifier")
	}




}

func TestBadHistory(t *testing.T) {
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


	var result bool = v.Verify()

	// assert false
	if result{
		t.Errorf("Bad history passes verifier")
	}
}

func TestValidHistory(t *testing.T) {
	// Generating 50 random transactions
	//var hexRunes = []rune("123456789abcdef")
	//var transactionSenders = make([]rune,16)
	//var transactionReceivers = make([]rune,16)
	var result bool
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

	// simple case we expect to pass
	v = NewVerifier()



	result = v.Verify()

	if ! result{
		t.Errorf("Good history failed verifier")
	}


	// TODO: make test case that we expect to pass
	v = NewVerifier()


	// TODO: transactions we expect to pass
	result = v.Verify()

	if ! result{
		t.Errorf("Good history failed verifier")
	}
}
