# WebTerm - Web-Based Terminal 🌐

**WebTerm** is a high-performance, production-ready web-based terminal that provides secure, real-time access to your system's command line interface from any modern web browser. Built with Go and featuring advanced session management, WebSocket communication, and comprehensive monitoring capabilities.

## ✨ Key Features

### 🚀 **Advanced Session Management**

- **Enhanced I/O Bridging**: Sophisticated named pipe-based communication for optimal performance
- **Multi-Session Support**: Create and manage multiple concurrent terminal sessions
- **Session Isolation**: Complete separation between sessions for security
- **Automatic Cleanup**: Intelligent resource management and cleanup
- **Retry Mechanisms**: Robust error handling with automatic retry logic

### ⚡ **High Performance Architecture**

- **Optimized Output Buffering**: Intelligent buffering for smooth real-time output
- **Connection Pooling**: Efficient WebSocket connection management
- **Memory Optimization**: Automatic garbage collection and memory management
- **Performance Monitoring**: Real-time metrics and performance tracking
- **Resource Limits**: Prevents runaway processes and resource exhaustion

### 🔧 **Production-Ready Features**

- **Comprehensive Logging**: Structured JSON logging with configurable levels
- **Health Monitoring**: Built-in health checks and system monitoring
- **Graceful Shutdown**: Proper cleanup and resource management
- **Cross-Platform Support**: Works on Linux, macOS, and Windows
- **Mobile Responsive**: Touch-optimized interface for mobile devices

### 🛡️ **Security & Reliability**

- **Session Isolation**: Each terminal session is completely separate
- **Input Validation**: Comprehensive input sanitization and validation
- **Resource Monitoring**: Real-time monitoring of system resources
- **Error Recovery**: Automatic error detection and recovery mechanisms
- **Secure Communication**: WebSocket-based encrypted communication

## Demo
https://github.com/user-attachments/assets/1397006a-df05-4620-bd75-25a43bbaff27


## 🎯 Who Is This For?

### 👨‍💻 **DevOps Engineers & System Administrators**

- Remote server management and monitoring
- Production environment troubleshooting
- Automated deployment and maintenance tasks
- Real-time system monitoring and alerting

### 🏢 **Development Teams**

- Remote development environment access
- Collaborative debugging and troubleshooting
- CI/CD pipeline management
- Production issue investigation

### 🏠 **Home Users & Enthusiasts**

- Remote home server management
- IoT device administration
- Personal project development
- Learning and experimentation

## 🚀 Quick Start

### Prerequisites

- Go 1.23.1 or later
- Linux, macOS, or Windows
- Modern web browser (Chrome, Firefox, Safari, Edge)

### Installation & Running

1. **Clone and Build**

   ```bash
   git clone https://github.com/piyushgupta53/webterm.git
   cd webterm
   go build -o webterm cmd/server/main.go
   ```

2. **Start WebTerm**

   ```bash
   ./webterm
   ```

3. **Access the Interface**
   Open your browser and navigate to `http://localhost:8080`

4. **Create Your First Session**
   Click "New Terminal" and start using your web-based terminal!

### Docker Deployment

```bash
# Build the Docker image
docker build -t webterm .

# Run the container
docker run -p 8080:8080 --rm webterm
```

## ⚙️ Configuration

### Environment Variables

| Variable                  | Default              | Description                              |
| ------------------------- | -------------------- | ---------------------------------------- |
| `WEBTERM_HOST`            | `localhost`          | Server host address                      |
| `WEBTERM_PORT`            | `8080`               | Server port                              |
| `WEBTERM_STATIC_DIR`      | `web/static`         | Static files directory                   |
| `WEBTERM_LOG_LEVEL`       | `info`               | Logging level (debug, info, warn, error) |
| `WEBTERM_PIPES_DIR`       | `/tmp/webterm-pipes` | Named pipes directory                    |
| `WEBTERM_SESSION_TIMEOUT` | `30m`                | Session timeout duration                 |

### Session Configuration Options

When creating a session, you can configure:

- **Shell**: Choose from bash, zsh, sh, or specify a custom shell path
- **Working Directory**: Set the initial working directory
- **Environment Variables**: Custom environment variables for the session
- **Initial Command**: Optional command to run when session starts

## 🔌 API Reference

### REST Endpoints

| Endpoint             | Method | Description                   |
| -------------------- | ------ | ----------------------------- |
| `/health`            | GET    | Health check endpoint         |
| `/api/sessions`      | GET    | List all active sessions      |
| `/api/sessions`      | POST   | Create a new terminal session |
| `/api/sessions/{id}` | GET    | Get session details           |
| `/api/sessions/{id}` | DELETE | Terminate a session           |

### WebSocket Endpoints

| Endpoint           | Description                      |
| ------------------ | -------------------------------- |
| `/ws?session={id}` | Real-time terminal communication |

### Message Types

- **Input**: Send terminal input to session
- **Output**: Receive terminal output from session
- **Resize**: Resize terminal dimensions
- **Status**: Session status updates
- **Error**: Error notifications

