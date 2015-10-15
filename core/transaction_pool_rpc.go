package core

import (
	"bytes"
	"fmt"

	"encoding/json"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rlp"
	rpc "github.com/ethereum/go-ethereum/rpc/v2"
	"gopkg.in/fatih/set.v0"
)

const (
	defaultGasPrice = uint64(10000000000000)
	defaultGas = uint64(90000)
)

// RPCTransaction represents a transaction that will serialize to the RPC representation of a transaction
type RPCTransaction struct {
	BlockHash        common.Hash     `json:"blockHash"`
	BlockNumber      *rpc.HexNumber  `json:"blockNumber"`
	From             common.Address  `json:"from"`
	Gas              *rpc.HexNumber  `json:"gas"`
	GasPrice         *rpc.HexNumber  `json:"gasPrice"`
	Hash             common.Hash     `json:"hash"`
	Input            string          `json:"input"`
	Nonce            *rpc.HexNumber  `json:"nonce"`
	To               *common.Address `json:"to"`
	TransactionIndex *rpc.HexNumber  `json:"transactionIndex"`
	Value            *rpc.HexNumber  `json:"value"`
}

// newRPCPendingTransaction returns a pending transaction that will serialize to the RPC representation
func newRPCPendingTransaction(tx *types.Transaction) *RPCTransaction {
	from, _ := tx.From()

	return &RPCTransaction{
		From:     from,
		Gas:      rpc.NewHexNumber(tx.Gas()),
		GasPrice: rpc.NewHexNumber(tx.GasPrice()),
		Hash:     tx.Hash(),
		Input:    fmt.Sprintf("0x%x", tx.Data()),
		Nonce:    rpc.NewHexNumber(tx.Nonce()),
		To:       tx.To(),
		Value:    rpc.NewHexNumber(tx.Value()),
	}
}

// newRPCTransaction returns a transaction that will serialize to the RPC representation.
func newRPCTransactionFromBlockIndex(b *types.Block, txIndex int) (*RPCTransaction, error) {
	if txIndex >= 0 && txIndex < len(b.Transactions()) {
		tx := b.Transactions()[txIndex]
		from, err := tx.From()
		if err != nil {
			return nil, err
		}

		return &RPCTransaction{
			BlockHash:        b.Hash(),
			BlockNumber:      rpc.NewHexNumber(b.Number()),
			From:             from,
			Gas:              rpc.NewHexNumber(tx.Gas()),
			GasPrice:         rpc.NewHexNumber(tx.GasPrice()),
			Hash:             tx.Hash(),
			Input:            fmt.Sprintf("0x%x", tx.Data()),
			Nonce:            rpc.NewHexNumber(tx.Nonce()),
			To:               tx.To(),
			TransactionIndex: rpc.NewHexNumber(txIndex),
			Value:            rpc.NewHexNumber(tx.Value()),
		}, nil
	}

	return nil, nil
}

// newRPCTransaction returns a transaction that will serialize to the RPC representation.
func newRPCTransaction(b *types.Block, txHash common.Hash) (*RPCTransaction, error) {
	for idx, tx := range b.Transactions() {
		if tx.Hash() == txHash {
			return newRPCTransactionFromBlockIndex(b, idx)
		}
	}

	return nil, nil
}

// TransactionPoolService exposes methods for the RPC interface
type TransactionPoolService struct {
	eventMux *event.TypeMux
	chainDb  ethdb.Database
	bc       *BlockChain
	am       *accounts.Manager
	txPool   *TxPool
	txMu     sync.Mutex
}

// NewTransactionPoolService creates a new RPC service with methods specific for the transaction pool.
func NewTransactionPoolService(txPool *TxPool, chainDb ethdb.Database, bc *BlockChain, am *accounts.Manager) *TransactionPoolService {
	return &TransactionPoolService{
		eventMux: txPool.eventMux,
		chainDb:  chainDb,
		bc:       bc,
		am:       am,
		txPool:   txPool,
	}
}

func getTransaction(chainDb ethdb.Database, txPool *TxPool, txHash common.Hash) (*types.Transaction, bool, error) {
	txData, err := chainDb.Get(txHash.Bytes())
	isPending := false
	tx := new(types.Transaction)

	if err == nil && len(txData) > 0 {
		if err := rlp.DecodeBytes(txData, tx); err != nil {
			return nil, isPending, err
		}
	} else {
		// pending transaction?
		tx = txPool.GetTransaction(txHash)
		isPending = true
	}

	return tx, isPending, nil
}

