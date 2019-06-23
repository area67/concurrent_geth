// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package increment

import (
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = abi.U256
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// TestaABI is the input ABI used to generate the binding from.
const TestaABI = "[{\"constant\":false,\"inputs\":[],\"name\":\"decrement\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"get\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"increment\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"

// TestaBin is the compiled bytecode used for deploying new contracts.
const TestaBin = `608060405234801561001057600080fd5b5060c68061001f6000396000f3fe6080604052348015600f57600080fd5b5060043610603c5760003560e01c80632baeceb71460415780636d4ce63c146049578063d09de08a146065575b600080fd5b6047606d565b005b604f607f565b6040518082815260200191505060405180910390f35b606b6088565b005b60016000808282540392505081905550565b60008054905090565b6001600080828254019250508190555056fea165627a7a72305820ed2cfa27f1752d694f5b0fac7b43254211cbe3ff9943c179d195af51919d4ff80029`

// DeployTesta deploys a new Ethereum contract, binding an instance of Testa to it.
func DeployTesta(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Testa, error) {
	parsed, err := abi.JSON(strings.NewReader(TestaABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(TestaBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Testa{TestaCaller: TestaCaller{contract: contract}, TestaTransactor: TestaTransactor{contract: contract}, TestaFilterer: TestaFilterer{contract: contract}}, nil
}

// Testa is an auto generated Go binding around an Ethereum contract.
type Testa struct {
	TestaCaller     // Read-only binding to the contract
	TestaTransactor // Write-only binding to the contract
	TestaFilterer   // Log filterer for contract events
}

// TestaCaller is an auto generated read-only Go binding around an Ethereum contract.
type TestaCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TestaTransactor is an auto generated write-only Go binding around an Ethereum contract.
type TestaTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TestaFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type TestaFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TestaSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type TestaSession struct {
	Contract     *Testa            // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// TestaCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type TestaCallerSession struct {
	Contract *TestaCaller  // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// TestaTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type TestaTransactorSession struct {
	Contract     *TestaTransactor  // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// TestaRaw is an auto generated low-level Go binding around an Ethereum contract.
type TestaRaw struct {
	Contract *Testa // Generic contract binding to access the raw methods on
}

// TestaCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type TestaCallerRaw struct {
	Contract *TestaCaller // Generic read-only contract binding to access the raw methods on
}

// TestaTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type TestaTransactorRaw struct {
	Contract *TestaTransactor // Generic write-only contract binding to access the raw methods on
}

// NewTesta creates a new instance of Testa, bound to a specific deployed contract.
func NewTesta(address common.Address, backend bind.ContractBackend) (*Testa, error) {
	contract, err := bindTesta(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Testa{TestaCaller: TestaCaller{contract: contract}, TestaTransactor: TestaTransactor{contract: contract}, TestaFilterer: TestaFilterer{contract: contract}}, nil
}

// NewTestaCaller creates a new read-only instance of Testa, bound to a specific deployed contract.
func NewTestaCaller(address common.Address, caller bind.ContractCaller) (*TestaCaller, error) {
	contract, err := bindTesta(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &TestaCaller{contract: contract}, nil
}

// NewTestaTransactor creates a new write-only instance of Testa, bound to a specific deployed contract.
func NewTestaTransactor(address common.Address, transactor bind.ContractTransactor) (*TestaTransactor, error) {
	contract, err := bindTesta(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &TestaTransactor{contract: contract}, nil
}

// NewTestaFilterer creates a new log filterer instance of Testa, bound to a specific deployed contract.
func NewTestaFilterer(address common.Address, filterer bind.ContractFilterer) (*TestaFilterer, error) {
	contract, err := bindTesta(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &TestaFilterer{contract: contract}, nil
}

// bindTesta binds a generic wrapper to an already deployed contract.
func bindTesta(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(TestaABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Testa *TestaRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Testa.Contract.TestaCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Testa *TestaRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Testa.Contract.TestaTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Testa *TestaRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Testa.Contract.TestaTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Testa *TestaCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Testa.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Testa *TestaTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Testa.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Testa *TestaTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Testa.Contract.contract.Transact(opts, method, params...)
}

// Get is a free data retrieval call binding the contract method 0x6d4ce63c.
//
// Solidity: function get() constant returns(uint256)
func (_Testa *TestaCaller) Get(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _Testa.contract.Call(opts, out, "get")
	return *ret0, err
}

// Get is a free data retrieval call binding the contract method 0x6d4ce63c.
//
// Solidity: function get() constant returns(uint256)
func (_Testa *TestaSession) Get() (*big.Int, error) {
	return _Testa.Contract.Get(&_Testa.CallOpts)
}

// Get is a free data retrieval call binding the contract method 0x6d4ce63c.
//
// Solidity: function get() constant returns(uint256)
func (_Testa *TestaCallerSession) Get() (*big.Int, error) {
	return _Testa.Contract.Get(&_Testa.CallOpts)
}

// Decrement is a paid mutator transaction binding the contract method 0x2baeceb7.
//
// Solidity: function decrement() returns()
func (_Testa *TestaTransactor) Decrement(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Testa.contract.Transact(opts, "decrement")
}

// Decrement is a paid mutator transaction binding the contract method 0x2baeceb7.
//
// Solidity: function decrement() returns()
func (_Testa *TestaSession) Decrement() (*types.Transaction, error) {
	return _Testa.Contract.Decrement(&_Testa.TransactOpts)
}

// Decrement is a paid mutator transaction binding the contract method 0x2baeceb7.
//
// Solidity: function decrement() returns()
func (_Testa *TestaTransactorSession) Decrement() (*types.Transaction, error) {
	return _Testa.Contract.Decrement(&_Testa.TransactOpts)
}

// Increment is a paid mutator transaction binding the contract method 0xd09de08a.
//
// Solidity: function increment() returns()
func (_Testa *TestaTransactor) Increment(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Testa.contract.Transact(opts, "increment")
}

// Increment is a paid mutator transaction binding the contract method 0xd09de08a.
//
// Solidity: function increment() returns()
func (_Testa *TestaSession) Increment() (*types.Transaction, error) {
	return _Testa.Contract.Increment(&_Testa.TransactOpts)
}

// Increment is a paid mutator transaction binding the contract method 0xd09de08a.
//
// Solidity: function increment() returns()
func (_Testa *TestaTransactorSession) Increment() (*types.Transaction, error) {
	return _Testa.Contract.Increment(&_Testa.TransactOpts)
}
