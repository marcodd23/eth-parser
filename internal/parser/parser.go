package parser

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// Ethereum node URL for JSON-RPC requests
const (
	ethereumNodeURL = "https://cloudflare-eth.com"
)

// Parser defines the interface for the Ethereum parser
type Parser interface {
	GetCurrentBlock() int
	Subscribe(address string) bool
	GetTransactions(address string) []Transaction
	WaitForShutdown()
}

// EthParser implements the Parser interface
type EthParser struct {
	currentBlock       int
	lastProcessedBlock int
	subscriptions      map[string]bool
	storage            Storage
	updateRatePeriod   int
	mu                 sync.Mutex
	wg                 sync.WaitGroup
	cancel             context.CancelFunc
}

// NewEthParser creates a new EthParser instance with initial settings and begins background tasks
// necessary for its operation. It takes a context (ctx) for handling cancellation of background operations,
// a storage interface to interact with the storage layer, and an updateRatePeriod which defines the frequency
// of updates in seconds. The function initializes an EthParser with a map to manage subscriptions, the provided
// storage, and sets the last processed block to zero.
// It returns a pointer to the newly created EthParser instance.
//
// Parameters:
//   - ctx: Parent context to which a new cancellable context is derived for background task management.
//     It allows the background tasks to be stopped externally.
//   - storage: Storage interface that the parser uses to interact with the underlying storage mechanism.
//   - updateRatePeriod: The interval in seconds at which the parser updates its data from the blockchain.
//
// Returns:
//   - *EthParser: A pointer to the newly created EthParser instance.
func NewEthParser(ctx context.Context, storage Storage, updateRatePeriod int) *EthParser {
	parser := &EthParser{
		subscriptions:      make(map[string]bool),
		storage:            storage,
		lastProcessedBlock: 0,
		updateRatePeriod:   updateRatePeriod,
	}

	parser.initializeCurrentBlock()

	ctx, cancel := context.WithCancel(ctx)
	parser.cancel = cancel

	// Start the background tasks
	parser.setupBackgroundUpdateTasks(ctx)

	return parser
}

func (p *EthParser) setupBackgroundUpdateTasks(ctx context.Context) {
	p.wg.Add(2)

	// runUpdateCurrentBlock updates the current block number periodically
	go func() {
		defer p.wg.Done()
		ticker := time.NewTicker(time.Second * time.Duration(p.updateRatePeriod))
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				log.Println("Updating current block")
				p.updateCurrentBlock()
			case <-ctx.Done():
				log.Println("Stopping runUpdateCurrentBlock")
				return
			}
		}
	}()

	// runFetchTransactions fetches transactions for subscribed addresses periodically
	go func() {
		defer p.wg.Done()
		ticker := time.NewTicker(time.Second * time.Duration(p.updateRatePeriod+5))
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				log.Println("Fetching new transactions")
				p.fetchTransactions()
			case <-ctx.Done():
				log.Println("Stopping runFetchTransactions")
				return
			}
		}
	}()
}

// WaitForShutdown waits for the background jobs to complete
func (p *EthParser) WaitForShutdown() {
	log.Println("Waiting for background jobs to complete...")
	p.cancel()
	p.wg.Wait()
	log.Println("Background jobs stopped")
}

// GetCurrentBlock returns the last parsed block number
func (p *EthParser) GetCurrentBlock() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.currentBlock
}

// Subscribe adds an address to the list of subscriptions
func (p *EthParser) Subscribe(address string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.subscriptions[address]; exists {
		return false
	}
	p.subscriptions[address] = true
	return true
}

// GetTransactions returns the list of transactions for a given address
func (p *EthParser) GetTransactions(address string) []Transaction {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.storage.GetTransactions(address)
}

// initializeCurrentBlock initialize the current block and last processed block
func (p *EthParser) initializeCurrentBlock() {
	if p.lastProcessedBlock == 0 {
		p.updateCurrentBlock()
		p.mu.Lock()
		p.lastProcessedBlock = p.currentBlock - 50 // Set to a recent block, 50 blocks back
		p.mu.Unlock()
	}
}