// GetBlockTransactionCountByNumber returns the number of transactions in the block with the given block number.
func (s *TransactionPoolService) GetBlockTransactionCountByNumber(blockNr rpc.BlockNumber) *rpc.HexNumber {
	if blockNr == rpc.PendingBlockNumber {
		return rpc.NewHexNumber(0)
	}

	if block := blockByNumber(s.bc, blockNr); block != nil {
		return rpc.NewHexNumber(len(block.Transactions()))
	}

	return nil
}

// GetBlockTransactionCountByHash returns the number of transactions in the block with the given hash.
func (s *TransactionPoolService) GetBlockTransactionCountByHash(blockHash common.Hash) *rpc.HexNumber {
	if block := s.bc.GetBlock(blockHash); block != nil {
		return rpc.NewHexNumber(len(block.Transactions()))
	}
	return nil
}

// GetTransactionByBlockNumberAndIndex returns the transaction for the given block number and index.
func (s *TransactionPoolService) GetTransactionByBlockNumberAndIndex(blockNr rpc.BlockNumber, index rpc.HexNumber) (*RPCTransaction, error) {
	if block := blockByNumber(s.bc, blockNr); block != nil {
		return newRPCTransactionFromBlockIndex(block, index.Int())
	}
	return nil, nil
}

// GetTransactionByBlockHashAndIndex returns the transaction for the given block hash and index.
func (s *TransactionPoolService) GetTransactionByBlockHashAndIndex(blockHash common.Hash, index rpc.HexNumber) (*RPCTransaction, error) {
	if block := s.bc.GetBlock(blockHash); block != nil {
		return newRPCTransactionFromBlockIndex(block, index.Int())
	}
	return nil, nil
}

// GetTransactionCount returns the number of transactions the given address has sent for the given block number
func (s *TransactionPoolService) GetTransactionCount(address common.Address, blockNr rpc.BlockNumber) (*rpc.HexNumber, error) {
	block := blockByNumber(s.bc, blockNr)
	if block == nil {
		return nil, nil
	}

	state, err := state.New(block.Root(), s.chainDb)
	if err != nil {
		return nil, err
	}
	return rpc.NewHexNumber(state.GetNonce(address)), nil
}

// getTransactionBlockData fetches the meta data for the given transaction from the chain database. This is useful to
// retrieve block information for a hash. It returns the block hash, block index and transaction index.
func getTransactionBlockData(chainDb ethdb.Database, txHash common.Hash) (common.Hash, uint64, uint64, error) {
	var txBlock struct {
		BlockHash  common.Hash
		BlockIndex uint64
		Index      uint64
	}

	blockData, err := chainDb.Get(append(txHash.Bytes(), 0x0001))
	if err != nil {
		return common.Hash{}, uint64(0), uint64(0), err
	}

	reader := bytes.NewReader(blockData)
	if err = rlp.Decode(reader, &txBlock); err != nil {
		return common.Hash{}, uint64(0), uint64(0), err
	}

	return txBlock.BlockHash, txBlock.BlockIndex, txBlock.Index, nil
}

// GetTransactionByHash returns the transaction for the given hash
func (s *TransactionPoolService) GetTransactionByHash(txHash common.Hash) (*RPCTransaction, error) {
	var tx *types.Transaction
	var isPending bool
	var err error

	if tx, isPending, err = getTransaction(s.chainDb, s.txPool, txHash); err != nil {
		glog.V(logger.Debug).Infof("%v\n", err)
		return nil, nil
	} else if tx == nil {
		return nil, nil
	}

	if isPending {
		return newRPCPendingTransaction(tx), nil
	}

	blockHash, _, _, err := getTransactionBlockData(s.chainDb, txHash)
	if err != nil {
		glog.V(logger.Debug).Infof("%v\n", err)
		return nil, nil
	}

	if block := s.bc.GetBlock(blockHash); block != nil {
		return newRPCTransaction(block, txHash)
	}

	return nil, nil
}

