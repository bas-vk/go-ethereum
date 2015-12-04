// Copyright 2015 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package eth

import (
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/ethash"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/compiler"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rlp"
	rpc "github.com/ethereum/go-ethereum/rpc/v2"
)

// EthService exposes the RPC methods for the ethereum package
type EthService struct {
	e   *Ethereum
	gpo *GasPriceOracle
}

// NewEthService creates a new RPC service for the ethereum package
func NewEthService(e *Ethereum) *EthService {
	return &EthService{e, NewGasPriceOracle(e)}
}

// GasPrice returns a suggestion for a gas price.
func (s *EthService) GasPrice() *big.Int {
	return s.gpo.SuggestPrice()
}

// GetCompilers returns the collection of available smart contract compilers
func (s *EthService) GetCompilers() ([]string, error) {
	solc, err := s.e.Solc()
	if err != nil {
		return nil, err
	}

	if solc != nil {
		return []string{"Solidity"}, nil
	}

	return nil, nil
}

// CompileSolidity compiles the given solidity source
func (s *EthService) CompileSolidity(source string) (map[string]*compiler.Contract, error) {
	solc, err := s.e.Solc()
	if err != nil {
		return nil, err
	}

	if solc == nil {
		return nil, errors.New("solc (solidity compiler) not found")
	}

	return solc.Compile(source)
}

// Etherbase is the address that mining rewards will be send to
func (s *EthService) Etherbase() (common.Address, error) {
	return s.e.Etherbase()
}

// see Etherbase
func (s *EthService) Coinbase() (common.Address, error) {
	return s.Etherbase()
}

// ProtocolVersion returns the current Ethereum protocol version this node supports
func (s *EthService) ProtocolVersion() *rpc.HexNumber {
	return rpc.NewHexNumber(s.e.EthVersion())
}

// Hashrate returns the POW hashrate
func (s *EthService) Hashrate() *rpc.HexNumber {
	return rpc.NewHexNumber(s.e.Miner().HashRate())
}

// Syncing returns false in case the node is currently not synching with the network. It can be up to date or has not
// yet received the latest block headers from its pears. In case it is synchronizing an object with 3 properties is
// returned:
// - startingBlock: block number this node started to synchronise from
// - currentBlock: block number this node is currently importing
// - highestBlock: block number of the highest block header this node has received from peers
func (s *EthService) Syncing() (interface{}, error) {
	origin, current, height := s.e.Downloader().Progress()
	if current < height {
		return map[string]interface{}{
			"startingBlock": rpc.NewHexNumber(origin),
			"currentBlock":  rpc.NewHexNumber(current),
			"highestBlock":  rpc.NewHexNumber(height),
		}, nil
	}
	return false, nil
}

// EthAdminPrivateApi is the collection of Etheruem APIs exposed over the private
// admin endpoint.
type EthAdminPrivateApi struct {
	eth *Ethereum
}

// SetSolc sets the Solidity compiler path to be used by the node.
func (api *EthAdminPrivateApi) SetSolc(path string) (string, error) {
	solc, err := api.eth.SetSolc(path)
	if err != nil {
		return "", err
	}
	return solc.Info(), nil
}

// EthDebugPublicApi is the collection of Etheruem APIs exposed over the public
// debugging endpoint.
type EthDebugPublicApi struct {
	eth *Ethereum
}

// DumpBlock retrieves the entire state of the database at a given block.
func (api *EthDebugPublicApi) DumpBlock(number uint64) (state.World, error) {
	block := api.eth.BlockChain().GetBlockByNumber(number)
	if block == nil {
		return state.World{}, fmt.Errorf("block #%d not found", number)
	}
	stateDb, err := state.New(block.Root(), api.eth.ChainDb())
	if err != nil {
		return state.World{}, err
	}
	return stateDb.RawDump(), nil
}

// GetBlockRlp retrieves the RLP encoded for of a single block.
func (api *EthDebugPublicApi) GetBlockRlp(number uint64) (string, error) {
	block := api.eth.BlockChain().GetBlockByNumber(number)
	if block == nil {
		return "", fmt.Errorf("block #%d not found", number)
	}
	encoded, err := rlp.EncodeToBytes(block)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", encoded), nil
}

// PrintBlock retrieves a block and returns its pretty printed form.
func (api *EthDebugPublicApi) PrintBlock(number uint64) (string, error) {
	block := api.eth.BlockChain().GetBlockByNumber(number)
	if block == nil {
		return "", fmt.Errorf("block #%d not found", number)
	}
	return fmt.Sprintf("%s", block), nil
}

// SeedHash retrieves the seed hash of a block.
func (api *EthDebugPublicApi) SeedHash(number uint64) (string, error) {
	block := api.eth.BlockChain().GetBlockByNumber(number)
	if block == nil {
		return "", fmt.Errorf("block #%d not found", number)
	}
	hash, err := ethash.GetSeedHash(number)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("0x%x", hash), nil
}

// EthDebugPrivageApi is the collection of Etheruem APIs exposed over the private
// debugging endpoint.
type EthDebugPrivateApi struct {
	eth *Ethereum
}

