package main

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/Cod-e-Codes/marchat/client/crypto"
	"github.com/Cod-e-Codes/marchat/shared"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gorilla/websocket"
)

// WebSocket message types for the Bubble Tea update loop
type wsConnected struct{}
type wsReaderClosed struct{}

type wsMsg struct {
	Type string
	Data json.RawMessage
}

type wsErr struct{ error }

type wsUsernameError struct{ message string }

func (e wsUsernameError) Error() string { return e.message }

type UserList struct {
	Users []string `json:"users"`
}

type quitMsg struct{}

func debugEncryptAndSend(recipients []string, plaintext string, ws *websocket.Conn, keystore *crypto.KeyStore, username string) error {
	log.Printf("DEBUG: Starting global encryption for %d recipients", len(recipients))
	log.Printf("DEBUG: Plaintext length: %d", len(plaintext))

	if keystore == nil {
		log.Printf("ERROR: Keystore is nil")
		return fmt.Errorf("keystore not initialized")
	}
	log.Printf("DEBUG: Keystore loaded: %t", keystore != nil)

	globalKey := keystore.GetSessionKey("global")
	if globalKey == nil {
		log.Printf("ERROR: Global key not found")
		return fmt.Errorf("global key not available - global E2E encryption not initialized")
	}
	log.Printf("DEBUG: Global key available (ID: %s)", globalKey.KeyID)

	conversationID := "global"
	encryptedMsg, err := keystore.EncryptMessage(username, plaintext, conversationID)
	if err != nil {
		log.Printf("ERROR: Global encryption failed: %v", err)
		return fmt.Errorf("global encryption failed: %v", err)
	}

	log.Printf("DEBUG: Global encryption successful - encrypted length: %d", len(encryptedMsg.Encrypted))

	if len(encryptedMsg.Encrypted) == 0 {
		log.Printf("ERROR: Encryption returned empty ciphertext")
		return fmt.Errorf("encryption returned empty ciphertext; aborting send")
	}

	combinedData := make([]byte, 0, len(encryptedMsg.Nonce)+len(encryptedMsg.Encrypted))
	combinedData = append(combinedData, encryptedMsg.Nonce...)
	combinedData = append(combinedData, encryptedMsg.Encrypted...)

	finalContent := base64.StdEncoding.EncodeToString(combinedData)
	log.Printf("DEBUG: Base64 encoded nonce+ciphertext - length: %d", len(finalContent))

	if len(finalContent) == 0 {
		log.Printf("ERROR: Final content is empty after encoding")
		return fmt.Errorf("final content is empty after encoding")
	}

	msg := shared.Message{
		Content:   finalContent,
		Sender:    username,
		CreatedAt: time.Now(),
		Type:      shared.TextMessage,
		Encrypted: true,
	}

	log.Printf("DEBUG: Final message - Content length: %d, Type: %s",
		len(msg.Content), msg.Type)

	if err := ws.WriteJSON(msg); err != nil {
		log.Printf("ERROR: WebSocket write failed: %v", err)
		return err
	}

	log.Printf("DEBUG: Global encrypted message sent successfully")
	return nil
}

func validateEncryptionRoundtrip(keystore *crypto.KeyStore, username string) error {
	testPlaintext := "Hello, global encryption test!"

	log.Printf("DEBUG: Testing global encryption roundtrip")

	conversationID := "global"

	globalKey := keystore.GetSessionKey(conversationID)
	if globalKey == nil {
		return fmt.Errorf("global key not found - global E2E encryption not available")
	}

	log.Printf("DEBUG: Global key found (ID: %s)", globalKey.KeyID)

	encryptedMsg, err := keystore.EncryptMessage(username, testPlaintext, conversationID)
	if err != nil {
		return fmt.Errorf("global encryption test failed: %v", err)
	}

	if len(encryptedMsg.Encrypted) == 0 {
		return fmt.Errorf("global encryption test produced empty ciphertext")
	}

	log.Printf("DEBUG: Global encryption test successful - ciphertext length: %d", len(encryptedMsg.Encrypted))

	decryptedMsg, err := keystore.DecryptMessage(encryptedMsg, conversationID)
	if err != nil {
		return fmt.Errorf("global decryption test failed: %v", err)
	}

	if decryptedMsg.Content != testPlaintext {
		return fmt.Errorf("global decryption roundtrip failed: expected '%s', got '%s'", testPlaintext, decryptedMsg.Content)
	}

	log.Printf("DEBUG: Global encryption roundtrip test successful")
	return nil
}

