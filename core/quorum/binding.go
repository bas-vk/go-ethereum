// This file is an automatically generated Go binding. Do not modify as any
// change will likely be lost upon the next re-generation!

package quorum

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// VotingContractABI is the input ABI used to generate the binding from.
const VotingContractABI = `[{"constant":true,"inputs":[],"name":"voterCount","outputs":[{"name":"","type":"uint256"}],"type":"function"},{"constant":false,"inputs":[{"name":"addr","type":"address"}],"name":"removeVoter","outputs":[],"type":"function"},{"constant":true,"inputs":[{"name":"p","type":"uint256"},{"name":"n","type":"uint256"}],"name":"getEntry","outputs":[{"name":"","type":"bytes32"}],"type":"function"},{"constant":false,"inputs":[{"name":"hash","type":"bytes32"}],"name":"vote","outputs":[],"type":"function"},{"constant":true,"inputs":[{"name":"","type":"address"}],"name":"canVote","outputs":[{"name":"","type":"bool"}],"type":"function"},{"constant":true,"inputs":[],"name":"getSize","outputs":[{"name":"","type":"uint256"}],"type":"function"},{"constant":false,"inputs":[{"name":"addr","type":"address"}],"name":"addVoter","outputs":[],"type":"function"},{"constant":true,"inputs":[],"name":"getCanonHash","outputs":[{"name":"","type":"bytes32"}],"type":"function"},{"inputs":[],"type":"constructor"}]`

// VotingContract is an auto generated Go binding around an Ethereum contract.
type VotingContract struct {
	VotingContractCaller     // Read-only binding to the contract
	VotingContractTransactor // Write-only binding to the contract
}

// VotingContractCaller is an auto generated read-only Go binding around an Ethereum contract.
type VotingContractCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// VotingContractTransactor is an auto generated write-only Go binding around an Ethereum contract.
type VotingContractTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// VotingContractSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type VotingContractSession struct {
	Contract     *VotingContract   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// VotingContractCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type VotingContractCallerSession struct {
	Contract *VotingContractCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// VotingContractTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type VotingContractTransactorSession struct {
	Contract     *VotingContractTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// VotingContractRaw is an auto generated low-level Go binding around an Ethereum contract.
type VotingContractRaw struct {
	Contract *VotingContract // Generic contract binding to access the raw methods on
}

// VotingContractCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type VotingContractCallerRaw struct {
	Contract *VotingContractCaller // Generic read-only contract binding to access the raw methods on
}

// VotingContractTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type VotingContractTransactorRaw struct {
	Contract *VotingContractTransactor // Generic write-only contract binding to access the raw methods on
}

// NewVotingContract creates a new instance of VotingContract, bound to a specific deployed contract.
func NewVotingContract(address common.Address, backend bind.ContractBackend) (*VotingContract, error) {
	contract, err := bindVotingContract(address, backend.(bind.ContractCaller), backend.(bind.ContractTransactor))
	if err != nil {
		return nil, err
	}
	return &VotingContract{VotingContractCaller: VotingContractCaller{contract: contract}, VotingContractTransactor: VotingContractTransactor{contract: contract}}, nil
}

// NewVotingContractCaller creates a new read-only instance of VotingContract, bound to a specific deployed contract.
func NewVotingContractCaller(address common.Address, caller bind.ContractCaller) (*VotingContractCaller, error) {
	contract, err := bindVotingContract(address, caller, nil)
	if err != nil {
		return nil, err
	}
	return &VotingContractCaller{contract: contract}, nil
}

// NewVotingContractTransactor creates a new write-only instance of VotingContract, bound to a specific deployed contract.
func NewVotingContractTransactor(address common.Address, transactor bind.ContractTransactor) (*VotingContractTransactor, error) {
	contract, err := bindVotingContract(address, nil, transactor)
	if err != nil {
		return nil, err
	}
	return &VotingContractTransactor{contract: contract}, nil
}

