# AzureTLS Server

The AzureTLS Server provides HTTP/REST and WebSocket APIs for making HTTP requests through the AzureTLS client library. This allows you to use AzureTLS functionality from any programming language that can make HTTP requests or connect to WebSockets.

## Quick Start

### 1. Start the Server

```bash
# Default configuration (localhost:8080)
go run cmd/server/main.go

# Custom host and port
go run cmd/server/main.go -host=0.0.0.0 -port=8080

# Custom configuration with limits
go run cmd/server/main.go -host=0.0.0.0 -port=8080 -max_sessions=500 -max_concurrent_requests=50 -read_timeout=60 -write_timeout=60
```

### 2. Make Your First Request

```bash
# Create a session
curl -X POST http://localhost:8080/api/v1/session/create

# Make a request using the session
curl -X POST http://localhost:8080/api/v1/session/{session_id}/request \
  -H "Content-Type: application/json" \
  -d '{"method": "GET", "url": "https://httpbin.org/get"}'
```

## Configuration

### Command Line Options

| Flag | Default     | Description |
|------|-------------|-------------|
| `-host` | `localhost` | Server bind address |
| `-port` | `8080`      | Server port |
| `-max_sessions` | `1000`      | Maximum concurrent sessions |
| `-max_concurrent_requests` | `100`       | Maximum concurrent requests per session |
| `-read_timeout` | `30`        | Server read timeout (seconds) |
| `-write_timeout` | `30`        | Server write timeout (seconds) |

## REST API Reference

### Health Check

```http
GET /health
```

**Response:**
```json
{
  "status": "healthy",
  "sessions": 0,
  "timestamp": "2024-01-01T00:00:00Z",
  "version": "1.0.0"
}
```

### Session Management

#### Create Session

```http
POST /api/v1/session/create
```

**Response:**
```json
{
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "created"
}
```

#### Delete Session

```http
DELETE /api/v1/session/{session_id}
```

**Response:** `204 No Content`

### Making Requests

#### Session-Based Request

```http
POST /api/v1/session/{session_id}/request
Content-Type: application/json

{
  "method": "GET",
  "url": "https://httpbin.org/get",
  "headers": {
    "User-Agent": "MyApp/1.0",
    "Authorization": "Bearer token"
  },
  "body": "{\"key\": \"value\"}",
  "options": {
    "timeout": 30,
    "follow_redirects": true,
    "max_redirects": 5,
    "proxy": "http://proxy:8080",
    "no_cookie": false,
    "browser": "chrome",
    "force_http1": false,
    "force_http3": false,
    "insecure_skip_verify": false
  }
}
```

#### Stateless Request

```http
POST /api/v1/request
Content-Type: application/json

{
  "method": "POST",
  "url": "https://httpbin.org/post",
  "headers": {
    "Content-Type": "application/json"
  },
  "body": "{\"message\": \"Hello World\"}"
}
```

### Request Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `timeout` | int | 30 | Request timeout in seconds |
| `follow_redirects` | bool | true | Follow HTTP redirects |
| `max_redirects` | int | 10 | Maximum number of redirects |
| `proxy` | string | "" | Proxy URL (http/https/socks5) |
| `no_cookie` | bool | false | Disable cookie handling |
| `browser` | string | "" | Browser profile (chrome, firefox, safari, edge) |
| `force_http1` | bool | false | Force HTTP/1.1 |
| `force_http3` | bool | false | Force HTTP/3 |
| `insecure_skip_verify` | bool | false | Skip TLS certificate verification |

### Response Format

```json
{
  "id": "request-123",
  "status_code": 200,
  "status": "200 OK",
  "headers": {
    "Content-Type": "application/json",
    "Content-Length": "1024"
  },
  "body": "{\"response\": \"data\"}",
  "cookies": [
    {
      "name": "session",
      "value": "abc123",
      "domain": "example.com",
      "path": "/",
      "expires": "2024-12-31T23:59:59Z",
      "secure": true,
      "http_only": true,
      "same_site": "Strict"
    }
  ],
  "url": "https://httpbin.org/get",
  "error": ""
}
```

## WebSocket API

### Connection

```javascript
const ws = new WebSocket('ws://localhost:8080/ws');
```

### Message Format

All WebSocket messages follow this format:

```json
{
  "type": "message_type",
  "id": "optional_request_id",
  "payload": { ... }
}
```

### Message Types

#### Session Info (Server → Client)

Sent automatically when connecting:

```json
{
  "type": "session",
  "payload": {
    "session_id": "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

#### Request (Client → Server)

```json
{
  "type": "request",
  "id": "req-1",
  "payload": {
    "method": "GET",
    "url": "https://httpbin.org/get",
    "headers": {
      "User-Agent": "WebSocket-Client/1.0"
    },
    "options": {
      "follow_redirects": true
    }
  }
}
```

#### Response (Server → Client)

```json
{
  "type": "response",
  "id": "req-1",
  "payload": {
    "status_code": 200,
    "status": "200 OK",
    "headers": {
      "Content-Type": "application/json"
    },
    "body": "{\"result\": \"success\"}",
    "url": "https://httpbin.org/get"
  }
}
```

#### Error (Server → Client)

```json
{
  "type": "error",
  "id": "req-1",
  "payload": {
    "error": "Connection timeout"
  }
}
```

#### Ping/Pong (Heartbeat)

The server sends ping messages every 30 seconds:

```json
{
  "type": "ping",
  "id": "ping-123"
}
```

Client should respond with:

```json
{
  "type": "pong",
  "id": "ping-123"
}
```

## Examples

### Basic Usage (Go)

```bash
cd examples/server
go run basic.go
```

This example demonstrates:
- Creating and deleting sessions
- Making GET and POST requests
- Stateless requests
- Error handling

### WebSocket Client (Go)

```bash
cd examples/server
go run websocket.go
```

This example demonstrates:
- WebSocket connection
- Session management
- Multiple concurrent requests
- Ping/pong handling

### Advanced Features (Go)

```bash
cd examples/server
go run advanced.go
```

This example demonstrates:
- Browser fingerprinting
- Proxy configuration
- Cookie management
- Custom headers and authentication
- Protocol version control
- Error handling and timeouts

### Python Example

```python
import requests
import json

