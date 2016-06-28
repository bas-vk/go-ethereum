package quorum

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

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
	"gopkg.in/fatih/set.v0"
)

const definition = `[{"constant":false,"inputs":[{"name":"addr","type":"address"}],"name":"removeVoter","outputs":[],"type":"function"},{"constant":true,"inputs":[{"name":"p","type":"uint256"},{"name":"n","type":"uint256"}],"name":"getEntry","outputs":[{"name":"","type":"bytes32"}],"type":"function"},{"constant":false,"inputs":[{"name":"hash","type":"bytes32"}],"name":"vote","outputs":[],"type":"function"},{"constant":true,"inputs":[{"name":"","type":"address"}],"name":"canVote","outputs":[{"name":"","type":"bool"}],"type":"function"},{"constant":true,"inputs":[],"name":"getSize","outputs":[{"name":"","type":"uint256"}],"type":"function"},{"constant":false,"inputs":[{"name":"addr","type":"address"}],"name":"addVoter","outputs":[],"type":"function"},{"constant":true,"inputs":[],"name":"getCanonHash","outputs":[{"name":"","type":"bytes32"}],"type":"function"},{"inputs":[],"type":"constructor"}]`
const DeployCode = `6060604052361561008a576000357c01000000000000000000000000000000000000000000000000000000009004806342169e481461008c57806386c1ff68146100af57806398ba676d146100c7578063a69beaba14610100578063adfaa72e14610118578063de8fa43114610146578063f4ab9adf14610169578063f8d11a57146101815761008a565b005b61009960048050506101a8565b6040518082815260200191505060405180910390f35b6100c560048080359060200190919050506101b1565b005b6100e660048080359060200190919080359060200190919050506102b6565b604051808260001916815260200191505060405180910390f35b610116600480803590602001909190505061030c565b005b61012e60048080359060200190919050506105ac565b60405180821515815260200191505060405180910390f35b61015360048050506105d1565b6040518082815260200191505060405180910390f35b61017f60048080359060200190919050506105e6565b005b61018e60048050506106de565b604051808260001916815260200191505060405180910390f35b60016000505481565b600260005060003373ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060009054906101000a900460ff1615610259576001600160005054141561020357610002565b600260005060008273ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060006101000a81549060ff02191690556001600081815054809291906001900391905055506102b2565b7f054bb91f2e6a3ea70c2704666a0b9c440fcdd463a0fceccb4ae7b21222232c6b3360014303604051808373ffffffffffffffffffffffffffffffffffffffff1681526020018281526020019250505060405180910390a15b5b50565b60006000600060005084815481101561000257906000526020600020906002020160005b5090508060010160005083815481101561000257906000526020600020900160005b50549150610305565b5092915050565b6000600260005060003373ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060009054906101000a900460ff161561054e574360006000508054905010156103f9576000600050805480919060010190908154818355818115116103f4576002028160020283600052602060002091820191016103f39190610398565b808211156103ef57600060018201600050805460008255906000526020600020908101906103e491906103c6565b808211156103e057600081815060009055506001016103c6565b5090565b5b5050600201610398565b5090565b5b505050505b600060005060014303815481101561000257906000526020600020906002020160005b5090506000816000016000506000846000191681526020019081526020016000206000505414156104b75780600101600050805480600101828181548183558181151161049b5781836000526020600020918201910161049a919061047c565b80821115610496576000818150600090555060010161047c565b5090565b5b5050509190906000526020600020900160005b84909190915055505b806000016000506000836000191681526020019081526020016000206000818150548092919060010191905055507f3d03ba7f4b5227cdb385f2610906e5bcee147171603ec40005b30915ad20e258336001430384604051808473ffffffffffffffffffffffffffffffffffffffff16815260200183815260200182600019168152602001935050505060405180910390a16105a7565b7f054bb91f2e6a3ea70c2704666a0b9c440fcdd463a0fceccb4ae7b21222232c6b3360014303604051808373ffffffffffffffffffffffffffffffffffffffff1681526020018281526020019250505060405180910390a15b5b5050565b600260005060205280600052604060002060009150909054906101000a900460ff1681565b600060006000508054905090506105e3565b90565b600260005060003373ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060009054906101000a900460ff1615610681576001600260005060008373ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060006101000a81548160ff0219169083021790555060016000818150548092919060010191905055506106da565b7f054bb91f2e6a3ea70c2704666a0b9c440fcdd463a0fceccb4ae7b21222232c6b3360014303604051808373ffffffffffffffffffffffffffffffffffffffff1681526020018281526020019250505060405180910390a15b5b50565b60006000600060006000600050600160006000508054905003815481101561000257906000526020600020906002020160005b509250600090505b82600101600050805490508110156107c5578260000160005060008460010160005083815481101561000257906000526020600020900160005b505460001916815260200190815260200160002060005054836000016000506000846000191681526020019081526020016000206000505410156107b7578260010160005081815481101561000257906000526020600020900160005b5054915081505b5b8080600101915050610719565b8193506107cd565b5050509056`

