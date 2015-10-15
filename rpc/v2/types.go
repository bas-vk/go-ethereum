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

package v2

import (
	"reflect"
)

type rpcRequest struct {
	service string
	method  string
	id      int64
	params  interface{}
}

// RPCError implements RPC error, is add support for error codec over regular go errors
type RPCError interface {
	// RPC error code
	Code() int
	// Error message
	Error() string
}

// ServerCodec implements reading, parsing and writing RPC messages for the server side of a RPC session.
type ServerCodec interface {
	// Read next request
	ReadRequestHeaders() ([]rpcRequest, bool, RPCError)
	// Parse request argument to the given types
	ParseRequestArgument([]reflect.Type, interface{}) ([]reflect.Value, RPCError)
	// Assemble success response
	CreateResponse(int64, interface{}) interface{}
	// Assemble error response
	CreateErrorResponse(int64, RPCError) interface{}
	// Write msg to client, write can be called concurrently and must be thread safe
	Write(interface{}) error
	// Close underlaying data stream
	Close()
}
