package core

import (
	"github.com/ethereum/go-ethereum/core/types"
	"fmt"
)

type Chain struct {
	mgr *ChainManager
}

func NewChainProxy(mgr *ChainManager) *Chain {
	return &Chain{mgr}
}

// no args
func (c *Chain) CurrentBlock() *types.Block {
	return c.mgr.CurrentBlock()
}

// single arg
func (c *Chain) GetBlockByNumber(number uint64) *types.Block {
	return c.mgr.GetBlockByNumber(number)
}

type FooArgs struct {
	S1 string
	S2 string
}

// no return value
func (c *Chain) NoReturnValue(args FooArgs) {
	fmt.Printf("Foo s1: %s, s2: %s\n", args.S1, args.S2)
}

// error response
func (c Chain) ErrorReturn(args FooArgs) error {
	return fmt.Errorf("Some error [%s/%s]", args.S1, args.S2)
}