var (
	contractAddress = common.HexToAddress("0x0000000000000000000000000000000000000020")

	// temporary hard coded, must be supplied
	MasterKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	masterAddr   = crypto.PubkeyToAddress(MasterKey.PublicKey)

	ErrSynchronising = errors.New("could not create block: synchronising blockchain")
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

// pendingState holds the state that will be converted into a "mined" block.
type work struct {
	header             *types.Header
	state              *state.StateDB
	receipts           types.Receipts
	ignoredTransactors *set.Set
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

	active bool
	work   *work

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
		work:        new(work),
		quit:        make(chan struct{}),
	}

	// initialize pending state
	head := bv.blockchain.CurrentBlock()
	bv.reset(head)

	return bv
}

// DeployVotingContract writes the genesis block with the voting contract included.
func deployVotingContract(db ethdb.Database) {
	balance, _ := new(big.Int).SetString("0xffffffffffffffffffffffff", 0)
	contract := core.GenesisAccount{contractAddress, new(big.Int), common.FromHex(DeployCode), map[string]string{
		"0000000000000000000000000000000000000000000000000000000000000001": "01",
		// add addr1 as a voter
		common.Bytes2Hex(crypto.Keccak256(append(common.LeftPadBytes(masterAddr[:], 32), common.LeftPadBytes([]byte{2}, 32)...))): "01",
	}}
	acc1 := core.GenesisAccount{masterAddr, balance, nil, nil}
	acc2 := core.GenesisAccount{common.HexToAddress("0x740d08a357b075e4161f88574a0a56ab82ed951b"), balance, nil, nil}

	glog.Infof("Deploy voting contract at %s", contract.Address.Hex())
	glog.Infof("account: %s balance: %d", acc1.Address.Hex(), acc1.Balance)
	glog.Infof("account: %s balance: %d", acc2.Address.Hex(), acc2.Balance)

	core.WriteGenesisBlockForTesting(db, contract, acc1, acc2)
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

const blockTime = 30 * time.Second

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
				bv.reset(e.Block) // new head, reset pending state
			case core.TxPreEvent:
				bv.applyTransactions(bv.work, types.Transactions{e.Tx}) // tx entered tx pool, apply to pending state
			case MakeBlock:
				bv.createBlock() // asked to generate new block
			}
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

	// 1. lookup canon hash
	ch, err := bv.CanonHash()
	if err != nil {
		glog.Errorf("unable to retrieve canonical hash: %v", err)
		ch = bv.blockchain.CurrentBlock().Hash()
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
	glog.V(logger.Debug).Infof("block %s found, base new block on top of that", cBlock.Hash().Hex())

	transactions := bv.txpool.GetTransactions()
	types.SortByPriceAndNonce(transactions)

	nBlock, _, _, _, err := bv.Create(cBlock, transactions)
	if err != nil {
		glog.Errorf("unable to create new block: %v", err)
		return
	}

	// 4. insert into chain
	if _, err := bv.blockchain.InsertChain(types.Blocks{nBlock}); err != nil {
		glog.Errorf("unable to insert new block: %v", err)
		return
	}

	glog.Infof("created new block %s", nBlock.Hash().Hex())

	go bv.mux.Post(core.NewMinedBlockEvent{Block: nBlock})
}

