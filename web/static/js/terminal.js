// Terminal management with Xterm.js integration
class TerminalManager {
  constructor() {
    this.terminal = null;
    this.fitAddon = null;
    this.isInitialized = false;
    this.sessionId = null;
    this.websocketClient = null;

    // Terminal configuration
    this.config = {
      theme: {
        background: "#000000",
        foreground: "#ffffff",
        cursor: "#4CAF50",
        cursorAccent: "#4CAF50",
        selection: "rgba(76, 175, 80, 0.3)",
        black: "#000000",
        red: "#ff5555",
        green: "#50fa7b",
        yellow: "#f1fa8c",
        blue: "#bd93f9",
        magenta: "#ff79c6",
        cyan: "#8be9fd",
        white: "#bfbfbf",
        brightBlack: "#4d4d4d",
        brightRed: "#ff6e67",
        brightGreen: "#5af78e",
        brightYellow: "#f4f99d",
        brightBlue: "#caa9fa",
        brightMagenta: "#ff92d0",
        brightCyan: "#9aedfe",
        brightWhite: "#e6e6e6",
      },
      fontSize: 14,
      fontFamily:
        "'SF Mono', 'Monaco', 'Inconsolata', 'Roboto Mono', monospace",
      cursorBlink: true,
      cursorStyle: "block",
      scrollback: 1000,
      tabStopWidth: 4,
      bellStyle: "none",
    };

    // Bind methods
    this.handleResize = this.handleResize.bind(this);
    this.handleData = this.handleData.bind(this);
  }

  // Initialize terminal
  async initialize(containerElement, websocketClient) {
    try {
      if (this.isInitialized) {
        await this.cleanup();
      }

      this.websocketClient = websocketClient;

      // Create terminal instance
      this.terminal = new Terminal(this.config);

      // Create fit addon
      this.fitAddon = new FitAddon.FitAddon();
      this.terminal.loadAddon(this.fitAddon);

      // Open terminal in container
      this.terminal.open(containerElement);

      // Setup event handlers
      this.setupEventHandlers();

      // Initial fit
      this.fit();

      this.isInitialized = true;
      console.log("Terminal initialized successfully");

      return true;
    } catch (error) {
      console.error("Failed to initialize terminal:", error);
      throw error;
    }
  }

  setupEventHandlers() {
    // Handle user input
    this.terminal.onData(this.handleData);

    // Handle terminal resize
    this.terminal.onResize(this.handleResize);

    // Handle selection changes
    this.terminal.onSelectionChange(() => {
      // Could implement copy functionality here
    });

    // Setup WebSocket event handlers
    if (this.websocketClient) {
      this.websocketClient.on("output", (data) => {
        this.writeOutput(data);
      });

      this.websocketClient.on("disconnected", () => {
        this.showConnectionStatus("Disconnected", "error");
      });

      this.websocketClient.on("connected", () => {
        this.showConnectionStatus("Connected", "success");
      });

      this.websocketClient.on("error", (error) => {
        this.showConnectionStatus(`Error: ${error}`, "error");
      });
    }

    // Handle window resize
    window.addEventListener("resize", () => {
      clearTimeout(this.resizeTimeout);
      this.resizeTimeout = setTimeout(() => {
        this.fit();
      }, 100);
    });
  }

  handleData(data) {
    if (this.websocketClient && this.websocketClient.isConnected()) {
      this.websocketClient.sendInput(data);
    } else {
      console.warn("Cannot send input: WebSocket not connected");
    }
  }

  handleResize(size) {
    if (this.websocketClient && this.websocketClient.isConnected()) {
      this.websocketClient.sendResize(size.rows, size.cols);
    }
  }

  // Connect to session
  async connectToSession(sessionId) {
    try {
      if (this.sessionId === sessionId && this.websocketClient.isConnected()) {
        return true;
      }

      this.sessionId = sessionId;

      // Clear terminal
      this.clear();
      this.showConnectionStatus("Connecting...", "warning");

      // Connect WebSocket
      await this.websocketClient.connect(sessionId);

      // Focus terminal
      this.focus();

      return true;
    } catch (error) {
      console.error("Failed to connect to session:", error);
      this.showConnectionStatus(`Connection failed: ${error.message}`, "error");
      throw error;
    }
  }

