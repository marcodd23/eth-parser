package parser

import "log"

// NotificationFunc defines a function to send notifications
type NotificationFunc func(address string, transactions []Transaction)

// NotifyOnConsole simulates sending a notification about new transactions
func NotifyOnConsole(address string, transactions []Transaction) {
	// Simulate sending a notification (e.g., print to console)
	for _, tx := range transactions {
		log.Printf("Notification - Address: %s, Transaction: %s, From: %s, To: %s, Value: %s, Block: %s\n",
			address, tx.Hash, tx.From, tx.To, tx.Value, tx.BlockNumber)
	}
}
