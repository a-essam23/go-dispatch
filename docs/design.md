# Architecture, Design, and Philosophy

This document outlines the core principles guiding the development of GoDispatch. Adherence to these rules ensures the project remains modular, testable, and maintainable.

## I. Core Philosophy

1.  **Configuration over Code**: The engine's behavior must be driven by external configuration (YAML) wherever possible. Application logic should interpret configuration, not contain hardcoded business rules.
2.  **Explicit over Implicit**: Dependencies and data flow must be explicit. Avoid "magical" behavior. A function's signature should clearly state what it needs to operate.
3.  **Clarity over Premature Optimization**: Write clean, readable, and well-structured code first. Optimize only after profiling reveals a clear bottleneck.

## II. Layered Architecture

1.  **Strictly Adhere to Layering**: The system is composed of distinct layers (State, Transport, Gateway, Router, etc.). Each has a single, well-defined responsibility.
2.  **Layers Communicate Only with Adjacent Layers**: A layer should not bypass its neighbors. For example, the `EventRouter` (Layer 3) should interact with the `StateManager` (Layer 0) but not directly with the `Transport` (Layer 1).
3.  **Define Clear Responsibilities for Each Layer**:
    *   **Layer 0 (State Core)**: Manages the lifecycle and relationships of all canonical data objects (Users, Connections, Rooms).
    *   **Layer 1 (Transport Core)**: Manages the raw WebSocket I/O (read/write pumps). It is completely stateless.
    *   **Layer 2 (Connection Gateway)**: Guards the HTTP endpoint. It authenticates and authorizes *connection attempts*, but does not handle post-connection message logic.
    *   **Layer 3 (Event Router)**: Interprets messages from connected clients and delegates actions to the State Core.

## III. Dependency Management & Inversion of Control

1.  **Always Use Constructor-Based Dependency Injection**: Dependencies (like loggers or state managers) must be passed as arguments into a component's constructor function (e.g., `NewAuthMiddleware(logger, config)`).
2.  **Depend on Abstractions, Not Concretions**: Components should depend on interfaces or function types, not concrete structs.
    *   **Example**: The `RateLimiter` middleware depends on the `IPConnectionCounter` function type, not the concrete `*state.InMemoryManager`. This decouples the middleware from the state implementation and makes testing trivial.
3.  **The `server` Package is the Composition Root**: The `internal/server` package, specifically the `NewApp` function, is the *only* place where concrete dependencies are instantiated and wired together.
4.  **Avoid Global State**: Do not use global variables or singletons for application components. Pass all dependencies explicitly.

## IV. Middleware Design Pattern

1.  **Follow the "Blueprint" Pattern**: Middleware does not modify core application state directly. Instead, it creates and validates a "blueprint" of the request (our `RequestMetadata` struct). This inert struct is passed down the chain via the context.
2.  **The Final Handler Commits the State**: The final `http.Handler` in the chain (`upgradeHandler`) receives the validated blueprint and is responsible for making the single, atomic change to the `StateManager`.
3.  **Do Not Mutate Global State Within Middleware**: Violating this rule breaks transactional integrity. If a middleware modified the state and a subsequent middleware rejected the request, the state would be left corrupted.
4.  **Use `context.WithValue` for Request-Scoped Blueprints Only**: The context is for passing request-scoped, inert data like `RequestMetadata`. It is **not** for passing application dependencies like loggers, configuration, or state managers.

## V. State Management

1.  **The State Core is the Single Source of Truth**: All information about the system's current state (who is connected, who is in what room) must reside within the State Manager. No other component should hold its own version of this state.
2.  **Use a Pointer-Based Model for Canonical Objects**: Storing pointers to `ConnectionProfile`, `UserSession`, etc., ensures that any change to an object is instantly reflected everywhere it is referenced, preventing data duplication and inconsistency.
3.  **Ensure Thread Safety at the Boundary**: All methods on the `StateManager` interface must be safe for concurrent use. The implementation (`InMemoryManager`) is responsible for its own internal locking (e.g., `sync.RWMutex`).
4.  **Abstract the State Backend via Interfaces**: The `StateManager` interface is the public contract. This allows the underlying implementation (`InMemoryManager`) to be swapped out in the future without changing any of the consuming code.

## VI. Package Organization

1.  **Distinguish `pkg` and `internal`**:
    *   `pkg/`: Contains truly generic, reusable libraries that could theoretically be published as standalone modules (e.g., our state management or transport primitives).
    *   `internal/`: Contains all private application logic specific to the GoDispatch service. This code cannot be imported by external applications.
2.  **Organize by Feature or Responsibility**: Create dedicated packages for clear areas of responsibility (e.g., `internal/server/middleware`, `pkg/state`).
