package parser

import (
	"context"
	"testing"
	"time"
)

// TestSubscribe tests the subscription functionality
func TestSubscribe(t *testing.T) {
	storage := NewMemoryStorage()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	parser := NewEthParser(ctx, storage, 10)
	success := parser.Subscribe("0xSomeAddress")
	if !success {
		t.Errorf("Expected subscription to be successful")
	}
	success = parser.Subscribe("0xSomeAddress")
	if success {
		t.Errorf("Expected subscription to fail for already subscribed address")
	}
	parser.WaitForShutdown()
}

// TestGetTransactions tests fetching transactions for a subscribed address
func TestGetTransactions(t *testing.T) {
	storage := NewMemoryStorage()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	parser := NewEthParser(ctx, storage, 10)
	parser.Subscribe("0xSomeAddress")
	transactions := parser.GetTransactions("0xSomeAddress")
	if len(transactions) != 0 {
		t.Errorf("Expected no transactions for new address")
	}
	parser.WaitForShutdown()
}

// TestUpdateCurrentBlock tests the updateCurrentBlock functionality
func TestUpdateCurrentBlock(t *testing.T) {
	storage := NewMemoryStorage()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	parser := NewEthParser(ctx, storage, 10)
	parser.updateCurrentBlock()
	currentBlock := parser.GetCurrentBlock()
	if currentBlock <= 0 {
		t.Errorf("Expected current block to be greater than 0")
	}
	parser.WaitForShutdown()
}

// TestFetchTransactions tests the fetchTransactions functionality
func TestFetchTransactions(t *testing.T) {
	storage := NewMemoryStorage()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	parser := NewEthParser(ctx, storage, 10)
	parser.Subscribe("0xSomeAddress")
	parser.updateCurrentBlock()
	parser.fetchTransactions()
	transactions := parser.GetTransactions("0xSomeAddress")
	if len(transactions) != 0 {
		t.Errorf("Expected no transactions for new address after fetching")
	}
	// Ensure the background tasks are stopped gracefully
	cancel()
	time.Sleep(time.Second)
	parser.WaitForShutdown()
}
