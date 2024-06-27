package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// EthereumNodeURL Ethereum node URL for JSON-RPC requests
const (
	EthereumNodeURL = "https://cloudflare-eth.com"
)

type JsonRpcClient interface {
	SendRequest(req JSONRPCRequest) (JSONRPCResponse, error)
}

// DefaultClient is the default implementation JsonRpcClient
type DefaultClient struct {
}

// NewJsonRpcClient is the default constructor for JsonRpcClient
func NewJsonRpcClient() *DefaultClient {
	return &DefaultClient{}
}

// SendRequest is the default implementation for sending JSON-RPC requests
func (c *DefaultClient) SendRequest(req JSONRPCRequest) (JSONRPCResponse, error) {
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return JSONRPCResponse{}, err
	}

	resp, err := http.Post(EthereumNodeURL, "application/json", bytes.NewBuffer(reqBytes))
	if err != nil {
		return JSONRPCResponse{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return JSONRPCResponse{}, err
	}

	var rpcResp JSONRPCResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return JSONRPCResponse{}, err
	}

	if rpcResp.Error != nil {
		return rpcResp, fmt.Errorf("JSON-RPC error: %v", rpcResp.Error)
	}

	return rpcResp, nil
}
