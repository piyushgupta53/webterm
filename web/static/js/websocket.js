// WebSocket client for real-time terminal communication
class WebSocketClient {
  constructor() {
    this.ws = null;
    this.connected = false;
    this.connecting = false;
    this.sessionId = null;
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = 5;
    this.reconnectDelay = 1000;
    this.messageHandlers = new Map();
    this.connectionCallbacks = new Set();
    this.terminated = false; // Flag to prevent reconnection for terminated sessions

    // Heartbeat
    this.pingInterval = null;
    this.pongTimeout = null;
    this.heartbeatInterval = 30000; // 30 seconds
    this.pongTimeoutDuration = 10000; // 10 seconds
  }

  // Event handler registration
  on(event, handler) {
    if (!this.messageHandlers.has(event)) {
      this.messageHandlers.set(event, new Set());
    }
    this.messageHandlers.get(event).add(handler);
  }

  off(event, handler) {
    if (this.messageHandlers.has(event)) {
      this.messageHandlers.get(event).delete(handler);
    }
  }

  emit(event, data) {
    if (this.messageHandlers.has(event)) {
      this.messageHandlers.get(event).forEach((handler) => {
        try {
          handler(data);
        } catch (error) {
          console.error("Error in event handler:", error);
        }
      });
    }
  }

  // Connection management
  connect(sessionId) {
    if (this.connecting || this.connected) {
      return Promise.resolve();
    }

    // Don't connect if session is terminated
    if (this.terminated) {
      return Promise.reject(new Error("Session is terminated"));
    }

    this.connecting = true;
    this.sessionId = sessionId;

    return new Promise((resolve, reject) => {
      const wsUrl = `ws://${window.location.host}/api/ws?session=${sessionId}`;
      this.ws = new WebSocket(wsUrl);

      this.setupEventHandlers(resolve, reject);
    });
  }

  setupEventHandlers(resolve, reject) {
    const connectionTimeout = setTimeout(() => {
      if (this.connecting) {
        this.connecting = false;
        this.ws.close();
        reject(new Error("Connection timeout"));
      }
    }, 10000);

    this.ws.onopen = () => {
      clearTimeout(connectionTimeout);
      this.connected = true;
      this.connecting = false;
      this.reconnectAttempts = 0;

      console.log("WebSocket connected to session:", this.sessionId);
      this.startHeartbeat();
      this.emit("connected", { sessionId: this.sessionId });
      resolve();
    };

    this.ws.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data);
        this.handleMessage(message);
      } catch (error) {
        console.error("Error parsing WebSocket message:", error);
      }
    };

    this.ws.onclose = (event) => {
      clearTimeout(connectionTimeout);
      this.connected = false;
      this.connecting = false;
      this.stopHeartbeat();

      console.log("WebSocket disconnected:", event.code, event.reason);
      this.emit("disconnected", { code: event.code, reason: event.reason });

      // Auto-reconnect if not a normal closure and session is not terminated
      if (
        event.code !== 1000 &&
        this.reconnectAttempts < this.maxReconnectAttempts &&
        !this.terminated
      ) {
        this.scheduleReconnect();
      }
    };

    this.ws.onerror = (error) => {
      clearTimeout(connectionTimeout);
      console.error("WebSocket error:", error);
      this.emit("error", error);

      if (this.connecting) {
        this.connecting = false;
        reject(error);
      }
    };
  }

  scheduleReconnect() {
    this.reconnectAttempts++;
    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);

    console.log(
      `Scheduling reconnect attempt ${this.reconnectAttempts} in ${delay}ms`
    );

    setTimeout(() => {
      if (!this.connected && this.sessionId) {
        console.log("Attempting to reconnect...");
        this.connect(this.sessionId).catch((error) => {
          console.error("Reconnection failed:", error);
        });
      }
    }, delay);
  }

  handleMessage(message) {
    console.log("Received WebSocket message:", message);

    switch (message.type) {
      case "output":
        this.emit("output", message.data);
        break;
      case "status":
        this.emit("status", {
          sessionId: message.session_id,
          status: message.status,
        });

        // Handle session termination
        if (message.status === "stopped" || message.status === "error") {
          console.log("Session terminated via WebSocket:", message.session_id);
          this.terminated = true; // Prevent reconnection attempts
          this.emit("session_terminated", {
            sessionId: message.session_id,
            status: message.status,
          });
        }
        break;
      case "error":
        this.emit("error", message.error);
        break;
      case "pong":
        this.handlePong();
        break;
      case "connected":
        this.emit("session_connected", { sessionId: message.session_id });
        break;
      default:
        console.log("Unknown message type:", message.type);
    }
  }

  // Message sending
  send(type, data = {}) {
    if (!this.connected) {
      console.warn("Cannot send message: WebSocket not connected");
      return false;
    }

    const message = {
      type,
      timestamp: new Date().toISOString(),
      ...data,
    };

    try {
      this.ws.send(JSON.stringify(message));
      return true;
    } catch (error) {
      console.error("Error sending WebSocket message:", error);
      return false;
    }
  }

  sendInput(data) {
    return this.send("input", { data });
  }

  sendResize(rows, cols) {
    return this.send("resize", { rows, cols });
  }

  sendPing() {
    return this.send("ping");
  }

  // Heartbeat management
  startHeartbeat() {
    this.stopHeartbeat();

    this.pingInterval = setInterval(() => {
      if (this.connected) {
        this.sendPing();

        // Set timeout for pong response
        this.pongTimeout = setTimeout(() => {
          console.warn("Pong timeout - connection may be dead");
          this.ws.close();
        }, this.pongTimeoutDuration);
      }
    }, this.heartbeatInterval);
  }

  stopHeartbeat() {
    if (this.pingInterval) {
      clearInterval(this.pingInterval);
      this.pingInterval = null;
    }

    if (this.pongTimeout) {
      clearTimeout(this.pongTimeout);
      this.pongTimeout = null;
    }
  }

  handlePong() {
    if (this.pongTimeout) {
      clearTimeout(this.pongTimeout);
      this.pongTimeout = null;
    }
  }

  // Connection status
  isConnected() {
    return this.connected;
  }

  isConnecting() {
    return this.connecting;
  }

  getCurrentSessionId() {
    return this.sessionId;
  }

  // Cleanup
  disconnect() {
    this.stopHeartbeat();

    if (this.ws) {
      this.ws.close(1000, "Client disconnect");
      this.ws = null;
    }

    this.connected = false;
    this.connecting = false;
    this.sessionId = null;
    this.reconnectAttempts = 0;
    this.terminated = false; // Reset terminated flag on disconnect
  }

  destroy() {
    this.disconnect();
    this.messageHandlers.clear();
    this.connectionCallbacks.clear();
  }
}

// Export for use in other modules
window.WebSocketClient = WebSocketClient;