// ProcessBlock reprocesses an already owned block.
func (api *EthDebugPrivateApi) ProcessBlock(number uint64) (bool, error) {
	// Fetch the block that we aim to reprocess
	block := api.eth.BlockChain().GetBlockByNumber(number)
	if block == nil {
		return false, fmt.Errorf("block #%d not found", number)
	}
	// Temporarily enable debugging
	defer func(old bool) { vm.Debug = old }(vm.Debug)
	vm.Debug = true

	// Validate and reprocess the block
	var (
		blockchain = api.eth.BlockChain()
		validator  = blockchain.Validator()
		processor  = blockchain.Processor()
	)
	if err := core.ValidateHeader(blockchain.AuxValidator(), block.Header(), blockchain.GetHeader(block.ParentHash()), true, false); err != nil {
		return false, err
	}
	statedb, err := state.New(blockchain.GetBlock(block.ParentHash()).Root(), api.eth.ChainDb())
	if err != nil {
		return false, err
	}
	receipts, _, usedGas, err := processor.Process(block, statedb)
	if err != nil {
		return false, err
	}
	if err := validator.ValidateState(block, blockchain.GetBlock(block.ParentHash()), statedb, receipts, usedGas); err != nil {
		return false, err
	}
	return true, nil
}

// SetHead rewinds the head of the blockchain to a previous block.
func (api *EthDebugPrivateApi) SetHead(number uint64) {
	api.eth.BlockChain().SetHead(number)
}

// MinerManagementService provides private RPC methods to control the miner
type MinerManagementService struct {
	e *Ethereum
}

// NewMinerManagementService create a new RPC service which controls the miner of this node.
func NewMinerManagementService(e *Ethereum) *MinerManagementService {
	return &MinerManagementService{e: e}
}

// Start the miner with the given number of threads
func (s *MinerManagementService) Start(threads rpc.HexNumber) (bool, error) {
	s.e.StartAutoDAG()
	err := s.e.StartMining(threads.Int(), "")
	if err == nil {
		return true, nil
	}
	return false, err
}

// Stop the miner
func (s *MinerManagementService) Stop() bool {
	s.e.StopMining()
	return true
}

// SetExtra sets the extra data string that is included when this miner mines a block.
func (s *MinerManagementService) SetExtra(extra string) (bool, error) {
	if err := s.e.Miner().SetExtra([]byte(extra)); err != nil {
		return false, err
	}
	return true, nil
}

// SetGasPrice sets the minimum accepted gas price for the miner.
func (s *MinerManagementService) SetGasPrice(gasPrice rpc.Number) bool {
	s.e.Miner().SetGasPrice(gasPrice.BigInt())
	return true
}

// SetEtherbase sets the etherbase of the miner
func (s *MinerManagementService) SetEtherbase(etherbase common.Address) bool {
	s.e.SetEtherbase(etherbase)
	return true
}

// StartAutoDAG starts auto DAG generation. This will prevent the DAG generating on epoch change
// which will cause the node to stop mining during the generation process.
func (s *MinerManagementService) StartAutoDAG() bool {
	s.e.StartAutoDAG()
	return true
}

// StopAutoDAG stops auto DAG generation
func (s *MinerManagementService) StopAutoDAG() bool {
	s.e.StopAutoDAG()
	return true
}

// MakeDAG creates the new DAG for the given block number
func (s *MinerManagementService) MakeDAG(blockNr rpc.BlockNumber) (bool, error) {
	if err := ethash.MakeDAG(uint64(blockNr.Int64()), ""); err != nil {
		return false, err
	}
	return true, nil
}

// TxPoolService offers and API for the
type TxPoolService struct {
	e *Ethereum
}

// NewTxPoolService creates a new tx pool service that gives information about the transaction pool.
func NewTxPoolService(e *Ethereum) *TxPoolService {
	return &TxPoolService{e}
}

// Status returns the number of pending and queued transaction in the pool.
func (s *TxPoolService) Status() map[string]*rpc.HexNumber {
	pending, queue := s.e.TxPool().Stats()
	return map[string]*rpc.HexNumber{
		"pending": rpc.NewHexNumber(pending),
		"queued":  rpc.NewHexNumber(queue),
	}
}

// AccountService represents a RPC service with support for account specific actions.
type AccountService struct {
	am *accounts.Manager
}

// NewAccountService creates a new Account RPC service instance.
func NewAccountService(am *accounts.Manager) *AccountService {
	return &AccountService{am: am}
}

// Accounts returns the collection of accounts this node manages
func (s *AccountService) Accounts() ([]accounts.Account, error) {
	return s.am.Accounts()
}

// PersonalService represents a RPC service with support for personal methods.
type PersonalService struct {
	am *accounts.Manager
}

// NewPersonalService creates a new RPC service with support for personal actions.
func NewPersonalService(am *accounts.Manager) *PersonalService {
	return &PersonalService{am}
}

// ListAccounts will return a list of addresses for accounts this node manages.
func (s *PersonalService) ListAccounts() ([]common.Address, error) {
	accounts, err := s.am.Accounts()
	if err != nil {
		return nil, err
	}

	addresses := make([]common.Address, len(accounts))
	for i, acc := range accounts {
		addresses[i] = acc.Address
	}
	return addresses, nil
}

// NewAccount will create a new account and returns the address for the new account.
func (s *PersonalService) NewAccount(password string) (common.Address, error) {
	acc, err := s.am.NewAccount(password)
	if err == nil {
		return acc.Address, nil
	}
	return common.Address{}, err
}

// UnlockAccount will unlock the account associated with the given address with the given password for duration seconds.
// It returns an indication if the action was successful.
func (s *PersonalService) UnlockAccount(addr common.Address, password string, duration int) bool {
	if err := s.am.TimedUnlock(addr, password, time.Duration(duration)*time.Second); err != nil {
		glog.V(logger.Info).Infof("%v\n", err)
		return false
	}
	return true
}

// LockAccount will lock the account associated with the given address when it's unlocked.
func (s *PersonalService) LockAccount(addr common.Address) bool {
	return s.am.Lock(addr) == nil
}
