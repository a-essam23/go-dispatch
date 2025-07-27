# Architecture, Design, and Philosophy

This document outlines the core principles guiding the development of GoDispatch. Adherence to these rules ensures the project remains modular, testable, and maintainable.

## I. Core Philosophy

1.  **Configuration over Code**: The engine's behavior must be driven by external configuration (YAML) wherever possible. Application logic should interpret configuration, not contain hardcoded business rules. The `config.yaml` is the definition of the application's real-time logic.
2.  **Explicit over Implicit**: Dependencies and data flow must be explicit. Avoid "magical" behavior. A function's signature should clearly state what it needs to operate.
3.  **Clarity over Premature Optimization**: Write clean, readable, and well-structured code first. Optimize only after profiling reveals a clear bottleneck.

## II. Layered Architecture

1.  **Strictly Adhere to Layering**: The system is composed of distinct layers, each with a single, well-defined responsibility.
2.  **Layers Communicate Through Interfaces**: A layer should not depend on the concrete implementation of another layer, but rather on an interface or function type that defines a contract.
3.  **Define Clear Responsibilities for Each Layer**:
    *   **Layer 0 (State Core - `pkg/state`)**: Manages the lifecycle and relationships of all canonical data objects (Users, Connections, Rooms). It is the single source of truth for the application's run-time state.
    *   **Layer 1 (Transport Core - `pkg/transport`)**: Manages the raw WebSocket I/O (read/write pumps). It is completely unaware of application logic or state.
    *   **Layer 2 (Connection Gateway - `internal/server`)**: Guards the HTTP endpoint. It authenticates and authorizes *connection attempts* using a chain of middlewares, but does not handle post-connection message logic.
    *   **Layer 3 (Event Router & Pipeline - `internal/router`, `internal/engine`)**: The primary processing hub. It receives raw messages from the transport layer, parses them, finds the correct action pipeline, and executes it step-by-step.
    *   **Layer 4 (Configuration - `pkg/config`)**: Responsible for loading, parsing, and compiling the declarative YAML configuration into the executable pipelines used by the Event Router.

## III. Dependency Management & Inversion of Control

1.  **Always Use Constructor-Based Dependency Injection**: Dependencies (like loggers, configurations, or state managers) must be passed as arguments into a component's constructor function (e.g., `NewAuthMiddleware(logger, jwtSecret, permissionCompiler)`).
2.  **Depend on Abstractions, Not Concretions**: Components should depend on interfaces or function types, not concrete structs.
    *   **Example**: The `ConnectionLimiter` middleware depends on the `UserConnectionCounter` and `UserConnectionCycler` function types, not the concrete `*statemanager.InMemoryManager`. This decouples the middleware from the state implementation and makes testing trivial.
3.  **The `server` Package is the Composition Root**: The `internal/server` package, specifically the `NewApp` function, is the *only* place where concrete dependencies are instantiated and wired together.
4.  **Avoid Global State**: Do not use global variables or singletons for application components (with the rare exception of truly global registries like the one for actions). Pass all dependencies explicitly.

## IV. Middleware Design Pattern

1.  **Follow the "Blueprint" Pattern**: Middleware does not modify core application state directly. Instead, it creates and validates a "blueprint" of the request (the `RequestMetadata` struct). This inert struct is passed down the chain via the request's context.
2.  **The Final Handler Commits the State**: The final `http.Handler` in the chain (`upgradeHandler`) receives the fully populated and validated blueprint. It is responsible for making the single, atomic change to the `StateManager` (e.g., registering the connection and associating the user).
3.  **Do Not Mutate Global State Within Middleware**: Violating this rule breaks transactional integrity. If a middleware modified the state and a subsequent middleware rejected the request, the state would be left in an inconsistent state. The blueprint pattern prevents this.
4.  **Use `context.WithValue` for Request-Scoped Data Only**: The context is for passing request-scoped, inert data like `RequestMetadata`. It is **not** for passing application dependencies like loggers, configuration, or state managers.

## V. State Management

1.  **The `StateManager` is the Single Source of Truth**: All information about the system's current state (who is connected, who is in what room, what permissions they have) must reside within the `StateManager`. No other component should hold its own version of this state.
2.  **Use a Pointer-Based Model for Canonical Objects**: Storing pointers to `User`, `Connection`, etc., ensures that any change to an object is instantly reflected everywhere it is referenced, preventing data duplication and inconsistency.
3.  **Ensure Thread Safety at the Boundary**: All methods on the `StateManager` interface must be safe for concurrent use. The implementation (`statemanager.InMemoryManager`) is responsible for its own internal locking (e.g., `sync.RWMutex`).
4.  **Abstract the State Backend via Interfaces**: The `state.Manager` interface is the public contract. This allows the underlying `InMemoryManager` implementation to be swapped out in the future (e.g., with a Redis-backed manager) without changing any of the consuming code.

## VI. Package Organization

1.  **Distinguish `pkg` and `internal`**:
    *   `pkg/`: Contains truly generic, reusable libraries that could theoretically be used in other projects (e.g., our state management primitives, transport wrapper, or configuration models).
    *   `internal/`: Contains all private application logic specific to the GoDispatch service itself (e.g., the server setup, specific action implementations, and the event router). This code cannot be imported by external applications.
2.  **Organize by Feature or Responsibility**: Create dedicated packages for clear areas of responsibility (e.g., `internal/server/middleware`, `pkg/state/statemanager`).
