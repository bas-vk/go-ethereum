package quorum

import (
	"crypto/ecdsa"
	"errors"
	"math/big"
	"math/rand"
	"strings"
	"sync"
	"time"

	"gopkg.in/fatih/set.v0"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/pow"
)

const definition = `[{"constant":false,"inputs":[{"name":"addr","type":"address"}],"name":"removeVoter","outputs":[],"type":"function"},{"constant":true,"inputs":[{"name":"p","type":"uint256"},{"name":"n","type":"uint256"}],"name":"getEntry","outputs":[{"name":"","type":"bytes32"}],"type":"function"},{"constant":false,"inputs":[{"name":"hash","type":"bytes32"}],"name":"vote","outputs":[],"type":"function"},{"constant":true,"inputs":[{"name":"","type":"address"}],"name":"canVote","outputs":[{"name":"","type":"bool"}],"type":"function"},{"constant":true,"inputs":[],"name":"getSize","outputs":[{"name":"","type":"uint256"}],"type":"function"},{"constant":false,"inputs":[{"name":"addr","type":"address"}],"name":"addVoter","outputs":[],"type":"function"},{"constant":true,"inputs":[],"name":"getCanonHash","outputs":[{"name":"","type":"bytes32"}],"type":"function"},{"inputs":[],"type":"constructor"}]`
const DeployCode = `6060604052361561008a576000357c01000000000000000000000000000000000000000000000000000000009004806342169e481461008c57806386c1ff68146100af57806398ba676d146100c7578063a69beaba14610100578063adfaa72e14610118578063de8fa43114610146578063f4ab9adf14610169578063f8d11a57146101815761008a565b005b61009960048050506101a8565b6040518082815260200191505060405180910390f35b6100c560048080359060200190919050506101b1565b005b6100e6600480803590602001909190803590602001909190505061025e565b604051808260001916815260200191505060405180910390f35b61011660048080359060200190919050506102b4565b005b61012e6004808035906020019091905050610498565b60405180821515815260200191505060405180910390f35b61015360048050506104bd565b6040518082815260200191505060405180910390f35b61017f60048080359060200190919050506104d2565b005b61018e6004805050610572565b604051808260001916815260200191505060405180910390f35b60016000505481565b600260005060003373ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060009054906101000a900460ff1615610259576001600160005054141561020357610002565b600260005060008273ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060006101000a81549060ff021916905560016000818150548092919060019003919050555061025a565b5b5b50565b60006000600060005084815481101561000257906000526020600020906002020160005b5090508060010160005083815481101561000257906000526020600020900160005b505491506102ad565b5092915050565b6000600260005060003373ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060009054906101000a900460ff1615610492574360006000508054905010156103a15760006000508054809190600101909081548183558181151161039c5760020281600202836000526020600020918201910161039b9190610340565b80821115610397576000600182016000508054600082559060005260206000209081019061038c919061036e565b80821115610388576000818150600090555060010161036e565b5090565b5b5050600201610340565b5090565b5b505050505b600060005060014303815481101561000257906000526020600020906002020160005b50905060008160000160005060008460001916815260200190815260200160002060005054141561045f57806001016000508054806001018281815481835581811511610443578183600052602060002091820191016104429190610424565b8082111561043e5760008181506000905550600101610424565b5090565b5b5050509190906000526020600020900160005b84909190915055505b80600001600050600083600019168152602001908152602001600020600081815054809291906001019190505550610493565b5b5b5050565b600260005060205280600052604060002060009150909054906101000a900460ff1681565b600060006000508054905090506104cf565b90565b600260005060003373ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060009054906101000a900460ff161561056d576001600260005060008373ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060006101000a81548160ff02191690830217905550600160008181505480929190600101919050555061056e565b5b5b50565b60006000600060006000600050600160006000508054905003815481101561000257906000526020600020906002020160005b509250600090505b8260010160005080549050811015610659578260000160005060008460010160005083815481101561000257906000526020600020900160005b5054600019168152602001908152602001600020600050548360000160005060008460001916815260200190815260200160002060005054101561064b578260010160005081815481101561000257906000526020600020900160005b5054915081505b5b80806001019150506105ad565b819350610661565b5050509056`

