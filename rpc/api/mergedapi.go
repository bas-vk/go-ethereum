// Copyright 2015 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.

package api

import (
	"math/rand"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rpc/shared"
)

const (
	MergedApiVersion = "1.0"
)

// combines multiple API's
type MergedApi struct {
	apis    map[string]string
	methods map[string]shared.EthereumApi
}

// create new merged api instance
func newMergedApi(apis ...shared.EthereumApi) *MergedApi {
	mergedApi := new(MergedApi)
	mergedApi.apis = make(map[string]string, len(apis))
	mergedApi.methods = make(map[string]shared.EthereumApi)

	for _, api := range apis {
		mergedApi.apis[api.Name()] = api.ApiVersion()
		for _, method := range api.Methods() {
			mergedApi.methods[method] = api
		}
	}
	return mergedApi
}

// Supported RPC methods
func (self *MergedApi) Methods() []string {
	all := make([]string, len(self.methods))
	for method, _ := range self.methods {
		all = append(all, method)
	}
	return all
}

// Call the correct API's Execute method for the given request
func (self *MergedApi) Execute(req *shared.Request) (res interface{}, err error) {
	reqId := rand.Intn(100000000)
	glog.V(logger.Detail).Infof("REQ[%d]: %s %s", reqId, req.Method, req.Params)
	defer func() {
		glog.V(logger.Detail).Infof("RES[%d]: res %v, err - %v", reqId, res, err)
	}()

	if res, err = self.handle(req); res != nil {
		return 
	}
	if api, found := self.methods[req.Method]; found {
		res, err = api.Execute(req)
		return
	}
	res, err = nil, shared.NewNotImplementedError(req.Method)
	return
}

func (self *MergedApi) Name() string {
	return shared.MergedApiName
}

func (self *MergedApi) ApiVersion() string {
	return MergedApiVersion
}

func (self *MergedApi) handle(req *shared.Request) (interface{}, error) {
	if req.Method == "modules" { // provided API's
		return self.apis, nil
	}

	return nil, nil
}