// bindVotingContract binds a generic wrapper to an already deployed contract.
func bindVotingContract(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(VotingContractABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_VotingContract *VotingContractRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _VotingContract.Contract.VotingContractCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_VotingContract *VotingContractRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _VotingContract.Contract.VotingContractTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_VotingContract *VotingContractRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _VotingContract.Contract.VotingContractTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_VotingContract *VotingContractCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _VotingContract.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_VotingContract *VotingContractTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _VotingContract.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_VotingContract *VotingContractTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _VotingContract.Contract.contract.Transact(opts, method, params...)
}

// CanVote is a free data retrieval call binding the contract method 0xadfaa72e.
//
// Solidity: function canVote( address) constant returns(bool)
func (_VotingContract *VotingContractCaller) CanVote(opts *bind.CallOpts, arg0 common.Address) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _VotingContract.contract.Call(opts, out, "canVote", arg0)
	return *ret0, err
}

// CanVote is a free data retrieval call binding the contract method 0xadfaa72e.
//
// Solidity: function canVote( address) constant returns(bool)
func (_VotingContract *VotingContractSession) CanVote(arg0 common.Address) (bool, error) {
	return _VotingContract.Contract.CanVote(&_VotingContract.CallOpts, arg0)
}

// CanVote is a free data retrieval call binding the contract method 0xadfaa72e.
//
// Solidity: function canVote( address) constant returns(bool)
func (_VotingContract *VotingContractCallerSession) CanVote(arg0 common.Address) (bool, error) {
	return _VotingContract.Contract.CanVote(&_VotingContract.CallOpts, arg0)
}

// GetCanonHash is a free data retrieval call binding the contract method 0xf8d11a57.
//
// Solidity: function getCanonHash() constant returns(bytes32)
func (_VotingContract *VotingContractCaller) GetCanonHash(opts *bind.CallOpts) ([32]byte, error) {
	var (
		ret0 = new([32]byte)
	)
	out := ret0
	err := _VotingContract.contract.Call(opts, out, "getCanonHash")
	return *ret0, err
}

// GetCanonHash is a free data retrieval call binding the contract method 0xf8d11a57.
//
// Solidity: function getCanonHash() constant returns(bytes32)
func (_VotingContract *VotingContractSession) GetCanonHash() ([32]byte, error) {
	return _VotingContract.Contract.GetCanonHash(&_VotingContract.CallOpts)
}

// GetCanonHash is a free data retrieval call binding the contract method 0xf8d11a57.
//
// Solidity: function getCanonHash() constant returns(bytes32)
func (_VotingContract *VotingContractCallerSession) GetCanonHash() ([32]byte, error) {
	return _VotingContract.Contract.GetCanonHash(&_VotingContract.CallOpts)
}

// GetEntry is a free data retrieval call binding the contract method 0x98ba676d.
//
// Solidity: function getEntry(p uint256, n uint256) constant returns(bytes32)
func (_VotingContract *VotingContractCaller) GetEntry(opts *bind.CallOpts, p *big.Int, n *big.Int) ([32]byte, error) {
	var (
		ret0 = new([32]byte)
	)
	out := ret0
	err := _VotingContract.contract.Call(opts, out, "getEntry", p, n)
	return *ret0, err
}

// GetEntry is a free data retrieval call binding the contract method 0x98ba676d.
//
// Solidity: function getEntry(p uint256, n uint256) constant returns(bytes32)
func (_VotingContract *VotingContractSession) GetEntry(p *big.Int, n *big.Int) ([32]byte, error) {
	return _VotingContract.Contract.GetEntry(&_VotingContract.CallOpts, p, n)
}

// GetEntry is a free data retrieval call binding the contract method 0x98ba676d.
//
// Solidity: function getEntry(p uint256, n uint256) constant returns(bytes32)
func (_VotingContract *VotingContractCallerSession) GetEntry(p *big.Int, n *big.Int) ([32]byte, error) {
	return _VotingContract.Contract.GetEntry(&_VotingContract.CallOpts, p, n)
}

// GetSize is a free data retrieval call binding the contract method 0xde8fa431.
//
// Solidity: function getSize() constant returns(uint256)
func (_VotingContract *VotingContractCaller) GetSize(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _VotingContract.contract.Call(opts, out, "getSize")
	return *ret0, err
}

// GetSize is a free data retrieval call binding the contract method 0xde8fa431.
//
// Solidity: function getSize() constant returns(uint256)
func (_VotingContract *VotingContractSession) GetSize() (*big.Int, error) {
	return _VotingContract.Contract.GetSize(&_VotingContract.CallOpts)
}

