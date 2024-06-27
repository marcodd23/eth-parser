package parser_test

import (
	"encoding/json"
	"eth-parser/internal/parser"
	"fmt"
	"strconv"
	"sync"
)

// ============================================
// Mock Storage
// ============================================

// MockStorage implements the Storage interface for testing purposes
type MockStorage struct {
	data map[string][]parser.Transaction
	mu   sync.Mutex
}

// NewMockStorage creates a new instance of MockStorage
func NewMockStorage() *MockStorage {
	return &MockStorage{
		data: make(map[string][]parser.Transaction),
	}
}

// SaveTransactions saves transactions to the mock storage
func (m *MockStorage) SaveTransactions(address string, transactions []parser.Transaction) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[address] = append(m.data[address], transactions...)
}

// GetTransactions returns transactions from the mock storage
func (m *MockStorage) GetTransactions(address string) []parser.Transaction {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.data[address]
}

// MockBlockchain simulates blockchain data for testing
type MockBlockchain struct {
	Blocks map[int]parser.Block
	mu     sync.Mutex
}

// ============================================
// Mock a Blockchain
// ============================================

// NewMockBlockchain creates a new instance of MockBlockchain
func NewMockBlockchain() *MockBlockchain {
	return &MockBlockchain{
		Blocks: make(map[int]parser.Block),
	}
}

// AddBlock adds a block to the mock data
func (m *MockBlockchain) AddBlock(blockNumber int, block parser.Block) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Blocks[blockNumber] = block
}

// GetBlockByNumber simulates fetching a block by its number
func (m *MockBlockchain) GetBlockByNumber(number int) (parser.Block, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	block, exists := m.Blocks[number]
	if !exists {
		return parser.Block{}, fmt.Errorf("block number %d not found", number)
	}
	return block, nil
}

// ============================================
// MOCK JSONRPC Client
// ============================================

type MockClient struct {
	*MockBlockchain
}

func NewMockClient(blockchain *MockBlockchain) *MockClient {
	return &MockClient{
		blockchain,
	}
}

// MockJSONRPCRequest simulates sending a JSON-RPC request and returns the mocked response
func (m *MockClient) SendRequest(req parser.JSONRPCRequest) (parser.JSONRPCResponse, error) {
	if req.Method == "eth_blockNumber" {
		latestBlock := len(m.Blocks)
		return parser.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  fmt.Sprintf("0x%x", latestBlock),
		}, nil
	}

	if req.Method == "eth_getBlockByNumber" {
		blockNumberHex := req.Params[0].(string)
		blockNumber, err := strconv.ParseInt(blockNumberHex[2:], 16, 64)
		if err != nil {
			return parser.JSONRPCResponse{}, err
		}
		block, err := m.GetBlockByNumber(int(blockNumber))
		if err != nil {
			return parser.JSONRPCResponse{}, err
		}
		resultBytes, err := json.Marshal(block)
		if err != nil {
			return parser.JSONRPCResponse{}, err
		}
		var result interface{}
		if err := json.Unmarshal(resultBytes, &result); err != nil {
			return parser.JSONRPCResponse{}, err
		}
		return parser.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  result,
		}, nil
	}

	return parser.JSONRPCResponse{}, fmt.Errorf("unsupported method: %s", req.Method)
}
