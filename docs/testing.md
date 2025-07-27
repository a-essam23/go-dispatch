## Prerequisites

- [ ] The server is running locally via `make run`.
- [ ] The `config.yaml` file is configured with the test events (especially `join_private_room` and `send_public_message`).
- [ ] The `.env` file is present and contains the correct `GODISPATCH_SERVER_AUTH_JWTSECRET`.
- [ ] A browser with its developer console is open for running the client-side JavaScript snippet.
- [ ] [jwt.io](https://jwt.io/) is open for generating tokens.

---

## ✅ Test Suite 1: Connection Handshake & Authentication

**Goal:** Verify the server correctly handles session JWTs at the connection gateway.

- [ ] **1a. Valid Connection:**
    -   **Action:** Connect using a valid session JWT with the correct secret.
    -   **Expected:** WebSocket connection succeeds. Browser console logs `✅ WebSocket connection established!`.

- [ ] **1b. Invalid Signature:**
    -   **Action:** Connect using a session JWT signed with the wrong secret.
    -   **Expected:** WebSocket connection fails immediately. Console logs `❌ WebSocket connection closed`.

- [ ] **1c. Missing Token:**
    -   **Action:** Attempt to connect without providing the `Authorization` protocol header.
    -   **Expected:** WebSocket connection fails immediately. Console logs `❌ WebSocket connection closed`.

- [ ] **1d. Malformed Token:**
    -   **Action:** Attempt to connect using a random, non-JWT string as the token.
    -   **Expected:** WebSocket connection fails immediately. Console logs `❌ WebSocket connection closed`.

- [ ] **1e. Expired Token:**
    -   **Action:** On `jwt.io`, create a valid session token but set the `exp` claim to a timestamp in the past. Attempt to connect.
    -   **Expected:** WebSocket connection fails. Server log should show an "invalid token" or "token is expired" error.

---

## ✅ Test Suite 2: Connection Limiter

**Goal:** Verify the `connectionLimit` feature works as configured in `config.yaml`. (Assumes `maxPerUser: 5`, `mode: "cycle"`).

- [ ] **2a. Under the Limit:**
    -   **Action:** Using a single session JWT (`sub: "user-limiter"`), open 4 separate browser tabs and establish a connection in each.
    -   **Expected:** All 4 connections are established and remain active.

- [ ] **2b. Hitting and Exceeding the Limit (`cycle` mode):**
    -   **Action:** With 4 connections active for `user-limiter`, open a 5th tab and connect. Then open a 6th tab and connect.
    -   **Expected:** The 5th and 6th connections succeed. One of the initial tabs should now show a `❌ WebSocket connection closed` message, indicating it was cycled out.

- [ ] **2c. Server Logging for Cycling:**
    -   **Action:** While performing test 2b, monitor the server logs.
    -   **Expected:** When the 6th connection is made, the server log should contain a message like "Cycling connection: closing oldest...".

---

## ✅ Test Suite 3: Modifier - `rate_limit`

**Goal:** Verify the `rate_limit` modifier correctly blocks and allows events. (Tests the `send_public_message` event with a `20/m` limit).

- [ ] **3a. Rapid Fire Events to Trigger Limit:**
    -   **Action:** Establish a connection. Use a `for` loop in the browser console to send the `send_public_message` event 25 times in quick succession.
    -   **Expected:** The server log shows 20 successful `_notify_room` action executions, followed by 5 `Modifier check failed... rate limit... exceeded` warnings.

- [ ] **3b. Window Expiration and Reset:**
    -   **Action:** Perform test 3a to get rate-limited. Wait for just over one minute. Send the `send_public_message` event one more time.
    -   **Expected:** The final event is processed successfully. The server log should show a successful action execution.

- [ ] **3c. Separate Limits for Different Users:**
    -   **Action:** Connect with two different clients using two different session tokens (`user-A` and `user-B`). Get `user-A` rate-limited by spamming events. Immediately after, send an event from `user-B`.
    -   **Expected:** The event from `user-B` is processed successfully, proving that rate limits are tracked per user.

---

## ✅ Test Suite 4: Modifier - `secure`

**Goal:** Verify the `secure` modifier correctly validates event-specific JWTs. (Tests the `join_private_room` event).

- [ ] **4a. Valid Event Token:**
    -   **Action:** From your backend (or jwt.io), generate a short-lived (e.g., 60s expiry) JWT with the correct secret. Send the `join_private_room` event, including this token in the `token` field of the message.
    -   **Expected:** The server log shows the `_log` action running successfully and printing the message with the claims from the token (e.g., `User user-123 joining room room-private-A`).

- [ ] **4b. Missing Event Token:**
    -   **Action:** Send the `join_private_room` event *without* the `token` field in the message.
    -   **Expected:** The server log shows a `Modifier check failed` error with a message like "request payload missing required 'token' field". The action pipeline does not run.

- [ ] **4c. Invalid Signature on Event Token:**
    -   **Action:** Generate an event JWT using the wrong secret. Send the `join_private_room` event with this invalid token.
    -   **Expected:** The server log shows a `Modifier check failed` error with a message like "token validation failed: signature is invalid".

- [ ] **4d. Expired Event Token:**
    -   **Action:** Generate an event JWT with an `exp` claim in the past. Send the `join_private_room` event with this expired token.
    -   **Expected:** The server log shows a `Modifier check failed` error with a message like "token validation failed: token is expired".

- [ ] **4e. Templating with Token Claims:**
    -   **Action:** Generate a valid event JWT with custom claims, e.g., `{"room_id": "secret-lounge", "grant_perms": "read-write"}`. Send the `join_private_room` event.
    -   **Expected:** The `_notify_origin` action sends back a payload that correctly resolves the `{$token.room_id}` template, e.g., `{"room":"secret-lounge"}`.
