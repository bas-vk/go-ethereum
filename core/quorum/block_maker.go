package quorum

import (
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/pow"
)

// BlockMaker is an algorithm to generate blocks.
type BlockMaker interface {
	Start()
	Stop()
	Verify(pow.Block) bool

	Pending() (*types.Block, *state.StateDB)

	// Attach block maker to node (abigen)
	Attach(*node.Node) error
}