// GetTransactionReceipt returns the transaction receipt for the given transaction hash.
func (s *TransactionPoolService) GetTransactionReceipt(txHash common.Hash) (map[string]interface{}, error) {
	receipt := GetReceipt(s.chainDb, txHash)
	if receipt == nil {
		glog.V(logger.Debug).Infof("receipt not found for transaction %s", txHash.Hex())
		return nil, nil
	}

	tx, _, err := getTransaction(s.chainDb, s.txPool, txHash)
	if err != nil {
		glog.V(logger.Debug).Infof("%v\n", err)
		return nil, nil
	}

	txBlock, blockIndex, index, err := getTransactionBlockData(s.chainDb, txHash)
	if err != nil {
		glog.V(logger.Debug).Infof("%v\n", err)
		return nil, nil
	}

	from, err := tx.From()
	if err != nil {
		glog.V(logger.Debug).Infof("%v\n", err)
		return nil, nil
	}

	fields := map[string]interface{}{
		"blockHash":         txBlock,
		"blockNumber":       rpc.NewHexNumber(blockIndex),
		"transactionHash":   txHash,
		"transactionIndex":  rpc.NewHexNumber(index),
		"from":              from,
		"to":                tx.To(),
		"gasUsed":           rpc.NewHexNumber(receipt.GasUsed),
		"cumulativeGasUsed": rpc.NewHexNumber(receipt.CumulativeGasUsed),
		"contractAddress":   nil,
		"logs":              receipt.Logs,
	}

	if receipt.Logs == nil {
		fields["logs"] = []vm.Logs{}
	}

	// If the ContractAddress is 20 0x0 bytes, assume it is not a contract creation
	if bytes.Compare(receipt.ContractAddress.Bytes(), bytes.Repeat([]byte{0}, 20)) != 0 {
		fields["contractAddress"] = receipt.ContractAddress
	}

	return fields, nil
}

// sign is a helper function that signs a transaction with the private key of the given address.
func (s *TransactionPoolService) sign(address common.Address, tx *types.Transaction) (*types.Transaction, error) {
	acc := accounts.Account{address}
	signature, err := s.am.Sign(acc, tx.SigHash().Bytes())
	if err != nil {
		return nil, err
	}
	return tx.WithSignature(signature)
}

type SendTxArgs struct {
	From     common.Address `json:"from"`
	To       common.Address `json:"to"`
	Gas      *rpc.HexNumber `json:"gas"`
	GasPrice *rpc.HexNumber `json:"gasPrice"`
	Value    *rpc.HexNumber `json:"value"`
	Data     string         `json:"data"`
	Nonce    *rpc.HexNumber `json:"nonce"`
}

// SendTransaction will create a transaction for the given transaction argument, sign it and submit it to the
// transaction pool.
func (s *TransactionPoolService) SendTransaction(args SendTxArgs) (common.Hash, error) {
	if args.Gas == nil {
		args.Gas = rpc.NewHexNumber(defaultGas)
	}
	if args.GasPrice == nil {
		args.GasPrice = rpc.NewHexNumber(defaultGasPrice)
	}
	if args.Value == nil {
		args.Value = rpc.NewHexNumber(0)
	}

	s.txMu.Lock()
	defer s.txMu.Unlock()

	if args.Nonce == nil {
		args.Nonce = rpc.NewHexNumber(s.txPool.State().GetNonce(args.From))
	}

	var tx *types.Transaction
	contractCreation := (args.To == common.Address{})

	if contractCreation {
		tx = types.NewContractCreation(args.Nonce.Uint64(), args.Value.BigInt(), args.Gas.BigInt(), args.GasPrice.BigInt(), common.FromHex(args.Data))
	} else {
		tx = types.NewTransaction(args.Nonce.Uint64(), args.To, args.Value.BigInt(), args.Gas.BigInt(), args.GasPrice.BigInt(), common.FromHex(args.Data))
	}

	signedTx, err := s.sign(args.From, tx)
	if err != nil {
		return common.Hash{}, err
	}

	if err := s.txPool.Add(signedTx); err != nil {
		return common.Hash{}, nil
	}

	if contractCreation {
		addr := crypto.CreateAddress(args.From, args.Nonce.Uint64())
		glog.V(logger.Info).Infof("Tx(%s) created: %s\n", signedTx.Hash().Hex(), addr.Hex())
	} else {
		glog.V(logger.Info).Infof("Tx(%s) to: %s\n", signedTx.Hash().Hex(), tx.To().Hex())
	}

	return signedTx.Hash(), nil
}

// SendRawTransaction will add the signed transaction to the transaction pool.
// The sender is responsible for signing the transaction and using the correct nonce.
func (s *TransactionPoolService) SendRawTransaction(encodedTx string) (string, error) {
	tx := new(types.Transaction)
	if err := rlp.DecodeBytes(common.FromHex(encodedTx), tx); err != nil {
		return "", err
	}

	if err := s.txPool.Add(tx); err != nil {
		return "", err
	}

	if tx.To() == nil {
		from, err := tx.From()
		if err != nil {
			return "", err
		}
		addr := crypto.CreateAddress(from, tx.Nonce())
		glog.V(logger.Info).Infof("Tx(%x) created: %x\n", tx.Hash(), addr)
	} else {
		glog.V(logger.Info).Infof("Tx(%x) to: %x\n", tx.Hash(), tx.To())
	}

	return tx.Hash().Hex(), nil
}