## 📊 Monitoring & Metrics

### Built-in Metrics

WebTerm provides comprehensive monitoring capabilities:

- **Session Metrics**: Active sessions, creation/termination rates
- **Connection Metrics**: WebSocket connections, throughput
- **Performance Metrics**: Response times, request rates
- **Resource Metrics**: Memory usage, goroutines, file descriptors
- **Error Metrics**: Error rates by type

### Health Checks

```bash
# Check application health
curl http://localhost:8080/health

# Response example:
{
  "status": "healthy",
  "version": "1.0.0",
  "uptime": "2h30m15s",
  "active_sessions": 3,
  "active_connections": 5
}
```

## 🏗️ Architecture

### Core Components

1. **Session Manager** (`internal/terminal/manager.go`)

   - Manages terminal session lifecycle
   - Handles PTY creation and process management
   - Implements session isolation and cleanup

2. **WebSocket Hub** (`internal/websocket/hub.go`)

   - Manages real-time client connections
   - Handles message routing and broadcasting
   - Implements connection pooling and optimization

3. **Enhanced I/O Bridge** (`internal/terminal/session.go`)

   - Sophisticated named pipe-based communication
   - Optimized output buffering and streaming
   - Robust error handling and retry mechanisms

4. **Performance Optimizer** (`internal/performance/optimizer.go`)

   - Connection pooling and management
   - Memory optimization and garbage collection
   - Performance monitoring and metrics collection

5. **Monitoring System** (`internal/monitoring/metrics.go`)
   - Real-time metrics collection
   - Resource monitoring and alerting
   - Performance analysis and reporting

### Data Flow

```
Browser ←→ WebSocket Hub ←→ Session Manager ←→ PTY ←→ Shell Process
                ↓                    ↓
         Connection Pool      Enhanced I/O Bridge
                ↓                    ↓
         Performance Monitor    Named Pipes
```

## 🔒 Security Considerations

### Built-in Security Features

- **Session Isolation**: Complete separation between terminal sessions
- **Input Validation**: Comprehensive sanitization of all user inputs
- **Resource Limits**: Prevents resource exhaustion attacks
- **Secure Communication**: WebSocket-based encrypted data transmission
- **Automatic Cleanup**: Prevents resource leaks and security issues

### Production Security Recommendations

1. **Use HTTPS**: Deploy behind a reverse proxy with SSL/TLS
2. **Network Security**: Restrict access with firewall rules
3. **Authentication**: Implement user authentication for production use
4. **Access Control**: Use VPN or private networks for sensitive environments
5. **Regular Updates**: Keep the application updated with security patches

## 🚀 Performance Features

### Optimization Techniques

- **Output Buffering**: Intelligent buffering for smooth real-time output
- **Connection Pooling**: Efficient WebSocket connection management
- **Memory Management**: Automatic garbage collection and optimization
- **Resource Monitoring**: Real-time system resource tracking
- **Performance Metrics**: Comprehensive performance analysis

### Scalability

- **Horizontal Scaling**: Stateless design allows multiple instances
- **Load Balancing**: Compatible with standard load balancers
- **Resource Efficiency**: Minimal memory and CPU footprint
- **Connection Management**: Efficient handling of multiple concurrent users

## 💡 Use Cases

### 🏢 **Enterprise Environments**

```bash
# Production server monitoring
htop
df -h
systemctl status nginx mysql redis

# Application deployment
cd /var/www/production
git pull origin main
docker-compose up -d
systemctl restart services
```

### 🖥️ **Development Workflows**

```bash
# Project development
cd /home/dev/project
git status
npm install
npm run test
docker build -t myapp .
```

### 📊 **System Administration**

```bash
# System maintenance
apt update && apt upgrade
systemctl restart problematic-service
journalctl -f -u nginx
free -h && df -h
```

### 🔧 **Emergency Response**

```bash
# Critical issue resolution
systemctl stop failing-service
find /var/log -name "*.log" -size +100M -delete
systemctl restart critical-services
```

## 🎨 User Interface

### Modern Web Interface

- **Responsive Design**: Works seamlessly on desktop and mobile devices
- **Dark Theme**: Easy on the eyes for extended use
- **Session Tabs**: Easy switching between multiple terminal sessions
- **Real-time Status**: Live connection and session status indicators
- **Terminal Controls**: Clear, disconnect, and terminate session buttons

### Browser Compatibility

- **Desktop**: Chrome, Firefox, Safari, Edge (latest versions)
- **Mobile**: iOS Safari, Android Chrome
- **Features**: Full keyboard support, touch optimization, responsive design

## 🔮 Roadmap

### Planned Features

- **User Authentication**: Multi-user support with role-based access
- **Session Sharing**: Collaborative terminal sessions
- **File Transfer**: Upload/download files through the browser
- **Plugin System**: Extensible architecture for custom functionality
- **Cloud Integration**: One-click deployment to cloud platforms
- **Advanced Monitoring**: Integration with external monitoring systems
