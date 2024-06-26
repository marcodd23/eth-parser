package parser

// JSONRPCRequest represents the structure of a JSON-RPC request
type JSONRPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

// JSONRPCResponse represents the structure of a JSON-RPC response
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Result  interface{} `json:"result"`
	Error   interface{} `json:"error"`
}

// Transaction represents a simplified Ethereum transaction
type Transaction struct {
	Hash           string `json:"hash"`
	From           string `json:"from"`
	To             string `json:"to"`
	Value          string `json:"value"`
	BlockNumber    string `json:"blockNumber"`
	BlockNumberInt int    `json:"-"`
}

// Block represents a simplified Ethereum block
type Block struct {
	Number       string        `json:"number"`
	Transactions []Transaction `json:"transactions"`
}