// reset current state with the given block as parent block.
func (bv *BlockVoting) reset(parent *types.Block) {
	//now := time.Now()
	//tstamp := now.Unix()
	//
	//// timestamp of parent in future, adjust
	//if parent.Time().Cmp(new(big.Int).SetInt64(tstamp)) >= 0 {
	//	tstamp = parent.Time().Int64() + 1
	//}
	//
	//// make sure we are not too far in the future
	//deadline := now.Unix() + 4
	//if tstamp > deadline {
	//	wait := time.Duration(tstamp-deadline) * time.Second
	//	glog.Infof("too far in the future, wait for %d second", wait.Seconds())
	//	time.Sleep(wait)
	//}
	//
	//// create new header for the pending state
	//header := &types.Header{
	//	ParentHash: parent.Hash(),
	//	Number:     new(big.Int).Add(parent.Number(), common.Big1),
	//	Difficulty: core.CalcDifficulty(bv.chainConfig, uint64(tstamp), parent.Time().Uint64(), parent.Number(), parent.Difficulty()),
	//	GasLimit:   core.CalcGasLimit(parent),
	//	GasUsed:    new(big.Int),
	//	Coinbase:   crypto.PubkeyToAddress(bv.key.PublicKey),
	//	Extra:      []byte(""),
	//	Time:       big.NewInt(tstamp),
	//}
	//
	//state, err := state.New(parent.Root(), bv.db)
	//if err != nil {
	//	glog.Errorf("unable to make pending state: %v", err)
	//	return
	//}
	//
	//// generate pending state
	//w := &work{
	//	header:             header,
	//	state:              state,
	//	ignoredTransactors: set.New(),
	//}
	//
	//// apply transaction to pending state
	//txs := bv.txpool.GetTransactions()
	//types.SortByPriceAndNonce(txs)
	//
	//bv.applyTransactions(w, txs)
	//
	//header.Root = state.IntermediateRoot()
	//bv.work = w
}

func applyTransaction(tx *types.Transaction, bc *core.BlockChain, gp *core.GasPool) (vm.Logs, error) {
	return nil, nil

	/*
		func (env *Work) commitTransaction(tx *types.Transaction, bc *core.BlockChain, gp *core.GasPool) (error, vm.Logs) {
				snap := env.state.Copy()

				// this is a bit of a hack to force jit for the miners
				config := env.config.VmConfig
				if !(config.EnableJit && config.ForceJit) {
					config.EnableJit = false
				}
				config.ForceJit = false // disable forcing jit

				receipt, logs, _, err := core.ApplyTransaction(env.config, bc, gp, env.state, env.header, tx, env.header.GasUsed, config)
				if err != nil {
					env.state.Set(snap)
					return err, nil
				}
				env.txs = append(env.txs, tx)
				env.receipts = append(env.receipts, receipt)

				return nil, logs
			}
	*/
}

// applyTransaction executes the given transaction on the given pending state
func (bv *BlockVoting) applyTransactions(w *work, txs types.Transactions) {
	//gp := new(core.GasPool).AddGas(w.header.GasLimit)
	//var receipts types.Receipts
	//var allLogs vm.Logs
	//
	//for _, tx := range txs {
	//	// Error may be ignored here. The error has already been checked
	//	// during transaction acceptance is the transaction pool.
	//	from, _ := tx.From()
	//
	//	// Move on to the next transaction when the transactor is in ignored transactions set
	//	// This may occur when a transaction hits the gas limit. When a gas limit is hit and
	//	// the transaction is processed (that could potentially be included in the block) it
	//	// will throw a nonce error because the previous transaction hasn't been processed.
	//	// Therefor we need to ignore any transaction after the ignored one.
	//	if w.ignoredTransactors.Has(from) {
	//		continue
	//	}
	//
	//	w.state.StartRecord(tx.Hash(), common.Hash{}, 0)
	//
	//	// make copy of state in case transaction fails to execute
	//	snapshot := w.state.Copy()
	//
	//	config := bv.chainConfig.VmConfig
	//
	//	receipt, logs, _, err := core.ApplyTransaction(bv.chainConfig, bv.blockchain, gp, w.state, w.header, tx, w.header.GasUsed, config)
	//	if err != nil {
	//		glog.Errorf("unable to execute tx %s: %v", tx.Hash(), err)
	//		w.state = snapshot
	//		continue
	//	}
	//
	//	receipts = append(receipts, receipt)
	//}
	//
	//w.receipts = append(w.receipts, receipts...)

	/*
		func (env *Work) commitTransactions(mux *event.TypeMux, transactions types.Transactions, gasPrice *big.Int, bc *core.BlockChain) {
			gp := new(core.GasPool).AddGas(env.header.GasLimit)

			var coalescedLogs vm.Logs
			for _, tx := range transactions {

				env.state.StartRecord(tx.Hash(), common.Hash{}, 0)

				err, logs := env.commitTransaction(tx, bc, gp)
				switch {
				case core.IsGasLimitErr(err):
					// ignore the transactor so no nonce errors will be thrown for this account
					// next time the worker is run, they'll be picked up again.
					env.ignoredTransactors.Add(from)

					glog.V(logger.Detail).Infof("Gas limit reached for (%x) in this block. Continue to try smaller txs\n", from[:4])
				case err != nil:
					env.remove.Add(tx.Hash())

					if glog.V(logger.Detail) {
						glog.Infof("TX (%x) failed, will be removed: %v\n", tx.Hash().Bytes()[:4], err)
					}
				default:
					env.tcount++
					coalescedLogs = append(coalescedLogs, logs...)
				}
			}
			if len(coalescedLogs) > 0 || env.tcount > 0 {
				go func(logs vm.Logs, tcount int) {
					if len(logs) > 0 {
						mux.Post(core.PendingLogsEvent{Logs: logs})
					}
					if tcount > 0 {
						mux.Post(core.PendingStateEvent{})
					}
				}(coalescedLogs, env.tcount)
			}
		}

		func (env *Work) commitTransaction(tx *types.Transaction, bc *core.BlockChain, gp *core.GasPool) (error, vm.Logs) {
			snap := env.state.Copy()

			// this is a bit of a hack to force jit for the miners
			config := env.config.VmConfig
			if !(config.EnableJit && config.ForceJit) {
				config.EnableJit = false
			}
			config.ForceJit = false // disable forcing jit

			receipt, logs, _, err := core.ApplyTransaction(env.config, bc, gp, env.state, env.header, tx, env.header.GasUsed, config)
			if err != nil {
				env.state.Set(snap)
				return err, nil
			}
			env.txs = append(env.txs, tx)
			env.receipts = append(env.receipts, receipt)

			return nil, logs
		}
	*/
}

