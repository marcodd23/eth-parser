package parser

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"
)

// initialLookBackBlocksCount specifies the number of blocks to check backwards from the current block when the app starts for the first time
const initialLookBackBlocksCount = 10

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
	fetchPeriod        int
	client             JsonRpcClient
	notify             NotificationFunc
	mu                 sync.Mutex
	wg                 sync.WaitGroup
	cancel             context.CancelFunc
}

// NewEthParser creates a new EthParser instance with initial settings and begins background tasks
// necessary for its operation. It takes a context (ctx) for handling cancellation of background operations,
// a storage interface to interact with the storage layer, and an fetchPeriod which defines the frequency
// of updates in seconds. The function initializes an EthParser with a map to manage subscriptions, the provided
// storage, and initialize the lastProcessedBlock and currentBlock.
// It returns a pointer to the newly created EthParser instance.
//
// Parameters:
//   - ctx: Parent context to which a new cancellable context is derived for background task management.
//     It allows the background tasks to be stopped externally.
//   - storage: Storage interface that the parser uses to interact with the underlying storage mechanism.
//   - fetchPeriod: The interval in seconds at which the parser updates its data from the blockchain.
//   - client: A function type for sending JSON-RPC requests
//   - notify: a function to send custom notifications
//
// Returns:
//   - *EthParser: A pointer to the newly created EthParser instance.
//
// NewEthParser creates a new EthParser instance
func NewEthParser(
	cancellableCtx context.Context,
	storage Storage,
	fetchPeriod int,
	client JsonRpcClient,
	notify NotificationFunc) *EthParser {
	parser := &EthParser{
		subscriptions:      make(map[string]bool),
		storage:            storage,
		lastProcessedBlock: 0,
		fetchPeriod:        fetchPeriod,
		client:             client,
		notify:             notify,
	}

	parser.initializeCurrentBlock()

	// Create a new Cancellable Context and set it in the parser the cancel() function
	cancellableCtx, cancel := context.WithCancel(cancellableCtx)
	parser.cancel = cancel

	// Start the background tasks under the cancellableCtx
	parser.setupBackgroundUpdateTasks(cancellableCtx)

	return parser
}

func (p *EthParser) setupBackgroundUpdateTasks(cancelCtx context.Context) {
	p.wg.Add(2)

	// updates the current block number periodically
	go func() {
		defer p.wg.Done()
		ticker := time.NewTicker(time.Second * time.Duration(p.fetchPeriod))
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				log.Println("Updating current block")
				p.updateCurrentBlock()
			case <-cancelCtx.Done():
				log.Println("Stopping runUpdateCurrentBlock")
				return
			}
		}
	}()

	// fetches transactions for subscribed addresses periodically
	go func() {
		defer p.wg.Done()
		ticker := time.NewTicker(time.Second * time.Duration(p.fetchPeriod))
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				log.Println("Fetching new transactions")
				p.fetchTransactions()
			case <-cancelCtx.Done():
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
	return p.storage.GetTransactions(address)
}

// initializeCurrentBlock initialize the current block and last processed block
func (p *EthParser) initializeCurrentBlock() {
	if p.lastProcessedBlock == 0 {
		p.updateCurrentBlock()
		p.mu.Lock()
		p.lastProcessedBlock = p.currentBlock - initialLookBackBlocksCount

		// Ensure lastProcessedBlock is not negative
		if p.lastProcessedBlock < 0 {
			p.lastProcessedBlock = 0
		}

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

	resp, err := p.client.SendRequest(req)
	if err != nil {
		log.Println("Error fetching block number:", err)
		return
	}

	blockNumberHex := resp.Result.(string) // ex. 0x4b7
	blockNumberDecimal, err := convertHexNumberToDecimal(blockNumberHex)
	if err != nil {
		log.Println("Error parsing block number:", err)
		return
	}

	p.mu.Lock()
	p.currentBlock = blockNumberDecimal
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

		blockNumberDecimal, err := convertHexNumberToDecimal(block.Number)
		if err != nil {
			log.Println("Error parsing block number:", err)
			continue
		}

		transactionsForAddresses := make(map[string][]Transaction)

		for _, tx := range block.Transactions {
			if subscribedAddresses[tx.From] || subscribedAddresses[tx.To] {
				tx.BlockNumberDecimal = blockNumberDecimal
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
			err := p.storage.SaveTransactions(address, transactions)
			if err != nil {
				log.Printf("error saving transaction for addres %s", address)
			}
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

	resp, err := p.client.SendRequest(req)
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

func convertHexNumberToDecimal(hexNumber string) (int, error) {
	blockNumber, err := strconv.ParseInt(hexNumber[2:], 16, 64)
	if err != nil {
		log.Println("Error parsing hex number:", err)
		return -1, err
	}

	return int(blockNumber), nil
}