// updateCurrentBlock fetches and updates the current block number from the Ethereum blockchain
func (p *EthParser) updateCurrentBlock() {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "eth_blockNumber",
		Params:  []interface{}{},
		ID:      1,
	}

	resp, err := sendJSONRPCRequest(req)
	if err != nil {
		log.Println("Error fetching block number:", err)
		return
	}

	blockNumberHex := resp.Result.(string) // ex. 0x4b7
	blockNumber, err := strconv.ParseInt(blockNumberHex[2:], 16, 64)
	if err != nil {
		log.Println("Error parsing block number:", err)
		return
	}

	p.mu.Lock()
	p.currentBlock = int(blockNumber)
	p.mu.Unlock()
}

// fetchTransactions fetches transactions for all subscribed addresses
func (p *EthParser) fetchTransactions() {
	log.Println("Starting fetchTransactions")

	p.mu.Lock()
	subscribedAddresses := make(map[string]bool)
	for address := range p.subscriptions {
		subscribedAddresses[address] = true
	}
	startBlock := p.lastProcessedBlock + 1
	currentBlock := p.currentBlock
	p.mu.Unlock()

	log.Printf("Fetching transactions from block %d to %d\n", startBlock, currentBlock)

	for i := startBlock; i <= currentBlock; i++ {
		block, err := p.getBlockByNumber(i)
		if err != nil {
			log.Println("Error fetching block number:", i, err)
			continue
		}

		blockNumberInt, err := strconv.ParseInt(block.Number[2:], 16, 64)
		if err != nil {
			log.Println("Error parsing block number:", err)
			continue
		}

		transactionsForAddresses := make(map[string][]Transaction)

		for _, tx := range block.Transactions {
			if subscribedAddresses[tx.From] || subscribedAddresses[tx.To] {
				tx.BlockNumberInt = int(blockNumberInt)
				if subscribedAddresses[tx.From] {
					transactionsForAddresses[tx.From] = append(transactionsForAddresses[tx.From], tx)
				}
				if subscribedAddresses[tx.To] {
					transactionsForAddresses[tx.To] = append(transactionsForAddresses[tx.To], tx)
				}
			}
		}

		for address, transactions := range transactionsForAddresses {
			log.Printf("Found %d transactions for address %s in block %d\n", len(transactions), address, i)
			p.notify(address, transactions)
			p.storage.SaveTransactions(address, transactions)
		}
	}

	p.mu.Lock()
	p.lastProcessedBlock = currentBlock
	p.mu.Unlock()

	log.Println("Completed fetchTransactions")
}

// getBlockByNumber fetches a block by its number
func (p *EthParser) getBlockByNumber(number int) (Block, error) {
	numberHex := fmt.Sprintf("0x%x", number)
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "eth_getBlockByNumber",
		Params:  []interface{}{numberHex, true},
		ID:      1,
	}

	resp, err := sendJSONRPCRequest(req)
	if err != nil {
		return Block{}, err
	}

	resultMap, ok := resp.Result.(map[string]interface{})
	if !ok {
		return Block{}, fmt.Errorf("unexpected result format")
	}

	var block Block
	resultBytes, err := json.Marshal(resultMap)
	if err != nil {
		return Block{}, err
	}
	err = json.Unmarshal(resultBytes, &block)
	if err != nil {
		return Block{}, err
	}

	return block, nil
}

// sendJSONRPCRequest sends a JSON-RPC request to the Ethereum node and returns the response
func sendJSONRPCRequest(req JSONRPCRequest) (JSONRPCResponse, error) {
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return JSONRPCResponse{}, err
	}

	resp, err := http.Post(ethereumNodeURL, "application/json", bytes.NewBuffer(reqBytes))
	if err != nil {
		return JSONRPCResponse{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
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

// notify simulates sending a notification about new transactions
func (p *EthParser) notify(address string, transactions []Transaction) {
	// Simulate sending a notification (e.g., print to console)
	for _, tx := range transactions {
		log.Printf("Notification - Address: %s, Transaction: %s, From: %s, To: %s, Value: %s, Block: %s\n",
			address, tx.Hash, tx.From, tx.To, tx.Value, tx.BlockNumber)
	}

	// In a real implementation, you would send an HTTP request to a notification service here.
}