func verifyKeystoreUnlocked(keystore *crypto.KeyStore) error {
	if keystore == nil {
		return fmt.Errorf("keystore is nil")
	}

	globalKey := keystore.GetGlobalKey()
	if globalKey == nil {
		return fmt.Errorf("global key not available")
	}

	log.Printf("DEBUG: Keystore properly unlocked for global encryption")
	return nil
}

func debugWebSocketWrite(ws *websocket.Conn, msg interface{}) error {
	jsonData, err := json.Marshal(msg)
	if err != nil {
		log.Printf("ERROR: JSON marshal failed: %v", err)
		return err
	}

	log.Printf("DEBUG: Sending WebSocket message - length: %d bytes", len(jsonData))

	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonData, &parsed); err == nil {
		if content, exists := parsed["content"]; exists {
			if contentStr, ok := content.(string); ok {
				log.Printf("DEBUG: Message content length: %d", len(contentStr))
				if len(contentStr) == 0 {
					log.Printf("WARNING: Sending message with empty content!")
				}
			}
		}
	}

	return ws.WriteJSON(msg)
}

func (m *model) deliverWSMsg(msg tea.Msg) {
	select {
	case m.msgChan <- msg:
	default:
		if len(m.msgChan) > 0 {
			<-m.msgChan
		}
		select {
		case m.msgChan <- msg:
		default:
			log.Printf("WARNING: Unable to deliver WebSocket message, channel full")
		}
	}
}

func (m *model) abortPartialWebSocketConnect() {
	if m.conn != nil {
		m.conn.Close()
		m.conn = nil
	}
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	m.connected = false
}

