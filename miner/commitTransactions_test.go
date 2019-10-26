package miner

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/clique"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
	"math/big"
	"testing"
	"time"
)

var(
	txsMap = make(map[common.Address]types.Transactions)
	numAccounts = 15
	txsPerAccount = 10
	keys = make([]*ecdsa.PrivateKey,numAccounts)
	addresses = make([]common.Address,numAccounts)
	initBalance = big.NewInt(100000000000) // inital balance for all accounts
	)

func init() {
	testTxPoolConfig = core.DefaultTxPoolConfig
	testTxPoolConfig.Journal = ""
	ethashChainConfig = params.TestChainConfig
	cliqueChainConfig = params.TestChainConfig
	cliqueChainConfig.Clique = &params.CliqueConfig{
		Period: 10,
		Epoch:  30000,
	}


	// generate account keys and init balances
	for i := 0; i<numAccounts; i++{
		keys[i],_ = crypto.GenerateKey()
		addresses[i] = crypto.PubkeyToAddress(keys[i].PublicKey)

		fmt.Println("")
	}

	//nonce := uint64(0)
	for a := range addresses{
		txs := types.Transactions{}
		for i:=0; i < txsPerAccount; i++{
			recipientIndex := (a+i+1)% len(addresses)
			t , _ := types.SignTx(types.NewTransaction(uint64(i), addresses[recipientIndex], big.NewInt(1), params.TxGas, nil, nil), types.HomesteadSigner{}, keys[a])
			//nonce++
			txs = append(txs, t )
		}
		txsMap[addresses[a]] = txs
	}


	tx1, _ := types.SignTx(types.NewTransaction(0, testUserAddress, big.NewInt(1000), params.TxGas, nil, nil), types.HomesteadSigner{}, testBankKey)
	pendingTxs = append(pendingTxs, tx1)
	tx2, _ := types.SignTx(types.NewTransaction(1, testUserAddress, big.NewInt(1000), params.TxGas, nil, nil), types.HomesteadSigner{}, testBankKey)
	newTxs = append(newTxs, tx2)

	// generate account keys and transactions


	// need map of transactions for txn by price an nonce
	/*
	for each transaction{
		txsMap[account] = append(txsMap[account],txs)
	}
	*/
}


func TestEmptyWorkEthashConcurrent(t *testing.T) {
	testEmptyWorkConcurrent(t, ethashChainConfig, ethash.NewFaker())
}
func TestEmptyWorkCliqueConcurrent(t *testing.T) {
	testEmptyWorkConcurrent(t, cliqueChainConfig, clique.New(cliqueChainConfig.Clique, ethdb.NewMemDatabase()))
}

func testEmptyWorkConcurrent(t *testing.T, chainConfig *params.ChainConfig, engine consensus.Engine) {
	defer engine.Close()

	w, _ := newTestWorker(t, chainConfig, engine, 0)
	defer w.close()

	var (
		taskCh    = make(chan struct{}, 2)
		taskIndex int
	)

	checkEqual := func(t *testing.T, task *task, index int) {
		receiptLen, balance := 0, big.NewInt(0)
		if index == 1 {
			receiptLen, balance = 1, big.NewInt(1000)
		}
		if len(task.receipts) != receiptLen {
			t.Errorf("receipt number mismatch: have %d, want %d", len(task.receipts), receiptLen)
		}
		if task.state.GetBalance(testUserAddress).Cmp(balance) != 0 {
			t.Errorf("account balance mismatch: have %d, want %d", task.state.GetBalance(testUserAddress), balance)
		}
	}

	w.newTaskHook = func(task *task) {
		if task.block.NumberU64() == 1 {
			checkEqual(t, task, taskIndex)
			taskIndex += 1
			taskCh <- struct{}{}
		}
	}
	w.fullTaskHook = func() {
		time.Sleep(100 * time.Millisecond)
	}

	// Ensure worker has finished initialization
	for {
		b := w.pendingBlock()
		if b != nil && b.NumberU64() == 1 {
			break
		}
	}

	w.start()
	for i := 0; i < 2; i += 1 {
		select {
		case <-taskCh:
		case <-time.NewTimer(4 * time.Second).C:
			t.Error("new task timeout")
		}
	}
}

func TestCommitTransactionsPerformance(t *testing.T){
	testCommitTransactionsPerformance(t, cliqueChainConfig, clique.New(cliqueChainConfig.Clique, ethdb.NewMemDatabase()))
}

func testCommitTransactionsPerformance(t *testing.T, chainConfig *params.ChainConfig, engine consensus.Engine){
	// need worker
	w, _ := newTestWorker(t, chainConfig, engine, 0)
	defer w.close()

	var (
		taskCh    = make(chan struct{}, 2)
		taskIndex int
		interrupt int32 = 0
	)

	checkEqual := func(t *testing.T, task *task, index int) {
		receiptLen := 0
		if index == 1 {
			receiptLen = numAccounts*txsPerAccount//, big.NewInt(1000)
		}
		if len(task.receipts) != receiptLen {
			t.Errorf("receipt number mismatch: have %d, want %d", len(task.receipts), receiptLen)
		}
		/*
		if task.state.GetBalance(testUserAddress).Cmp(balance) != 0 {
			t.Errorf("account balance mismatch: have %d, want %d", task.state.GetBalance(testUserAddress), balance)
		}*/
	}

	w.newTaskHook = func(task *task) {
		if task.block.NumberU64() == 1 {
			checkEqual(t, task, taskIndex)
			taskIndex += 1
			taskCh <- struct{}{}
		}
	}
	w.fullTaskHook = func() {
		time.Sleep(100 * time.Millisecond)
	}

	// Ensure worker has finished initialization
	for {
		b := w.pendingBlock()
		if b != nil && b.NumberU64() == 1 {
			break
		}
	}

	w.start()

	//tem := w.current
	//print(tem)
	testTxs:=types.NewTransactionsByPriceAndNonce(w.current.signer, txsMap)

	// start time
	w.commitTransactions(testTxs, w.coinbase, &interrupt)
	// end time
}