# Create session
response = requests.post('http://localhost:8080/api/v1/session/create')
session_id = response.json()['session_id']

# Make request
payload = {
    'method': 'GET',
    'url': 'https://httpbin.org/get',
    'headers': {'User-Agent': 'Python-Client/1.0'}
}

response = requests.post(
    f'http://localhost:8080/api/v1/session/{session_id}/request',
    json=payload
)

result = response.json()
print(f"Status: {result['status_code']}")
print(f"Body: {result['body']}")

# Delete session
requests.delete(f'http://localhost:8080/api/v1/session/{session_id}')
```

### JavaScript/Node.js Example

```javascript
const axios = require('axios');

async function example() {
  // Create session
  const sessionResp = await axios.post('http://localhost:8080/api/v1/session/create');
  const sessionId = sessionResp.data.session_id;

  // Make request
  const payload = {
    method: 'GET',
    url: 'https://httpbin.org/get',
    headers: { 'User-Agent': 'Node-Client/1.0' }
  };

  const response = await axios.post(
    `http://localhost:8080/api/v1/session/${sessionId}/request`,
    payload
  );

  console.log('Status:', response.data.status_code);
  console.log('Body:', response.data.body);

  // Delete session
  await axios.delete(`http://localhost:8080/api/v1/session/${sessionId}`);
}

example().catch(console.error);
```

## Error Codes

| HTTP Code | Description | Example |
|-----------|-------------|---------|
| 200 | OK | Successful request |
| 201 | Created | Session created |
| 204 | No Content | Session deleted |
| 400 | Bad Request | Invalid JSON payload |
| 404 | Not Found | Session not found |
| 415 | Unsupported Media Type | Invalid Content-Type |
| 429 | Too Many Requests | Concurrent request limit exceeded |
| 500 | Internal Server Error | Server processing error |

## Error Response Format

```json
{
  "error": "Session not found",
  "status": 404
}
```

## Browser Profiles

The server supports these browser profiles for fingerprinting:

- `chrome` - Latest Chrome browser
- `firefox` - Latest Firefox browser
- `safari` - Latest Safari browser
- `edge` - Latest Edge browser

Example:
```json
{
  "method": "GET",
  "url": "https://httpbin.org/headers",
  "options": {
    "browser": "chrome"
  }
}
```

## Proxy Support

Supports HTTP, HTTPS, and SOCKS5 proxies:

```json
{
  "options": {
    "proxy": "http://proxy.example.com:8080"
  }
}
```

```json
{
  "options": {
    "proxy": "socks5://127.0.0.1:1080"
  }
}
```

## Session Lifecycle

1. **Creation**: `POST /api/v1/session/create` returns session ID
2. **Usage**: Make requests using session ID to maintain cookies and connection pooling
3. **Automatic Cleanup**: Sessions are automatically cleaned up on server shutdown
4. **Manual Cleanup**: `DELETE /api/v1/session/{id}` to explicitly delete

For WebSocket connections:
1. **Connection**: Connect to `/ws` endpoint
2. **Session Creation**: Server automatically creates session and sends session ID
3. **Request/Response**: Send request messages, receive response messages
4. **Cleanup**: Session automatically deleted when WebSocket connection closes

## Performance Considerations

- **Session Reuse**: Use sessions for multiple requests to the same domain for better performance
- **Connection Pooling**: Sessions automatically pool connections for efficiency
- **Concurrent Limits**: Server limits concurrent requests per session (configurable)
- **Memory Usage**: Sessions store cookies and connection state - clean up unused sessions
- **WebSocket Overhead**: WebSocket connections have less overhead than REST for multiple requests

## Troubleshooting

### Server Won't Start
- Check if port is already in use: `netstat -an | grep :8080`
- Try different port: `go run cmd/server/main.go -port=8081`

### Session Not Found
- Verify session was created successfully
- Check session ID in request URL
- Sessions may expire on server restart

### Request Timeouts
- Increase timeout in request options
- Check network connectivity
- Verify target server is responding

### WebSocket Connection Issues
- Ensure WebSocket URL is correct: `ws://localhost:8080/ws`
- Check for firewall blocking WebSocket connections
- Verify server is running and accessible

### Proxy Issues
- Test proxy connectivity independently
- Verify proxy URL format
- Check proxy authentication if required

## Development

### Running Tests

```bash
# Start server in one terminal
go run cmd/server/main.go

# Run examples in another terminal
cd examples/server
go run basic.go
go run websocket.go
go run advanced.go
```

### Building

```bash
# Build server binary
go build -o azuretls-server cmd/server/main.go

# Run built binary
./azuretls-server -host=0.0.0.0 -port=8080
```

This server provides a language-agnostic way to use AzureTLS functionality, making it easy to integrate advanced HTTP client features into any application stack.