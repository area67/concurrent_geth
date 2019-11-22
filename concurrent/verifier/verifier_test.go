package correctness_tool

import (
	"math/rand"
	"testing"
)

type Data struct {
	sender string
	receiver string
	amount int
}

var data []Data

func TestVerifier(t *testing.T) {
	// Generating 50 random transactions
	var hexRunes = []rune("123456789abcdef")
	var transactionSenders = make([]rune,16)
	var transactionReceivers = make([]rune,16)
	//var transactions [50]TransactionData

	for i := 0; i < 50; i++ {
		for j := 0; j < 16; j++ {
			transactionSenders[j] = hexRunes[rand.Intn(len(hexRunes))]
			transactionReceivers[j] = hexRunes[rand.Intn(len(hexRunes))]
		}
		data[i].sender = string(transactionSenders)
		data[i].receiver = string(transactionReceivers)
		data[i].amount = rand.Intn(50)
	}
}