// Sign will sign the given data string with the given address. The account corresponding with the address needs to
// be unlocked.
func (s *TransactionPoolService) Sign(address common.Address, data string) (string, error) {
	signature, error := s.am.Sign(accounts.Account{Address: address}, common.HexToHash(data).Bytes())
	return common.ToHex(signature), error
}

type SignTransactionArgs struct {
	From        common.Address
	To          common.Address
	Nonce       *rpc.HexNumber
	Value       *rpc.HexNumber
	Gas         *rpc.HexNumber
	GasPrice    *rpc.HexNumber
	Data        string

	BlockNumber int64
}

// Tx is a helper object for argument and return values
type Tx struct {
	tx       *types.Transaction

	To       *common.Address `json:"to"`
	From     common.Address  `json:"from"`
	Nonce    *rpc.HexNumber  `json:"nonce"`
	Value    *rpc.HexNumber  `json:"value"`
	Data     string          `json:"data"`
	GasLimit *rpc.HexNumber  `json:"gas"`
	GasPrice *rpc.HexNumber  `json:"gasPrice"`
	Hash     common.Hash     `json:"hash"`
}

func (tx *Tx) UnmarshalJSON(b []byte) (err error) {
	req := struct {
		To       common.Address `json:"to"`
		From     common.Address `json:"from"`
		Nonce    *rpc.HexNumber `json:"nonce"`
		Value    *rpc.HexNumber `json:"value"`
		Data     string         `json:"data"`
		GasLimit *rpc.HexNumber `json:"gas"`
		GasPrice *rpc.HexNumber `json:"gasPrice"`
		Hash     common.Hash    `json:"hash"`
	}{}

	if err := json.Unmarshal(b, &req); err != nil {
		return err
	}

	contractCreation := (req.To == (common.Address{}))

	tx.To = &req.To
	tx.From = req.From
	tx.Nonce = req.Nonce
	tx.Value = req.Value
	tx.Data = req.Data
	tx.GasLimit = req.GasLimit
	tx.GasPrice = req.GasPrice
	tx.Hash = req.Hash

	data := common.Hex2Bytes(tx.Data)

	if tx.Nonce == nil {
		return fmt.Errorf("need nonce")
	}
	if tx.Value == nil {
		tx.Value = rpc.NewHexNumber(0)
	}
	if tx.GasLimit == nil {
		tx.GasLimit = rpc.NewHexNumber(0)
	}
	if tx.GasPrice == nil {
		tx.GasPrice = rpc.NewHexNumber(defaultGasPrice)
	}

	if contractCreation {
		tx.tx = types.NewContractCreation(tx.Nonce.Uint64(), tx.Value.BigInt(), tx.GasLimit.BigInt(), tx.GasPrice.BigInt(), data)
	} else {
		if tx.To == nil {
			return fmt.Errorf("need to address")
		}
		tx.tx = types.NewTransaction(tx.Nonce.Uint64(), *tx.To, tx.Value.BigInt(), tx.GasLimit.BigInt(), tx.GasPrice.BigInt(), data)
	}

	return nil
}

type SignTransactionResult struct {
	Raw string `json:"raw"`
	Tx  *Tx    `json:"tx"`
}

func newTx(t *types.Transaction) *Tx {
	from, _ := t.From()
	return &Tx{
		tx:       t,
		To:       t.To(),
		From:     from,
		Value:    rpc.NewHexNumber(t.Value()),
		Nonce:    rpc.NewHexNumber(t.Nonce()),
		Data:     "0x" + common.Bytes2Hex(t.Data()),
		GasLimit: rpc.NewHexNumber(t.Gas()),
		GasPrice: rpc.NewHexNumber(t.GasPrice()),
		Hash:     t.Hash(),
	}
}

