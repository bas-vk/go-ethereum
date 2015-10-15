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
	"encoding/json"
	"io"
	"reflect"
	"strings"
	"sync"
)

const (
	jsonRPCVersion = "2.0"
)

type jsonRequest struct {
	Method  string          `json:"method"`
	Version string          `json:"jsonrpc"`
	Id      int64           `json:"id"`
	Payload json.RawMessage `json:"params"`
}

type jsonError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type jsonErrResponse struct {
	Version string    `json:"jsonrpc"`
	Id      int64     `json:"id"`
	Error   jsonError `json:"error"`
}

type jsonSuccessResponse struct {
	Version string      `json:"jsonrpc"`
	Id      int64       `json:"id"`
	Result  interface{} `json:"result,omitempty"`
}

type jsonCodec struct {
	d   *json.Decoder
	mu  sync.Mutex // guards e
	e   *json.Encoder
	req jsonRequest
	rw  io.ReadWriteCloser
}

// NewJSONCodec creates a new RPC server codec with support for JSON-RPC 2.0
func NewJSONCodec(rwc io.ReadWriteCloser) ServerCodec {
	d := json.NewDecoder(rwc)
	d.UseNumber()
	return &jsonCodec{d: d, e: json.NewEncoder(rwc), rw: rwc}
}

func (c *jsonCodec) ReadRequestHeaders() ([]rpcRequest, bool, RPCError) {
	var data json.RawMessage
	if err := c.d.Decode(&data); err != nil {
		return nil, false, &invalidRequest{err.Error()}
	}

	if data[0] == '[' {
		return parseBatchRequest(data)
	}

	return parseRequest(data)
}

func parseRequest(data json.RawMessage) ([]rpcRequest, bool, RPCError) {
	var in jsonRequest
	if err := json.Unmarshal(data, &in); err != nil {
		return nil, false, &invalidMessageError{err.Error()}
	}

	elems := strings.Split(in.Method, ".")
	if len(elems) != 2 {
		return nil, false, &unknownServiceError{in.Method, ""}
	}

	if len(in.Payload) == 0 {
		return []rpcRequest{rpcRequest{service: elems[0], method: elems[1], id: in.Id, params: nil}}, false, nil
	}

	return []rpcRequest{rpcRequest{service: elems[0], method: elems[1], id: in.Id, params: in.Payload}}, false, nil
}

func parseBatchRequest(data json.RawMessage) ([]rpcRequest, bool, RPCError) {
	var in []jsonRequest
	if err := json.Unmarshal(data, &in); err != nil {
		return nil, false, &invalidMessageError{err.Error()}
	}

	requests := make([]rpcRequest, len(in))
	for i, r := range in {
		elems := strings.Split(r.Method, ".")
		if len(elems) != 2 {
			return nil, true, &unknownServiceError{r.Method, ""}
		}

		if len(r.Payload) == 0 {
			requests[i] = rpcRequest{service: elems[0], method: elems[1], id: r.Id, params: nil}
		} else {
			requests[i] = rpcRequest{service: elems[0], method: elems[1], id: r.Id, params: r.Payload}
		}
	}

	return requests, true, nil
}

func (c *jsonCodec) ParseRequestArgument(argTypes []reflect.Type, params interface{}) ([]reflect.Value, RPCError) {
	if data, ok := params.(json.RawMessage); !ok {
		return nil, &invalidParamsError{"Invalid params supplied"}
	} else {
		return parsePositionalArgument(data, argTypes)
	}
}

func parsePositionalArgument(data json.RawMessage, argTypes []reflect.Type) ([]reflect.Value, RPCError) {
	argValues := make([]reflect.Value, len(argTypes))
	params := make([]interface{}, len(argTypes))
	for i, t := range argTypes {
		if t.Kind() == reflect.Ptr {
			argValues[i] = reflect.New(t.Elem())
			params[i] = argValues[i].Interface()
		} else {
			argValues[i] = reflect.New(t)
			params[i] = argValues[i].Interface()
		}
	}

	if err := json.Unmarshal(data, &params); err != nil {
		return nil, &invalidParamsError{err.Error()}
	}

	for i, a := range argValues {
		if a.Kind() != argTypes[i].Kind() {
			argValues[i] = reflect.Indirect(argValues[i])
		}
	}

	return argValues, nil
}

func (c *jsonCodec) CreateResponse(id int64, reply interface{}) interface{} {
	return &jsonSuccessResponse{Version: jsonRPCVersion, Id: id, Result: reply}
}

func (c *jsonCodec) CreateErrorResponse(id int64, err RPCError) interface{} {
	return &jsonErrResponse{Version: jsonRPCVersion, Id: id, Error: jsonError{Code: err.Code(), Message: err.Error()}}
}

func (c *jsonCodec) Write(res interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.e.Encode(&res)
}

func (c *jsonCodec) Close() {
	c.rw.Close()
}
