let ws = null;
let currentUserId = null;
function generateRandomString(length) {
  const chars =
    "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
  let result = "";
  for (let i = 0; i < length; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length));
  }
  return result;
}
const userID = generateRandomString(10);
// Generate JWT for GoDispatch
function generateJWT(userId) {
  const secret = "a-very-secret-key"; // WARNING: Don't use this in production
  const header = { alg: "HS256", typ: "JWT" };

  const now = Math.floor(Date.now() / 1000);
  const payload = {
    iss: "your-app", // Issuer
    sub: userId, // Subject
    aud: ["your-client-id"], // Audience (as an array)
    exp: now + 3600, // Expires in 1 hour
    nbf: now, // Not before
    iat: now, // Issued at
    jti: crypto.randomUUID(), // JWT ID (unique token ID)
  };

  const token = KJUR.jws.JWS.sign(
    null,
    JSON.stringify(header),
    JSON.stringify(payload),
    { utf8: secret },
  );
  return token;
}
function setSessionCookie(token) {
  document.cookie = `session-token=${token}; path=/; max-age=3600; samesite=lax`;
}
// Connect to GoDispatch WebSocket
function connectWebSocket() {
  const usernameInput = document.getElementById("username-input");
  const connectionError = document.getElementById("connection-error");
  currentUserId = usernameInput.value.trim();

  if (!currentUserId) {
    connectionError.textContent = "Please enter a username";
    connectionError.style.display = "block";
    return;
  }

  const sessionToken = generateJWT(userID);
  setSessionCookie(sessionToken);
  ws = new WebSocket(`ws://127.0.0.1:8080/ws`);

  ws.onopen = () => {
    updateConnectionStatus("connected");
    displayMessage("system", "Connected to GoDispatch!");
    // Hide modal and show chat
    document.getElementById("username-modal").style.display = "none";
    document.getElementById("chat-container").style.display = "flex";
    // Join the lobby room
    const joinMessage = {
      event: "join_room",
      target: "room:global",
      payload: {
        name: currentUserId,
      },
    };
    ws.send(JSON.stringify(joinMessage));
  };

  ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    console.log(data);
    if (data.event === "join_success") {
      displayMessage("system", `Joined room: ${data.payload.room}`);
    } else if (data.event === "user_joined") {
      displayMessage(
        "system",
        `User ${data.payload.user} joined ${data.payload.room}`,
      );
    } else if (data.event === "new_message") {
      displayMessage(
        data.payload.user,
        data.payload.message,
        data.payload.user === currentUserId,
      );
    }
  };

  ws.onclose = () => {
    updateConnectionStatus("disconnected");
    displayMessage("system", "Disconnected from server");
    // Show modal again if disconnected
    document.getElementById("username-modal").style.display = "flex";
    document.getElementById("chat-container").style.display = "none";
    ws = null;
  };

  ws.onerror = (error) => {
    console.error("WebSocket error:", error);
    connectionError.textContent = "Failed to connect. Please try again.";
    connectionError.style.display = "block";
  };
}

// Send a message
function sendMessage() {
  if (!ws || ws.readyState !== WebSocket.OPEN) {
    displayMessage("system", "Not connected to server");
    return;
  }
  const messageText = document.getElementById("message-input").value.trim();
  if (!messageText) {
    return;
  }
  const message = {
    event: "send_message",
    target: "room:global",
    payload: { message: messageText },
  };
  ws.send(JSON.stringify(message));
  document.getElementById("message-input").value = ""; // Clear input
}

// Display a message in the chat area
function displayMessage(user, message, isSender = false) {
  const chatArea = document.getElementById("chat-area");
  const messageDiv = document.createElement("div");
  messageDiv.className = `message ${isSender ? "sender" : "other"}`;
  const timestamp = new Date().toISOString().slice(0, 16).replace("T", " ");
  messageDiv.innerHTML = `
    <div class="username">${user}</div>
    <div>${message}</div>
    <div class="timestamp">${timestamp}</div>
  `;
  chatArea.appendChild(messageDiv);
  chatArea.scrollTop = chatArea.scrollHeight; // Auto-scroll to bottom
}

// Update connection status
function updateConnectionStatus(status) {
  const statusElement = document.getElementById("connection-status");
  statusElement.textContent = status.charAt(0).toUpperCase() + status.slice(1);
  statusElement.classList.toggle("connected", status === "connected");
}

// Event listeners
document.getElementById("message-input").addEventListener("keypress", (e) => {
  if (e.key === "Enter") {
    sendMessage();
  }
});

document.getElementById("username-input").addEventListener("keypress", (e) => {
  if (e.key === "Enter") {
    connectWebSocket();
  }
});

// Show modal on page load
document.addEventListener("DOMContentLoaded", () => {
  document.getElementById("username-modal").style.display = "flex";
  document.getElementById("chat-container").style.display = "none";
});
