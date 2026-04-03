package server

import (
	"database/sql"
	"log"
	"strings"
	"time"

	"github.com/Cod-e-Codes/marchat/shared"
)

func normalizeReactionTarget(msg shared.Message) int64 {
	if msg.MessageID > 0 {
		return msg.MessageID
	}
	if msg.Reaction != nil && msg.Reaction.TargetID > 0 {
		return msg.Reaction.TargetID
	}
	return 0
}

func PersistReaction(db *sql.DB, msg shared.Message) {
	targetID := normalizeReactionTarget(msg)
	if db == nil || msg.Reaction == nil || targetID == 0 || strings.TrimSpace(msg.Sender) == "" {
		return
	}
	if msg.Reaction.IsRemoval {
		if _, err := dbExec(db, `DELETE FROM message_reactions WHERE message_id = ? AND username = ? AND emoji = ?`, targetID, msg.Sender, msg.Reaction.Emoji); err != nil {
			log.Printf("warning: delete reaction failed: %v", err)
		}
		return
	}
	if _, err := dbExec(db, insertIgnoreReactionSQL(db), targetID, msg.Sender, msg.Reaction.Emoji); err != nil {
		log.Printf("warning: persist reaction failed: %v", err)
	}
}

func LoadReactionsForMessages(db *sql.DB, messageIDs []int64) []shared.Message {
	if db == nil || len(messageIDs) == 0 {
		return nil
	}
	args := make([]interface{}, 0, len(messageIDs))
	ph := make([]string, 0, len(messageIDs))
	for _, id := range messageIDs {
		ph = append(ph, "?")
		args = append(args, id)
	}
	rows, err := dbQuery(db, `SELECT message_id, username, emoji FROM message_reactions WHERE message_id IN (`+strings.Join(ph, ",")+`) ORDER BY created_at ASC`, args...)
	if err != nil {
		log.Printf("warning: load reactions failed: %v", err)
		return nil
	}
	defer rows.Close()

	out := make([]shared.Message, 0)
	for rows.Next() {
		var messageID int64
		var username, emoji string
		if err := rows.Scan(&messageID, &username, &emoji); err != nil {
			continue
		}
		out = append(out, shared.Message{
			Type:      shared.ReactionMessage,
			Sender:    username,
			CreatedAt: time.Now(),
			MessageID: messageID,
			Reaction: &shared.ReactionMeta{
				Emoji:     emoji,
				TargetID:  messageID,
				IsRemoval: false,
			},
		})
	}
	return out
}

func PersistUserChannel(db *sql.DB, username, channel string) {
	if db == nil || strings.TrimSpace(username) == "" || strings.TrimSpace(channel) == "" {
		return
	}
	if _, err := dbExec(db, upsertUserChannelSQL(db), strings.ToLower(username), strings.ToLower(channel)); err != nil {
		log.Printf("warning: persist user channel failed: %v", err)
	}
}

func LoadUserChannel(db *sql.DB, username string) string {
	if db == nil || strings.TrimSpace(username) == "" {
		return ""
	}
	var channel string
	if err := dbQueryRow(db, `SELECT channel FROM user_channels WHERE username = ?`, strings.ToLower(username)).Scan(&channel); err != nil {
		return ""
	}
	if channel == "" {
		return ""
	}
	return channel
}

func PersistReadReceipt(db *sql.DB, username string, messageID int64) {
	if db == nil || strings.TrimSpace(username) == "" || messageID <= 0 {
		return
	}
	if _, err := dbExec(db, insertIgnoreReadReceiptSQL(db), strings.ToLower(username), messageID); err != nil {
		log.Printf("warning: persist read receipt failed: %v", err)
	}
}

func LoadReadReceiptsForMessages(db *sql.DB, username string, messageIDs []int64) []shared.Message {
	if db == nil || strings.TrimSpace(username) == "" || len(messageIDs) == 0 {
		return nil
	}

	args := make([]interface{}, 0, len(messageIDs)+1)
	args = append(args, strings.ToLower(username))
	ph := make([]string, 0, len(messageIDs))
	for _, id := range messageIDs {
		ph = append(ph, "?")
		args = append(args, id)
	}

	rows, err := dbQuery(db, `SELECT message_id, read_at FROM read_receipts WHERE username = ? AND message_id IN (`+strings.Join(ph, ",")+`) ORDER BY read_at ASC`, args...)
	if err != nil {
		log.Printf("warning: load read receipts failed: %v", err)
		return nil
	}
	defer rows.Close()

	var out []shared.Message
	for rows.Next() {
		var messageID int64
		var readAt time.Time
		if err := rows.Scan(&messageID, &readAt); err != nil {
			continue
		}
		out = append(out, shared.Message{
			Type:      shared.ReadReceiptType,
			Sender:    username,
			MessageID: messageID,
			CreatedAt: readAt,
		})
	}
	return out
}
