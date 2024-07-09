package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"eth-parser/internal/parser"
)

func main() {
	// Initialize the memory storage
	storage := parser.NewMemoryStorage()

	// Create a context that will be canceled on shutdown
	ctx := context.Background()

	// Initialize the Ethereum parser with the memory storage and JsonRpc Client
	ethParser := parser.NewEthParser(ctx, storage, 10, parser.NewJsonRpcClient(), parser.NotifyOnConsole)

	//Setup Routes
	SetupRoutes(ethParser)

	// Start the HTTP server in a goroutine
	server := &http.Server{Addr: ":8080"}
	go func() {
		log.Println("Starting the HTTP server")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not listen on :8080: %v\n", err)
		}

		log.Println("HTTP server stopped")
	}()

	// Set up a channel to listen for interrupt or terminate signals from the OS
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Wait for a signal to gracefully shut down
	<-stop
	log.Println("Received shutdown signal")

	// Shut down the server gracefully
	log.Println("Shutting down the server...")
	if err := server.Close(); err != nil {
		log.Fatalf("Server Close: %v", err)
	}

	// Wait for parser goroutines to terminate.
	ethParser.WaitForShutdown()
	log.Println("Application gracefully stopped")
}

func SetupRoutes(ethParser parser.Parser) {
	// Endpoint to get the current block number
	http.HandleFunc("/current_block", func(w http.ResponseWriter, r *http.Request) {
		block := ethParser.GetCurrentBlock()
		json.NewEncoder(w).Encode(map[string]int{"current_block": block})
	})

	// Endpoint to subscribe to an Ethereum address
	http.HandleFunc("/subscribe", func(w http.ResponseWriter, r *http.Request) {
		var request map[string]string
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}
		address, ok := request["address"]
		if !ok {
			http.Error(w, "Address field is required", http.StatusBadRequest)
			return
		}
		success := ethParser.Subscribe(address)
		json.NewEncoder(w).Encode(map[string]bool{"success": success})
	})

	// Endpoint to get transactions for a subscribed address
	http.HandleFunc("/transactions", func(w http.ResponseWriter, r *http.Request) {
		var request map[string]string
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}
		address, ok := request["address"]
		if !ok {
			http.Error(w, "Address field is required", http.StatusBadRequest)
			return
		}
		transactions := ethParser.GetTransactions(address)
		if len(transactions) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		json.NewEncoder(w).Encode(transactions)
	})
}
