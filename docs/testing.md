## Setup & Prerequisites

-   [x] GoDispatch server is running locally (`make run`).
-   [ ] `config.yaml` includes all necessary `events` for testing (e.g., `join_private_room`, `leave_room`, `admin_kick_user`, `send_public_message`, `send_message_to_room`, etc., as per recent actions).
-   [x] `.env` file is configured with `GODISPATCH_SERVER_AUTH_JWTSECRET`.
-   [x] Frontend application (if any) is running and correctly configured to set the `session-token` cookie and connect to `ws://127.0.0.1:8080/ws`.

---

## 🚀 Test Suite 1: Core Connection & Authentication (Layer 2)

**Goal:** Verify initial WebSocket connection establishment, session authentication via cookies, and connection limiting.

-   [x] **1.1 Valid Cookie Authentication:**
    -   [x] **Action:** Log in via your backend (or manually set `document.cookie = "session-token=<VALID_JWT>; path=/;"` in browser console). Then, connect to `ws://127.0.0.1:8080/ws`.
    -   [x] **Expected:** WebSocket connection succeeds. Browser console logs "Connected". Server logs show "User connection fully established".
-   [x] **1.2 Missing Cookie:**
    -   [x] **Action:** Clear all cookies for `127.0.0.1` in your browser. Attempt to connect.
    -   [x] **Expected:** WebSocket connection fails. Browser console logs "Disconnected". Server logs show "Authorization cookie 'session-token' not found".
-   [ ] **1.3 Invalid/Expired Cookie:**
    -   [ ] **Action:** Set a cookie `session-token` with a malformed JWT string or an expired JWT. Attempt to connect.
    -   [ ] **Expected:** WebSocket connection fails. Server logs show "Invalid JWT token presented" or "token validation failed: token is expired".
-   [ ] **1.4 Connection Limiter (Cycle Mode - assuming `maxPerUser: 5`, `mode: "cycle"` in `config.yaml`):**
    -   [ ] **Action:** Using one `UserID`, open 5 separate browser tabs and establish connections. Then, open a 6th tab and connect.
    -   [ ] **Expected:** The 6th connection succeeds. One of the first 5 tabs logs "connection cycled by new connection". Server logs show "Cycling connection: closing oldest".
-   [ ] **1.5 Connection Limiter (Reject Mode - requires `mode: "reject"` in `config.yaml` restart):**
    -   [ ] **Action:** After changing config and restarting, using one `UserID`, open 5 tabs and connect. Then, open a 6th tab and connect.
    -   [ ] **Expected:** The 6th connection is rejected. Server logs show "User connection limit reached".

---

## ⚡ Test Suite 2: Modifiers (Layer 3 - Validation)

**Goal:** Verify `secure` and `rate_limit` modifiers correctly guard event pipelines.

-   [ ] **2.1 `secure` Modifier - Valid Token:**
    -   [ ] **Action:** Use a client connected with a session token. Generate a **short-lived, event-specific JWT** (e.g., 60s expiry) on jwt.io using `GODISPATCH_SERVER_AUTH_JWTSECRET` with claims like `{"sub": "user-123", "room_id": "private-chat", "grant_perms": "read,write"}`. Send the `join_private_room` event including this token in the `token` field of the message.
    -   [ ] **Expected:** The event pipeline executes (e.g., `_log` action runs). Server logs confirm "Secure modifier check passed".
-   [ ] **2.2 `secure` Modifier - Missing Token:**
    -   [ ] **Action:** Send the `join_private_room` event without the `token` field.
    -   [ ] **Expected:** Server logs show "Modifier check failed, pipeline halted... missing required 'token' field". Action pipeline does not run.
-   [ ] **2.3 `secure` Modifier - Invalid Signature on Token:**
    -   [ ] **Action:** Generate an event JWT with the wrong secret. Send the `join_private_room` event with this invalid token.
    -   [ ] **Expected:** Server logs show "Modifier check failed... token validation failed: signature is invalid".
-   [ ] **2.4 `secure` Modifier - Expired Token:**
    -   [ ] **Action:** Generate an event JWT with `exp` in the past. Send the `join_private_room` event.
    -   [ ] **Expected:** Server logs show "Modifier check failed... token validation failed: token is expired".
-   [ ] **2.5 `rate_limit` Modifier - Under Limit:**
    -   [ ] **Action:** Establish a connection. Send the `send_public_message` event 5 times within 10 seconds (assuming a `10/m` limit).
    -   [ ] **Expected:** All 5 events are successfully processed. Server logs show corresponding `_notify_room` action executions.
-   [ ] **2.6 `rate_limit` Modifier - At Limit and Exceeded:**
    -   [ ] **Action:** Send the `send_public_message` event 12 times within 10 seconds.
    -   [ ] **Expected:** The first 10 events succeed. The 11th and 12th events are rejected. Server logs show "Modifier check failed... rate limit for event 'send_public_message' exceeded".
-   [ ] **2.7 `rate_limit` Modifier - Window Expiration & Reset:**
    -   [ ] **Action:** Trigger the rate limit as in 2.6. Wait for just over one minute (or the configured duration). Send `send_public_message` again.
    -   [ ] **Expected:** The event is processed successfully. Server logs show the action execution.
-   [ ] **2.8 `rate_limit` Modifier - Separate Limits Per User:**
    -   [ ] **Action:** Connect two different clients (User A, User B) with different `UserID`s. Get User A rate-limited. Immediately after, send an event from User B.
    -   [ ] **Expected:** User B's event is processed successfully, demonstrating isolated rate limits.
