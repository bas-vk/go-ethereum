package core

import (
	"crypto/ecdsa"
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
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/pow"
)

type BlockMaker struct {
	chainConfig *core.ChainConfig

	blockchain *core.BlockChain
	mux        *event.TypeMux
	db         ethdb.Database

	address common.Address
	abi     abi.ABI
	key     *ecdsa.PrivateKey
	session *VotingContractSession

	quit chan struct{} // quit chan
}

func startVoteSession(node *node.Node, addr common.Address, key *ecdsa.PrivateKey) (*VotingContractSession, error) {
	client, err := node.Attach()
	if err != nil {
		return nil, err
	}

	contract, err := NewVotingContract(addr, backends.NewRPCBackend(client))
	if err != nil {
		panic(err)
	}

	txOpts := bind.NewKeyedTransactor(key)
	txOpts.GasLimit = big.NewInt(5000000)
	txOpts.GasPrice = new(big.Int)
	txOpts.Value = new(big.Int)

	session := &VotingContractSession{
		Contract:     contract,
		TransactOpts: *txOpts,
	}

	return session, nil
}

// NewBlockMaker creates a new block maker.
func NewBlockMaker(chainConfig *core.ChainConfig, contractAddr common.Address, bc *core.BlockChain, db ethdb.Database,
	mux *event.TypeMux, node *node.Node, key *ecdsa.PrivateKey) *BlockMaker {

	voteSession, err := startVoteSession(node, contractAddr, key)

	abi, err := abi.JSON(strings.NewReader(definition))
	if err != nil {
		panic(err)
	}

	bm := &BlockMaker{
		chainConfig: chainConfig,
		db:          db,
		blockchain:  bc,
		abi:         abi,
		key:         key,
		mux:         mux,
		session:     voteSession,
		quit:        make(chan struct{}),
	}
	go bm.update()

	return bm
}

const blockTime = 5 * time.Second

func (bm *BlockMaker) update() {
	eventSub := bm.mux.Subscribe(core.ChainHeadEvent{})
	eventCh := eventSub.Chan()

	lastHeader := bm.blockchain.CurrentHeader()
	for {
		select {
		case event, ok := <-eventCh:
			if !ok {
				// Event subscription closed, set the channel to nil to stop spinning
				eventCh = nil
				continue
			}

			switch ev := event.Data.(type) {
			case core.ChainHeadEvent:
				if ev.Block.Hash() != lastHeader.Hash() && ev.Block.Number().Cmp(lastHeader.Number) > 0 {
				}
			}
		case <-bm.quit:
			return
		}
	}
}

func (bm *BlockMaker) Stop() {
	close(bm.quit)
}

func (bm *BlockMaker) createHeader() (*types.Block, *types.Header) {
	parent := findDecendant(bm.CanonHash(), bm.blockchain)

	tstamp := time.Now().Unix()
	if parent.Time().Int64() >= tstamp {
		tstamp++
	}

	return parent, &types.Header{
		ParentHash: parent.Hash(),
		Number:     new(big.Int).Add(parent.Number(), common.Big1),
		Difficulty: core.CalcDifficulty(bm.chainConfig, uint64(tstamp), parent.Time().Uint64(), parent.Number(), parent.Difficulty()),
		GasLimit:   core.CalcGasLimit(parent),
		GasUsed:    new(big.Int),
		Time:       big.NewInt(tstamp),
	}
}

// Create a new block and include the given transactions.
func (bm *BlockMaker) Create(txs types.Transactions) (*types.Block, *state.StateDB) {
	parent, header := bm.createHeader()

	gp := new(core.GasPool).AddGas(header.GasLimit)
	statedb, _ := state.New(parent.Root(), bm.db)
	var receipts types.Receipts

	for i, tx := range txs {
		snap := statedb.Copy()
		receipt, _, _, err := core.ApplyTransaction(bm.chainConfig, bm.blockchain, gp, statedb, header, tx, header.GasUsed, bm.chainConfig.VmConfig)
		if err != nil {
			switch {
			case core.IsGasLimitErr(err):
				from, _ := tx.From()
				glog.Infof("Gas limit reached for (%x) in this block. Continue to try smaller txs\n", from)
			case err != nil:
				glog.Infof("TX (%x) failed, will be removed: %v\n", tx.Hash().Bytes()[:4], err)
			}
			statedb.Set(snap)

			txs = txs[:i]
			break
		}
		receipts = append(receipts, receipt)
	}
	core.AccumulateRewards(statedb, header, nil)
	header.Root = statedb.IntermediateRoot()

	return types.NewBlock(header, txs, nil, receipts), statedb
}

// CanonHash returns the hash of the latest block within the canonical chain.
func (bm *BlockMaker) CanonHash() common.Hash {
	hash, _ := bm.session.GetCanonHash()
	return hash
}

