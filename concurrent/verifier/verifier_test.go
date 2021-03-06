package correctness_tool

import (
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/concurrent"
	"time"

	//"github.com/ethereum/go-ethereum/concurrent"
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

//var transactionData [MAXTXNS]TxnTestData
var numAccounts = 50
var transactionSenders = make([]common.Address,numAccounts)
var transactionReceivers = make([]common.Address,numAccounts)
/*
called before each test case?
*/
func init(){
	var senderKeys = make([]*ecdsa.PrivateKey,numAccounts)
	var receiverKeys = make([]*ecdsa.PrivateKey,numAccounts)
	for i := 0; i<numAccounts; i++{
		senderKeys[i],_ = crypto.GenerateKey()
		receiverKeys[i],_ = crypto.GenerateKey()
		transactionSenders[i] = crypto.PubkeyToAddress(senderKeys[i].PublicKey)
		transactionReceivers[i] = crypto.PubkeyToAddress(receiverKeys[i].PublicKey)
	}
}

/*
The dumbest, smallest test case possible
*/
func TestSimpleVerifierFunction(t *testing.T){

	//var numAccounts = 2

	var txnDatum TxnTestData
	//var numTestThreads = 2
	var result bool
	v := NewVerifier()



	// single thread, single txn
	txnDatum.sender = transactionSenders[0].String()
	txnDatum.receiver =  transactionReceivers[0].String()
	txnDatum.tId = 0
	txnDatum.amount = 50
	v.LockFreeAddTxn(NewTxData(txnDatum.sender,txnDatum.receiver,big.NewInt(int64(txnDatum.amount)),txnDatum.tId))

	result = v.Verify()
	// expect true
	if !result{
		t.Errorf("Single transaction on sigle thread failed verifier")
	}
}

func TestBadHistory(t *testing.T) {
	var txnDatum TxnTestData
	//var numTestThreads = 2
	var result bool
	v := NewVerifier()

	// single thread, 2 txns
	txnDatum.sender = "alice"
	txnDatum.receiver = "lily"
	txnDatum.tId = 0
	txnDatum.amount = 50
	v.LockFreeAddTxn(NewTxData(txnDatum.sender,txnDatum.receiver,big.NewInt(int64(txnDatum.amount)),txnDatum.tId))


	txnDatum.sender = "alice"
	txnDatum.receiver =  "bob"
	txnDatum.tId = 0
	txnDatum.amount = 55
	// larger transaction comes after smaller on. should fail verifier
	v.LockFreeAddTxn(NewTxData(txnDatum.sender,txnDatum.receiver,big.NewInt(int64(txnDatum.amount)),txnDatum.tId))


	result = v.Verify()

	// expect to fail verifier
	if result{
		t.Errorf("Bad history passes verifier")
	}
}

func TestValidHistory(t *testing.T) {
	var txnDatum TxnTestData
	//var numTestThreads = 2
	var result bool
	v := NewVerifier()



	// single thread, 2 txns
	txnDatum.sender = transactionSenders[0].String()
	txnDatum.receiver =  transactionReceivers[0].String()
	txnDatum.tId = 0
	txnDatum.amount = 50
	v.LockFreeAddTxn(NewTxData(txnDatum.sender,txnDatum.receiver,big.NewInt(int64(txnDatum.amount)),txnDatum.tId))


	txnDatum.sender = transactionSenders[0].String()
	txnDatum.receiver =  transactionReceivers[1].String()
	txnDatum.tId = 0
	txnDatum.amount = 45
	// larger transaction comes after smaller on. should fail verifier
	v.LockFreeAddTxn(NewTxData(txnDatum.sender,txnDatum.receiver,big.NewInt(int64(txnDatum.amount)),txnDatum.tId))


	result = v.Verify()

	// expect to pass verifier
	if !result{
		t.Errorf("Good history fails verifier")
	}
}


func TestBackgroundSimpleVerifierFunction(t *testing.T){

	//var numAccounts = 2

	var txnDatum TxnTestData
	//var numTestThreads = 2
	var result bool
	v := NewVerifier()
	v.ConcurrentVerify()


	// single thread, single txn
	txnDatum.sender = "alice"
	txnDatum.receiver = "lily"
	txnDatum.tId = 0
	txnDatum.amount = 50
	v.LockFreeAddTxn(NewTxData(txnDatum.sender,txnDatum.receiver,big.NewInt(int64(txnDatum.amount)),txnDatum.tId))
	for i := 0; i < concurrent.NumThreads; i++ {
		v.ThreadFinished(i)
	}
	time.Sleep(1 * time.Second)

	result = v.WaitVerifier()
	// expect true
	if !result{
		t.Errorf("Single transaction on sigle thread failed verifier")
	}
}

func TestBackgroundBadHistory(t *testing.T) {
	var txnDatum TxnTestData
	//var numTestThreads = 2
	var result bool
	v := NewVerifier()
	v.ConcurrentVerify()


	// single thread, 2 txns
	txnDatum.sender = "alice"
	txnDatum.receiver = "lily"
	txnDatum.tId = 0
	txnDatum.amount = 50
	v.LockFreeAddTxn(NewTxData(txnDatum.sender,txnDatum.receiver,big.NewInt(int64(txnDatum.amount)),txnDatum.tId))


	txnDatum.sender = "alice"
	txnDatum.receiver =  "bob"
	txnDatum.tId = 0
	txnDatum.amount = 55
	// larger transaction comes after smaller on. should fail verifier
	v.LockFreeAddTxn(NewTxData(txnDatum.sender,txnDatum.receiver,big.NewInt(int64(txnDatum.amount)),txnDatum.tId))
	for i := 0; i < concurrent.NumThreads; i++ {
		v.ThreadFinished(i)
	}
	time.Sleep(1 * time.Second)

	result = v.WaitVerifier()

	// expect to fail verifier
	if result{
		t.Errorf("Bad history passes verifier")
	}
}

func TestBackgroundValidHistory(t *testing.T) {
	var txnDatum TxnTestData
	//var numTestThreads = 2
	var result bool
	v := NewVerifier()



	// single thread, 2 txns
	txnDatum.sender = "alice"
	txnDatum.receiver = "lily"
	txnDatum.tId = 0
	txnDatum.amount = 50
	v.LockFreeAddTxn(NewTxData(txnDatum.sender,txnDatum.receiver,big.NewInt(int64(txnDatum.amount)),txnDatum.tId))


	txnDatum.sender = transactionSenders[0].String()
	txnDatum.receiver =  transactionReceivers[1].String()
	txnDatum.tId = 0
	txnDatum.amount = 45
	// larger transaction comes before smaller on. should pass verifier
	v.LockFreeAddTxn(NewTxData(txnDatum.sender,txnDatum.receiver,big.NewInt(int64(txnDatum.amount)),txnDatum.tId))
	for i := 0; i < concurrent.NumThreads; i++ {
		v.ThreadFinished(i)
	}
	time.Sleep(1 * time.Second)

	result = v.WaitVerifier()

	// expect to pass verifier
	if !result{
		t.Errorf("Good history fails verifier")
	}
}

/*
simulates 2 threads working on transactions
*/
func TestTwoThreadHistory(t *testing.T){
	numTxns := 10
	var txnDatum TxnTestData
	//var numTestThreads = 2
	var result bool
	v := NewVerifier()

	//for i := 0; i < numTxns; i++{
		txnDatum.sender = "alice"
		txnDatum.receiver = "lily"
		txnDatum.tId = 0
		txnDatum.amount = 50
		v.LockFreeAddTxn(NewTxData(txnDatum.sender,txnDatum.receiver,big.NewInt(int64(txnDatum.amount)),txnDatum.tId))


		txnDatum.sender = "alice"
		txnDatum.receiver =  "lily"
		txnDatum.tId = 1
		txnDatum.amount = 45
		// larger transaction comes first smaller on. should pass verifier
		v.LockFreeAddTxn(NewTxData(txnDatum.sender,txnDatum.receiver,big.NewInt(int64(txnDatum.amount)),txnDatum.tId))
	//}

	result = v.Verify()

	if !result{
		t.Errorf("Good simple multi thread history fails verifier")
	}

	v = NewVerifier()
	// process unrelated transactions in separate threads
	for i := 0; i < numTxns; i++{
		txnDatum.sender = "alice"
		txnDatum.receiver = "lily"
		txnDatum.tId = 0
		txnDatum.amount = 50-i
		v.LockFreeAddTxn(NewTxData(txnDatum.sender,txnDatum.receiver,big.NewInt(int64(txnDatum.amount)),txnDatum.tId))


		txnDatum.sender = "bob"
		txnDatum.receiver =  "ross"
		txnDatum.tId = 1
		txnDatum.amount = 45-i
		// larger transaction comes first smaller on. should pass verifier
		v.LockFreeAddTxn(NewTxData(txnDatum.sender,txnDatum.receiver,big.NewInt(int64(txnDatum.amount)),txnDatum.tId))
	}

	result = v.Verify()

	if !result{
		t.Errorf("Good seperate multi thread history fails verifier")
	}

	v = NewVerifier()
	// process unrelated transactions in separate threads
	for i := 0; i < numTxns; i++{
		txnDatum.sender = "alice"
		txnDatum.receiver = "lily"
		txnDatum.tId = 0
		txnDatum.amount = 50+i
		v.LockFreeAddTxn(NewTxData(txnDatum.sender,txnDatum.receiver,big.NewInt(int64(txnDatum.amount)),txnDatum.tId))


		txnDatum.sender = "bob"
		txnDatum.receiver =  "ross"
		txnDatum.tId = 1
		txnDatum.amount = 45+i

		v.LockFreeAddTxn(NewTxData(txnDatum.sender,txnDatum.receiver,big.NewInt(int64(txnDatum.amount)),txnDatum.tId))
	}

	result = v.Verify()

	if result{
		t.Errorf("Bad seperate multi thread history passes verifier")
	}

	// interleaved history
	v = NewVerifier()
	// process unrelated transactions in alternating threads
	for i := 0; i < numTxns; i++{
		txnDatum.sender = "alice"
		txnDatum.receiver = "lily"
		txnDatum.tId = i%2
		txnDatum.amount = 50-i
		v.LockFreeAddTxn(NewTxData(txnDatum.sender,txnDatum.receiver,big.NewInt(int64(txnDatum.amount)),txnDatum.tId))

		txnDatum.sender = "bob"
		txnDatum.receiver =  "ross"
		txnDatum.tId = (i+1)%2
		txnDatum.amount = 45-i

		v.LockFreeAddTxn(NewTxData(txnDatum.sender,txnDatum.receiver,big.NewInt(int64(txnDatum.amount)),txnDatum.tId))
	}

	// expect to pass verifier. txnx added in descending order
	result = v.Verify()

	if !result{
		t.Errorf("Good interleved multi thread history fails verifier")
	}

	v = NewVerifier()
	// process unrelated transactions in alternating threads
	for i := 0; i < numTxns; i++{
		txnDatum.sender = "alice"
		txnDatum.receiver = "lily"
		txnDatum.tId = i%2
		txnDatum.amount = 50+i
		v.LockFreeAddTxn(NewTxData(txnDatum.sender,txnDatum.receiver,big.NewInt(int64(txnDatum.amount)),txnDatum.tId))

		txnDatum.sender = "bob"
		txnDatum.receiver =  "ross"
		txnDatum.tId = (i+1)%2
		txnDatum.amount = 45+i

		v.LockFreeAddTxn(NewTxData(txnDatum.sender,txnDatum.receiver,big.NewInt(int64(txnDatum.amount)),txnDatum.tId))
	}

	// expect to fail verifier. txnx added in acceding order
	result = v.Verify()

	if result{
		t.Errorf("Bad interleved multi thread history passes verifier")
	}
}