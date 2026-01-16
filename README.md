# In-Memory Message Queue

A lightweight, high-performance in-memory message queue written in Go. It supports multi-tenancy, API key authentication, message visibility timeouts, and dead-letter queues.

## 🚀 Features

- **Multi-Tenancy**: Isolated queues for different tenants using secure API keys.
- **At-Least-Once Delivery**: Messages are kept in-flight until acknowledged.
- **Visibility Timeouts**: Automatically requeues messages if they are not acknowledged within a specified timeframe.
- **Dead Letter Queue (DLQ)**: Messages that exceed the maximum delivery attempts are moved to a DLQ for later inspection.
- **Blocking Claims (Long Polling)**: Consumers can wait for messages if the queue is empty.
- **Secure Authentication**: API keys are hashed with SHA256 and verified using constant-time comparison.
- **Real-time Statistics**: Monitor pending, in-flight, and dead-letter message counts.
- **Graceful Shutdown**: Handles OS signals to ensure a clean exit.

## 🛠 Project Structure

- `internal/auth`: Manages API keys, hashing, and tenant association.
- `internal/broker`: Coordinates queue operations and handles authentication logic.
- `internal/queue`: Core in-memory data structures and message lifecycle management.
- `internal/server`: HTTP handlers and server configuration.
- `internal/utils`: JSON response helpers.

## ⚙️ How It Works

1.  **Publish**: A producer sends a message to a named queue.
2.  **Claim**: A consumer requests a message. The message moves to an "In-Flight" state and is assigned a **Visibility Deadline**.
3.  **Process & Ack**:
    - If the consumer successfully processes the message, it sends an **Ack**. The message is removed from the system.
    - If the consumer fails or takes too long, the **requeue goroutine** detects the expired deadline and moves the message back to "Pending".
4.  **Max Retries**: If a message fails (is requeued) more than the allowed `maxDelivery` limit, it is moved to the **Dead Letter Queue**.

## 🚀 Getting Started

### Prerequisites

- Go 1.21 or later

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/sanke08/in-mem-message-queue.git
   cd in-mem-message-queue
   ```

2. Install dependencies (if any):
   ```bash
   go mod tidy
   ```

### Running the Server

```bash
go run main.go
```
The server will start on `http://localhost:8080`.

## 📖 API Usage

All requests (except `/create_key`) require an `Authorization` header:
`Authorization: ApiKey <your_api_key>`

### 1. Create an API Key
Used to create a tenant and get a secret key.
- **Endpoint**: `GET /create_key?tenant=<tenant_id>`
- **Response**:
  ```json
  {
    "api_key": "kid.secret",
    "tenant": "my-tenant"
  }
  ```

### 2. Publish a Message
- **Endpoint**: `POST /publish?queue=<queue_name>`
- **Header**: `Authorization: ApiKey <key>`
- **Body**: Plain text or JSON payload.
- **Response**:
  ```json
  {
    "message_id": "unique-id"
  }
  ```

### 3. Claim a Message
Requests a message from the queue. Blocks for 5 seconds if no messages are available.
- **Endpoint**: `GET /claim?queue=<queue_name>`
- **Header**: `Authorization: ApiKey <key>`
- **Response**:
  ```json
  {
    "message_id": "unique-id",
    "payload": "message content"
  }
  ```

### 4. Acknowledge a Message
Confirms the message was processed and removes it from the queue.
- **Endpoint**: `POST /ack?queue=<queue_name>&message_id=<id>`
- **Header**: `Authorization: ApiKey <key>`
- **Response**:
  ```json
  {
    "status": "ok"
  }
  ```

### 5. Get Queue Statistics
- **Endpoint**: `GET /stats?queue=<queue_name>`
- **Header**: `Authorization: ApiKey <key>`
- **Response**:
  ```json
  {
    "Pending": 5,
    "InFlight": 2,
    "DeadLetter": 0
  }
  ```

## 🛡 Security

- Keys are stored as hashes (SHA256).
- Verification uses `hmac.Equal` to prevent timing attacks.
- Tenant IDs are prepended to queue names to ensure data isolation.

## 📝 Configuration

Configuration is currently handled in `main.go`. You can modify:
- **Lease Duration**: How long a message stays "in-flight" before being requeued (default: `5s`).
- **Max Delivery**: How many times a message can be retries before moving to DLQ (default: `3`).
- **Port**: Default is `:8080`.
