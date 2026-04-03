package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strings"

	"github.com/Cod-e-Codes/marchat/shared"
	"github.com/alecthomas/chroma/quick"
	"github.com/charmbracelet/lipgloss"
)

// sortMessagesByTimestamp ensures messages are displayed in chronological order
// This provides client-side protection against server ordering issues
func sortMessagesByTimestamp(messages []shared.Message) {
	sort.Slice(messages, func(i, j int) bool {
		if !messages[i].CreatedAt.Equal(messages[j].CreatedAt) {
			return messages[i].CreatedAt.Before(messages[j].CreatedAt)
		}
		if messages[i].Sender != messages[j].Sender {
			return messages[i].Sender < messages[j].Sender
		}
		return messages[i].Content < messages[j].Content
	})
}

func renderMessages(msgs []shared.Message, styles themeStyles, username string, users []string, width int, twentyFourHour bool, reactions ...map[int64]map[string]map[string]bool) string {
	var reactionMap map[int64]map[string]map[string]bool
	if len(reactions) > 0 {
		reactionMap = reactions[0]
	}
	if len(msgs) == 0 {
		return styles.Info.Render("No messages yet. Say hi!")
	}

	var b strings.Builder
	sortMessagesByTimestamp(msgs)

	var lastDate string
	for _, msg := range msgs {
		if msg.Type == shared.TypingMessage || msg.Type == shared.ReadReceiptType {
			continue
		}

		dateStr := msg.CreatedAt.Format("January 2, 2006")
		if dateStr != lastDate {
			b.WriteString("\n" + styles.Info.Width(width).Align(lipgloss.Center).Render("─── "+dateStr+" ───") + "\n\n")
			lastDate = dateStr
		}

		timeFormat := "3:04 PM"
		if twentyFourHour {
			timeFormat = "15:04"
		}
		timeStr := msg.CreatedAt.Format(timeFormat)
		timestamp := styles.Timestamp.Render("[" + timeStr + "]")

		var prefix string
		if msg.Type == shared.DirectMessage {
			prefix += lipgloss.NewStyle().Foreground(lipgloss.Color("#FF69B4")).Render("[DM] ")
		}
		if msg.Edited {
			prefix += styles.Timestamp.Render("(edited) ")
		}

		content := msg.Content
		if msg.Type == shared.DeleteMessage {
			content = styles.Timestamp.Render("[deleted]")
		} else if msg.Type == shared.FileMessageType && msg.File != nil {
			content = fmt.Sprintf("File: %s (%d bytes); use :savefile %s to save", msg.File.Filename, msg.File.Size, msg.File.Filename)
		} else {
			content = renderEmojis(content)
			content = renderCodeBlocks(content)
			content = renderHyperlinks(content, styles)
			if mentionRegex != nil {
				content = mentionRegex.ReplaceAllStringFunc(content, func(match string) string {
					mentionName := strings.TrimPrefix(match, "@")
					for _, u := range users {
						if strings.EqualFold(u, mentionName) {
							return styles.Mention.Render(match)
						}
					}
					return match
				})
			}
		}

		var metadata []string
		if msg.MessageID > 0 {
			metadata = append(metadata, fmt.Sprintf("id:%d", msg.MessageID))
		}
		if msg.Encrypted {
			metadata = append(metadata, "encrypted")
		}
		metaSuffix := ""
		if len(metadata) > 0 {
			metaSuffix = " " + styles.Timestamp.Render("["+strings.Join(metadata, ", ")+"]")
		}

		switch msg.Sender {
		case "System":
			b.WriteString(timestamp + " " + prefix + styles.Info.Render(content) + metaSuffix + "\n")
		case username:
			b.WriteString(timestamp + " " + prefix + styles.Me.Render(msg.Sender) + ": " + content + metaSuffix + "\n")
		default:
			b.WriteString(timestamp + " " + prefix + styles.Other.Render(msg.Sender) + ": " + content + metaSuffix + "\n")
		}

		if reactionMap != nil && msg.MessageID > 0 {
			if emojiMap, ok := reactionMap[msg.MessageID]; ok && len(emojiMap) > 0 {
				var parts []string
				for emoji, users := range emojiMap {
					if len(users) > 0 {
						parts = append(parts, fmt.Sprintf("%s %d", emoji, len(users)))
					}
				}
				if len(parts) > 0 {
					reactionLine := "       " + lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(strings.Join(parts, "  "))
					b.WriteString(reactionLine + "\n")
				}
			}
		}
	}
	return b.String()
}

