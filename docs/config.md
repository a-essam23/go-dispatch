# System Design Guideline

## I. Core Philosophy

1.  **Backend is the Policy Authority:** The application backend is the sole decider of business logic. It determines who can access what resources and under which conditions by issuing signed **Action Tickets**.
2.  **GoDispatch is the Enforcement Engine:** GoDispatch's role is to act as a high-performance, real-time enforcement point. It validates tickets and executes the action pipelines defined in `config.yaml`.
3.  **The Room is the Universe:** All communication, state, and permissions are scoped to a "room." This provides a unified and predictable model for all interactions.
4.  **Configuration as Code:** All engine behavior is defined declaratively. No custom Go code is required to define application logic.

## II. The `UserID` as the Canonical Identifier

The `UserID` is the primary key for all user-related operations within GoDispatch.

*   **Source:** The `UserID` is provided by the backend via the primary **Session Token (JWT)** upon initial connection.
*   **Uniqueness:** It is the developer's responsibility to ensure `UserID`s are unique.
*   **Connection Grouping:** GoDispatch internally maps all active WebSocket connections to a single `UserID`. This allows actions targeting a `UserID` to be delivered to all of that user's connected devices (e.g., phone, laptop, tablet) simultaneously.
*   **Reserved Namespaces:** While most room names are developer-defined, GoDispatch reserves certain prefixes for internal functionality. The primary reserved namespace is `user:`.

## III. The User Room: A Fundamental Primitive

Every authenticated user is automatically associated with a **User Room**, named `user:<UserID>`.

*   **Purpose:** This room acts as the user's personal notification inbox and the direct target for user-to-user communication. It is the mechanism for presence tracking and cross-device synchronization.
*   **Membership:** All connections authenticated with a given `UserID` are automatically members of the corresponding User Room. This is an internal engine behavior and does not require an explicit `join` event.

## IV. The Two-Token Authorization Model

The system employs a two-token model to separate **identity** from **contextual authority**.

*   **Session Token (The "ID Card"):** A long-lived JWT issued by the backend that validates a user's identity (`UserID`) and global attributes for a session.
*   **Action Ticket (The "Boarding Pass"):** A short-lived, single-purpose, HMAC-signed payload issued by the backend to grant temporary, contextual authority for a privileged action (e.g., joining a room with specific permissions).

## V. The Logic & Configuration Structure

### 1. Events: The Entrypoint
Events are the public API, triggered by clients. They are defined in `config.yaml`.

```yaml
events:
  kick_user:
     This event is privileged and requires proof of backend authority.
    modifiers: [ proof: "HMAC" ]
    ayload: { ... } # Defines required user_id and room_id
    actions:
       The pipeline to execute if the ticket is valid.
      - _revoke_permissions("all", {.payload.room_id}, {.payload.user_id})
      - _leave({.payload.room_id}, {.payload.user_id})
```

### 2. Modifiers: The Security Guardrail
Declarative constraints enforced *before* actions are executed.
*   **`proof: "HMAC"`:** Mandates a valid, unexpired **Action Ticket**. The engine validates its signature, expiry, and context.
*   **`rate: "count/duration"`:** Enforces per-connection rate limiting.

### 3. Action Pipelines: The Logic
A sequence of built-in primitives that define the event's logic.

**Built-in Action Primitives:**

*   **Permission Management:**
    *   `_set_permissions(grants_bitmap, room_id, [user_id])`: Applies a full permission bitmap to a user in a room. The `user_id` is optional and defaults to the originating user.
    *   `_add_permissions(permissions_list, room_id, [user_id])`: Adds specific permissions to a user's grant.
    *   `_revoke_permissions(permissions_list | "all", room_id, [user_id])`: Revokes specific (or all) permissions from a user's grant.

*   **Membership Management:**
    *   `_join(room_id, [user_id])`: Adds a user (all their connections) to a room's membership list.
    *   `_leave(room_id, [user_id])`: Removes a user (all their connections) from a room.

*   **Communication:**
    *   `_notify(room_id, event_name, payload)`: Sends a new event to all members of a target room.
    *   `_notify_origin(event_name, payload)`: Sends an event back to the original connection that triggered the pipeline.

*   **State & Logic:**
    *   `_check_permissions(permissions_list)`: Halts the pipeline if the originating user does not have the required permissions for the current room context.
    *   `_log(level, message)`: Writes to the engine's logs.

**Templating:**
Action parameters are made dynamic via a simple templating syntax.
*   `{.payload.field}`: Accesses data from the incoming event's payload.
*   `{$ticket.field}`: Accesses data from the validated Action Ticket (e.g., `{$ticket.grants}`).
*   `{$connection.user_id}`: Accesses the UserID of the originating connection.

## VI. Example Flow: The Authoritative Kick

This demonstrates how the "Control Plane" is simply a well-secured event, not a separate system.

1.  A room admin in an application clicks "Kick User B." This sends an HTTP request to the application backend.
2.  The backend verifies the admin's authority through its own business logic.
3.  The backend issues a short-lived **Action Ticket** authorizing the `kick_user` action for User B in that specific room.
4.  The backend, acting as a privileged client, sends the `kick_user` event to GoDispatch, including the ticket in the `proof` field.
5.  GoDispatch receives the event.
    a. The `proof: "HMAC"` modifier validates the ticket is authentic and unexpired.
    b. The action pipeline begins.
    c. `_revoke_permissions("all", "group:dev-chat", "user-b-id")` is executed, clearing all of User B's permissions for that room from the live state.
    d. `_leave("group:dev-chat", "user-b-id")` is executed, removing User B from the room's membership and terminating their connections to it.
6.  The kick is complete and enforced at the engine level, securely and instantly.
