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
	"fmt"
	"net"
	"reflect"
	"sync"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

type callback struct {
	method   reflect.Method // callback
	argTypes []reflect.Type // input argument types
	errPos   int            // err return idx, of -1 when method cannot return error
}

type service struct {
	name      string        // name for service
	rcvr      reflect.Value // receiver of methods for the service
	typ       reflect.Type  // receiver type
	callbacks callbacks     // registered handlers
}

type serverRequest struct {
	id      int64
	svcname string
	rcvr    reflect.Value
	callb   *callback
	args    []reflect.Value
}

type serviceRegistry map[string]*service
type callbacks map[string]*callback

// Server represents a RPC server
type Server struct {
	listener   net.Listener
	serviceMap serviceRegistry
	mu         sync.Mutex // protects codec collection
}

// NewServer will create a new server instance with no registered handlers.
func NewServer() *Server {
	return &Server{
		serviceMap: make(serviceRegistry),
	}
}

// Register publishes suitable methods of rcvr. Methods will be published
// with a dot between the rcrv and method, e.g. Calculator.Add. The supplied
// rcvr must be a pointer.
func (s *Server) Register(rcvr interface{}) error {
	return s.register(rcvr, "", false)
}

// Register publishes suitable methods of rcvr. Methods will be published
// with a dot between the given name and method, e.g. Calculator.Add. The
// supplied rcvr must be a pointer.
func (s *Server) RegisterName(name string, rcvr interface{}) error {
	return s.register(rcvr, name, true)
}

func (s *Server) register(rcvr interface{}, name string, useName bool) error {
	if s.serviceMap == nil {
		s.serviceMap = make(map[string]*service)
	}

	svc := new(service)
	svc.typ = reflect.TypeOf(rcvr)
	svc.rcvr = reflect.ValueOf(rcvr)

	sname := reflect.Indirect(svc.rcvr).Type().Name()
	if useName {
		sname = name
	}
	if sname == "" {
		return fmt.Errorf("no service name for type %s", svc.typ.String())
	}
	if !isExported(sname) && !useName {
		return fmt.Errorf("%s is not exported", sname)
	}

	if _, present := s.serviceMap[sname]; present {
		return fmt.Errorf("%s already registered", sname)
	}

	svc.name = sname
	svc.callbacks = suitableCallbacks(svc.typ)

	if len(svc.callbacks) == 0 {
		return fmt.Errorf("Service doesn't have any suitable methods to expose")
	}

	s.serviceMap[svc.name] = svc

	return nil
}

func (s *Server) ServeCodec(codec ServerCodec) {
	defer codec.Close()

	for {
		reqs, batch, reqid, err := s.readRequest(codec)
		if err != nil {
			glog.V(logger.Info).Infof("%v\n", err)
			codec.Write(codec.CreateErrorResponse(reqid, err))
			break
		}

		if batch {
			go s.execBatch(codec, reqs)
		} else {
			go s.exec(codec, reqs[0])
		}
	}
}

func (s *Server) execBatch(codec ServerCodec, requests []*serverRequest) {
	responses := make([]interface{}, len(requests))

	for i, req := range requests {
		var reply []reflect.Value

		if len(req.args) != len(req.callb.argTypes) {
			rpcErr := &invalidParamsError{fmt.Sprintf("%s.%s expects %d parameters, got %d",
				req.svcname, req.callb.method.Name, len(req.callb.argTypes), len(req.args))}
			responses[i] = codec.CreateErrorResponse(req.id, rpcErr)
			continue
		}

		arguments := []reflect.Value{req.rcvr}
		if len(req.args) > 0 {
			arguments = append(arguments, req.args...)
		}

		reply = req.callb.method.Func.Call(arguments)

		if len(reply) == 0 {
			responses[i] = codec.CreateResponse(req.id, nil)
			continue
		}

		if req.callb.errPos >= 0 {
			if !reply[req.callb.errPos].IsNil() {
				if e, ok := reply[req.callb.errPos].Interface().(error); ok {
					rpcErr := &callbackError{e.Error()}
					responses[i] = codec.CreateErrorResponse(req.id, rpcErr)
					continue
				}
			}
		}

		responses[i] = codec.CreateResponse(req.id, reply[0].Interface())
	}

	if err := codec.Write(responses); err != nil {
		glog.V(logger.Error).Infof("%v\n", err)
		codec.Close()
	}
}

func (s *Server) exec(codec ServerCodec, req *serverRequest) {
	var reply []reflect.Value

	if len(req.args) != len(req.callb.argTypes) {
		rpcErr := &invalidParamsError{fmt.Sprintf("%s.%s expects %d parameters, got %d",
			req.svcname, req.callb.method.Name, len(req.callb.argTypes), len(req.args))}
		res := codec.CreateErrorResponse(req.id, rpcErr)
		if err := codec.Write(res); err != nil {
			codec.Close()
		}
		return
	}

	arguments := []reflect.Value{req.rcvr}
	if len(req.args) > 0 {
		arguments = append(arguments, req.args...)
	}

	reply = req.callb.method.Func.Call(arguments)

	if len(reply) == 0 {
		res := codec.CreateResponse(req.id, nil)
		if err := codec.Write(res); err != nil {
			glog.V(logger.Error).Infof("%v\n", err)
			codec.Close()
		}
		return
	}

	if req.callb.errPos >= 0 { // test if method returned an error
		if !reply[req.callb.errPos].IsNil() {
			if e, ok := reply[req.callb.errPos].Interface().(error); ok {
				glog.V(logger.Debug).Infof("%v\n", e)
				rpcErr := &callbackError{e.Error()}
				res := codec.CreateErrorResponse(req.id, rpcErr)
				if err := codec.Write(res); err != nil {
					glog.V(logger.Error).Infof("%v\n", err)
					codec.Close()
				}
				return
			}
		}
	}

	res := codec.CreateResponse(req.id, reply[0].Interface())
	if err := codec.Write(res); err != nil {
		glog.V(logger.Error).Infof("%v\n", err)
		codec.Close()
	}
}

func (s *Server) readRequest(codec ServerCodec) ([]*serverRequest, bool, int64, RPCError) {
	reqs, batch, err := codec.ReadRequestHeaders()
	if err != nil {
		return nil, batch, 0, err
	}

	requests := make([]*serverRequest, len(reqs))

	// verify requests
	for i, r := range reqs {
		var ok bool
		var svc *service
		var callb *callback

		if svc, ok = s.serviceMap[r.service]; !ok {
			return nil, batch, r.id, &unknownServiceError{r.service, r.method}
		}

		if callb, ok = svc.callbacks[r.method]; !ok {
			return nil, batch, r.id, &unknownServiceError{r.service, r.method}
		}

		req := &serverRequest{id: r.id, svcname: svc.name, rcvr: svc.rcvr, callb: callb}
		if r.params != nil && len(callb.argTypes) > 0 {
			if args, err := codec.ParseRequestArgument(callb.argTypes, r.params); err == nil {
				req.args = args
			} else {
				return nil, false, r.id, err
			}
		}

		requests[i] = req
	}

	return requests, batch, 0, err
}