// Vote for a block hash to be part of the canonical chain.
func (bm *BlockMaker) Vote(hash common.Hash) (*types.Transaction, error) {
	return bm.session.Vote(hash)
}

// AddVoter adds an address to the collection of addresses that are allowed to make votes.
func (bm *BlockMaker) AddVoter(address common.Address) (*types.Transaction, error) {
	return bm.session.AddVoter(address)
}

// RemoveVoter deletes an address from the collection of addresses that are allowed to vote.
func (bm *BlockMaker) RemoveVoter(address common.Address) (*types.Transaction, error) {
	return bm.session.RemoveVoter(address)
}

func (bm *BlockMaker) Verify(block pow.Block) bool {
	newBlock, _ := bm.Create(nil)
	return newBlock.ParentHash() == block.(*types.Block).Hash()
}

func findDecendant(hash common.Hash, blockchain *core.BlockChain) *types.Block {
	if hash == (common.Hash{}) {
		return blockchain.Genesis()
	}

	block := blockchain.GetBlockByHash(hash)
	// get next in line
	return blockchain.GetBlockByNumber(block.NumberU64() + 1)
}

const definition = `[{"constant":false,"inputs":[{"name":"addr","type":"address"}],"name":"removeVoter","outputs":[],"type":"function"},{"constant":true,"inputs":[{"name":"p","type":"uint256"},{"name":"n","type":"uint256"}],"name":"getEntry","outputs":[{"name":"","type":"bytes32"}],"type":"function"},{"constant":false,"inputs":[{"name":"hash","type":"bytes32"}],"name":"vote","outputs":[],"type":"function"},{"constant":true,"inputs":[{"name":"","type":"address"}],"name":"canVote","outputs":[{"name":"","type":"bool"}],"type":"function"},{"constant":true,"inputs":[],"name":"getSize","outputs":[{"name":"","type":"uint256"}],"type":"function"},{"constant":false,"inputs":[{"name":"addr","type":"address"}],"name":"addVoter","outputs":[],"type":"function"},{"constant":true,"inputs":[],"name":"getCanonHash","outputs":[{"name":"","type":"bytes32"}],"type":"function"},{"inputs":[],"type":"constructor"}]`
const DeployCode = `6060604052361561008a576000357c01000000000000000000000000000000000000000000000000000000009004806342169e481461008c57806386c1ff68146100af57806398ba676d146100c7578063a69beaba14610100578063adfaa72e14610118578063de8fa43114610146578063f4ab9adf14610169578063f8d11a57146101815761008a565b005b61009960048050506101a8565b6040518082815260200191505060405180910390f35b6100c560048080359060200190919050506101b1565b005b6100e66004808035906020019091908035906020019091905050610259565b604051808260001916815260200191505060405180910390f35b61011660048080359060200190919050506102af565b005b61012e600480803590602001909190505061048e565b60405180821515815260200191505060405180910390f35b61015360048050506104b3565b6040518082815260200191505060405180910390f35b61017f60048080359060200190919050506104c8565b005b61018e6004805050610563565b604051808260001916815260200191505060405180910390f35b60016000505481565b600260005060003373ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060009054906101000a900460ff1615610255576001600160005054141561020357610002565b600260005060008273ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060006101000a81549060ff02191690556001600081815054809291906001900391905055505b5b50565b60006000600060005084815481101561000257906000526020600020906002020160005b5090508060010160005083815481101561000257906000526020600020900160005b505491506102a8565b5092915050565b6000600260005060003373ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060009054906101000a900460ff16156104895743600060005080549050101561039c5760006000508054809190600101909081548183558181151161039757600202816002028360005260206000209182019101610396919061033b565b8082111561039257600060018201600050805460008255906000526020600020908101906103879190610369565b808211156103835760008181506000905550600101610369565b5090565b5b505060020161033b565b5090565b5b505050505b600060005060014303815481101561000257906000526020600020906002020160005b50905060008160000160005060008460001916815260200190815260200160002060005054141561045a5780600101600050805480600101828181548183558181151161043e5781836000526020600020918201910161043d919061041f565b80821115610439576000818150600090555060010161041f565b5090565b5b5050509190906000526020600020900160005b84909190915055505b806000016000506000836000191681526020019081526020016000206000818150548092919060010191905055505b5b5050565b600260005060205280600052604060002060009150909054906101000a900460ff1681565b600060006000508054905090506104c5565b90565b600260005060003373ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060009054906101000a900460ff161561055f576001600260005060008373ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060006101000a81548160ff0219169083021790555060016000818150548092919060010191905055505b5b50565b60006000600060006000600050600160006000508054905003815481101561000257906000526020600020906002020160005b509250600090505b826001016000508054905081101561064a578260000160005060008460010160005083815481101561000257906000526020600020900160005b5054600019168152602001908152602001600020600050548360000160005060008460001916815260200190815260200160002060005054101561063c578260010160005081815481101561000257906000526020600020900160005b5054915081505b5b808060010191505061059e565b819350610652565b5050509056`