var (
	contractAddress = common.HexToAddress("0x0000000000000000000000000000000000000020")

	// temporary hard coded, must be supplied
	MasterKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	masterAddr   = crypto.PubkeyToAddress(MasterKey.PublicKey)

	ErrSynchronising = errors.New("could not create block: synchronising blockchain")

	gasPrice = common.Big0 // use a zero gas price for consortium chain
)

type PrivateBlockVotingAPI struct {
	vote *BlockVoting
	mux  *event.TypeMux
}

// NewPrivateBlockVotingAPI creates a new voting API.
func NewPrivateBlockVotingAPI(vote *BlockVoting, mux *event.TypeMux) *PrivateBlockVotingAPI {
	return &PrivateBlockVotingAPI{vote, mux}
}

func (api *PrivateBlockVotingAPI) Vote(h common.Hash) (common.Hash, error) {
	tx, err := api.vote.Vote(h)
	if err != nil {
		return common.Hash{}, err
	}

	api.vote.txpool.Add(tx)
	return tx.Hash(), nil
}

// CanonicalHash returns the canonical hash for the block voting algorithm.
func (api *PrivateBlockVotingAPI) CanonicalHash() (common.Hash, error) {
	return api.vote.CanonHash()
}

func (api *PrivateBlockVotingAPI) Period() (*big.Int, error) {
	return api.vote.GetSize()
}

func (api *PrivateBlockVotingAPI) VoterCount() (*big.Int, error) {
	return api.vote.VoterCount()
}

// MakeBlock orders block voting to generate a new block.
func (api *PrivateBlockVotingAPI) MakeBlock() (bool, error) {
	return true, api.mux.Post(MakeBlock{})
}

// MakeBlock is posted to the event mux when BlockVoting must generate a block.
type MakeBlock struct{}

// pendingState holds the state where a new block will be based on.
type pendingState struct {
	header   *types.Header
	state    *state.StateDB
	receipts types.Receipts
	txs      types.Transactions
	config   *core.ChainConfig

	removed            *set.Set // transactions which are removed due to an error on execution
	ignoredTransactors *set.Set // tx sender which are temporary ignored to prevent nonce errors when a previous tx failed
}

type BlockVoting struct {
	chainConfig *core.ChainConfig

	blockchain *core.BlockChain
	txpool     *core.TxPool
	mux        *event.TypeMux
	db         ethdb.Database

	address      common.Address
	abi          abi.ABI
	key          *ecdsa.PrivateKey
	voteContract *VotingContractSession

	active   bool
	pStateMu sync.RWMutex
	pState   *pendingState

	quit chan struct{} // quit chan
}

type BlockVotingVerifier struct {
}

func NewVerifier() *BlockVotingVerifier {
	return new(BlockVotingVerifier)
}

func (bvv *BlockVotingVerifier) Verify(block pow.Block) bool {
	// TODO, for now assume all blocks are valid
	// in the future verify if signature in extra field is an allowed block maker
	// since we don't have a pow nonce for block voting
	return true
}

// NewBlockVoting creates a new block maker.
func NewBlockVoting(config *core.ChainConfig, db ethdb.Database, bc *core.BlockChain, txpool *core.TxPool, mux *event.TypeMux, key *ecdsa.PrivateKey) *BlockVoting {
	deployVotingContract(db)

	abi, err := abi.JSON(strings.NewReader(definition))
	if err != nil {
		panic(err)
	}

	bv := &BlockVoting{
		chainConfig: config,
		blockchain:  bc,
		abi:         abi,
		key:         key,
		mux:         mux,
		db:          db,
		txpool:      txpool,
		active:      true,
		quit:        make(chan struct{}),
	}

	// initialize pending state
	head := bv.blockchain.CurrentBlock()
	bv.resetPendingState(head)

	return bv
}

