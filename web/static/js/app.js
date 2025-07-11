// Main WebTerm application
class WebTermApp {
  constructor() {
    this.sessionManager = null;
    this.terminalManager = null;
    this.websocketClient = null;
    this.isInitialized = false;

    // Connection status
    this.connectionStatus = "disconnected";

    // DOM elements
    this.elements = {};

    // Bind methods
    this.handleSessionSwitch = this.handleSessionSwitch.bind(this);
    this.handleTerminalClear = this.handleTerminalClear.bind(this);
    this.handleTerminalDisconnect = this.handleTerminalDisconnect.bind(this);
    this.updateConnectionStatus = this.updateConnectionStatus.bind(this);
  }

  // Initialize the application
  async initialize() {
    try {
      console.log("Initializing WebTerm application...");

      this.bindElements();
      this.setupEventHandlers();

      // Initialize managers
      await this.initializeManagers();

      // Setup initial state
      this.setupInitialState();

      this.isInitialized = true;
      console.log("WebTerm application initialized successfully");
    } catch (error) {
      console.error("Failed to initialize WebTerm application:", error);
      this.showErrorState(error);
    }
  }

  bindElements() {
    this.elements = {
      connectionIndicator: document.getElementById("connection-indicator"),
      connectionStatus: document.getElementById("connection-status"),
      terminalContainer: document.getElementById("terminal-container"),
      terminalPlaceholder: document.getElementById("terminal-placeholder"),
    };
  }

  setupEventHandlers() {
    // Application-level event handlers
    window.addEventListener("sessionSwitch", this.handleSessionSwitch);
    window.addEventListener("terminalClear", this.handleTerminalClear);
    window.addEventListener(
      "terminalDisconnect",
      this.handleTerminalDisconnect
    );

    // Connection status updates
    window.addEventListener(
      "connectionStatusChange",
      this.updateConnectionStatus
    );

    // Window events
    window.addEventListener("beforeunload", () => {
      this.cleanup();
    });

    // Visibility change (for connection management)
    document.addEventListener("visibilitychange", () => {
      if (document.visibilityState === "visible") {
        this.handlePageVisible();
      } else {
        this.handlePageHidden();
      }
    });
  }

  async initializeManagers() {
    // Initialize WebSocket client
    this.websocketClient = new WebSocketClient();
    this.setupWebSocketEventHandlers();

    // Initialize terminal manager
    this.terminalManager = new TerminalManager();

    // Initialize session manager
    this.sessionManager = new SessionManager();
    this.sessionManager.initialize();

    console.log("All managers initialized");
  }

  setupWebSocketEventHandlers() {
    this.websocketClient.on("connected", () => {
      this.setConnectionStatus("connected");
    });

    this.websocketClient.on("disconnected", () => {
      this.setConnectionStatus("disconnected");
    });

    this.websocketClient.on("error", (error) => {
      console.error("WebSocket error:", error);
      this.setConnectionStatus("error");
    });

    // Handle session status updates
    this.websocketClient.on("status", (data) => {
      console.log("Session status update:", data);
      if (this.sessionManager) {
        this.sessionManager.updateSessionStatus(data.sessionId, data.status);
      }
    });

    // Handle session termination
    this.websocketClient.on("session_terminated", (data) => {
      console.log("Session terminated via WebSocket:", data);
      if (this.sessionManager) {
        // Remove the terminated session from the UI
        this.sessionManager.sessions.delete(data.sessionId);
        this.sessionManager.updateSessionsList(
          Array.from(this.sessionManager.sessions.values())
        );
        this.sessionManager.removeSessionTab(data.sessionId);

        // If this was the current session, switch to another or show placeholder
        if (this.sessionManager.currentSessionId === data.sessionId) {
          this.sessionManager.currentSessionId = null;
          const remainingSessions = Array.from(
            this.sessionManager.sessions.keys()
          );
          if (remainingSessions.length > 0) {
            this.sessionManager.switchToSession(remainingSessions[0]);
          } else {
            this.sessionManager.showPlaceholder();
          }
        }

        this.sessionManager.showNotification("Session terminated", "info");
      }
    });
  }

  setupInitialState() {
    // Set initial connection status
    this.setConnectionStatus("disconnected");

    // Show placeholder initially
    this.showPlaceholder();

    // Load initial sessions
    this.sessionManager.loadSessions();
  }

  // Event handlers
  async handleSessionSwitch(event) {
    const { sessionId, session } = event.detail;

    try {
      console.log("Switching to session:", sessionId);

      // Initialize terminal if not already done
      if (!this.terminalManager.isInitialized) {
        await this.terminalManager.initialize(
          this.elements.terminalContainer,
          this.websocketClient
        );
      }

      // Connect to session
      await this.terminalManager.connectToSession(sessionId);

      // Hide placeholder
      this.hidePlaceholder();

      // Show welcome message for new sessions
      if (session.status === "running") {
        this.terminalManager.showWelcomeMessage();
      }

      console.log("Successfully switched to session:", sessionId);
    } catch (error) {
      console.error("Failed to switch session:", error);
      this.sessionManager.showNotification(
        `Failed to connect to session: ${error.message}`,
        "error"
      );
    }
  }

  handleTerminalClear() {
    if (this.terminalManager) {
      this.terminalManager.clear();
    }
  }

  handleTerminalDisconnect() {
    if (this.terminalManager) {
      this.terminalManager.disconnect();
      this.setConnectionStatus("disconnected");
    }
  }

  handlePageVisible() {
    // Optionally reconnect or refresh when page becomes visible
    console.log("Page became visible");
  }

  handlePageHidden() {
    // Optionally handle page hide (e.g., pause heartbeat)
    console.log("Page became hidden");
  }

