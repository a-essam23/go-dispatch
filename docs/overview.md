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
The foundation of the system. It's a high-performance, in-memory data grid that stores canonical, inter-linked objects for all connections, users, IPs, and rooms. By using a pointer-based model, it ensures data consistency and efficiency. It is built behind a pluggable `StateManager` interface, allowing the default in-memory backend to be swapped for alternatives like Redis in the future.

**Layer 1: Transport Core**
A pure, stateless transport layer responsible for WebSocket connections and message pumps. It's built using the high-performance `github.com/coder/websocket` library, handling connection lifecycles and timeouts gracefully. It has no knowledge of application state and is completely decoupled from other layers.

**Layer 2: Connection Gateway**
A standard Go `http.Handler` middleware chain that guards the WebSocket upgrade endpoint. It follows a "blueprint" pattern where each request's details are captured in an inert `RequestMetadata` struct. This chain is composed of pluggable, single-responsibility middlewares that are wired together using constructor-based dependency injection. Implemented middlewares include:
*   **RequestMetadata Middleware**: Injects the initial metadata blueprint into the request context.
*   **Logging Middleware**: Provides observability for every connection attempt.
*   **Rate Limiting Middleware**: Prevents abuse by limiting connections per IP. It is decoupled from the state core via a function-type dependency (`IPConnectionCounter`), making it highly testable.
*   **Authentication Middleware**: Enforces a mandatory JWT policy for all incoming connections.

**Layer 3: Event Router** (Next Stage)
Provides room-based message routing with subscription management. It leverages the State Core to efficiently look up members of a room and deliver messages. Complex routing patterns and broadcasts are defined declaratively in configuration files.

**Layer 4: Configuration Engine** (Future)
Loads and validates YAML configuration files with schema validation. Supports hot-reloading for configuration changes without service restarts. Manages configuration inheritance and environment-specific overrides.

**Layer 5: Event Processing Pipeline** (Future)
Handles message transformation, input validation, and conditional processing rules defined in configuration. Provides data filtering and format conversion capabilities before messages are routed.

**Layer 6: Action Ecosystem** (Future)
An extensible plugin system for external integrations, custom actions, and third-party service connections. Designed for future expansion without modifying the core system.

## Structure
```
.
├── cmd/                          # Application Entrypoints (Orchestration)
│   └── go-dispatch/
│       └── main.go               # --- Main function to start the server.
├── docs/                         # Project Documentation
├── internal/                     # Private Application Logic (Not for external import)
│   └── server/                   # Layer 2: Connection Gateway & Server Orchestration [COMPLETE]
│       ├── middleware/           # --- Single-responsibility middlewares for the gateway.
│       │   ├── auth.go
│       │   ├── chain.go
│       │   ├── logger.go
│       │   ├── metadata.go
│       │   └── ratelimiter.go
│       └── server.go             # --- App struct, server setup, and final HTTP->WebSocket upgrade handler.
├── pkg/                          # Reusable Libraries (Can be imported by other projects)
│   ├── logging/                  # --- Cross-cutting concern: Standardized logger setup.
│   │   └── logger.go
│   ├── state/                    # Layer 0: State Core [COMPLETE]
│   │   ├── interface.go          # --- Defines the contract for all state management backends.
│   │   ├── manager_inmemory.go   # --- The default in-memory implementation of the state manager.
│   │   └── models.go             # --- Canonical data models (ConnectionProfile, UserSession, etc.).
│   └── transport/                # Layer 1: Transport Core [COMPLETE]
│       └── connection.go         # --- Manages a single WebSocket connection (read/write pumps).
├── go.mod
└── go.sum
```
## Security Model

The system enforces a **mandatory JWT policy** for all connections. The application backend is the sole authority for issuing tokens. A token's only responsibility is to provide a **stable, verified User ID** (from the `sub` claim); it does not carry permission data. This powerful model allows guest access to be managed entirely by the backend (by issuing tokens with temporary or guest-specific IDs) without changing GoDispatch's logic.

Upon successful token validation, GoDispatch creates or retrieves a canonical `UserSession` object in the **State Core**, making it the single source of truth for that user's identity.

## Permission Management

Authorization operates on a capability-based model where the central `UserSession` object is the single source of truth for all user permissions. Administrative events can be sent to GoDispatch to modify these session objects in real-time. This enables instant enforcement of bans, mutes, or privilege changes without requiring user reconnection, as all authorization checks will consult this live state object.

## Configuration Philosophy

All system behavior is defined through YAML configurations rather than code. This includes event routing rules, permission requirements, message transformation logic, room management policies, and integration patterns. The configuration system supports complex conditional logic and data processing while remaining readable and maintainable.

## Project Structure

The codebase follows Go project layout best practices. Core, reusable library components reside in `pkg/` (e.g., `state`, `transport`). Application-specific wiring and business logic are encapsulated within `internal/`. For example, the server entrypoint is in `internal/server`, and all HTTP middlewares are organized in the `internal/server/middleware` package. This ensures a clean separation between the reusable engine and the running service.

## Development Goals

This project serves as both a practical tool and a learning exercise in professional Go development. The architecture emphasizes a clean separation of concerns, pluggable backends via interfaces, explicit dependency injection, and comprehensive error handling. The open-source nature encourages community contributions, while the modular design allows for the selective adoption of individual components.
