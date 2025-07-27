# System Configuration Guide

## I. Core Philosophy

1.  **Configuration as Code:** GoDispatch is designed to be controlled by configuration, not by custom Go code. This file, `config.yaml`, defines your application's real-time behavior, including event handling, permissions, and server settings.
2.  **Stateless Logic, Stateful Core:** The logic you define in this file is stateless. It operates on the live state of users and connections managed by the GoDispatch engine's in-memory core.
3.  **The Room is the Universe:** All communication is scoped to a "room." A room can represent a group chat, a document collaboration session, or any logical grouping of users.
4.  **The User Room is the Direct Channel:** In addition to custom rooms, every user automatically gets a private "user room" for direct messaging and personal notifications, enabling easy cross-device synchronization.

## II. The `UserID` and Session Token

The `UserID` is the primary key for all user-related operations. It is provided by your backend via a standard **Session Token (JWT)** when a client connects.

*   **Source:** The `sub` (Subject) claim of the JWT must contain the unique `UserID`.
*   **Permissions:** An optional `perms` claim in the JWT, containing an array of permission names (strings), can be used to grant a user global permissions for their session.
*   **Uniqueness:** Your application backend is responsible for ensuring `UserID`s are unique and for issuing valid JWTs. GoDispatch only enforces the JWT's validity.

## III. Configuration Structure

The `config.yaml` is divided into several main sections that control different layers of the engine.

### 1. Server Layer: Connection & Transport

This section defines the behavior of the HTTP/WebSocket server itself.

```yaml
# ====== SERVER LAYER ======
server:
  # The address and port for the HTTP server to listen on.
  address: ":8080"

  # The secret key for signing and validating JWTs.
  # This MUST be set via an environment variable in production (e.g., GODISPATCH_SERVER_AUTH_JWTSECRET).
  auth:
    jwtSecret: "a-very-secret-key"

  # Configure limits on concurrent connections per UserID.
  connectionLimit:
    maxPerUser: 3
    # What to do when the limit is reached:
    # "reject": Reject the new connection attempt.
    # "cycle":  Close the user's oldest connection and accept the new one.
    mode: "cycle"

transport:
  # The maximum duration for waiting for a message from a client
  # before the connection is considered dead.
  readTimeout: "60m"
```

### 2. Router Layer: Events & Actions

This is where you define your application's core logic. An **Event** is a named entrypoint triggered by a client message. Each event is composed of an **Action Pipeline**—a sequence of built-in functions that execute in order.

```yaml
# ====== ROUTER LAYER ======
events:
  # Event triggered when a client wants to send a message to a group room.
  send_message_to_room:
    actions:
      # The target room ID is taken from the incoming message's `target` field.
      - name: _notify_room
        params: ["new_message", "{.payload.message}"] # This action implicitly sends to the `target` room.

  # Event triggered when a client wants to send a direct message.
  send_direct_message:
    actions:
      # This works because a user's private room (e.g., "user:some-id") is treated
      # just like any other room by the notification action.
      - name: _notify_room
        params: ["new_dm", "{.payload.message}"]

  # A simple event for logging client-side analytics or debug info.
  log_client_event:
    actions:
      - name: _log
        params: ["Client analytics event received"]
```

### 3. Permissions

This section allows you to define custom, application-specific permissions. GoDispatch will assign a unique bitmap value to each, allowing for efficient permission checks.

```yaml
# The engine will assign a unique bitmap value to each permission.
permissions:
  - "kick"
  - "ban"
  - "grant_perms"
  - "delete_message"
  # Add any other custom permissions your application needs.
```
These permissions can be included in a user's JWT `perms` claim to grant them global capabilities.

## IV. Action Primitives & Templating

**Actions** are the built-in functions you can use to build your pipelines.

**Available Actions:**

*   `_notify_room(event_name, payload)`: Sends a new message to all members of the room specified in the incoming event's `target` field.
*   `_notify_origin(event_name, payload)`: Sends a message back only to the specific client connection that triggered the event.
*   `_log(message)`: Writes a message to the GoDispatch server's standard log output. Useful for debugging pipelines.

**Templating:**

Action parameters are made dynamic using a simple templating syntax that pulls data from the context of the incoming message.

*   `{.target}`: Accesses the top-level `target` field from the client's message. This is used implicitly by `_notify_room`.
*   `{.payload}`: Accesses the entire JSON payload of the message as a string.
*   `{.payload.<field>}`: Uses GJSON path syntax to access a specific field within the JSON payload (e.g., `{.payload.message.text}`).
*   `{.user.id}`: Accesses the UserID of the originating connection.
*   `{.connection.id}`: Accesses the unique UUID of the originating connection.

## V. Example Flow: Sending a Direct Message

This example shows how the system components work together.

1.  **Client A** wants to send a message to **Client B**. Client A's application knows Client B's UserID is `user-b-id`.
2.  Client A constructs and sends a WebSocket message to GoDispatch:
    ```json
    {
      "target": "user:user-b-id",
      "event": "send_direct_message",
      "payload": {
        "from": "user-a-id",
        "message": "Hello!"
      }
    }
    ```
3.  GoDispatch receives the message.
    a. The **Event Router** sees the event name is `send_direct_message` and looks up its action pipeline in the configuration.
    b. The router finds one action: `_notify_room` with params `["new_dm", "{.payload.message}"]`.
    c. The **Pipeline Engine** resolves the parameters. `{.payload.message}` is resolved to the string `"Hello!"`.
    d. The `_notify_room` action is executed. It looks at the message's `target` (`user:user-b-id`) and finds all connections that belong to that room (in this case, all of Client B's connected devices).
4.  GoDispatch sends a new message to all of Client B's connections:
    ```json
    {
      "event": "new_dm",
      "payload": "Hello!"
    }
    ```
5.  Client B's application receives the `new_dm` event and displays the message.
