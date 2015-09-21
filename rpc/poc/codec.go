package poc

import (
	"bufio"
	"io"

	"github.com/ethereum/go-ethereum/rlp"
"math/big"
	"fmt"
)

const (
	sep = ' ' // separator
	successWithNoResponse = '-'
	successWithResponse = '+'
	errorResponse = '!'
)

type rpcCodec struct {
	c io.ReadWriteCloser // underlying stream
	r *bufio.Reader
	w *bufio.Writer

	protocol string // serialization protocol for next request
}

// NewRPCCodec returns a new codec which supports RPC stream on c
func NewRPCCodec(c io.ReadWriteCloser) *rpcCodec {
	return &rpcCodec{
		c: c,
		r: bufio.NewReader(c),
		w: bufio.NewWriter(c),
	}
}

func (c *rpcCodec) ReadRequestHeader(r *Request) (err error) {
	if r.ServiceMethod, err = c.readWord(); err != nil {
		return err
	}

	if c.protocol, err = c.readWord(); err != nil {
		return err
	}

	return nil
}

// readWord read a string till the next separator
// it will strip the separator from the stream and returns the word without the separator
func (c *rpcCodec) readWord() (word string, err error) {
	word, err = c.r.ReadString(sep)
	if err == nil && word[len(word)-1] == ' ' {
		word = word[:len(word)-1]
	}
	return
}

func (c *rpcCodec) ReadRequestBody(x interface{}) error {
	return rlp.Decode(c.r, x)
}

func (c *rpcCodec) WriteResponse(r *Response, v interface{}) error {
	defer c.w.Flush()

	if v == nil { // no response value
		return c.w.WriteByte(successWithNoResponse)
	}

	if err := c.w.WriteByte(successWithResponse); err != nil {
		return err
	}

	return rlp.Encode(c.w, v)
}

func (c *rpcCodec) WriteErrorResponse(r *Response, code int, err error) error {
	defer c.w.Flush()

	if err := c.w.WriteByte(errorResponse); err != nil {
		return err
	}

	response := struct {
		Code *big.Int
		Message string
	}{
		big.NewInt(int64(code)),
		err.Error(),
	}

	err = rlp.Encode(c.w, &response)
	fmt.Printf("err writing error esponse: %v\n", err)
	return err
}

func (c *rpcCodec) Close() error {
	c.w.Flush()
	return c.c.Close()
}
