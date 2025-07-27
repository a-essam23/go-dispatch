Of course. This is a great idea. A good `overview.md` should be both an accurate snapshot and an inspiring roadmap.

Here is the updated `overview.md` file. It has been rewritten to be consistent with your current codebase while preserving the future vision. Key changes include:
*   **Correcting the architecture layers** to reflect what is already built.
*   **Updating the file structure** to include the implemented `router` and `engine`.
*   **Completely rewriting the Security and Permission models** to be factually correct based on your code.
*   **Reframing "Future" features** as logical extensions of the existing, solid foundation.

---

# GoDispatch: Configuration-Driven Real-Time WebSocket Engine

## Project Overview

GoDispatch is an open-source WebSocket engine built in Go that eliminates the need for custom real-time application development. Instead of writing WebSocket handling code, developers define event routing, permissions, and message transformations through YAML configuration files. The system handles connection management, authentication, authorization, and message routing based entirely on these configurations.

## Core Problem

Real-time applications typically require developers to write repetitive WebSocket boilerplate code for connection management, event routing, room subscriptions, and permission checks. Each project reinvents the same patterns, leading to inconsistent implementations and maintenance overhead.

## Solution Approach

GoDispatch provides a declarative approach where developers specify their real-time requirements in YAML configurations. The engine interprets these configurations to handle WebSocket connections, route messages between clients, manage rooms and subscriptions, and enforce permissions without requiring custom code.

## Architecture

The system follows a layered architecture designed for performance, stability, and extensibility. The core of the system is a centralized, in-memory state manager that acts as the single source of truth.

**Layer 0: State Core**
The foundation of the system. It's a high-performance, in-memory data grid that stores canonical, inter-linked objects for all connections, users, and rooms. By using a pointer-based model, it ensures data consistency and efficiency. It is built behind a pluggable `StateManager` interface, allowing the default in-memory backend to be swapped for alternatives like Redis in the future.

**Layer 1: Transport Core**
A pure, stateless transport layer responsible for WebSocket connections and message pumps. It's built using the high-performance `github.com/coder/websocket` library, handling connection lifecycles and timeouts gracefully. It has no knowledge of application state and is completely decoupled from other layers.

**Layer 2: Connection Gateway**
A standard Go `http.Handler` middleware chain that guards the WebSocket upgrade endpoint. It follows a "blueprint" pattern where each request's details are captured in an inert `RequestMetadata` struct. This chain is composed of pluggable, single-responsibility middlewares, including:
*   **RequestMetadata Middleware**: Injects the initial metadata blueprint into the request context.
*   **Logging Middleware**: Provides observability for every connection attempt.
*   **Connection Limiting Middleware**: Prevents abuse by limiting connections per user, with configurable "reject" or "cycle" strategies.
*   **Authentication Middleware**: Enforces a mandatory JWT policy for all incoming connections.

**Layer 3: Event Router**
The core message processing hub. When a message arrives from a client, the router is responsible for parsing it, identifying the event type, and looking up the corresponding pre-compiled action pipeline from the configuration.

**Layer 4: Configuration Engine**
Loads, parses, and compiles YAML configuration files into executable pipelines on startup. This engine turns declarative rules into an efficient in-memory representation. The foundation is designed to support future enhancements like schema validation and hot-reloading.

**Layer 5: Event Processing Pipeline**
Executes the action pipelines defined in the configuration. The current implementation includes a Just-In-Time (JIT) template resolver that dynamically injects data from the incoming message (`payload`, `target`, `user.id`, etc.) into the parameters of each action before it's executed.

**Layer 6: Action Ecosystem**
The set of built-in, executable functions that can be used in a pipeline. The core is in place with primitives for communication (`_notify_room`, `_notify_origin`) and introspection (`_log`). The system is designed for easy extension, allowing new actions (e.g., for permission management or third-party integrations) to be added without modifying the core engine.

## Structure
```
.
├── cmd/                          # Application Entrypoints
│   └── go-dispatch/
│       └── main.go               # --- Main function to start the server.
├── internal/                     # Private Application Logic
│   ├── engine/                   # Layer 6: Action Ecosystem & Registry
│   │   ├── actions.go
│   │   └── registry.go
│   ├── router/                   # Layer 3 & 5: Event Router & Pipeline Execution
│   │   ├── router.go
│   │   └── models.go
│   └── server/                   # Layer 2: Connection Gateway & Server Orchestration
│       ├── middleware/
│       └── server.go
├── pkg/                          # Reusable Libraries
│   ├── config/                   # Layer 4: Configuration Loading & Compiling
│   │   ├── compiler.go
│   │   ├── loader.go
│   │   └── models.go
│   ├── logging/
│   ├── pipeline/                 # --- Defines the core pipeline constructs (Cargo, Step).
│   ├── state/                    # Layer 0: State Core
│   │   ├── interface.go
│   │   ├── statemanager/
│   │   └── models.go
│   └── transport/                # Layer 1: Transport Core
│       └── connection.go
├── config.yaml
└── go.mod
```
## Security Model

The system enforces a **mandatory JWT policy** for all connections, positioning the application backend as the sole authority for issuing tokens. A token's claims are used to establish the connection's identity and base capabilities:
*   The `sub` (Subject) claim is **required** and provides the stable, verified **UserID**.
*   The `perms` claim is **optional** and can contain a list of string names for any global permissions the user should have for the duration of their session.

Upon successful token validation, GoDispatch creates or retrieves a canonical `User` object in the **State Core**, making it the single source of truth for that user's identity and permissions for the lifetime of the connection.

## Permission Management

Authorization operates on a capability-based model where the central `User` object is the single source of truth for a user's permissions. Permissions loaded from the initial JWT are stored in this live state object.

All subsequent permission checks during the session will consult this live object. This architecture ensures that permission changes can be enforced instantly. The model is designed to support future administrative events (e.g., `_set_permissions`, `_join`) that can modify these user objects in real-time, enabling features like instant enforcement of bans, mutes, or privilege changes without requiring user reconnection.

## Configuration Philosophy

All system behavior is defined through YAML configurations rather than code. This includes event routing rules, permission definitions, connection limits, and the sequence of actions that constitute an event's logic. This approach makes the engine highly adaptable to different use cases without code changes.

## Development Goals

This project serves as both a practical tool and a learning exercise in professional Go development. The architecture emphasizes a clean separation of concerns, pluggable backends via interfaces, explicit dependency injection, and comprehensive error handling. The open-source nature encourages community contributions, while the modular design allows for the selective adoption of individual components.
