package parser_test

import (
	"context"
	"eth-parser/internal/parser"
	"sync"
	"testing"
	"time"
)

func TestEthParser(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockBlockchain := NewMockBlockchain()
	storage := NewMockStorage()

	// Simulate blocks with transactions
	block1 := parser.Block{
		Number: "0x1",
		Transactions: []parser.Transaction{
			{Hash: "0xabc", From: "0x1", To: "0x2", Value: "100"},
		},
	}
	block2 := parser.Block{
		Number: "0x2",
		Transactions: []parser.Transaction{
			{Hash: "0xdef", From: "0x2", To: "0x3", Value: "200"},
		},
	}

	mockBlockchain.AddBlock(1, block1)
	mockBlockchain.AddBlock(2, block2)

	notifications := make(map[string][]parser.Transaction)
	var mu sync.Mutex

	notifyFunc := func(address string, transactions []parser.Transaction) {
		mu.Lock()
		defer mu.Unlock()
		notifications[address] = append(notifications[address], transactions...)
	}

	ethParser := parser.NewEthParser(ctx, storage, 1, NewMockClient(mockBlockchain), notifyFunc)

	// Subscribe to addresses
	if !ethParser.Subscribe("0x1") {
		t.Fatal("Failed to subscribe to address 0x1")
	}
	if !ethParser.Subscribe("0x2") {
		t.Fatal("Failed to subscribe to address 0x2")
	}

	// Wait for background tasks to process the mock data
	time.Sleep(2 * time.Second)

	// Check transactions for subscribed addresses
	transactions := ethParser.GetTransactions("0x1")
	if len(transactions) != 1 || transactions[0].Hash != "0xabc" {
		t.Fatalf("Unexpected transactions for address 0x1: %v", transactions)
	}

	transactions = ethParser.GetTransactions("0x2")
	if len(transactions) != 2 || transactions[1].Hash != "0xdef" {
		t.Fatalf("Unexpected transactions for address 0x2: %v", transactions)
	}

	// Verify notifications
	mu.Lock()
	defer mu.Unlock()
	if len(notifications["0x1"]) != 1 || notifications["0x1"][0].Hash != "0xabc" {
		t.Fatalf("Unexpected notifications for address 0x1: %v", notifications["0x1"])
	}
	if len(notifications["0x2"]) != 2 || notifications["0x2"][1].Hash != "0xdef" {
		t.Fatalf("Unexpected notifications for address 0x2: %v", notifications["0x2"])
	}
}