// DeployVotingContract writes the genesis block with the voting contract included.
func deployVotingContract(db ethdb.Database) {
	balance, _ := new(big.Int).SetString("0xffffffffffffffffffffffff", 0)
	acc1 := core.GenesisAccount{masterAddr, balance, nil, nil}
	acc2 := core.GenesisAccount{common.HexToAddress("0x391694e7e0b0cce554cb130d723a9d27458f9298"), balance, nil, nil}
	acc3 := core.GenesisAccount{common.HexToAddress("0xafa3f8684e54059998bc3a7b0d2b0da075154d66"), balance, nil, nil}

	contract := core.GenesisAccount{contractAddress, new(big.Int), common.FromHex(DeployCode), map[string]string{
		"0000000000000000000000000000000000000000000000000000000000000001": "01",
		// add addr1 as a voter
		common.Bytes2Hex(crypto.Keccak256(append(common.LeftPadBytes(masterAddr[:], 32), common.LeftPadBytes([]byte{2}, 32)...))): "01",
		// add acc2 as voter
		common.Bytes2Hex(crypto.Keccak256(append(common.LeftPadBytes(acc2.Address[:], 32), common.LeftPadBytes([]byte{2}, 32)...))): "01",
		// add acc3 as voter
		common.Bytes2Hex(crypto.Keccak256(append(common.LeftPadBytes(acc3.Address[:], 32), common.LeftPadBytes([]byte{2}, 32)...))): "01",
	}}

	glog.Infof("Deploy voting contract at %s", contract.Address.Hex())
	glog.Infof("Master addr: %s\n", masterAddr.Hex())
	glog.Infof("account: %s balance: %d", acc1.Address.Hex(), acc1.Balance)
	glog.Infof("account: %s balance: %d", acc2.Address.Hex(), acc2.Balance)
	glog.Infof("account: %s balance: %d", acc3.Address.Hex(), acc3.Balance)

	core.WriteGenesisBlockForTesting(db, contract, acc1, acc2, acc3)
}

func (bv *BlockVoting) Attach(node *node.Node) error {
	client, err := node.Attach()
	if err != nil {
		return err
	}

	contract, err := NewVotingContract(contractAddress, backends.NewRPCBackend(client))
	if err != nil {
		return err
	}

	glog.Infof("voting contract tx from: %s\n", crypto.PubkeyToAddress(bv.key.PublicKey).Hex())

	txOpts := bind.NewKeyedTransactor(bv.key)
	txOpts.GasLimit = big.NewInt(250000)
	txOpts.GasPrice = new(big.Int)
	txOpts.Value = new(big.Int)

	bv.voteContract = &VotingContractSession{
		Contract:     contract,
		TransactOpts: *txOpts,
	}

	go bv.update()

	return nil
}

func (bv *BlockVoting) update() {
	eventSub := bv.mux.Subscribe(
		downloader.StartEvent{},
		downloader.DoneEvent{},
		downloader.FailedEvent{},
		core.ChainHeadEvent{},
		core.TxPreEvent{},
		MakeBlock{},
	)

	defer eventSub.Unsubscribe()

	ticker := time.After(time.Duration(10+rand.Intn(11)) * time.Second)
	for {
		select {
		case event, ok := <-eventSub.Chan():
			if !ok {
				return
			}

			switch e := event.Data.(type) {
			case downloader.StartEvent:
				bv.active = false // during syncing don't vote/generate blocks
			case downloader.DoneEvent, downloader.FailedEvent:
				bv.active = true // syncing stopped, allow voting/block generation
			case core.ChainHeadEvent:
				if bv.active {
					bv.resetPendingState(e.Block)      // new head, reset pending state
					tx, err := bv.Vote(e.Block.Hash()) // vote for our head
					if err != nil {
						bv.txpool.Add(tx)
					}
					ticker = time.After(time.Duration(10+rand.Intn(11)) * time.Second)
				}
			case core.TxPreEvent:
				bv.applyTransactions(types.Transactions{e.Tx}) // tx entered tx pool, apply to pending state
			case MakeBlock:
				bv.createBlock() // asked to generate new block
				ticker = time.After(time.Duration(10+rand.Intn(11)) * time.Second)
			}
		case <-ticker:
			bv.createBlock() // asked to generate new block
			ticker = time.After(time.Duration(10+rand.Intn(11)) * time.Second)
		case <-bv.quit:
			return
		}
	}
}

