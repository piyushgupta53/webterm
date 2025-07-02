// WebTerm Application - Stage 1
class WebTermApp {
  constructor() {
    this.statusIndicator = document.getElementById("status-indicator");
    this.statusText = document.getElementById("status-text");
    this.newSessionBtn = document.getElementById("new-session-btn");
    this.refreshBtn = document.getElementById("refresh-btn");
    this.sessionInfo = document.getElementById("session-info");

    this.init();
  }

  init() {
    console.log("WebTerm App initializing...");

    // Setup event listeners
    this.newSessionBtn.addEventListener("click", () => this.createSession());
    this.refreshBtn.addEventListener("click", () => this.refreshStatus());

    // Check server status
    this.checkServerStatus();

    // Update status every 30 seconds
    setInterval(() => this.checkServerStatus(), 30000);
  }

  async checkServerStatus() {
    try {
      console.log("Checking server status...");
      const response = await fetch("/health");

      if (response.ok) {
        const health = await response.json();
        this.updateStatus("connected", `Server healthy (v${health.version})`);
        this.newSessionBtn.disabled = false;
        console.log("Health check response:", health);
      } else {
        throw new Error(`Server returned ${response.status}`);
      }
    } catch (error) {
      console.error("Health check failed:", error);
      this.updateStatus("disconnected", "Server unavailable");
      this.newSessionBtn.disabled = true;
    }
  }

  updateStatus(status, text) {
    this.statusText.textContent = text;

    if (status === "connected") {
      this.statusIndicator.style.color = "#4CAF50";
      this.statusIndicator.style.animation = "pulse 2s infinite";
    } else {
      this.statusIndicator.style.color = "#f44336";
      this.statusIndicator.style.animation = "none";
    }
  }

  createSession() {
    // Placeholder for Stage 2
    console.log("Create session clicked - will be implemented in Stage 2");
    alert(
      "Session creation will be implemented in Stage 2!\n\nFor now, enjoy the working HTTP server 🎉"
    );
  }

  refreshStatus() {
    console.log("Refresh status clicked");
    this.checkServerStatus();
  }
}

// Initialize app when DOM is loaded
document.addEventListener("DOMContentLoaded", () => {
  window.webTermApp = new WebTermApp();
  console.log("WebTerm App loaded successfully!");
});