func (m *model) connectWebSocket(serverURL string) error {
	escapedUsername := url.QueryEscape(m.cfg.Username)
	fullURL := serverURL + "?username=" + escapedUsername

	log.Printf("Attempting to connect to: %s", fullURL)
	log.Printf("Username: %s, Admin: %v", m.cfg.Username, *isAdmin)

	u, parseErr := url.Parse(serverURL)
	if parseErr != nil {
		return fmt.Errorf("invalid server URL: %w", parseErr)
	}
	dialer := *websocket.DefaultDialer
	// Gorilla sends an HTTP/1.1 Upgrade on the TLS connection; if ALPN negotiates
	// h2/h3 (common behind Caddy), reads/writes can fail after connect. Force HTTP/1.1.
	if u.Scheme == "wss" {
		tlsCfg := &tls.Config{NextProtos: []string{"http/1.1"}}
		if *skipTLSVerify {
			tlsCfg.InsecureSkipVerify = true
		}
		// TLS SNI must match a name on the cert. Caddy "tls internal" for localhost
		// fails when the URL uses 127.0.0.1 / ::1 (SNI = IP). Dial IP but SNI localhost.
		sni := u.Hostname()
		if sni == "127.0.0.1" || sni == "::1" {
			sni = "localhost"
		}
		tlsCfg.ServerName = sni
		dialer.TLSClientConfig = tlsCfg
	}

	log.Printf("Attempting WebSocket connection to: %s", fullURL)
	conn, resp, err := dialer.Dial(fullURL, nil)
	if err != nil {
		log.Printf("WebSocket dial failed - Error: %v (Type: %T)", err, err)
		if resp != nil {
			log.Printf("HTTP Response - Status: %d, Headers: %v", resp.StatusCode, resp.Header)
			if resp.Body != nil {
				body := make([]byte, 1024)
				if n, readErr := resp.Body.Read(body); readErr == nil && n > 0 {
					log.Printf("Response body: %s", string(body[:n]))
				}
				resp.Body.Close()
			}
			if resp.StatusCode == 403 {
				log.Printf("Connection forbidden - likely duplicate username")
				return wsUsernameError{message: "Username already taken - please choose a different username"}
			}
		}
		return err
	}
	log.Printf("WebSocket connection established")

	m.conn = conn
	m.connected = true
	m.banner = "✅ Connected to server!"
	m.ctx, m.cancel = context.WithCancel(context.Background())
	m.msgChan = make(chan tea.Msg, 256)

	// Send handshake
	handshake := shared.Handshake{
		Username: m.cfg.Username,
		Admin:    *isAdmin,
		AdminKey: "",
	}
	if *isAdmin {
		handshake.AdminKey = *adminKey
	}
	if err := m.conn.WriteJSON(handshake); err != nil {
		log.Printf("Handshake write failed: %v", err)
		m.abortPartialWebSocketConnect()
		return fmt.Errorf("handshake failed: %v", err)
	}
	log.Printf("Handshake sent successfully")

	// WebSocket reader goroutine
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		defer func() {
			m.deliverWSMsg(wsReaderClosed{})
		}()
		for {
			_, rawMsg, readErr := m.conn.ReadMessage()
			if readErr != nil {
				if websocket.IsCloseError(readErr, websocket.CloseNormalClosure) {
					log.Printf("WebSocket closed normally")
					return
				}
				re := readErr.Error()
				if strings.Contains(strings.ToLower(re), "username already taken") ||
					strings.Contains(strings.ToLower(re), "username is already taken") ||
					strings.Contains(strings.ToLower(re), "duplicate username") {
					m.deliverWSMsg(wsUsernameError{message: readErr.Error()})
					return
				}
				m.deliverWSMsg(wsErr{readErr})
				return
			}
			log.Printf("Raw message received - length: %d", len(rawMsg))

			// Try parsing as shared.Message first (has sender field).
			// This must come before the envelope check because both have a "type"
			// JSON key, but only chat messages have "sender".
			var chatMsg shared.Message
			if err := json.Unmarshal(rawMsg, &chatMsg); err == nil && chatMsg.Sender != "" {
				log.Printf("Received chat message from %s, encrypted: %t", chatMsg.Sender, chatMsg.Encrypted)

				if m.useE2E && chatMsg.Encrypted {
					if chatMsg.Type == shared.FileMessageType && chatMsg.File != nil && m.keystore != nil {
						decData, decErr := m.keystore.DecryptRaw(chatMsg.File.Data, "global")
						if decErr != nil {
							log.Printf("ERROR: Failed to decrypt file from %s: %v", chatMsg.Sender, decErr)
						} else {
							chatMsg.File.Data = decData
							chatMsg.File.Size = int64(len(decData))
						}
					} else {
						log.Printf("DEBUG: Attempting to decrypt message from %s", chatMsg.Sender)

						combinedData, decodeErr := base64.StdEncoding.DecodeString(chatMsg.Content)
						if decodeErr != nil {
							log.Printf("ERROR: Base64 decode failed: %v", decodeErr)
							chatMsg.Content = "[ENCRYPTED - DECRYPTION FAILED]"
						} else if len(combinedData) < 12 {
							log.Printf("ERROR: Combined data too short (%d bytes)", len(combinedData))
							chatMsg.Content = "[ENCRYPTED - DECRYPTION FAILED]"
						} else {
							nonce := combinedData[:12]
							ciphertext := combinedData[12:]

							encMsg := &shared.EncryptedMessage{
								Sender:      chatMsg.Sender,
								Nonce:       nonce,
								Encrypted:   ciphertext,
								IsEncrypted: true,
							}

							conversationID := "global"
							decrypted, decryptErr := m.keystore.DecryptMessage(encMsg, conversationID)
							if decryptErr != nil {
								log.Printf("ERROR: Decryption failed for message from %s: %v", chatMsg.Sender, decryptErr)
								chatMsg.Content = "[ENCRYPTED - DECRYPTION FAILED]"
							} else {
								log.Printf("DEBUG: Successfully decrypted message from %s", chatMsg.Sender)
								chatMsg.Content = decrypted.Content
							}
						}
					}
				}

				m.deliverWSMsg(chatMsg)
				continue
			}

			// Try parsing as WSMessage envelope (server-generated, has type+data but no sender)
			var envelope struct {
				Type string          `json:"type"`
				Data json.RawMessage `json:"data"`
			}
			if err := json.Unmarshal(rawMsg, &envelope); err == nil && envelope.Type != "" {
				log.Printf("Received server envelope: type=%s", envelope.Type)
				m.deliverWSMsg(wsMsg{Type: envelope.Type, Data: envelope.Data})
				continue
			}

			log.Printf("WARNING: Unrecognized message format")
		}
	}()

	// Ping goroutine
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if m.conn != nil {
					if err := m.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
						log.Printf("Ping failed: %v", err)
						return
					}
				}
			case <-m.ctx.Done():
				return
			}
		}
	}()

	return nil
}

func (m *model) closeWebSocket() {
	if m.cancel != nil {
		m.cancel()
	}
	if m.conn != nil {
		m.conn.Close()
		m.conn = nil
	}
	m.wg.Wait()
}

func (m *model) listenWebSocket() tea.Cmd {
	return func() tea.Msg {
		return <-m.msgChan
	}
}

func (m *model) renderMessagesContent() string {
	var content strings.Builder
	for _, msg := range m.messages {
		content.WriteString(msg.Content)
		content.WriteString(" ")
	}
	return content.String()
}

func (m *model) findURLAtClickPosition(clickX, clickY int) string {
	allURLs := urlRegex.FindAllString(m.renderMessagesContent(), -1)
	if len(allURLs) == 0 {
		return ""
	}

	adjustedY := clickY - 3

	if adjustedY >= 0 && adjustedY < m.viewport.Height && clickX >= 0 && clickX < m.viewport.Width {
		return allURLs[0]
	}

	return ""
}
