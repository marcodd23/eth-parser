# EthParser

EthParser is a Go-based application designed to monitor Ethereum blockchain addresses for incoming and outgoing transactions. The application leverages the Ethereum JSON-RPC API to fetch the latest blocks and transactions and provides a REST API to subscribe to addresses and retrieve transactions.

## Features

- Subscribe to Ethereum addresses to monitor transactions.
- Fetch the latest Ethereum block number periodically.
- Fetch transactions for subscribed addresses periodically.
- Provide a REST API to manage subscriptions and retrieve transactions.
- Use in-memory storage for transaction data, easily extendable to support other storage mechanisms.
- Graceful shutdown handling with context and wait groups.

## Design

The application is designed with modularity and encapsulation in mind, using a clear separation of concerns:

- **cmd/**: Contains the main application entry point.
- **internal/parser/**: Contains the core parsing logic, background task management, and storage interface.
- **internal/parser/parser.go**: Implements the Ethereum parser with background task management.
- **internal/parser/storage.go**: Implements in-memory storage for transactions.
- **internal/parser/models.go**: Defines models for JSON-RPC requests and responses, as well as Ethereum transactions.
- **internal/parser/parser_test.go**: Contains tests for the parser functionalities.

## Folder Structure

```go
eth-parser/
├── cmd/
│   └── main.go
├── internal/
│   ├── parser/
│   │   ├── parser.go
│   │   ├── storage.go
│   │   ├── models.go
│   │   └── parser_test.go
└── go.mod
```



## Installation

1. Clone the repository:
    ```sh
    git clone https://github.com/your-username/eth-parser.git
    cd eth-parser
    ```

2. Initialize a new Go module (if not already done):
    ```sh
    go mod init eth-parser
    ```

3. Install dependencies:
    ```sh
    go mod tidy
    ```

## Usage

1. Start the application:
    ```sh
    go run cmd/main.go
    ```

2. Use the following endpoints to interact with the application:

   - **GET /current_block**: Get the current block number.
   - **POST /subscribe**: Subscribe to an Ethereum address. Example request body:
     ```json
     {
         "address": "0xYourEthereumAddress"
     }
     ```
   - **POST /transactions**: Get transactions for a subscribed address. Example request body:
     ```json
     {
         "address": "0xYourEthereumAddress"
     }
     ```

## Implementation Details

### `cmd/main.go`

The main entry point initializes the memory storage and the Ethereum parser, sets up HTTP endpoints, and starts the HTTP server. It handles graceful shutdown by using a context and a wait group.

### `internal/parser/parser.go`

- **EthParser**: Implements the `Parser` interface, managing subscriptions, fetching blocks, and transactions. It uses a separate wait group for background jobs and starts them in the constructor.
- **Background Jobs**:
   - `runUpdateCurrentBlock`: Periodically fetches the current Ethereum block number.
   - `runFetchTransactions`: Periodically fetches transactions for subscribed addresses.
- **Synchronization**: Uses mutexes to ensure thread safety when accessing shared resources.

### `internal/parser/storage.go`

Implements an in-memory storage mechanism for transactions. It provides methods to save and retrieve transactions, ensuring thread safety with mutexes.

### `internal/parser/models.go`

Defines models for JSON-RPC requests and responses, and Ethereum transactions, ensuring clear data structures for communication with the Ethereum node.

### `internal/parser/parser_test.go`

Contains tests for the parser functionalities:
- **Subscription Management**: Tests subscribing to and unsubscribing from addresses.
- **Transaction Fetching**: Tests fetching transactions for subscribed addresses.
- **Background Task Management**: Ensures background tasks are started and stopped gracefully.

## Extending the Storage Mechanism

To extend the application to support other storage mechanisms (e.g., a database), implement the `Storage` interface defined in `internal/parser/storage.go`. Replace the in-memory storage with your implementation in the `main` function.