func renderEmojis(s string) string {
	emojis := map[string]string{
		":)": "😊",
		":(": "🙁",
		":D": "😃",
		"<3": "❤️",
		":P": "😛",
	}
	for k, v := range emojis {
		s = strings.ReplaceAll(s, k, v)
	}
	return s
}

var reactionAliases = map[string]string{
	"+1":     "👍",
	"-1":     "👎",
	"heart":  "❤️",
	"laugh":  "😂",
	"fire":   "🔥",
	"party":  "🎉",
	"eyes":   "👀",
	"check":  "✅",
	"x":      "❌",
	"think":  "🤔",
	"clap":   "👏",
	"rocket": "🚀",
	"wave":   "👋",
	"100":    "💯",
	"sad":    "😢",
	"wow":    "😮",
	"angry":  "😡",
	"skull":  "💀",
	"pray":   "🙏",
	"star":   "⭐",
}

func resolveReactionEmoji(input string) string {
	if emoji, ok := reactionAliases[strings.ToLower(input)]; ok {
		return emoji
	}
	return input
}

func renderHyperlinks(content string, styles themeStyles) string {
	return urlRegex.ReplaceAllStringFunc(content, func(url string) string {
		return styles.Hyperlink.Render(url)
	})
}

func openURL(u string) error {
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		u = "https://" + u
	}

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", u)
		if err := cmd.Start(); err != nil {
			cmd = exec.Command("cmd", "/c", "start", u)
			return cmd.Start()
		}
		return nil
	case "darwin":
		cmd = exec.Command("open", u)
	case "linux":
		cmd = exec.Command("xdg-open", u)
		if err := cmd.Start(); err != nil {
			cmd = exec.Command("sensible-browser", u)
			if err := cmd.Start(); err != nil {
				cmd = exec.Command("firefox", u)
				return cmd.Start()
			}
			return nil
		}
		return nil
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}

func renderCodeBlocks(content string) string {
	codeBlockRegex := regexp.MustCompile("```([a-zA-Z0-9+]*)\n([\\s\\S]*?)```")

	return codeBlockRegex.ReplaceAllStringFunc(content, func(match string) string {
		parts := codeBlockRegex.FindStringSubmatch(match)
		if len(parts) < 3 {
			return match
		}

		language := parts[1]
		code := parts[2]

		var sb strings.Builder
		err := quick.Highlight(&sb, code, language, "terminal256", "monokai")
		if err != nil {
			return match
		}

		return sb.String()
	})
}

func renderUserList(users []string, me string, styles themeStyles, width int, isAdmin bool, selectedUserIndex int) string {
	var b strings.Builder
	title := " Users "
	b.WriteString(styles.UserList.Width(width).Render(title) + "\n")
	max := maxUsersDisplay
	for i, u := range users {
		if i >= max {
			b.WriteString(lipgloss.NewStyle().Italic(true).Faint(true).Width(width).Render(fmt.Sprintf("+%d more", len(users)-max)) + "\n")
			break
		}

		var userStyle lipgloss.Style
		var prefix string

		if u == me {
			userStyle = styles.Me
			prefix = "• "
		} else {
			userStyle = styles.Other
			prefix = "• "

			if isAdmin && selectedUserIndex == i {
				userStyle = userStyle.Background(lipgloss.Color("#444444")).Bold(true)
				prefix = "► "
			}
		}

		b.WriteString(userStyle.Render(prefix+u) + "\n")
	}
	return b.String()
}
