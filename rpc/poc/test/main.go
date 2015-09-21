package main

import (
	"bufio"
	"fmt"
	"net"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"math/big"
)

type Error struct {
	Code *big.Int
	Msg  string
}

func getResponse(conn *bufio.ReadWriter, res interface{}) error {
	marker, err := conn.ReadByte()
	if err != nil {
		return err
	}

	if marker == 0x2b { // call successful with response
		return rlp.Decode(conn, res)
	}

	if marker == 0x2d { // call successful with no response
		return nil
	}

	if marker == 0x21 { // received error
		fmt.Println("asdf")
		err := rlp.Decode(conn, res)
		fmt.Println("322332")
		return err
	}

	return fmt.Errorf("Protocol error, indication byte %x %c", res, res)
}

func getBlockRLP(blockNr uint64, conn *bufio.ReadWriter) {
	params, err := rlp.EncodeToBytes(blockNr)
	if err != nil {
		fmt.Printf("RLP encode failed: %v\n", err)
	}

	req := []byte("chain.GetBlockByNumber RLP ")
	req = append(req, params...)

	_, err = conn.Write(req)
	if err != nil {
		fmt.Printf("write err: %v\n", err)
		return
	}
	conn.Flush()

	var block types.Block
	err = getResponse(conn, &block)
	if err != nil {
		fmt.Printf("err response: %v\n", err)
	} else {
		fmt.Printf("GetBlock(%d).Number() = %d %x\n", blockNr, block.Number(), block.Hash())
	}
}

func NoReturnValue(conn *bufio.ReadWriter) {
	var args [2]interface{}
	args[0] = "cat"
	args[1] = "dog"
	params, err := rlp.EncodeToBytes(args)
	if err != nil {
		fmt.Printf("foo encode err: %v\n", err)
		return
	}

	req := []byte("chain.NoReturnValue RLP ")
	req = append(req, params...)

	conn.Write(req)
	conn.Flush()

	err = getResponse(conn, nil)
	if err != nil {
		panic(err)
	}
}

func ErrorReturn(conn *bufio.ReadWriter) {
	var args [2]interface{}
	args[0] = "cat"
	args[1] = "dog"
	params, err := rlp.EncodeToBytes(args)
	if err != nil {
		fmt.Printf("ErrorReturn encode err: %v\n", err)
		return
	}

	req := []byte("chain.ErrorReturn RLP ")
	req = append(req, params...)

	conn.Write(req)
	conn.Flush()

	var response Error
	if err := getResponse(conn, &response); err != nil {
		panic(err)
	}

	fmt.Printf("error response: [%d] [%s]\n", response.Code, response.Msg)
}

func main() {
	c, err := net.DialUnix("unix", nil, &net.UnixAddr{"/tmp/poc.sock", "unix"})
	if err != nil {
		panic(err)
	}
	defer c.Close()

	rw := bufio.NewReadWriter(bufio.NewReader(c), bufio.NewWriter(c))

	for i := uint64(0); i < 3; i++ {
		getBlockRLP(i, rw)
		NoReturnValue(rw)
		ErrorReturn(rw)
	}
}
