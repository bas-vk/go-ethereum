package comms

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rpc/shared"
	"golang.org/x/net/websocket"
	"github.com/rs/cors"
	"strings"
)

var (
	wsHttpListener         *stoppableTCPListener
)

// JSON-RPC websocket handler
func WSHandler(ws *websocket.Conn, api shared.EthereumApi) {
	var err error
	var requests []*shared.Request

	for {
		var incoming json.RawMessage
		if err = websocket.JSON.Receive(ws, &incoming); err != nil {
			glog.V(logger.Error).Infof("Error receiving WS request %v", err)
			ws.WriteClose(http.StatusBadRequest)
			return
		}

		isBatch := len(incoming) > 0 && incoming[0] == '['
		if isBatch {
			requests = make([]*shared.Request, 0)
			err = json.Unmarshal(incoming, &requests)
		} else {
			requests = make([]*shared.Request, 1)
			var singleRequest shared.Request
			if err = json.Unmarshal(incoming, &singleRequest); err == nil {
				requests[0] = &singleRequest
			}
		}

		if err != nil {
			glog.V(logger.Error).Info("Error parsing WS-JSON request %v", err)
			ws.WriteClose(http.StatusBadRequest)
			return
		}

		glog.V(logger.Debug).Infof("Received %d WS request(s)", len(requests))
		responses := make([]*interface{}, len(requests))
		i := 0
		for _, request := range requests {
			response, apiErr := api.Execute(request)
			if request.Id != nil {
				rpcResponse := shared.NewRpcResponse(request.Id, request.Jsonrpc, response, apiErr)
				responses[i] = rpcResponse
				i += 1
			} else {
				glog.V(logger.Debug).Info("Send no WS response for request (req.Id == nil)")
			}
		}

		if isBatch {
			websocket.JSON.Send(ws, responses)
		} else if len(responses) == 1 {
			websocket.JSON.Send(ws, responses[0])
		}
	}
}

// starts the websocket RPC interface
// this interface currently supports only JSON-RPC
func StartWS(cfg HttpConfig, api shared.EthereumApi) error {
	if wsHttpListener != nil {
		if fmt.Sprintf("%s:%d", cfg.ListenAddress, cfg.ListenPort) != wsHttpListener.Addr().String() {
			return fmt.Errorf("Websocket RPC service already running on %s ", wsHttpListener.Addr().String())
		}
		return nil // RPC service already running on given host/port
	}

	l, err := newStoppableTCPListener(fmt.Sprintf("%s:%d", cfg.ListenAddress, cfg.ListenPort))
	if err != nil {
		glog.V(logger.Error).Infof("Can't listen on %s:%d: %v", cfg.ListenAddress, cfg.ListenPort, err)
		return err
	}
	wsHttpListener = l

	var handler http.Handler
	if len(cfg.CorsDomain) > 0 {
		var opts cors.Options
		opts.AllowedMethods = []string{"POST"}
		opts.AllowedOrigins = strings.Split(cfg.CorsDomain, " ")

		c := cors.New(opts)
		handler = newStoppableHandler(c.Handler(websocket.Handler(func(ws *websocket.Conn) {
			WSHandler(ws, api)
		})), l.stop)
	} else {
		handler = newStoppableHandler(websocket.Handler(func(ws *websocket.Conn) {
			WSHandler(ws, api)
		}), l.stop)
	}

	glog.V(logger.Info).Infof("Start WS JSON-RPC on %s:%d", cfg.ListenAddress, cfg.ListenPort)
	go http.Serve(l, handler)

	return nil
	/*

	http.Handle("/", websocket.Handler(func(ws *websocket.Conn) {
		WSHandler(ws, api)
	}))

	glog.V(logger.Info).Infof("Start WS JSON-RPC on %s:%d", cfg.ListenAddress, cfg.ListenPort)
	if err := http.ListenAndServe(fmt.Sprintf("%s:%d", cfg.ListenAddress, cfg.ListenPort), nil); err != nil {
		return err
	}

	return nil
	*/
}

func StopWS() {
	if wsHttpListener != nil {
		wsHttpListener.Stop()
		wsHttpListener = nil
		glog.V(logger.Info).Infoln("WS RPC interface stopped")
	}
}