  // Terminal operations
  writeOutput(data) {
    if (this.terminal) {
      this.terminal.write(data);
    }
  }

  clear() {
    if (this.terminal) {
      this.terminal.clear();
    }
  }

  focus() {
    if (this.terminal) {
      this.terminal.focus();
    }
  }

  blur() {
    if (this.terminal) {
      this.terminal.blur();
    }
  }

  fit() {
    if (this.fitAddon && this.terminal) {
      try {
        this.fitAddon.fit();
      } catch (error) {
        console.warn("Failed to fit terminal:", error);
      }
    }
  }

  // Utility methods
  showConnectionStatus(message, type = "info") {
    if (!this.terminal) return;

    const colors = {
      info: "\x1b[36m", // Cyan
      success: "\x1b[32m", // Green
      warning: "\x1b[33m", // Yellow
      error: "\x1b[31m", // Red
      reset: "\x1b[0m", // Reset
    };

    const color = colors[type] || colors.info;
    const statusMessage = `${color}[WebTerm] ${message}${colors.reset}\r\n`;

    this.terminal.write(statusMessage);
  }

  showWelcomeMessage() {
    if (!this.terminal) return;

    const welcomeMessage = [
      "\x1b[32m╭─────────────────────────────────────╮\x1b[0m\r\n",
      "\x1b[32m│       Welcome to WebTerm! 🚀       │\x1b[0m\r\n",
      "\x1b[32m╰─────────────────────────────────────╯\x1b[0m\r\n",
      "\x1b[36mTerminal session ready. Happy coding!\x1b[0m\r\n\r\n",
    ].join("");

    this.terminal.write(welcomeMessage);
  }

  // Theming
  setTheme(theme) {
    if (this.terminal) {
      this.terminal.setOption("theme", { ...this.config.theme, ...theme });
    }
  }

  setFontSize(fontSize) {
    if (this.terminal) {
      this.terminal.setOption("fontSize", fontSize);
      this.fit();
    }
  }

  // Clipboard operations
  async copySelection() {
    if (this.terminal && this.terminal.hasSelection()) {
      const selection = this.terminal.getSelection();
      try {
        await navigator.clipboard.writeText(selection);
        return true;
      } catch (error) {
        console.error("Failed to copy to clipboard:", error);
        return false;
      }
    }
    return false;
  }

  async paste() {
    try {
      const text = await navigator.clipboard.readText();
      if (text && this.websocketClient && this.websocketClient.isConnected()) {
        this.websocketClient.sendInput(text);
        return true;
      }
    } catch (error) {
      console.error("Failed to paste from clipboard:", error);
    }
    return false;
  }

  // Session management
  disconnect() {
    if (this.websocketClient) {
      this.websocketClient.disconnect();
    }
    this.sessionId = null;
    this.showConnectionStatus("Disconnected", "warning");
  }

  getCurrentSessionId() {
    return this.sessionId;
  }

  isConnected() {
    return this.websocketClient && this.websocketClient.isConnected();
  }

  // Cleanup
  async cleanup() {
    console.log("Cleaning up terminal...");

    // Disconnect WebSocket
    if (this.websocketClient) {
      this.websocketClient.disconnect();
    }

    // Dispose terminal
    if (this.terminal) {
      this.terminal.dispose();
      this.terminal = null;
    }

    // Clear addons
    this.fitAddon = null;

    // Remove event listeners
    window.removeEventListener("resize", this.handleResize);

    // Clear timeouts
    if (this.resizeTimeout) {
      clearTimeout(this.resizeTimeout);
    }

    this.isInitialized = false;
    this.sessionId = null;
  }
}

// Export for use in other modules
window.TerminalManager = TerminalManager;