// SignTransaction will sign the given transaction with the from account.
// The node needs to have the private key of the account corresponding with
// the given from address and it needs to be unlocked.
func (s *TransactionPoolService) SignTransaction(args *SignTransactionArgs) (*SignTransactionResult, error) {
	if args.Gas == nil {
		args.Gas = rpc.NewHexNumber(defaultGas)
	}
	if args.GasPrice == nil {
		args.GasPrice = rpc.NewHexNumber(defaultGasPrice)
	}
	if args.Value == nil {
		args.Value = rpc.NewHexNumber(0)
	}

	s.txMu.Lock()
	defer s.txMu.Unlock()

	if args.Nonce == nil {
		args.Nonce = rpc.NewHexNumber(s.txPool.State().GetNonce(args.From))
	}

	var tx *types.Transaction
	contractCreation := (args.To == common.Address{})

	if contractCreation {
		tx = types.NewContractCreation(args.Nonce.Uint64(), args.Value.BigInt(), args.Gas.BigInt(), args.GasPrice.BigInt(), common.FromHex(args.Data))
	} else {
		tx = types.NewTransaction(args.Nonce.Uint64(), args.To, args.Value.BigInt(), args.Gas.BigInt(), args.GasPrice.BigInt(), common.FromHex(args.Data))
	}

	signedTx, err := s.sign(args.From, tx)
	if err != nil {
		return nil, err
	}

	data, err := rlp.EncodeToBytes(signedTx)
	if err != nil {
		return nil, err
	}

	return &SignTransactionResult{"0x" + common.Bytes2Hex(data), newTx(tx)}, nil
}

// PendingTransactions returns the transactions that are in the transaction pool and have a from address that is one of
// the accounts this node manages.
func (s *TransactionPoolService) PendingTransactions() ([]*RPCTransaction, error) {
	accounts, err := s.am.Accounts()
	if err != nil {
		return nil, err
	}

	accountSet := set.New()
	for _, account := range accounts {
		accountSet.Add(account.Address)
	}

	pending := s.txPool.GetTransactions()
	transactions := make([]*RPCTransaction, 0)
	for _, tx := range pending {
		if from, _ := tx.From(); accountSet.Has(from) {
			transactions = append(transactions, newRPCPendingTransaction(tx))
		}
	}

	return transactions, nil
}

// NewPendingTransaction creates a subscription that is triggered each time a transaction enters the transaction pool
// and is send from one of the transactions this nodes manages.
func (s *TransactionPoolService) NewPendingTransactions() (rpc.Subscription, error) {
	sub := s.eventMux.Subscribe(TxPreEvent{})

	accounts, err := s.am.Accounts()
	if err != nil {
		return rpc.Subscription{}, err
	}
	accountSet := set.New()
	for _, account := range accounts {
		accountSet.Add(account.Address)
	}
	accountSetLastUpdates := time.Now()

	output := func(transaction interface{}) interface{} {
		if time.Since(accountSetLastUpdates) > (time.Duration(2) * time.Second) {
			if accounts, err = s.am.Accounts(); err != nil {
				accountSet.Clear()
				for _, account := range accounts {
					accountSet.Add(account.Address)
				}
				accountSetLastUpdates = time.Now()
			}
		}

		tx := transaction.(TxPreEvent)
		if from, err := tx.Tx.From(); err == nil {
			if accountSet.Has(from) {
				return tx.Tx.Hash()
			}
		}
		return nil
	}

	return rpc.NewSubscriptionWithOutputFormat(sub, output), nil
}

// Resend accepts an existing transaction and a new gas price and limit. It will remove the given transaction from the
// pool and reinsert it with the new gas price and limit.
func (s *TransactionPoolService) Resend(tx *Tx, gasPrice, gasLimit *rpc.HexNumber) (common.Hash, error) {

	pending := s.txPool.GetTransactions()
	for _, p := range pending {
		if pFrom, err := p.From(); err == nil && pFrom == tx.From && p.SigHash() == tx.tx.SigHash() {
			if gasPrice == nil {
				gasPrice = rpc.NewHexNumber(tx.tx.GasPrice())
			}
			if gasLimit == nil {
				gasLimit = rpc.NewHexNumber(tx.tx.Gas())
			}

			var newTx *types.Transaction
			contractCreation := (*tx.tx.To() == common.Address{})
			if contractCreation {
				newTx = types.NewContractCreation(tx.tx.Nonce(), tx.tx.Value(), gasPrice.BigInt(), gasLimit.BigInt(), tx.tx.Data())
			} else {
				newTx = types.NewTransaction(tx.tx.Nonce(), *tx.tx.To(), tx.tx.Value(), gasPrice.BigInt(), gasLimit.BigInt(), tx.tx.Data())
			}

			signedTx, err := s.sign(tx.From, newTx)
			if err != nil {
				return common.Hash{}, err
			}

			s.txPool.RemoveTx(tx.Hash)
			if err = s.txPool.Add(signedTx); err != nil {
				return common.Hash{}, err
			}

			return signedTx.Hash(), nil
		}
	}

	return common.Hash{}, fmt.Errorf("Transaction %#x not found", tx.Hash)
}
