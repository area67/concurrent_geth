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
	var control string
	//var transactions [50]TransactionData

	for i := 0; i < 50; i++ {
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
			data[i].amount = rand.Intn(50)
		} else {
			data[i].sender = string(transactionSenders)
			data[i].receiver = string(transactionReceivers)
			data[i].amount = rand.Intn(50)
		}
	}
}
