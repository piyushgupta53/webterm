// Session management and UI interactions
class SessionManager {
  constructor() {
    this.sessions = new Map();
    this.currentSessionId = null;
    this.apiBaseUrl = "/api";

    // DOM elements
    this.elements = {};

    // Bind methods
    this.handleSessionCreate = this.handleSessionCreate.bind(this);
    this.handleSessionSelect = this.handleSessionSelect.bind(this);
    this.handleTabClick = this.handleTabClick.bind(this);
    this.handleTabClose = this.handleTabClose.bind(this);
  }

  // Initialize session manager
  initialize() {
    this.bindElements();
    this.setupEventHandlers();
    this.loadSessions();

    console.log("Session manager initialized");
  }

  bindElements() {
    this.elements = {
      // Buttons
      newSessionBtn: document.getElementById("new-session-btn"),
      refreshBtn: document.getElementById("refresh-btn"),
      quickNewSessionBtn: document.getElementById("quick-new-session"),
      tabAddBtn: document.getElementById("tab-add-btn"),

      // Session controls
      sessionSelect: document.getElementById("session-select"),
      currentSessionId: document.getElementById("current-session-id"),
      currentSessionStatus: document.getElementById("current-session-status"),

      // Terminal controls
      terminalClear: document.getElementById("terminal-clear"),
      terminalDisconnect: document.getElementById("terminal-disconnect"),
      terminalTerminate: document.getElementById("terminal-terminate"),

      // Modal
      modalOverlay: document.getElementById("modal-overlay"),
      sessionModal: document.getElementById("session-modal"),
      sessionForm: document.getElementById("session-form"),
      sessionShell: document.getElementById("session-shell"),
      sessionWorkdir: document.getElementById("session-workdir"),
      sessionCancel: document.getElementById("session-cancel"),
      modalClose: document.getElementById("modal-close"),

      // Tabs
      tabsContainer: document.getElementById("tabs-container"),

      // Loading
      loadingOverlay: document.getElementById("loading-overlay"),
      loadingText: document.getElementById("loading-text"),

      // Notifications
      notifications: document.getElementById("notifications"),
    };
  }

  setupEventHandlers() {
    // Session creation buttons
    this.elements.newSessionBtn?.addEventListener("click", () =>
      this.showCreateSessionModal()
    );
    this.elements.quickNewSessionBtn?.addEventListener("click", () =>
      this.createSession({})
    );
    this.elements.tabAddBtn?.addEventListener("click", () =>
      this.showCreateSessionModal()
    );

    // Session controls
    this.elements.refreshBtn?.addEventListener("click", () =>
      this.loadSessions()
    );
    this.elements.sessionSelect?.addEventListener(
      "change",
      this.handleSessionSelect
    );

    // Terminal controls
    this.elements.terminalClear?.addEventListener("click", () =>
      this.clearTerminal()
    );
    this.elements.terminalDisconnect?.addEventListener("click", () =>
      this.disconnectSession()
    );
    this.elements.terminalTerminate?.addEventListener("click", () =>
      this.terminateCurrentSession()
    );

    // Modal handlers
    this.elements.modalClose?.addEventListener("click", () =>
      this.hideCreateSessionModal()
    );
    this.elements.sessionCancel?.addEventListener("click", () =>
      this.hideCreateSessionModal()
    );
    this.elements.sessionForm?.addEventListener(
      "submit",
      this.handleSessionCreate
    );
    this.elements.modalOverlay?.addEventListener("click", (e) => {
      if (e.target === this.elements.modalOverlay) {
        this.hideCreateSessionModal();
      }
    });

    // Keyboard shortcuts
    document.addEventListener("keydown", (e) => {
      // Ctrl/Cmd + T: New session
      if ((e.ctrlKey || e.metaKey) && e.key === "t") {
        e.preventDefault();
        this.createSession({});
      }

      // Ctrl/Cmd + W: Close current session
      if ((e.ctrlKey || e.metaKey) && e.key === "w") {
        e.preventDefault();
        this.terminateCurrentSession();
      }

      // Escape: Close modal
      if (e.key === "Escape") {
        this.hideCreateSessionModal();
      }
    });
  }

