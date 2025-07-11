# WebTerm - Access Your Terminal from Anywhere 🌐

**WebTerm** is a web-based terminal that lets you run commands and manage your system directly from your browser. Think of it as having a terminal window that works on any device with a web browser - no installation required!

## ✨ What Can You Do With WebTerm?

### 🖥️ **Access Your Terminal Anywhere**

- Open your terminal sessions from any computer, tablet, or phone
- No need to install SSH clients or terminal apps
- Works on Windows, Mac, Linux, iOS, and Android

### 🔧 **Manage Your System Remotely**

- Run system commands and scripts
- Monitor server processes and logs
- Manage files and directories
- Install and configure software

### 🚀 **Perfect for Remote Work**

- Access your development environment from anywhere
- Debug issues on remote servers
- Perform maintenance tasks on the go
- Share terminal access with team members

## 🎯 Who Is This For?

### 👨‍💻 **Developers**

- Access your development environment remotely
- Debug production issues from anywhere
- Run deployment scripts and maintenance tasks
- Collaborate with team members on server management

### 🏠 **Home Users**

- Access your home server from work
- Manage your media server remotely
- Control smart home devices via command line
- Backup and maintain your personal systems

## 🚀 Getting Started

### Quick Start (30 seconds!)

1. **Start WebTerm**

   ```bash
   go run cmd/server/main.go
   ```

2. **Open Your Browser**
   Navigate to `http://localhost:8080`

3. **Start Using Your Terminal**
   Click "New Terminal" and begin running commands!

### For Production Use

1. **Install on Your Server**

   ```bash
   git clone https://github.com/piyushgupta53/webterm.git
   cd webterm
   go build -o webterm cmd/server/main.go
   ```

2. **Configure for Your Environment**

   ```bash
   export WEBTERM_HOST=0.0.0.0
   export WEBTERM_PORT=8080
   ./webterm
   ```

3. **Access from Anywhere**
   Open `http://your-server-ip:8080` from any device

## 🔧 Current Features

### ✅ **Implemented Features**

- **Real-time Terminal Sessions**: Create and manage multiple terminal sessions
- **WebSocket Communication**: Instant command execution and output streaming
- **Session Management**: Start, stop, and switch between sessions via REST API
- **Multiple Shell Support**: Bash, Zsh, Sh, and custom shell configurations
- **Terminal Resizing**: Dynamic terminal size adjustment
- **Session Isolation**: Each terminal session is completely separate
- **Automatic Cleanup**: Sessions are automatically cleaned up when inactive
- **Health Monitoring**: Built-in health check endpoint at `/health`
- **Comprehensive Logging**: Structured JSON logging with configurable levels
- **Graceful Shutdown**: Proper cleanup of resources on application shutdown
- **Cross-platform Support**: Works on Linux, macOS, and Windows
- **Mobile Responsive**: Touch-friendly interface for mobile devices

### 🎨 **User Interface**

- **Modern Web Interface**: Clean, responsive design with dark theme
- **Session Tabs**: Easy switching between multiple terminal sessions
- **Real-time Status**: Live connection and session status indicators
- **Terminal Controls**: Clear, disconnect, and terminate session buttons
- **Session Creation Modal**: Configure shell, working directory, and environment
- **Loading Indicators**: Visual feedback during session operations
- **Error Notifications**: User-friendly error messages and notifications

### 🔌 **API Endpoints**

- `GET /health` - Health check endpoint
- `GET /api/sessions` - List all active sessions
- `POST /api/sessions` - Create a new terminal session
- `GET /api/sessions/{id}` - Get session details
- `DELETE /api/sessions/{id}` - Terminate a session
- `GET /ws` - WebSocket endpoint for real-time communication

## 🔒 Security & Safety

### ✅ **Built-in Security Features**

- Session isolation - each terminal session is completely separate
- Automatic cleanup - sessions are automatically closed when inactive
- Resource limits - prevents runaway processes from consuming all resources
- Secure communication - all data is transmitted via WebSocket
- Input validation - all user inputs are validated and sanitized

### 🛡️ **Best Practices**

- Run WebTerm behind a reverse proxy with HTTPS
- Use firewall rules to restrict access to trusted IPs
- Regularly update the application for security patches
- Monitor session activity and logs
- Configure appropriate session timeouts

## ⚙️ Configuration

### Environment Variables

- `WEBTERM_HOST` - Server host (default: localhost)
- `WEBTERM_PORT` - Server port (default: 8080)
- `WEBTERM_STATIC_DIR` - Static files directory (default: web/static)
- `WEBTERM_LOG_LEVEL` - Logging level (default: info)
- `WEBTERM_PIPES_DIR` - Named pipes directory (default: /tmp/webterm-pipes)

### Session Configuration

When creating a session, you can specify:

- **Shell**: Choose from bash, zsh, sh, or custom shell path
- **Working Directory**: Set the initial working directory
- **Environment Variables**: Custom environment variables for the session
- **Command**: Optional initial command to run

## 💡 Use Cases & Examples

### 🏢 **Remote Server Management**

```bash
# Check server health
htop
df -h
systemctl status nginx

# Update and maintain
apt update && apt upgrade
systemctl restart services
```

### 🖥️ **Development Workflow**

```bash
# Navigate to project
cd /var/www/myapp

# Check git status
git status
git pull origin main

# Run tests
npm test
python -m pytest

# Deploy
docker-compose up -d
```

### 📊 **System Monitoring**

```bash
# Monitor system resources
top
iotop
netstat -tulpn

# Check logs
tail -f /var/log/nginx/access.log
journalctl -f
```

### 🔧 **Emergency Maintenance**

```bash
# Stop problematic services
systemctl stop problematic-service

# Free up disk space
find /var/log -name "*.log" -size +100M -delete

# Restart critical services
systemctl restart mysql redis nginx
```

## 🎨 What Makes WebTerm Special?

### ⚡ **Lightning Fast**

- Instant terminal startup
- Real-time command execution
- Minimal resource usage
- Optimized for low-latency connections

### 🔧 **Easy to Use**

- No complex setup or configuration
- Works out of the box
- Familiar terminal experience
- Intuitive controls and navigation

### 🛠️ **Production Ready**

- Robust error handling and recovery
- Automatic resource cleanup
- Comprehensive logging and monitoring
- Scalable architecture
- Graceful shutdown handling

## 📱 Works Everywhere

### 💻 **Desktop Browsers**

- Chrome, Firefox, Safari, Edge
- Full keyboard support
- Mouse and touchpad navigation
- Multiple terminal sessions

### 📱 **Mobile Devices**

- iOS Safari and Android Chrome
- Touch-optimized interface
- Responsive design
- On-screen keyboard support

## 🔮 Future Features

### 🚀 **Planned Enhancements**

- **File Transfer**: Upload and download files through the browser
- **Session Sharing**: Share terminal sessions with team members
- **Multi-user Support**: User authentication and session isolation
- **Plugin System**: Extend functionality with custom plugins
- **API Integration**: Connect with other tools and services
- **Cloud Deployment**: One-click deployment to cloud platforms

## 🚀 **Contribute**

- Report bugs and request features
- Submit code improvements
- Help with documentation
- Share your use cases and success stories

## 📄 License

WebTerm is open source and available under the [MIT License](LICENSE). Feel free to use it for personal or commercial projects.

---

**Ready to access your terminal from anywhere?** 🚀

Start WebTerm today and experience the freedom of web-based terminal access!
