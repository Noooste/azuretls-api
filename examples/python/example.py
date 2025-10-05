"""
AzureTLS API - Synchronous REST API Example

This example demonstrates how to use the AzureTLS API with synchronous HTTP requests.
"""

import requests
import json
import base64


class AzureTLSClient:
    """Synchronous client for AzureTLS API"""

    def __init__(self, base_url="http://localhost:8080"):
        self.base_url = base_url
        self.session_id = None

    def create_session(self):
        """Create a new session"""
        response = requests.post(f"{self.base_url}/api/v1/session/create")
        response.raise_for_status()
        data = response.json()
        self.session_id = data["session_id"]
        print(f"✓ Session created: {self.session_id}")
        return self.session_id

    def delete_session(self, session_id=None):
        """Delete a session"""
        sid = session_id or self.session_id
        if not sid:
            raise ValueError("No session ID provided")

        response = requests.delete(f"{self.base_url}/api/v1/session/{sid}")
        response.raise_for_status()
        print(f"✓ Session deleted: {sid}")
        if sid == self.session_id:
            self.session_id = None

    def request(self, method, url, headers=None, body=None, options=None, session_id=None):
        """Make a request using a session"""
        sid = session_id or self.session_id
        if not sid:
            raise ValueError("No session ID. Create a session first.")

        payload = {
            "method": method,
            "url": url,
        }

        if headers:
            payload["headers"] = headers
        if body:
            # Encode body as base64
            if isinstance(body, bytes):
                payload["body_b64"] = base64.b64encode(body).decode('ascii')
            else:
                payload["body"] = body

        if options:
            payload["options"] = options

        response = requests.post(
            f"{self.base_url}/api/v1/session/{sid}/request",
            json=payload
        )
        response.raise_for_status()
        return response.json()

    def stateless_request(self, method, url, headers=None, body=None, options=None):
        """Make a stateless request without a session"""
        payload = {
            "method": method,
            "url": url,
        }

        if headers:
            payload["headers"] = headers
        if body:
            # Encode body as base64
            if isinstance(body, str):
                body = body.encode('utf-8')
            payload["body_b64"] = base64.b64encode(body).decode('ascii')
        if options:
            payload["options"] = options

        response = requests.post(
            f"{self.base_url}/api/v1/request",
            json=payload
        )
        response.raise_for_status()
        return response.json()

    def health_check(self):
        """Check server health"""
        response = requests.get(f"{self.base_url}/health")
        response.raise_for_status()
        return response.json()


def main():
    """Example usage"""
    print("=== AzureTLS API Synchronous Example ===\n")

    client = AzureTLSClient()

    # Health check
    print("1. Health Check")
    health = client.health_check()
    print(f"   Status: {health['status']}")
    print(f"   Active sessions: {health['sessions']}\n")

    # Create session
    print("2. Create Session")
    client.create_session()
    print()

    # Simple GET request
    print("3. Simple GET Request")
    result = client.request("GET", "https://httpbin.org/get")
    print(f"   Status: {result['status_code']}")
    print(f"   URL: {result['url']}")
    body = json.loads(result['body'])
    print(f"   User-Agent: {body.get('headers', {}).get('User-Agent', 'N/A')}\n")

    # POST request with custom headers and body
    print("4. POST Request with Headers and Body")
    result = client.request(
        "POST",
        "https://httpbin.org/post",
        headers={
            "Content-Type": "application/json",
            "User-Agent": "Python-AzureTLS-Client/1.0"
        },
        body=json.dumps({"message": "Hello from Python!", "test": True})
    )
    print(f"   Status: {result['status_code']}")
    body = json.loads(result['body'])
    print(f"   Sent data: {body.get('json', {})}\n")

    # Request with browser fingerprint
    print("5. Request with Browser Fingerprint")
    result = client.request(
        "GET",
        "https://httpbin.org/headers",
        options={
            "browser": "chrome",
            "timeout": 30
        }
    )
    print(f"   Status: {result['status_code']}")
    body = json.loads(result['body'])
    print(f"   Headers sent: {body.get('headers', {})}\n")

    # Request with cookies
    print("6. Request to Set Cookies")
    result = client.request(
        "GET",
        "https://httpbin.org/cookies/set?session=abc123&user=test"
    )
    if result.get('cookies'):
        print(f"   Cookies received: {len(result['cookies'])} cookie(s)")
        for cookie in result['cookies']:
            print(f"   - {cookie['name']}={cookie['value']}")
    print()

    # Verify cookies are maintained in session
    print("7. Verify Session Maintains Cookies")
    result = client.request("GET", "https://httpbin.org/cookies")
    print(f"   Status: {result['status_code']}")
    body = json.loads(result['body'])
    print(f"   Cookies in session: {body.get('cookies', {})}\n")

    # Stateless request (no session)
    print("8. Stateless Request (No Session)")
    result = client.stateless_request(
        "GET",
        "https://httpbin.org/get",
        headers={"X-Custom-Header": "Stateless-Request"}
    )
    print(f"   Status: {result['status_code']}")
    print(f"   Session used: No\n")

    # Error handling example
    print("9. Error Handling")
    try:
        result = client.request(
            "GET",
            "https://httpbin.org/status/404"
        )
        print(f"   Status: {result['status_code']}")
        print(f"   Error handling: Request completed (404 is not a request error)\n")
    except requests.exceptions.HTTPError as e:
        print(f"   Error: {e}\n")

    # Delete session
    print("10. Delete Session")
    client.delete_session()
    print()

    print("=== Example Complete ===")


if __name__ == "__main__":
    main()
