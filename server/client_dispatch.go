package server

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Cod-e-Codes/marchat/shared"
)

func (c *Client) dispatchInbound(msg *shared.Message) {
	if msg.Type == shared.FileMessageType && msg.File != nil {
		c.handleInboundFile(msg)
		return
	}

	if msg.Type == shared.EditMessageType && msg.MessageID > 0 {
		c.handleInboundEdit(msg)
		return
	}

	if msg.Type == shared.DeleteMessage && msg.MessageID > 0 {
		c.handleInboundDelete(msg)
		return
	}

	if msg.Type == shared.TypingMessage {
		c.handleInboundTyping(msg)
		return
	}

	if msg.Type == shared.ReactionMessage && msg.Reaction != nil {
		c.handleInboundReaction(msg)
		return
	}

	if msg.Type == shared.DirectMessage && msg.Recipient != "" {
		c.handleInboundDM(msg)
		return
	}

	if msg.Type == shared.SearchMessage {
		c.handleInboundSearch(msg)
		return
	}

	if msg.Type == shared.PinMessage {
		c.handleInboundPin(msg)
		return
	}

	if msg.Type == shared.ReadReceiptType {
		c.handleInboundReadReceipt(msg)
		return
	}

	if msg.Type == shared.JoinChannelType && msg.Channel != "" {
		c.handleInboundJoinChannel(msg)
		return
	}
	if msg.Type == shared.LeaveChannelType {
		c.handleInboundLeaveChannel(msg)
		return
	}
	if msg.Type == shared.ListChannelsType {
		c.handleInboundListChannels(msg)
		return
	}

	if strings.HasPrefix(msg.Content, ":") || msg.Type == shared.AdminCommandType {
		c.handleInboundCommand(msg)
		return
	}

	c.handleInboundText(msg)
}

func (c *Client) handleInboundFile(msg *shared.Message) {
	maxBytes := c.maxFileBytes
	if maxBytes <= 0 {
		maxBytes = 1024 * 1024
	}
	if msg.File.Size > maxBytes || int64(len(msg.File.Data)) > maxBytes {
		log.Printf("Rejected file from %s: too large (declared %d bytes, payload %d bytes)", c.username, msg.File.Size, len(msg.File.Data))
		c.send <- fileTooLargeSystemMessage(maxBytes)
		return
	}
	c.stampSenderTimedOutbound(msg)
	c.hub.broadcast <- *msg
}

func (c *Client) handleInboundEdit(msg *shared.Message) {
	if contentContainsNUL(msg.Content) {
		c.send <- shared.Message{
			Sender:    "System",
			Content:   "Message not sent: invalid character in content",
			CreatedAt: time.Now(),
			Type:      shared.TextMessage,
		}
		return
	}
	if plaintextContentEmpty(msg.Content, msg.Encrypted) {
		c.send <- shared.Message{
			Sender:    "System",
			Content:   "Message not sent: empty content",
			CreatedAt: time.Now(),
			Type:      shared.TextMessage,
		}
		return
	}
	if err := EditMessage(c.db, msg.MessageID, c.username, msg.Content, msg.Encrypted); err != nil {
		c.send <- shared.Message{
			Sender:    "System",
			Content:   "Edit failed: " + err.Error(),
			CreatedAt: time.Now(),
			Type:      shared.TextMessage,
		}
	} else {
		msg.Edited = true
		c.stampSenderTimedOutbound(msg)
		c.hub.broadcast <- *msg
	}
}

func (c *Client) handleInboundDelete(msg *shared.Message) {
	if err := DeleteMessage(c.db, msg.MessageID, c.username, c.isAdmin); err != nil {
		c.send <- shared.Message{
			Sender:    "System",
			Content:   "Delete failed: " + err.Error(),
			CreatedAt: time.Now(),
			Type:      shared.TextMessage,
		}
	} else {
		c.stampSenderTimedOutbound(msg)
		c.hub.broadcast <- *msg
	}
}

func (c *Client) handleInboundTyping(msg *shared.Message) {
	msg.Sender = c.username
	if strings.TrimSpace(msg.Recipient) != "" {
		c.hub.broadcastDM(*msg)
	} else {
		c.stampClientChannel(msg)
		c.hub.broadcast <- *msg
	}
}

func (c *Client) handleInboundReaction(msg *shared.Message) {
	c.stampSenderTimedOutbound(msg)
	PersistReaction(c.db, *msg)
	c.hub.broadcast <- *msg
}

func (c *Client) handleInboundDM(msg *shared.Message) {
	if contentContainsNUL(msg.Content) {
		c.send <- shared.Message{
			Sender:    "System",
			Content:   "Message not sent: invalid character in content",
			CreatedAt: time.Now(),
			Type:      shared.TextMessage,
		}
		return
	}
	if plaintextContentEmpty(msg.Content, msg.Encrypted) {
		c.send <- shared.Message{
			Sender:    "System",
			Content:   "Message not sent: empty content",
			CreatedAt: time.Now(),
			Type:      shared.TextMessage,
		}
		return
	}
	c.stampSenderTimedOutbound(msg)
	msgID, err := InsertMessage(c.db, *msg)
	if err != nil {
		log.Printf("Failed to persist DM from %s to %s: %v", c.username, msg.Recipient, err)
		c.send <- shared.Message{
			Sender:    "System",
			Content:   "Message not sent: could not save message",
			CreatedAt: time.Now(),
			Type:      shared.TextMessage,
		}
		return
	}
	msg.MessageID = msgID
	c.hub.broadcastDM(*msg)
}

