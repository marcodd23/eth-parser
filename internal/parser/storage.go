package parser

import (
	"sync"
)

// Storage defines the interface for transaction storage
type Storage interface {
	SaveTransactions(address string, transactions []Transaction)
	GetTransactions(address string) []Transaction
}

// MemoryStorage implements the Storage interface using in-memory storage
type MemoryStorage struct {
	data map[string][]Transaction
	mu   sync.Mutex
}

// NewMemoryStorage creates a new instance of MemoryStorage
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		data: make(map[string][]Transaction),
	}
}

// SaveTransactions saves transactions for a given address
func (s *MemoryStorage) SaveTransactions(address string, transactions []Transaction) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[address] = append(s.data[address], transactions...)
}

// GetTransactions retrieves transactions for a given address
func (s *MemoryStorage) GetTransactions(address string) []Transaction {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data[address]
}
