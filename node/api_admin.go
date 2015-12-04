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

package node

import (
	"fmt"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/rpc/comms"
)

// AdminPrivateApi is the collection of administrative API methods exposed only
// over a secure RPC channel.
type AdminPrivateApi struct {
	node *Node // Node interfaced by this API
}

// AddPeer requests connecting to a remote node, and also maintaining the new
// connection at all times, even reconnecting if it is lost.
func (api *AdminPrivateApi) AddPeer(url string) (bool, error) {
	// Make sure the server is running, fail otherwise
	server := api.node.Server()
	if server == nil {
		return false, ErrNodeStopped
	}
	// Try to add the url as a static peer and return
	node, err := discover.ParseNode(url)
	if err != nil {
		return false, fmt.Errorf("invalid enode: %v", err)
	}
	server.AddPeer(node)
	return true, nil
}

// StartRPC starts the HTTP RPC API server.
func (api *AdminPrivateApi) StartRPC(address string, port int, cors string, apis string) (bool, error) {
	/*// Parse the list of API modules to make available
	apis, err := api.ParseApiString(apis, codec.JSON, xeth.New(api.node, nil), api.node)
	if err != nil {
		return false, err
	}
	// Configure and start the HTTP RPC server
	config := comms.HttpConfig{
		ListenAddress: address,
		ListenPort:    port,
		CorsDomain:    cors,
	}
	if err := comms.StartHttp(config, self.codec, api.Merge(apis...)); err != nil {
		return false, err
	}
	return true, nil*/
	return false, fmt.Errorf("needs new RPC implementation to resolve circular dependency")
}

// StopRPC terminates an already running HTTP RPC API endpoint.
func (api *AdminPrivateApi) StopRPC() {
	comms.StopHttp()
}

// AdminPublicApi is the collection of administrative API methods exposed over
// both secure and unsecure RPC channels.
type AdminPublicApi struct {
	node *Node // Node interfaced by this API
}

// Peers retrieves all the information we know about each individual peer at the
// protocol granularity.
func (api *AdminPublicApi) Peers() ([]*p2p.PeerInfo, error) {
	server := api.node.Server()
	if server == nil {
		return nil, ErrNodeStopped
	}
	return server.PeersInfo(), nil
}

// NodeInfo retrieves all the information we know about the host node at the
// protocol granularity.
func (api *AdminPublicApi) NodeInfo() (*p2p.NodeInfo, error) {
	server := api.node.Server()
	if server == nil {
		return nil, ErrNodeStopped
	}
	return server.NodeInfo(), nil
}

// Datadir retrieves the current data directory the node is using.
func (api *AdminPublicApi) Datadir() string {
	return api.node.DataDir()
}