func (c *Client) handleInboundSearch(msg *shared.Message) {
	results := SearchMessages(c.db, msg.Content, 20)
	var sb strings.Builder
	if len(results) == 0 {
		fmt.Fprintf(&sb, "No results found for: %s", msg.Content)
	} else {
		sb.WriteString(fmt.Sprintf("Search results for '%s' (%d found):\n", msg.Content, len(results)))
		for _, r := range results {
			sb.WriteString(fmt.Sprintf("  [%s] %s: %s\n", r.CreatedAt.Format("01/02 15:04"), r.Sender, r.Content))
		}
	}
	c.send <- shared.Message{
		Sender:    "System",
		Content:   sb.String(),
		CreatedAt: time.Now(),
		Type:      shared.TextMessage,
	}
}

func (c *Client) handleInboundPin(msg *shared.Message) {
	if msg.MessageID == 0 {
		pinned := GetPinnedMessages(c.db)
		var sb strings.Builder
		if len(pinned) == 0 {
			sb.WriteString("No pinned messages")
		} else {
			sb.WriteString(fmt.Sprintf("Pinned messages (%d):\n", len(pinned)))
			for _, p := range pinned {
				sb.WriteString(fmt.Sprintf("  #%d [%s] %s: %s\n", p.MessageID, p.CreatedAt.Format("01/02 15:04"), p.Sender, p.Content))
			}
		}
		c.send <- shared.Message{
			Sender:    "System",
			Content:   sb.String(),
			CreatedAt: time.Now(),
			Type:      shared.TextMessage,
		}
		return
	}
	if !c.isAdmin {
		c.send <- shared.Message{
			Sender:    "System",
			Content:   "Only admins can pin messages",
			CreatedAt: time.Now(),
			Type:      shared.TextMessage,
		}
		return
	}
	pinned, err := TogglePinMessage(c.db, msg.MessageID)
	if err != nil {
		c.send <- shared.Message{
			Sender:    "System",
			Content:   "Pin failed: " + err.Error(),
			CreatedAt: time.Now(),
			Type:      shared.TextMessage,
		}
	} else {
		action := "pinned"
		if !pinned {
			action = "unpinned"
		}
		c.hub.broadcast <- shared.Message{
			Sender:    "System",
			Content:   fmt.Sprintf("Message %d %s by %s", msg.MessageID, action, c.username),
			CreatedAt: time.Now(),
			Type:      shared.TextMessage,
		}
	}
}

func (c *Client) handleInboundReadReceipt(msg *shared.Message) {
	msg.Sender = c.username
	if msg.MessageID > 0 {
		PersistReadReceipt(c.db, c.username, msg.MessageID)
	}
	c.stampClientChannel(msg)
	c.hub.broadcast <- *msg
}

func (c *Client) handleInboundJoinChannel(msg *shared.Message) {
	msg.Channel = strings.ToLower(strings.TrimSpace(msg.Channel))
	if msg.Channel == "" {
		return
	}
	old := c.hub.getClientChannel(c)
	if old != msg.Channel {
		c.hub.leaveChannel(c, old)
	}
	c.hub.joinChannel(c, msg.Channel)
	PersistUserChannel(c.db, c.username, msg.Channel)
	c.send <- shared.Message{
		Sender:    "System",
		Content:   "Joined channel #" + msg.Channel,
		CreatedAt: time.Now(),
		Type:      shared.TextMessage,
	}
}

func (c *Client) handleInboundLeaveChannel(msg *shared.Message) {
	current := c.hub.getClientChannel(c)
	if current != "general" {
		c.hub.leaveChannel(c, current)
		c.hub.joinChannel(c, "general")
		PersistUserChannel(c.db, c.username, "general")
		c.send <- shared.Message{
			Sender:    "System",
			Content:   "Left #" + current + ", back to #general",
			CreatedAt: time.Now(),
			Type:      shared.TextMessage,
		}
	}
}

func (c *Client) handleInboundListChannels(msg *shared.Message) {
	channels := c.hub.listChannels()
	current := c.hub.getClientChannel(c)
	var lines []string
	for _, ch := range channels {
		n := len(c.hub.getChannelUsers(ch))
		marker := ""
		if ch == current {
			marker = " (current)"
		}
		lines = append(lines, fmt.Sprintf("  #%s - %d user(s)%s", ch, n, marker))
	}
	c.send <- shared.Message{
		Sender:    "System",
		Content:   "Active channels:\n" + strings.Join(lines, "\n"),
		CreatedAt: time.Now(),
		Type:      shared.TextMessage,
	}
}

func (c *Client) handleInboundCommand(msg *shared.Message) {
	AdminLogger.Info("Command received", map[string]interface{}{
		"user":    c.username,
		"command": msg.Content,
		"admin":   c.isAdmin,
		"type":    msg.Type,
	})
	c.handleCommand(msg.Content)
}

func (c *Client) handleInboundText(msg *shared.Message) {
	if contentContainsNUL(msg.Content) {
		c.send <- shared.Message{
			Sender:    "System",
			Content:   "Message not sent: invalid character in content",
			CreatedAt: time.Now(),
			Type:      shared.TextMessage,
		}
		return
	}
	if plaintextContentEmpty(msg.Content, msg.Encrypted) {
		c.send <- shared.Message{
			Sender:    "System",
			Content:   "Message not sent: empty content",
			CreatedAt: time.Now(),
			Type:      shared.TextMessage,
		}
		return
	}
	c.stampSenderTimedOutbound(msg)
	if msg.Type == "" || msg.Type == shared.TextMessage {
		msgID, err := InsertMessage(c.db, *msg)
		if err != nil {
			log.Printf("Failed to persist message from %s: %v", c.username, err)
			c.send <- shared.Message{
				Sender:    "System",
				Content:   "Message not sent: could not save message",
				CreatedAt: time.Now(),
				Type:      shared.TextMessage,
			}
			return
		}
		msg.MessageID = msgID
	}
	c.hub.broadcast <- *msg
}
