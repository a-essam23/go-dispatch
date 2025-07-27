# GoDispatch: Configuration Guide

The `config.yaml` file is the heart of your GoDispatch server. It provides a declarative way to define your server's behavior, event logic, and security rules without writing any code.

This document serves as a complete reference for all available configuration options.

## Table of Contents

1.  [Server Layer](#1-server-layer)
    -   `server.address`
    -   `server.auth.jwtSecret`
    -   `server.connectionLimit`
2.  [Transport Layer](#2-transport-layer)
    -   `transport.readTimeout`
3.  [Router Layer](#3-router-layer)
    -   `events`
    -   `modifiers`
    -   `actions`
4.  [Permissions](#4-permissions)
5.  [Templating Syntax](#5-templating-syntax)
6.  [Full Example `config.yaml`](#6-full-example-configyaml)

---

## 1. Server Layer

This section configures the HTTP server and connection-level security.

### `server.address`

The address and port for the HTTP server to listen on.

-   **Type:** `string`
-   **Default:** `":8080"`
-   **Example:** `address: ":8080"`

### `server.auth.jwtSecret`

The secret key used to validate the signature of JWTs for the initial connection handshake.

> **Security Warning:** This value should **NEVER** be hardcoded in `config.yaml` for production. It **MUST** be provided via an environment variable.

-   **Type:** `string`
-   **Environment Variable:** `GODISPATCH_SERVER_AUTH_JWTSECRET`
-   **Example (for local dev only):** `jwtSecret: "a-very-secret-key-from-env-file"`

### `server.connectionLimit`

Configures the maximum number of concurrent WebSocket connections allowed for a single `UserID`.

-   `maxPerUser`: The maximum number of connections. A value of `0` or less means no limit.
-   `mode`: The behavior when the limit is reached:
    -   `"reject"`: Rejects the new connection attempt with an error.
    -   `"cycle"`: Closes the user's oldest active connection and accepts the new one.

-   **Example:**
    ```yaml
    connectionLimit:
      maxPerUser: 3
      mode: "cycle"
    ```

---

## 2. Transport Layer

This section configures low-level WebSocket connection settings.

### `transport.readTimeout`

The maximum duration the server will wait for a message from a client before considering the connection dead and closing it.

-   **Type:** `duration string` (e.g., `60s`, `10m`, `1h`)
-   **Default:** `"60m"`
-   **Example:** `readTimeout: "30m"`

---

## 3. Router Layer

This is where you define your application's core real-time logic.

### `events`

An `event` is a named entrypoint triggered by a client message. Each event consists of an optional `modifiers` chain and an `actions` pipeline.

-   The key (e.g., `send_message`) is the `event` name that the client must send.
-   The value is an object containing `modifiers` and/or `actions`.

### `modifiers`

**Modifiers are guards.** They are a list of validation steps that run *before* any actions. If any modifier fails, the entire pipeline for that event is halted immediately.

-   **Type:** `list` of modifier objects.
-   **Execution:** Modifiers run sequentially in the order they are defined.

#### Available Modifiers:

##### `secure`

Validates a short-lived JWT sent within the event's payload. This is used to authorize specific, privileged actions that have been pre-approved by your backend.

-   **Client Requirement:** The client must include a `token` field in the root of the JSON message.
-   **Params:** None.
-   **Example:**
    ```yaml
    modifiers:
      - name: "secure"
    ```

##### `rate_limit`

Restricts how frequently a single user can trigger a specific event. The rate limit is tracked per `UserID` and per `EventName`.

-   **Params:**
    -   `"count/unit"`: A single string parameter.
        -   `count`: An integer.
        -   `unit`: `s` (seconds), `m` (minutes), or `h` (hours).
-   **Example:**
    ```yaml
    modifiers:
      - name: "rate_limit"
        params: ["10/m"] # Allow this event 10 times per minute per user.
    ```

### `actions`

**Actions are verbs.** They are a sequence of functions that *do* things—send messages, log information, or change state. They only run if all modifiers pass.

-   **Type:** `list` of action objects.
-   **Execution:** Actions run sequentially in the order they are defined.

#### Available Actions:

##### `_log`

Writes a message to the GoDispatch server's standard log output. Useful for debugging pipelines.

-   **Params:** A single string message.
-   **Example:** `params: ["User triggered the 'join_room' event."]`

##### `_notify_room`

Sends a new message to all connected members of a target room. The target room is specified by the `target` field in the client's original message.

-   **Params:**
    1.  `event_name` (string): The name of the new event to send to the clients in the room.
    2.  `payload` (string): The payload for the new event. Often uses templating.
-   **Example:** `params: ["new_message", "{.payload.message}"]`

##### `_notify_origin`

Sends a message back only to the specific client connection that triggered the event.

-   **Params:**
    1.  `event_name` (string): The name of the event to send back.
    2.  `payload` (string): The payload for the event.
-   **Example:** `params: ["join_room_success", "{\"status\":\"ok\"}"]`

---

## 4. Permissions

A top-level list of custom, application-specific permission names. GoDispatch assigns a unique internal ID to each. These permissions can be included in a user's session JWT (`perms` claim) to grant them global capabilities.

> **Note:** The current set of actions does not yet perform permission checks. This feature is for defining the permissions that will be used by future actions (e.g., `_join`, `_kick`).

-   **Example:**
    ```yaml
    permissions:
      - "kick_user"
      - "delete_message"
      - "view_admin_dashboard"
    ```

---

## 5. Templating Syntax

Action parameters can be made dynamic by using a simple templating syntax to pull data from the context of the incoming message.

| Template                 | Description                                                                 | Example Use Case                            |
| ------------------------ | --------------------------------------------------------------------------- | ------------------------------------------- |
| `{.target}`              | The `target` field from the root of the client message.                     | Used implicitly by `_notify_room`.          |
| `{.payload}`             | The entire JSON `payload` object, as a string.                              | Sending a complex object back to the client.|
| `{.payload.<field>}`     | A specific field from the `payload`, using GJSON path syntax.               | `{.payload.message.text}`                   |
| `{.user.id}`             | The `UserID` of the originating connection.                                 | Logging which user performed an action.     |
| `{.connection.id}`       | The unique UUID of the originating connection.                              | For detailed debugging logs.                |
| `{$token.<claim>}`       | A claim from a JWT validated by the `secure` modifier. **This is secure.** | `{$token.room_id}`, `{$token.grant_perms}`  |

---

## 6. Full Example `config.yaml`

```yaml
# ====== SERVER LAYER ======
server:
  address: ":8080"
  connectionLimit:
    maxPerUser: 5
    mode: "cycle"

transport:
  readTimeout: "45m"

# ====== ROUTER LAYER ======
events:
  # A secure event to join a room, authorized by the backend.
  join_private_room:
    modifiers:
      # This event requires a valid, short-lived JWT from the backend.
      - name: "secure"
    actions:
      # A future "_join" action would use the validated token data.
      - name: "_log"
        params: ["User {$token.sub} joining room {$token.room_id}"]
      # Notify the user that their join was successful.
      - name: "_notify_origin"
        params: ["join_success", "{\"room\": \"{$token.room_id}\"}"]

  # A public message event with rate limiting.
  send_public_message:
    modifiers:
      - name: "rate_limit"
        params: ["20/m"] # 20 messages per minute
    actions:
      - name: "_notify_room"
        params: ["new_public_message", "{.payload}"]

# ====== PERMISSIONS ======
permissions:
  - "kick_user"
  - "ban_user"
  - "delete_message"
```