  // Connection status management
  setConnectionStatus(status) {
    this.connectionStatus = status;
    this.updateConnectionStatus();
  }

  updateConnectionStatus() {
    if (!this.elements.connectionIndicator || !this.elements.connectionStatus) {
      return;
    }

    // Remove existing status classes
    this.elements.connectionIndicator.classList.remove(
      "connected",
      "connecting",
      "disconnected",
      "error"
    );

    switch (this.connectionStatus) {
      case "connected":
        this.elements.connectionIndicator.classList.add("connected");
        this.elements.connectionStatus.textContent = "Connected";
        break;

      case "connecting":
        this.elements.connectionIndicator.classList.add("connecting");
        this.elements.connectionStatus.textContent = "Connecting...";
        break;

      case "disconnected":
        this.elements.connectionIndicator.classList.add("disconnected");
        this.elements.connectionStatus.textContent = "Disconnected";
        break;

      case "error":
        this.elements.connectionIndicator.classList.add("error");
        this.elements.connectionStatus.textContent = "Connection Error";
        break;

      default:
        this.elements.connectionIndicator.classList.add("disconnected");
        this.elements.connectionStatus.textContent = "Unknown";
    }
  }

  // UI management
  showPlaceholder() {
    if (this.elements.terminalPlaceholder) {
      this.elements.terminalPlaceholder.style.display = "flex";
    }
  }

  hidePlaceholder() {
    if (this.elements.terminalPlaceholder) {
      this.elements.terminalPlaceholder.style.display = "none";
    }
  }

  showErrorState(error) {
    console.error("Application error state:", error);

    // Show error notification
    if (this.sessionManager) {
      this.sessionManager.showNotification(
        `Application error: ${error.message}`,
        "error"
      );
    }

    // Update connection status
    this.setConnectionStatus("error");
  }

  // Health checking
  async checkHealth() {
    try {
      const response = await fetch("/health");
      if (response.ok) {
        const health = await response.json();
        console.log("Health check passed:", health);
        return true;
      } else {
        console.warn("Health check failed:", response.status);
        return false;
      }
    } catch (error) {
      console.error("Health check error:", error);
      return false;
    }
  }

  async startHealthChecking() {
    // Check health periodically
    setInterval(async () => {
      const healthy = await this.checkHealth();
      if (!healthy && this.connectionStatus === "connected") {
        this.setConnectionStatus("error");
      }
    }, 30000); // Check every 30 seconds
  }

  // Keyboard shortcuts
  setupKeyboardShortcuts() {
    document.addEventListener("keydown", (e) => {
      // Global shortcuts (when not in terminal)
      if (!e.target.closest(".xterm")) {
        // Ctrl/Cmd + K: Focus terminal
        if ((e.ctrlKey || e.metaKey) && e.key === "k") {
          e.preventDefault();
          if (this.terminalManager) {
            this.terminalManager.focus();
          }
        }

        // Ctrl/Cmd + R: Refresh sessions
        if ((e.ctrlKey || e.metaKey) && e.key === "r") {
          e.preventDefault();
          if (this.sessionManager) {
            this.sessionManager.loadSessions();
          }
        }
      }

      // Terminal shortcuts (global)
      if (this.terminalManager && this.terminalManager.isConnected()) {
        // Ctrl/Cmd + Shift + C: Copy selection
        if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key === "C") {
          e.preventDefault();
          this.terminalManager.copySelection();
        }

        // Ctrl/Cmd + Shift + V: Paste
        if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key === "V") {
          e.preventDefault();
          this.terminalManager.paste();
        }
      }
    });
  }

  // Cleanup
  async cleanup() {
    console.log("Cleaning up WebTerm application...");

    try {
      // Cleanup terminal
      if (this.terminalManager) {
        await this.terminalManager.cleanup();
      }

      // Cleanup WebSocket
      if (this.websocketClient) {
        this.websocketClient.destroy();
      }

      // Remove event listeners
      window.removeEventListener("sessionSwitch", this.handleSessionSwitch);
      window.removeEventListener("terminalClear", this.handleTerminalClear);
      window.removeEventListener(
        "terminalDisconnect",
        this.handleTerminalDisconnect
      );
      window.removeEventListener(
        "connectionStatusChange",
        this.updateConnectionStatus
      );

      console.log("WebTerm application cleanup completed");
    } catch (error) {
      console.error("Error during cleanup:", error);
    }
  }

  // Getters
  getConnectionStatus() {
    return this.connectionStatus;
  }

  isConnected() {
    return this.connectionStatus === "connected";
  }

  getCurrentSession() {
    return this.sessionManager ? this.sessionManager.getCurrentSession() : null;
  }
}

// Initialize application when DOM is loaded
document.addEventListener("DOMContentLoaded", async () => {
  try {
    // Create global app instance
    window.webTermApp = new WebTermApp();

    // Initialize the application
    await window.webTermApp.initialize();

    // Start health checking
    window.webTermApp.startHealthChecking();

    // Setup keyboard shortcuts
    window.webTermApp.setupKeyboardShortcuts();

    console.log("🚀 WebTerm is ready!");
  } catch (error) {
    console.error("Failed to start WebTerm application:", error);

    // Show basic error message
    document.body.innerHTML = `
          <div style="display: flex; align-items: center; justify-content: center; height: 100vh; background: #0f1419; color: #fff; font-family: sans-serif;">
              <div style="text-align: center;">
                  <h1>⚠️ WebTerm Failed to Load</h1>
                  <p>Please check the browser console for details and refresh the page.</p>
                  <button onclick="location.reload()" style="margin-top: 20px; padding: 10px 20px; background: #4CAF50; color: white; border: none; border-radius: 4px; cursor: pointer;">
                      Reload Page
                  </button>
              </div>
          </div>
      `;
  }
});
