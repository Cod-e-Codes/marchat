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
	"github.com/Cod-e-Codes/marchat/client/exthook"
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

// readReceiptFlushMsg fires after debounce to send one coalesced read_receipt.
type readReceiptFlushMsg struct{}

// encryptGlobalTextWireContent returns base64(nonce ‖ ciphertext) for global chat E2E text,
// matching the wire format produced for normal encrypted messages.
func encryptGlobalTextWireContent(keystore *crypto.KeyStore, username, plaintext string) (string, error) {
	if keystore == nil {
		return "", fmt.Errorf("keystore not initialized")
	}
	if keystore.GetSessionKey("global") == nil {
		return "", fmt.Errorf("global key not available - global E2E encryption not initialized")
	}
	encryptedMsg, err := keystore.EncryptMessage(username, plaintext, "global")
	if err != nil {
		return "", fmt.Errorf("global encryption failed: %w", err)
	}
	if len(encryptedMsg.Encrypted) == 0 {
		return "", fmt.Errorf("encryption returned empty ciphertext")
	}
	combinedData := make([]byte, 0, len(encryptedMsg.Nonce)+len(encryptedMsg.Encrypted))
	combinedData = append(combinedData, encryptedMsg.Nonce...)
	combinedData = append(combinedData, encryptedMsg.Encrypted...)
	finalContent := base64.StdEncoding.EncodeToString(combinedData)
	if len(finalContent) == 0 {
		return "", fmt.Errorf("final content is empty after encoding")
	}
	return finalContent, nil
}

// decryptEncryptedChatContent decrypts opaque wire content (base64 nonce || ciphertext) using the global key.
func decryptEncryptedChatContent(keystore *crypto.KeyStore, chatMsg shared.Message) (string, error) {
	if keystore == nil {
		return "", fmt.Errorf("keystore not initialized")
	}
	combinedData, err := base64.StdEncoding.DecodeString(chatMsg.Content)
	if err != nil {
		return "", fmt.Errorf("base64 decode: %w", err)
	}
	if len(combinedData) < 12 {
		return "", fmt.Errorf("combined data too short (%d bytes)", len(combinedData))
	}
	encMsg := &shared.EncryptedMessage{
		Sender:      chatMsg.Sender,
		Nonce:       combinedData[:12],
		Encrypted:   combinedData[12:],
		IsEncrypted: true,
	}
	decrypted, err := keystore.DecryptMessage(encMsg, "global")
	if err != nil {
		return "", err
	}
	return decrypted.Content, nil
}

// buildEncryptedOutboundMessage returns a chat message with opaque E2E content (base64 nonce || ciphertext).
func buildEncryptedOutboundMessage(keystore *crypto.KeyStore, username, plaintext string, msgType shared.MessageType, recipient string) (shared.Message, error) {
	if msgType == "" {
		msgType = shared.TextMessage
	}
	finalContent, err := encryptGlobalTextWireContent(keystore, username, plaintext)
	if err != nil {
		return shared.Message{}, err
	}
	msg := shared.Message{
		Content:   finalContent,
		Sender:    username,
		CreatedAt: time.Now(),
		Type:      msgType,
		Encrypted: true,
	}
	if recipient != "" {
		msg.Recipient = recipient
	}
	return msg, nil
}

func sendEncryptedChatMessage(ws *websocket.Conn, keystore *crypto.KeyStore, username, plaintext string, msgType shared.MessageType, recipient string) error {
	if keystore == nil {
		return fmt.Errorf("keystore not initialized")
	}
	msg, err := buildEncryptedOutboundMessage(keystore, username, plaintext, msgType, recipient)
	if err != nil {
		return err
	}
	return ws.WriteJSON(msg)
}

func sendDirectMessage(ws *websocket.Conn, keystore *crypto.KeyStore, username, recipient, content string, useE2E bool) error {
	if recipient == "" {
		return fmt.Errorf("dm recipient is required")
	}
	if useE2E {
		if err := verifyKeystoreUnlocked(keystore); err != nil {
			return err
		}
		return sendEncryptedChatMessage(ws, keystore, username, content, shared.DirectMessage, recipient)
	}
	return ws.WriteJSON(shared.Message{
		Type:      shared.DirectMessage,
		Sender:    username,
		Recipient: recipient,
		Content:   content,
		CreatedAt: time.Now(),
	})
}

// sendSnippetOutbound sends a code-snippet body as a DM when dmRecipient is set, otherwise as channel text.
func sendSnippetOutbound(ws *websocket.Conn, keystore *crypto.KeyStore, username, dmRecipient, content string, useE2E bool, channelRecipients []string) error {
	recipient := strings.TrimSpace(dmRecipient)
	if recipient != "" {
		return sendDirectMessage(ws, keystore, username, recipient, content, useE2E)
	}
	if useE2E {
		recipients := channelRecipients
		if len(recipients) == 0 {
			recipients = []string{username}
		}
		return debugEncryptAndSend(recipients, content, ws, keystore, username)
	}
	return ws.WriteJSON(shared.Message{
		Sender:  username,
		Content: content,
	})
}

