// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package utils

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	rpc "github.com/ethereum/go-ethereum/rpc/v2"
)

// Web3Service offers helper utils
type Web3Service struct {
	stack *node.Node
}

// NewWeb3Service creates a new Web3Service instance
func NewWeb3Service(stack *node.Node) *Web3Service {
	return &Web3Service{stack}
}

// ClientVersion returns the node name
func (s *Web3Service) ClientVersion() string {
	return s.stack.Server().Name
}

// Sha3 applies the ethereum sha3 implementation on the input.
// It assumes the input is hex encoded.
func (s *Web3Service) Sha3(input string) string {
	return common.ToHex(crypto.Sha3(common.FromHex(input)))
}

// NetService offers network related RPC methods
type NetService struct {
	net            *p2p.Server
	networkVersion int
}

func NewNetService(net *p2p.Server, networkVersion int) *NetService {
	return &NetService{net, networkVersion}
}

// Listening returns an indication if the node is listening for network connections.
func (s *NetService) Listening() bool {
	return true // always listening
}

// Peercount returns the number of connected peers
func (s *NetService) PeerCount() *rpc.HexNumber {
	return rpc.NewHexNumber(s.net.PeerCount())
}

// ProtocolVersion returns the current ethereum protocol version.
func (s *NetService) Version() string {
	return fmt.Sprintf("%d", s.networkVersion)
}