  // Session management
  async loadSessions() {
    try {
      const response = await fetch(`${this.apiBaseUrl}/sessions`);
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      const data = await response.json();
      this.updateSessionsList(data.sessions);
    } catch (error) {
      console.error("Failed to load sessions:", error);
      this.showNotification("Failed to load sessions", "error");
    }
  }

  async createSession(config = {}) {
    try {
      this.showLoading("Creating session...");

      const response = await fetch(`${this.apiBaseUrl}/sessions`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          shell: config.shell || "",
          working_dir: config.workingDir || "",
          env: config.env || {},
        }),
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      const data = await response.json();
      const session = data.session;

      // Add to sessions map
      this.sessions.set(session.id, session);

      // Update UI
      this.updateSessionsList(Array.from(this.sessions.values()));
      this.addSessionTab(session);

      // Switch to new session
      await this.switchToSession(session.id);

      this.showNotification("Session created successfully", "success");
      this.hideCreateSessionModal();

      return session;
    } catch (error) {
      console.error("Failed to create session:", error);
      this.showNotification(
        `Failed to create session: ${error.message}`,
        "error"
      );
      throw error;
    } finally {
      this.hideLoading();
    }
  }

  async terminateSession(sessionId) {
    try {
      // Immediately update UI to show termination in progress
      this.updateSessionStatus(sessionId, "stopping");

      const response = await fetch(`${this.apiBaseUrl}/sessions/${sessionId}`, {
        method: "DELETE",
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      // Remove from sessions map
      this.sessions.delete(sessionId);

      // Update UI
      this.updateSessionsList(Array.from(this.sessions.values()));
      this.removeSessionTab(sessionId);

      // Switch to another session if current was terminated
      if (this.currentSessionId === sessionId) {
        this.currentSessionId = null;
        const remainingSessions = Array.from(this.sessions.keys());
        if (remainingSessions.length > 0) {
          await this.switchToSession(remainingSessions[0]);
        } else {
          this.showPlaceholder();
        }
      }

      this.showNotification("Session terminated", "info");
    } catch (error) {
      console.error("Failed to terminate session:", error);
      this.showNotification(
        `Failed to terminate session: ${error.message}`,
        "error"
      );

      // Revert UI state if termination failed
      this.loadSessions();
    }
  }

  async terminateCurrentSession() {
    if (this.currentSessionId) {
      const confirmed = confirm(
        "Are you sure you want to terminate this session?"
      );
      if (confirmed) {
        await this.terminateSession(this.currentSessionId);
      }
    }
  }

  async switchToSession(sessionId) {
    if (this.currentSessionId === sessionId) {
      return;
    }

    try {
      const session = this.sessions.get(sessionId);
      if (!session) {
        throw new Error("Session not found");
      }

      // Check if session is in a valid state for switching
      if (session.status === "stopped" || session.status === "error") {
        throw new Error(`Cannot switch to session in ${session.status} state`);
      }

      this.currentSessionId = sessionId;

      // Update UI
      this.updateCurrentSessionInfo(session);
      this.updateSessionTabs();
      this.updateTerminalControls(true);
      this.hidePlaceholder();

      // Emit event for terminal manager
      window.dispatchEvent(
        new CustomEvent("sessionSwitch", {
          detail: { sessionId, session },
        })
      );

      console.log("Switched to session:", sessionId);
    } catch (error) {
      console.error("Failed to switch to session:", error);
      this.showNotification(
        `Failed to switch to session: ${error.message}`,
        "error"
      );

      // If switching failed, try to switch to another available session
      const availableSessions = Array.from(this.sessions.values()).filter(
        (s) => s.status === "running" || s.status === "starting"
      );

      if (availableSessions.length > 0) {
        await this.switchToSession(availableSessions[0].id);
      } else {
        this.showPlaceholder();
      }
    }
  }

  // UI Updates
  updateSessionsList(sessions) {
    // Update sessions map
    this.sessions.clear();
    sessions.forEach((session) => {
      this.sessions.set(session.id, session);
    });

    // Update select dropdown
    if (this.elements.sessionSelect) {
      this.elements.sessionSelect.innerHTML = "";

      if (sessions.length === 0) {
        const option = document.createElement("option");
        option.value = "";
        option.textContent = "No sessions";
        this.elements.sessionSelect.appendChild(option);
        this.elements.sessionSelect.disabled = true;
      } else {
        this.elements.sessionSelect.disabled = false;

        // Filter out stopped/error sessions for the dropdown
        const activeSessions = sessions.filter(
          (session) =>
            session.status === "running" || session.status === "starting"
        );

        activeSessions.forEach((session) => {
          const option = document.createElement("option");
          option.value = session.id;
          option.textContent = `${session.id.substring(0, 8)}... (${
            session.status
          })`;
          this.elements.sessionSelect.appendChild(option);
        });

        // Select current session if it's still active
        if (this.currentSessionId && this.sessions.has(this.currentSessionId)) {
          const currentSession = this.sessions.get(this.currentSessionId);
          if (
            currentSession.status === "running" ||
            currentSession.status === "starting"
          ) {
            this.elements.sessionSelect.value = this.currentSessionId;
          }
        }
      }
    }

    // Update tabs
    this.updateSessionTabs();
  }

  updateCurrentSessionInfo(session) {
    if (this.elements.currentSessionId) {
      this.elements.currentSessionId.textContent =
        session.id.substring(0, 8) + "...";
    }

    if (this.elements.currentSessionStatus) {
      this.elements.currentSessionStatus.textContent = "●";
      this.elements.currentSessionStatus.className = `session-status ${session.status}`;
    }
  }

  updateSessionStatus(sessionId, status) {
    const session = this.sessions.get(sessionId);
    if (session) {
      session.status = status;

      // Update UI if this is the current session
      if (this.currentSessionId === sessionId) {
        this.updateCurrentSessionInfo(session);
      }

      // Update tabs
      this.updateSessionTabs();
    }
  }

  updateTerminalControls(enabled) {
    const controls = [
      this.elements.terminalClear,
      this.elements.terminalDisconnect,
      this.elements.terminalTerminate,
    ];

    controls.forEach((control) => {
      if (control) {
        control.disabled = !enabled;
      }
    });
  }

  // Session tabs management
  addSessionTab(session) {
    if (!this.elements.tabsContainer) return;

    const tab = document.createElement("div");
    tab.className = "session-tab";
    tab.dataset.sessionId = session.id;

    // Add status indicator to tab
    const statusClass =
      session.status === "running" ? "active" : session.status;
    tab.classList.add(statusClass);

    tab.innerHTML = `
            <span class="tab-label">${session.id.substring(0, 8)}...</span>
            <button class="tab-close" data-session-id="${session.id}">×</button>
        `;

    // Tab click handler
    tab.addEventListener("click", (e) => {
      if (!e.target.classList.contains("tab-close")) {
        this.switchToSession(session.id);
      }
    });

    // Tab close handler
    tab.querySelector(".tab-close").addEventListener("click", (e) => {
      e.stopPropagation();
      this.terminateSession(session.id);
    });

    this.elements.tabsContainer.appendChild(tab);
  }

  removeSessionTab(sessionId) {
    if (!this.elements.tabsContainer) return;

    const tab = this.elements.tabsContainer.querySelector(
      `[data-session-id="${sessionId}"]`
    );
    if (tab) {
      tab.remove();
    }
  }

  updateSessionTabs() {
    if (!this.elements.tabsContainer) return;

    // Clear existing tabs
    this.elements.tabsContainer.innerHTML = "";

    // Add tabs for all sessions
    this.sessions.forEach((session) => {
      this.addSessionTab(session);
    });

    // Update active tab
    if (this.currentSessionId) {
      const activeTab = this.elements.tabsContainer.querySelector(
        `[data-session-id="${this.currentSessionId}"]`
      );
      if (activeTab) {
        activeTab.classList.add("active");
      }
    }
  }

  // Modal management
  showCreateSessionModal() {
    if (this.elements.modalOverlay) {
      this.elements.modalOverlay.classList.add("show");

      // Focus first input
      setTimeout(() => {
        this.elements.sessionShell?.focus();
      }, 100);
    }
  }

  hideCreateSessionModal() {
    if (this.elements.modalOverlay) {
      this.elements.modalOverlay.classList.remove("show");
    }

    // Reset form
    if (this.elements.sessionForm) {
      this.elements.sessionForm.reset();
    }
  }

  // Loading management
  showLoading(text = "Loading...") {
    if (this.elements.loadingOverlay) {
      this.elements.loadingOverlay.classList.add("show");
    }

    if (this.elements.loadingText) {
      this.elements.loadingText.textContent = text;
    }
  }

  hideLoading() {
    if (this.elements.loadingOverlay) {
      this.elements.loadingOverlay.classList.remove("show");
    }
  }

  // Placeholder management
  showPlaceholder() {
    const placeholder = document.getElementById("terminal-placeholder");
    const container = document.getElementById("terminal-container");

    if (placeholder) {
      placeholder.style.display = "flex";
    }

    this.updateTerminalControls(false);
    this.updateCurrentSessionInfo({ id: "No session", status: "none" });
  }

  hidePlaceholder() {
    const placeholder = document.getElementById("terminal-placeholder");

    if (placeholder) {
      placeholder.style.display = "none";
    }
  }

  // Notification system
  showNotification(message, type = "info") {
    if (!this.elements.notifications) return;

    const notification = document.createElement("div");
    notification.className = `notification ${type}`;
    notification.textContent = message;

    this.elements.notifications.appendChild(notification);

    // Show notification
    setTimeout(() => {
      notification.classList.add("show");
    }, 100);

    // Auto-hide notification
    setTimeout(() => {
      notification.classList.remove("show");
      setTimeout(() => {
        if (notification.parentNode) {
          notification.parentNode.removeChild(notification);
        }
      }, 300);
    }, 5000);
  }

  // Event handlers
  handleSessionCreate(e) {
    e.preventDefault();

    const config = {
      shell: this.elements.sessionShell?.value || "",
      workingDir: this.elements.sessionWorkdir?.value || "",
    };

    this.createSession(config);
  }

  handleSessionSelect(e) {
    const sessionId = e.target.value;
    if (sessionId) {
      this.switchToSession(sessionId);
    }
  }

  handleTabClick(e) {
    const tab = e.target.closest(".session-tab");
    if (tab && !e.target.classList.contains("tab-close")) {
      const sessionId = tab.dataset.sessionId;
      this.switchToSession(sessionId);
    }
  }

  handleTabClose(e) {
    e.stopPropagation();
    const sessionId = e.target.dataset.sessionId;
    if (sessionId) {
      this.terminateSession(sessionId);
    }
  }

  // Terminal operations
  clearTerminal() {
    window.dispatchEvent(new CustomEvent("terminalClear"));
  }

  disconnectSession() {
    window.dispatchEvent(new CustomEvent("terminalDisconnect"));
  }

  // Getters
  getCurrentSession() {
    return this.currentSessionId
      ? this.sessions.get(this.currentSessionId)
      : null;
  }

  getAllSessions() {
    return Array.from(this.sessions.values());
  }

  getSessionCount() {
    return this.sessions.size;
  }
}

// Export for use in other modules
window.SessionManager = SessionManager;