func (bv *BlockVoting) Start() {
	go bv.update()
}

func (bv *BlockVoting) Stop() {
	close(bv.quit)
}

func (bv *BlockVoting) createBlock() {
	if !bv.active {
		glog.V(logger.Debug).Infoln("don't generate block during sync")
		return
	}

	// 1. lookup canon hash from contract, if this fails use the local chain head as canon hash.
	ch, err := bv.CanonHash()
	if err != nil {
		glog.Errorf("unable to retrieve canonical hash: %v", err)
		ch = bv.blockchain.CurrentBlock().Hash()
	} else {
		glog.Errorln("found canonical hash")
	}

	// 2. if canonical hash is not within database vote for local chain head and return
	cBlock := bv.blockchain.GetBlockByHash(ch)
	if cBlock == nil {
		glog.V(logger.Debug).Infof("canonical hash %s not in db, vote for chain head", ch.Hex())
		cBlock = bv.blockchain.CurrentBlock()
		voteTx, err := bv.Vote(cBlock.Hash())
		if err != nil {
			glog.Errorf("unable to vote for %s: %v", cBlock.Hash().Hex(), err)
			return
		}
		if err := bv.txpool.Add(voteTx); err != nil {
			glog.Errorf("unable to vote for %s: %v", cBlock.Hash().Hex(), err)
		}
		return
	}

	// 3. block found, create new block on top of that
	glog.V(logger.Debug).Infof("base new block on top of %s", cBlock.Hash().Hex())

	transactions := bv.txpool.GetTransactions()
	types.SortByPriceAndNonce(transactions)

	nBlock, _, _, _, err := bv.Create(cBlock, transactions)
	if err != nil {
		glog.Errorf("unable to create new block: %v", err)
		return
	}

	// 4. insert into chain, do full validation
	if _, err := bv.blockchain.InsertChain(types.Blocks{nBlock}); err != nil {
		glog.Errorf("unable to insert new block: %v", err)
		return
	}

	glog.Infof("created new block %s", nBlock.Hash().Hex())

	go bv.mux.Post(core.NewMinedBlockEvent{Block: nBlock})
}

// resetPendingState create a new pending state with the given block as parent.
func (bv *BlockVoting) resetPendingState(parent *types.Block) {
	state, err := state.New(parent.Root(), bv.db)
	if err != nil {
		glog.Error("unable to create new pending state: %v", err)
		return
	}

	tstart := time.Now()
	tstamp := tstart.Unix()
	if parent.Time().Cmp(new(big.Int).SetInt64(tstamp)) >= 0 {
		tstamp = parent.Time().Int64() + 1
	}
	// this will ensure we're not going off too far in the future
	if now := time.Now().Unix(); tstamp > now+4 {
		wait := time.Duration(tstamp-now) * time.Second
		glog.V(logger.Info).Infoln("We are too far in the future. Waiting for", wait)
		time.Sleep(wait)
	}

	header := &types.Header{
		Number:     new(big.Int).Add(parent.Number(), common.Big1),
		ParentHash: parent.Hash(),
		Difficulty: core.CalcDifficulty(bv.chainConfig, uint64(tstamp), parent.Time().Uint64(), parent.Number(), parent.Difficulty()),
		GasLimit:   core.CalcGasLimit(parent),
		GasUsed:    new(big.Int),
		Coinbase:   masterAddr, // TODO, replace with local account addr
		Extra:      []byte(""), // TODO, when block is finilized sign it
		Time:       big.NewInt(tstamp),
	}

	pState := &pendingState{
		header:             header,
		state:              state,
		config:             bv.chainConfig,
		removed:            set.New(),
		ignoredTransactors: set.New(),
	}

	bv.pStateMu.Lock()
	bv.pState = pState
	bv.pStateMu.Unlock()

	glog.V(logger.Detail).Infoln("pending state is reset")
}

