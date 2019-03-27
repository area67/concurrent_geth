package main

import (
	"fmt"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

type dummyStatedb struct {
	state.StateDB
}

func main(){

	//

	evm1 := vm.NewEVM(vm.Context{}, &dummyStatedb{}, params.TestChainConfig, vm.Config{})

	evm2 := vm.NewEVM(vm.Context{}, &dummyStatedb{}, params.TestChainConfig, vm.Config{})

	fmt.Println(evm1.BlockNumber)
	fmt.Println(evm2.BlockNumber)
}
