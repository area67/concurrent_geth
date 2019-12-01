package correctness_tool

import "math/big"

type Status int
type Semantics int
type Types int
type State int32

const (
	PRESENT Status = iota
	ABSENT
	LESS = -1
	EQUAL = 0
	MORE = 1
)

const (
	FIFO Semantics = iota
	LIFO
	SET
	MAPP
	PRIORITY
)

const (
	PRODUCER Types = iota
	CONSUMER
	READER
	WRITER
)
const (
	WORKING = iota
	DONE
)


type Method struct {
	id          int       // atomic var
	itemAddrS    string       // sender account address
	itemAddrR    string   // receiver account address
	semantics   Semantics // hardcode as FIFO per last email
	types       Types     // producing/consuming  adding/subtracting
	status      bool
	requestAmnt *big.Int
}

// method constructor
func  NewMethod(id int, itemAddrS string, itemAddrR string, semantics Semantics, types Types, status bool, requestAmnt *big.Int) *Method {
	return &Method{
		id: id,
		itemAddrS: itemAddrS,
		itemAddrR: itemAddrR,
		semantics: semantics,
		types: types,
		status: status,
		requestAmnt: requestAmnt,
	}
}

type TransactionData struct {
	addrSender   string
	addrReceiver string
	balanceSender int
	balanceReceiver int
	amount       *big.Int
	tId          int
}


// constructor
func NewTxData(sender, receiver string, amount *big.Int, threadID int) *TransactionData {
	return  &TransactionData{
		addrSender:   sender,
		addrReceiver: receiver,
		amount:       new(big.Int).Set(amount),
		tId:          threadID,
	}
}