// applyTransaction the given transaction onto the pending state.
func (ps *pendingState) applyTransaction(tx *types.Transaction, bc *core.BlockChain, cc *core.ChainConfig, gp *core.GasPool) (vm.Logs, error) {
	// create recover point in case tx fails
	snapshot := ps.state.Copy()

	config := ps.config.VmConfig
	if !(config.EnableJit && config.ForceJit) {
		config.EnableJit = false
	}
	config.ForceJit = false // disable forcing jit

	receipt, logs, _, err := core.ApplyTransaction(cc, bc, gp, ps.state, ps.header, tx, ps.header.GasUsed, config)
	if err != nil {
		ps.state.Set(snapshot) // rollback
		return nil, err
	}

	// tx successful applied to current pending state
	ps.txs = append(ps.txs, tx)
	ps.receipts = append(ps.receipts, receipt)

	return logs, nil
}

// applyTransaction executes the given transactions on the pending state
func (bv *BlockVoting) applyTransactions(txs types.Transactions) {
	bv.pStateMu.Lock()
	defer bv.pStateMu.Unlock()

	gp := new(core.GasPool).AddGas(bv.pState.header.GasLimit)

	var coalescedLogs vm.Logs

	// shortcut
	pState := bv.pState
	nTx := 0

	for _, tx := range txs {
		from, _ := tx.From()

		// no gas price check, allow transaction with gas price of 0

		// Move on to the next transaction when the transactor is in ignored transactions set
		// This may occur when a transaction hits the gas limit. When a gas limit is hit and
		// the transaction is processed (that could potentially be included in the block) it
		// will throw a nonce error because the previous transaction hasn't been processed.
		// Therefor we need to ignore any transaction after the ignored one.
		if pState.ignoredTransactors.Has(from) {
			continue
		}

		pState.state.StartRecord(tx.Hash(), common.Hash{}, 0)

		logs, err := pState.applyTransaction(tx, bv.blockchain, bv.chainConfig, gp)
		switch {
		case core.IsGasLimitErr(err):
			// ignore the transactor so no nonce errors will be thrown for this account
			// when the pending state is reset (new/import block) it will be picked up again
			pState.ignoredTransactors.Add(from)
			glog.V(logger.Detail).Infof("Gas limit reached for (%x) in this block. Continue to try smaller txs\n", from[:4])
		case err == nil:
			coalescedLogs = append(coalescedLogs, logs...)
			nTx++
		default: // some error occured on execution, remove tx
			pState.removed.Add(tx.Hash())
			glog.V(logger.Detail).Infof("TX (%x) failed, will be removed: %v\n", tx.Hash().Bytes()[:4], err)
		}
	}

	// post pending state events if there where some txs succesful executed
	if len(coalescedLogs) > 0 || nTx > 0 {
		go func(logs vm.Logs, tCount int) {
			if len(logs) > 0 {
				bv.mux.Post(core.PendingLogsEvent{logs})
			}
			if tCount > 0 {
				bv.mux.Post(core.PendingStateEvent{})
			}
		}(coalescedLogs, nTx)
	}
}

func (bv *BlockVoting) createHeader(parent *types.Block) (*types.Block, *types.Header) {
	tstamp := time.Now().Unix()
	if parent.Time().Int64() >= tstamp {
		tstamp++
	}

	return parent, &types.Header{
		ParentHash: parent.Hash(),
		Number:     new(big.Int).Add(parent.Number(), common.Big1),
		Difficulty: core.CalcDifficulty(bv.chainConfig, uint64(tstamp), parent.Time().Uint64(), parent.Number(), parent.Difficulty()), /* TODO, remove difficulty */
		GasLimit:   core.CalcGasLimit(parent),
		GasUsed:    new(big.Int),
		Time:       big.NewInt(tstamp),
		Coinbase:   masterAddr,
	}
}

