# GoDispatch: Configuration-Driven Real-Time WebSocket Engine


GoDispatch is a high-performance WebSocket engine built in Go that replaces custom real-time backend code with a simple, declarative YAML configuration. Define your event logic, security rules, and message routing in `config.yaml`, and let the engine handle the rest.

## Core Features

- **Declarative Logic:** Define your application's real-time behavior (events, actions, security) in a `config.yaml` file instead of writing Go code.
- **Secure by Design:** Built-in support for JWT-based authentication for sessions and events.
- **Stateful Modifiers:** Implement complex validation logic like rate limiting with ease.
- **Extensible:** Add new custom "actions" and "modifiers" in Go to extend the engine's core functionality.

## Quick Start in 5 Minutes

### Prerequisites

- [Go](https://go.dev/doc/install) (version 1.22 or later)


### 1. Clone the Repository

```bash
git clone https://github.com/a-essam23/go-dispatch.git
cd go-dispatch
```

### 2. Set Up Your Environment

GoDispatch uses a `.env` file for local development to manage secrets.

Copy the example environment file:

```bash
cp .env.example .env
```

The default `.env` file contains the secret key (`a-very-secret-key-from-env-file`) that matches the `config.yaml`.

### 3. Run the Server

You can run the server directly or using the Makefile.

**With `make` (Recommended):**

```bash
make run
```

**Without `make`:**

```bash
go run ./cmd/go-dispatch/main.go
```

You should see log output indicating the server has started on `:8080`.

```
{"time":"...","level":"INFO","msg":"Server starting","addr":":8080"}
```

### 4. Connect and Send Events

Now you can connect to the server with any WebSocket client. Here's how to do it with your browser's developer console.

**Step A: Generate a Session Token**

GoDispatch requires a valid JWT for the initial connection.
1. Go to [jwt.io](https://jwt.io/).
2. Change the algorithm to `HS256`.
3. Set the payload to include a `sub` (Subject/UserID) claim:
   ```json
   {
     "sub": "user-123",
     "name": "Test User"
   }
   ```
4. In the "Verify Signature" section, paste the secret from your `.env` file: `a-very-secret-key-from-env-file`.
5. Copy the entire encoded token (the long string in the left panel).

**Step B: Connect via Browser Console**

Open a blank page in your browser, open the developer console (F12), and paste this JavaScript code. Replace `<YOUR_SESSION_TOKEN>` with the token you just generated.

```javascript
const sessionToken = "<YOUR_SESSION_TOKEN>";
const ws = new WebSocket("ws://localhost:8080/ws", ["Authorization", `Bearer ${sessionToken}`]);

ws.onopen = () => {
    console.log("✅ WebSocket connection established!");

    // Example 1: Send a simple message to a room
    const message = {
        event: "send_message_to_room",
        target: "room-lobby",
        payload: {
            message: "Hello from the browser!"
        }
    };
    ws.send(JSON.stringify(message));
    console.log("Sent message:", message);

    // Example 2: Trigger a rate-limited event (try this multiple times quickly)
    // See config.yaml for the 'send_message_rate_limited' event
};

ws.onmessage = (event) => {
    console.log("⬅️ Received from server:", JSON.parse(event.data));
};

ws.onclose = (event) => {
    console.error("❌ WebSocket connection closed:", event);
};

ws.onerror = (error) => {
    console.error("💥 WebSocket error:", error);
};
```
You have now successfully connected to and interacted with your GoDispatch server!

## Next Steps

- **Configuration:** Learn how to customize the server's behavior in the [Configuration Guide (CONFIG.md)](CONFIG.md).
- **Architecture:** Understand the internal design by reading the [Architecture Guide (ARCHITECTURE.md)](ARCHITECTURE.md).

## Contributing
Contributions are welcome! Please feel free to open an issue or submit a pull request.