func (bv *BlockVoting) createHeader(parent *types.Block) (*types.Block, *types.Header) {
	tstamp := time.Now().Unix()
	if parent.Time().Int64() >= tstamp {
		tstamp++
	}

	return parent, &types.Header{
		ParentHash: parent.Hash(),
		Number:     new(big.Int).Add(parent.Number(), common.Big1),
		Difficulty: core.CalcDifficulty(bv.chainConfig, uint64(tstamp), parent.Time().Uint64(), parent.Number(), parent.Difficulty()),
		GasLimit:   core.CalcGasLimit(parent),
		GasUsed:    new(big.Int),
		Time:       big.NewInt(tstamp),
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

	for i, tx := range txs {
		fmt.Printf(">> %s\n", gp)
		snap := statedb.Copy()
		receipt, logs, _, err := core.ApplyTransaction(bv.chainConfig, bv.blockchain, gp, statedb, header, tx, header.GasUsed, bv.chainConfig.VmConfig)
		if err != nil {
			switch {
			case core.IsGasLimitErr(err):
				glog.Infof("Gas limit reached for (%s) in this block. Continue to try smaller txs\n", tx.Hash().Hex())
			default:
				glog.Infof("TX (%x) failed, will be removed: %v\n", tx.Hash().Bytes()[:4], err)
			}
			statedb.Set(snap) // rollback state to state after last successful tx
			txs = txs[:i]     // successful included tx
			break
		}

		receipts = append(receipts, receipt)
		allLogs = append(allLogs, logs...)

		fmt.Printf("<< %s\n", gp)
	}

	core.AccumulateRewards(statedb, header, nil)
	header.Root = statedb.IntermediateRoot()

	// update block hash since it is now available and not when the receipt/log of individual transactions were created
	for _, r := range receipts {
		for _, l := range r.Logs {
			l.BlockHash = header.Hash()
		}
	}
	for _, log := range statedb.Logs() {
		log.BlockHash = header.Hash()
	}

	header.Bloom = types.CreateBloom(receipts)
	return types.NewBlock(header, txs, nil, receipts), statedb, receipts, allLogs, nil
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
	block := bv.blockchain.CurrentBlock()
	s, _ := state.New(block.Root(), bv.db)
	return block, s
}

func findDecendant(hash common.Hash, blockchain *core.BlockChain) *types.Block {
	if hash == (common.Hash{}) {
		return blockchain.Genesis()
	}

	block := blockchain.GetBlockByHash(hash)
	// get next in line
	return blockchain.GetBlockByNumber(block.NumberU64() + 1)
}