-   [ ] **2.9 Modifier Chain Execution Order:**
    -   [ ] **Action:** Create a test event with `rate_limit` first, then `secure`. Trigger it by spamming with an `invalid secure` token.
    -   [ ] **Expected:** The `rate_limit` modifier should log its message (if under limit), then the `secure` modifier should fail. If `rate_limit` is exceeded, `secure` should not even run.

---

## ✅ Test Suite 3: Actions (Layer 3 - Execution)

**Goal:** Verify all core actions correctly manipulate state or send messages.

-   [ ] **3.1 `_log` Action:**
    -   [ ] **Action:** Trigger an event that uses `_log`, e.g., `params: ["User {$user.id} logged in"]`.
    -   [ ] **Expected:** The exact message appears in the server's console logs.
-   [ ] **3.2 `_notify_origin` Action:**
    -   [ ] **Action:** Connect a client. Trigger an event that uses `_notify_origin` (e.g., `join_private_room`).
    -   [ ] **Expected:** The client receives a WebSocket message with the specified `event` and `payload`. Other connected clients do *not* receive this message.
-   [ ] **3.3 `_notify_room` Action:**
    -   [ ] **Action:** Connect two clients (User A, User B) to the same room (e.g., "lobby"). User A sends a message via `send_message_to_room`.
    -   [ ] **Expected:** Both User A and User B receive the `new_message` event with the correct payload. Clients not in "lobby" do not receive it.
-   [ ] **3.4 `_join` Action:**
    -   [ ] **Action:** Trigger `join_private_room` successfully (as in 2.1).
    -   [ ] **Expected:** The `_notify_room` action within `join_private_room` successfully sends "user_joined" message to all members of `{$token.room_id}` (including the newly joined user).
    -   [ ] **Verification (Manual/Server State):** If your `StateManager` had an inspection tool, verify `user-123` is now a member of `private-chat`.
-   [ ] **3.5 `_leave` Action:**
    -   [ ] **Action:** Connect a user to a room. Trigger `leave_room` event.
    -   [ ] **Expected:** User is successfully removed. `_notify_room` confirms "user_left" message sent to remaining room members.
    -   [ ] **Verification (Manual/Server State):** Confirm the user is no longer listed as a member. If they were the last member, the room should be removed from `StateManager`.
-   [ ] **3.6 `_set_permissions` Action:**
    -   [ ] **Action:** Trigger `join_private_room` (or a similar event that calls `_set_permissions`).
    -   [ ] **Expected:** Server logs confirm "Set permissions for user in room" with the specified `permissions`.
    -   [ ] **Verification (Manual/Server State/Future `_check_permissions` Action):** This requires direct inspection of `StateManager` or a future `_check_permissions` action to verify the bitmap was updated.

---

## ✅ Test Suite 4: Parameter Resolution & Compile-Time Validation

**Goal:** Verify template parsing (`{.payload...}`, `{$...}`) and compile-time checks for typos.

-   [ ] **4.1 `{.payload.<path>}` Resolution:**
    -   [ ] **Action:** Send a message with a nested payload, e.g., `{"event": "send_message", "target": "lobby", "payload": {"data": {"text": "hello"}}}` and an action parameter like `'{"text": "{.payload.data.text}"}'`.
    -   [ ] **Expected:** The resolved parameter correctly extracts "hello".
-   [ ] **4.2 `{.payload.<path>}` - Missing Payload Path:**
    -   [ ] **Action:** Create a new test event with an action using `{.payload.non.existent.field}`. Send a message that doesn't have this field.
    -   [ ] **Expected:** Server logs show "Failed to resolve params for action, halting pipeline" with an error like "path 'non.existent.field' not found in payload".
-   [ ] **4.3 `{$user.id}` Resolution:**
    -   [ ] **Action:** Trigger any action with `User {$user.id}` in its parameters.
    -   [ ] **Expected:** The `UserID` of the connected client is correctly interpolated.
-   [ ] **4.4 `{$target.id}` Resolution:**
    -   [ ] **Action:** Trigger any action with `Room {$target.id}` in its parameters.
    -   [ ] **Expected:** The `target` value from the client message is correctly interpolated.
-   [ ] **4.5 `{$token.<claim>}` Resolution:**
    -   [ ] **Action:** Trigger `join_private_room` with an event token containing custom claims (e.g., `{"secret_data": "xyz"}`). Use `{$token.secret_data}` in a follow-up action like `_log`.
    -   [ ] **Expected:** The `secret_data` value is correctly interpolated from the validated token.
-   [ ] **4.6 `{$token.<claim>}` - Missing Claim:**
    -   [ ] **Action:** Trigger `join_private_room` with a valid event token, but one that *does not contain* a claim referenced in the `config.yaml` (e.g., `{$token.non_existent_claim}`).
    -   [ ] **Expected:** Server logs show "Failed to resolve params for action... claim 'non_existent_claim' not found in validated token".
-   [ ] **4.7 `{$token.<claim>}` - No `secure` Modifier:**
    -   [ ] **Action:** Create a test event that uses `{$token.sub}` in an action *without* having the `secure` modifier on the event.
    -   [ ] **Expected:** Server logs show "Failed to resolve params for action... requires a validated token, but none was found. Is the 'secure' modifier missing?".
-   [ ] **4.8 Compile-Time Validation - Invalid Context Variable:**
    -   [ ] **Action:** Modify `config.yaml` to include an action with a typo in a context variable, e.g., `params: ["User {$user.idd}"]`. Attempt to start the server (`make run`).
    -   [ ] **Expected:** Server fails to start with a fatal error: "configuration compilation failed: invalid params for action '_log': invalid context variable '{$user.idd}'".