func debugEncryptAndSend(recipients []string, plaintext string, ws *websocket.Conn, keystore *crypto.KeyStore, username string) error {
	_ = recipients
	if keystore == nil {
		return fmt.Errorf("keystore not initialized")
	}
	if keystore.GetSessionKey("global") == nil {
		return fmt.Errorf("global key not available - global E2E encryption not initialized")
	}
	return sendEncryptedChatMessage(ws, keystore, username, plaintext, shared.TextMessage, "")
}

func validateEncryptionRoundtrip(keystore *crypto.KeyStore, username string) error {
	testPlaintext := "Hello, global encryption test!"
	conversationID := "global"

	globalKey := keystore.GetSessionKey(conversationID)
	if globalKey == nil {
		return fmt.Errorf("global key not found - global E2E encryption not available")
	}

	encryptedMsg, err := keystore.EncryptMessage(username, testPlaintext, conversationID)
	if err != nil {
		return fmt.Errorf("global encryption test failed: %v", err)
	}

	if len(encryptedMsg.Encrypted) == 0 {
		return fmt.Errorf("global encryption test produced empty ciphertext")
	}

	decryptedMsg, err := keystore.DecryptMessage(encryptedMsg, conversationID)
	if err != nil {
		return fmt.Errorf("global decryption test failed: %v", err)
	}

	if decryptedMsg.Content != testPlaintext {
		return fmt.Errorf("global decryption roundtrip failed: expected '%s', got '%s'", testPlaintext, decryptedMsg.Content)
	}

	return nil
}

func verifyKeystoreUnlocked(keystore *crypto.KeyStore) error {
	if keystore == nil {
		return fmt.Errorf("keystore is nil")
	}
	if keystore.GetGlobalKey() == nil {
		return fmt.Errorf("global key not available")
	}
	return nil
}

func debugWebSocketWrite(ws *websocket.Conn, msg interface{}) error {
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

// sanitizeServerURL trims whitespace and matching quotes often pasted from docs or PowerShell.
func sanitizeServerURL(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.Trim(s, `"'`+"`")
	// Unicode curly quotes (copy-paste from word processors)
	for {
		trimmed := strings.TrimPrefix(s, "\u2018")      // ‘
		trimmed = strings.TrimPrefix(trimmed, "\u201c") // “
		trimmed = strings.TrimSuffix(trimmed, "\u2019") // ’
		trimmed = strings.TrimSuffix(trimmed, "\u201d") // ”
		if trimmed == s {
			break
		}
		s = strings.TrimSpace(trimmed)
	}
	return strings.TrimSpace(s)
}

func (m *model) connectWebSocket(serverURL string) error {
	serverURL = sanitizeServerURL(serverURL)
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
	m.banner = "[OK] Connected to server."
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
						plaintext, decryptErr := decryptEncryptedChatContent(m.keystore, chatMsg)
						if decryptErr != nil {
							log.Printf("ERROR: Decryption failed for message from %s: %v", chatMsg.Sender, decryptErr)
							chatMsg.Content = "[ENCRYPTED - DECRYPTION FAILED]"
						} else {
							chatMsg.Content = plaintext
						}
					}
				}

				exthook.FireReceive(chatMsg)
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
	m.readReceiptFlushScheduled = false
	m.wg.Wait()
}

func (m *model) listenWebSocket() tea.Cmd {
	return func() tea.Msg {
		return <-m.msgChan
	}
}

// chatPanelOrigin returns terminal coordinates of the top-left of the chat transcript viewport.
func (m *model) chatPanelOrigin() (x0, y0 int) {
	x0 = userListWidth + 1 // user list + chat box left border
	y0 = 2                 // header row + chat box top border row
	if m.banner != "" || m.sending {
		bannerText := m.banner
		if m.sending && strings.TrimSpace(bannerText) == "" {
			bannerText = "[Sending...]"
		}
		fullW := chromeFullWidth(m.viewport.Width)
		shown := layoutBannerForStrip(bannerText, fullW)
		y0 += strings.Count(shown, "\n") + 1
	}
	return x0, y0
}

func trimViewportViewLines(lines []string) {
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}
}

func (m *model) findURLAtClickPosition(clickX, clickY int) string {
	x0, y0 := m.chatPanelOrigin()
	relX := clickX - x0
	relY := clickY - y0
	if relX < 0 || relY < 0 || relX >= m.viewport.Width || relY >= m.viewport.Height {
		return ""
	}

	lines := strings.Split(m.viewport.View(), "\n")
	if relY >= len(lines) {
		return ""
	}
	trimViewportViewLines(lines)

	lineIdx := m.viewport.YOffset + relY
	if u := urlFromTranscriptIndex(m.transcriptLineURLs, lineIdx, relX, lines[relY]); u != "" {
		return u
	}

	partial := findURLAtTranscriptClick(lines, relY, relX)
	return expandClickedURL(partial, m.visibleMessages())
}
