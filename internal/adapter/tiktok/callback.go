package tiktok

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// CallbackServer is a local HTTP server that handles the TikTok OAuth callback.
type CallbackServer struct {
	server     *http.Server
	listener   net.Listener
	port       int
	codeChan   chan string
	errorChan  chan error
	state      string
	codeVerifier string
}

// NewCallbackServer creates a new callback server on a random available port.
func NewCallbackServer(state, codeVerifier string) (*CallbackServer, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}

	// Extract port from the listener address
	addr := listener.Addr().(*net.TCPAddr)

	cs := &CallbackServer{
		listener:    listener,
		port:        addr.Port,
		codeChan:    make(chan string, 1),
		errorChan:   make(chan error, 1),
		state:       state,
		codeVerifier: codeVerifier,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/callback/", cs.handleCallback)

	cs.server = &http.Server{
		Handler: mux,
	}

	go cs.server.Serve(listener)

	return cs, nil
}

// URL returns the callback URL for the OAuth redirect.
func (cs *CallbackServer) URL() string {
	return fmt.Sprintf("http://localhost:%d/callback/", cs.port)
}

// WaitForCallback waits for the OAuth callback and returns the authorization code.
func (cs *CallbackServer) WaitForCallback(ctx context.Context) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case code := <-cs.codeChan:
		return code, nil
	case err := <-cs.errorChan:
		return "", err
	}
}

// Close shuts down the callback server.
func (cs *CallbackServer) Close() error {
	return cs.server.Shutdown(context.Background())
}

func (cs *CallbackServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	// Check for error in callback
	if errMsg := r.URL.Query().Get("error"); errMsg != "" {
		errDesc := r.URL.Query().Get("error_description")
		cs.errorChan <- fmt.Errorf("oauth error: %s - %s", errMsg, errDesc)
		cs.writeErrorPage(w, errMsg, errDesc)
		return
	}

	// Verify state to prevent CSRF
	state := r.URL.Query().Get("state")
	if state != cs.state {
		err := fmt.Errorf("state mismatch: expected %s, got %s", cs.state, state)
		cs.errorChan <- err
		cs.writeErrorPage(w, "state_mismatch", err.Error())
		return
	}

	// Extract authorization code
	code := r.URL.Query().Get("code")
	if code == "" {
		err := fmt.Errorf("no authorization code received")
		cs.errorChan <- err
		cs.writeErrorPage(w, "missing_code", err.Error())
		return
	}

	// Send code back on channel
	cs.codeChan <- code

	// Write success page
	cs.writeSuccessPage(w)
}

func (cs *CallbackServer) writeSuccessPage(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
    <title> TikTok Login Success</title>
    <style>
        body { font-family: Arial, sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; background: #f5f5f5; }
        .card { background: white; padding: 40px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); text-align: center; }
        h1 { color: #010101; margin-bottom: 10px; }
        p { color: #666; }
        .checkmark { color: #00f2ea; font-size: 48px; }
    </style>
</head>
<body>
    <div class="card">
        <div class="checkmark">✓</div>
        <h1>Login Successful</h1>
        <p>You can close this window and return to the CLI.</p>
    </div>
</body>
</html>`))
}

func (cs *CallbackServer) writeErrorPage(w http.ResponseWriter, err, desc string) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title> TikTok Login Error</title>
    <style>
        body { font-family: Arial, sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; background: #f5f5f5; }
        .card { background: white; padding: 40px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); text-align: center; }
        h1 { color: #fe2c55; margin-bottom: 10px; }
        p { color: #666; }
    </style>
</head>
<body>
    <div class="card">
        <h1>Login Failed</h1>
        <p><strong>Error:</strong> %s</p>
        <p><strong>Details:</strong> %s</p>
    </div>
</body>
</html>`, htmlEscape(err), htmlEscape(desc))))
}

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}

// WaitWithTimeout waits for callback with a default timeout.
func (cs *CallbackServer) WaitWithTimeout(timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return cs.WaitForCallback(ctx)
}