// Create a new block and include the given transactions.
func (bv *BlockVoting) Create(parent *types.Block, txs types.Transactions) (*types.Block, *state.StateDB, types.Receipts, vm.Logs, error) {
	if !bv.active {
		return nil, nil, nil, nil, ErrSynchronising
	}

	parent, header := bv.createHeader(parent)

	gp := new(core.GasPool).AddGas(header.GasLimit)
	statedb, _ := state.New(parent.Root(), bv.db)
	var receipts types.Receipts
	var allLogs vm.Logs

	var includedTxs types.Transactions

	for i, tx := range txs {
		checkpoint := statedb.Copy() // create checkpoint for rollback in case tx fails
		statedb.StartRecord(tx.Hash(), common.Hash{}, i)
		receipt, logs, _, err := core.ApplyTransaction(bv.chainConfig, bv.blockchain, gp, statedb, header, tx, header.GasUsed, bv.chainConfig.VmConfig)

		switch {
		case err == nil:
			glog.V(logger.Debug).Infof("tx %s applied", tx.Hash().Hex())
			includedTxs = append(includedTxs, tx)
			receipts = append(receipts, receipt)
			allLogs = append(allLogs, logs...)
		case core.IsGasLimitErr(err):
			glog.Infof("Gas limit reached for (%s) in this block. Continue to try smaller txs\n", tx.Hash().Hex())
			statedb.Set(checkpoint) // rollback state to state before failed tx
		default:
			glog.Infof("TX (%x) failed, will be removed: %v\n", tx.Hash().Bytes()[:4], err)
			bv.txpool.RemoveTransactions(types.Transactions{tx})
			statedb.Set(checkpoint) // rollback state to state before failed tx
		}
	}

	core.AccumulateRewards(statedb, header, nil) // TODO, maybe we should remove this and use a gasprice and static tx fee of 0
	header.Root, _ = statedb.Commit()

	// update block hash in receipts and logs now it is available
	for _, r := range receipts {
		for _, l := range r.Logs {
			l.BlockHash = header.Hash()
		}
	}
	for _, log := range statedb.Logs() {
		log.BlockHash = header.Hash()
	}

	header.Bloom = types.CreateBloom(receipts)

	// no uncles for consoritium chain
	return types.NewBlock(header, includedTxs, nil, receipts), statedb, receipts, allLogs, nil
}

// CanonHash returns the hash of the latest block within the canonical chain.
func (bv *BlockVoting) CanonHash() (common.Hash, error) {
	return bv.voteContract.GetCanonHash()
}

// Vote for a block hash to be part of the canonical chain.
func (bv *BlockVoting) Vote(hash common.Hash) (*types.Transaction, error) {
	return bv.voteContract.Vote(hash)
}

// AddVoter adds an address to the collection of addresses that are allowed to make votes.
func (bv *BlockVoting) AddVoter(address common.Address) (*types.Transaction, error) {
	return bv.voteContract.AddVoter(address)
}

// RemoveVoter deletes an address from the collection of addresses that are allowed to vote.
func (bv *BlockVoting) RemoveVoter(address common.Address) (*types.Transaction, error) {
	return bv.voteContract.RemoveVoter(address)
}

// GetSize returns the current period.
func (bv *BlockVoting) GetSize() (*big.Int, error) {
	return bv.voteContract.GetSize()
}

func (bv *BlockVoting) VoterCount() (*big.Int, error) {
	return bv.voteContract.VoterCount()
}

func (bv *BlockVoting) Verify(block pow.Block) bool {
	// TODO, add validation
	//newBlock, _, _ := bv.Create(nil, nil)
	//return newBlock.ParentHash() == block.(*types.Block).Hash()
	return true
}

// Pending returns block and state for the head of the canonical chain.
// For BlockVoting there is no pending state.
func (bv *BlockVoting) Pending() (*types.Block, *state.StateDB) {
	bv.pStateMu.RLock()
	defer bv.pStateMu.RUnlock()

	return types.NewBlock(
		bv.pState.header,
		bv.pState.txs,
		nil,
		bv.pState.receipts,
	), bv.pState.state
}

func findDecendant(hash common.Hash, blockchain *core.BlockChain) *types.Block {
	if hash == (common.Hash{}) {
		return blockchain.Genesis()
	}

	block := blockchain.GetBlockByHash(hash)
	// get next in line
	return blockchain.GetBlockByNumber(block.NumberU64() + 1)
}