// GetSize is a free data retrieval call binding the contract method 0xde8fa431.
//
// Solidity: function getSize() constant returns(uint256)
func (_VotingContract *VotingContractCallerSession) GetSize() (*big.Int, error) {
	return _VotingContract.Contract.GetSize(&_VotingContract.CallOpts)
}

// VoterCount is a free data retrieval call binding the contract method 0x42169e48.
//
// Solidity: function voterCount() constant returns(uint256)
func (_VotingContract *VotingContractCaller) VoterCount(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _VotingContract.contract.Call(opts, out, "voterCount")
	return *ret0, err
}

// VoterCount is a free data retrieval call binding the contract method 0x42169e48.
//
// Solidity: function voterCount() constant returns(uint256)
func (_VotingContract *VotingContractSession) VoterCount() (*big.Int, error) {
	return _VotingContract.Contract.VoterCount(&_VotingContract.CallOpts)
}

// VoterCount is a free data retrieval call binding the contract method 0x42169e48.
//
// Solidity: function voterCount() constant returns(uint256)
func (_VotingContract *VotingContractCallerSession) VoterCount() (*big.Int, error) {
	return _VotingContract.Contract.VoterCount(&_VotingContract.CallOpts)
}

// AddVoter is a paid mutator transaction binding the contract method 0xf4ab9adf.
//
// Solidity: function addVoter(addr address) returns()
func (_VotingContract *VotingContractTransactor) AddVoter(opts *bind.TransactOpts, addr common.Address) (*types.Transaction, error) {
	return _VotingContract.contract.Transact(opts, "addVoter", addr)
}

// AddVoter is a paid mutator transaction binding the contract method 0xf4ab9adf.
//
// Solidity: function addVoter(addr address) returns()
func (_VotingContract *VotingContractSession) AddVoter(addr common.Address) (*types.Transaction, error) {
	return _VotingContract.Contract.AddVoter(&_VotingContract.TransactOpts, addr)
}

// AddVoter is a paid mutator transaction binding the contract method 0xf4ab9adf.
//
// Solidity: function addVoter(addr address) returns()
func (_VotingContract *VotingContractTransactorSession) AddVoter(addr common.Address) (*types.Transaction, error) {
	return _VotingContract.Contract.AddVoter(&_VotingContract.TransactOpts, addr)
}

// RemoveVoter is a paid mutator transaction binding the contract method 0x86c1ff68.
//
// Solidity: function removeVoter(addr address) returns()
func (_VotingContract *VotingContractTransactor) RemoveVoter(opts *bind.TransactOpts, addr common.Address) (*types.Transaction, error) {
	return _VotingContract.contract.Transact(opts, "removeVoter", addr)
}

// RemoveVoter is a paid mutator transaction binding the contract method 0x86c1ff68.
//
// Solidity: function removeVoter(addr address) returns()
func (_VotingContract *VotingContractSession) RemoveVoter(addr common.Address) (*types.Transaction, error) {
	return _VotingContract.Contract.RemoveVoter(&_VotingContract.TransactOpts, addr)
}

// RemoveVoter is a paid mutator transaction binding the contract method 0x86c1ff68.
//
// Solidity: function removeVoter(addr address) returns()
func (_VotingContract *VotingContractTransactorSession) RemoveVoter(addr common.Address) (*types.Transaction, error) {
	return _VotingContract.Contract.RemoveVoter(&_VotingContract.TransactOpts, addr)
}

// Vote is a paid mutator transaction binding the contract method 0xa69beaba.
//
// Solidity: function vote(hash bytes32) returns()
func (_VotingContract *VotingContractTransactor) Vote(opts *bind.TransactOpts, hash [32]byte) (*types.Transaction, error) {
	return _VotingContract.contract.Transact(opts, "vote", hash)
}

// Vote is a paid mutator transaction binding the contract method 0xa69beaba.
//
// Solidity: function vote(hash bytes32) returns()
func (_VotingContract *VotingContractSession) Vote(hash [32]byte) (*types.Transaction, error) {
	return _VotingContract.Contract.Vote(&_VotingContract.TransactOpts, hash)
}

// Vote is a paid mutator transaction binding the contract method 0xa69beaba.
//
// Solidity: function vote(hash bytes32) returns()
func (_VotingContract *VotingContractTransactorSession) Vote(hash [32]byte) (*types.Transaction, error) {
	return _VotingContract.Contract.Vote(&_VotingContract.TransactOpts, hash)
}
