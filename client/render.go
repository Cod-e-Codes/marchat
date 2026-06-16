package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/Cod-e-Codes/marchat/shared"
	"github.com/alecthomas/chroma/quick"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// sortMessagesByTimestamp ensures messages are displayed in chronological order.
// Persisted chat (message_id > 0) sorts by id; server System lines (id == 0) use
// created_at; client-local System feedback (negative id) stays after persisted chat
// so clock skew cannot pin it above new messages.
func sortMessagesByTimestamp(messages []shared.Message) {
	sort.Slice(messages, func(i, j int) bool {
		return messageLess(messages[i], messages[j])
	})
}

func messageLess(a, b shared.Message) bool {
	aClient := a.MessageID < 0
	bClient := b.MessageID < 0
	if aClient != bClient {
		return !aClient
	}
	if aClient && bClient {
		if a.MessageID != b.MessageID {
			return a.MessageID < b.MessageID
		}
		return a.Content < b.Content
	}

	if a.MessageID > 0 && b.MessageID > 0 {
		if a.MessageID != b.MessageID {
			return a.MessageID < b.MessageID
		}
	}

	if !a.CreatedAt.Equal(b.CreatedAt) {
		return a.CreatedAt.Before(b.CreatedAt)
	}
	if a.MessageID > 0 && b.MessageID == 0 {
		return true
	}
	if a.MessageID == 0 && b.MessageID > 0 {
		return false
	}
	if a.Sender != b.Sender {
		return a.Sender < b.Sender
	}
	return a.Content < b.Content
}

// systemBannerText formats ephemeral System feedback for the banner strip.
func systemBannerText(content string) string {
	t := strings.TrimSpace(content)
	switch systemLineSeverityClass(t) {
	case systemLineErr:
		if !strings.HasPrefix(t, "[ERROR]") {
			return "[ERROR] " + t
		}
	case systemLineWarn:
		if !strings.HasPrefix(t, "[WARN]") {
			return "[WARN] " + t
		}
	}
	return t
}

// isTranscriptSystemNotice reports whether a System line should stay in the scrollable
// transcript. Command errors, usage, and other ephemeral feedback belong in the banner.
func isTranscriptSystemNotice(content string) bool {
	c := strings.TrimSpace(content)
	if c == "" {
		return false
	}
	cl := strings.ToLower(c)
	if c == e2eSearchNoResultsHint {
		return true
	}
	if strings.Count(c, "\n") >= 1 {
		return true
	}
	switch {
	case strings.HasPrefix(cl, "search results for "),
		strings.HasPrefix(cl, strings.ToLower(searchNoResultsPrefix)),
		strings.HasPrefix(cl, "pinned messages"),
		cl == "no pinned messages",
		strings.HasPrefix(cl, "active channels:"),
		strings.HasPrefix(cl, "joined channel #"),
		strings.HasPrefix(cl, "left #"),
		strings.HasPrefix(cl, "chat history cleared"),
		strings.HasPrefix(cl, "you have been kicked"),
		strings.HasPrefix(cl, "message ") && (strings.Contains(cl, " pinned by ") || strings.Contains(cl, " unpinned by ")),
		strings.Contains(cl, "has been kicked"),
		strings.Contains(cl, "permanently banned"),
		strings.Contains(cl, "has been unbanned"),
		strings.Contains(cl, "forcibly disconnected"),
		strings.HasPrefix(cl, "available themes:"),
		strings.HasPrefix(cl, "dm conversations:"):
		return true
	}
	return false
}

// isTranscriptSystemMessage reports whether a System wire/local message belongs in
// the transcript rather than the ephemeral banner.
func isTranscriptSystemMessage(msg shared.Message) bool {
	if msg.Sender != "System" {
		return true
	}
	if msg.MessageID < 0 {
		return isTranscriptSystemNotice(msg.Content)
	}
	if msg.MessageID > 0 {
		return true
	}
	return isTranscriptSystemNotice(msg.Content)
}

type systemLineSeverity int

const (
	systemLineInfo systemLineSeverity = iota
	systemLineWarn
	systemLineErr
)

// systemLineSeverityClass classifies System sender content for transcript coloring.
func systemLineSeverityClass(content string) systemLineSeverity {
	t := strings.TrimSpace(content)
	tl := strings.ToLower(t)
	switch {
	case strings.HasPrefix(tl, "[error]"):
		return systemLineErr
	case strings.HasPrefix(tl, "[warn]"):
		return systemLineWarn
	case strings.HasPrefix(tl, "unknown "),
		strings.HasPrefix(tl, "invalid "),
		tl == "error",
		strings.HasPrefix(tl, "error "),
		strings.HasPrefix(tl, "error:"),
		strings.Contains(tl, " not found"),
		strings.Contains(tl, " not allowed"),
		strings.Contains(tl, "failed"):
		return systemLineErr
	default:
		return systemLineInfo
	}
}

// systemLineStyle picks transcript styling for Server "System" lines so errors
// and warnings are not the same color as normal notices.
func systemLineStyle(styles themeStyles, content string) lipgloss.Style {
	switch systemLineSeverityClass(content) {
	case systemLineErr:
		return styles.SystemMsgError
	case systemLineWarn:
		return styles.SystemMsgWarn
	default:
		return styles.SystemMsg
	}
}

const (
	// urlNBHyphen is a non-breaking hyphen so ansi.Wrap does not split URLs at
	// domain/path hyphens (e.g. Cod-e-Codes). Restored for click-to-open matching.
	urlNBHyphen = '\u2011'
	// urlStartMarker and urlEndMarker are zero-width sentinels around URL spans so
	// hyperlink styling survives ansi.Wrap line breaks (removed before display).
	urlStartMarker = '\u200B'
	urlEndMarker   = '\u200C'
)

// wrapBreakpoints are characters where line wrapping may occur. Slashes and
// query delimiters let long URLs break at path boundaries instead of mid-token.
const wrapBreakpoints = " /?#&="

// prepareURLWrapping adjusts URL text so terminal wrapping prefers path segments
// over hyphens inside host/path components.
func prepareURLWrapping(s string) string {
	if urlRegex == nil {
		return s
	}
	return urlRegex.ReplaceAllStringFunc(s, func(url string) string {
		return strings.ReplaceAll(url, "-", string(urlNBHyphen))
	})
}

// markURLsForWrap inserts zero-width sentinels around detected URLs before wrap.
func markURLsForWrap(s string) string {
	if urlRegex == nil {
		return s
	}
	return urlRegex.ReplaceAllStringFunc(s, func(url string) string {
		return string(urlStartMarker) + url + string(urlEndMarker)
	})
}

// applyURLMarkers renders hyperlink style for marked URL spans on one wrapped line.
// open tracks an URL span that continues from the previous wrapped line.
func applyURLMarkers(line string, styles themeStyles, open *bool) string {
	var out strings.Builder
	var segment strings.Builder
	link := *open

	writeSegment := func() {
		if segment.Len() == 0 {
			return
		}
		if link {
			out.WriteString(styles.Hyperlink.Render(segment.String()))
		} else {
			out.WriteString(segment.String())
		}
		segment.Reset()
	}

	pos := 0
	for pos < len(line) {
		if line[pos] == '\x1b' {
			end := strings.IndexByte(line[pos:], 'm')
			if end < 0 {
				segment.WriteByte(line[pos])
				pos++
				continue
			}
			segment.WriteString(line[pos : pos+end+1])
			pos += end + 1
			continue
		}
		r, sz := utf8.DecodeRuneInString(line[pos:])
		switch r {
		case urlStartMarker:
			writeSegment()
			link = true
		case urlEndMarker:
			writeSegment()
			link = false
		default:
			segment.WriteRune(r)
		}
		pos += sz
	}
	writeSegment()
	*open = link
	return out.String()
}

// wrapStyledBlock word-wraps ANSI-styled chat body text to width, preserving escape codes.
// prefix is printed once on the first line; continuation lines align under the body column.
// Hyperlink styling is applied per wrapped line using URL span markers so underline does
// not bleed across breaks and wrapped segments stay styled.
func wrapStyledBlock(prefix, content, suffix string, width int, styles themeStyles) string {
	if content == "" {
		return prefix + suffix
	}
	if width <= 0 {
		open := false
		return prefix + applyURLMarkers(content, styles, &open) + suffix
	}

	prefixCells := ansi.StringWidth(prefix)
	lineWidth := width - prefixCells
	if lineWidth < 1 {
		lineWidth = width
		prefixCells = 0
		prefix = ""
	}
	continuationIndent := strings.Repeat(" ", prefixCells)

	var out strings.Builder
	first := true
	for _, paragraph := range strings.Split(content, "\n") {
		urlOpen := false
		wrapped := ansi.Wrap(paragraph, lineWidth, wrapBreakpoints)
		for _, wl := range strings.Split(wrapped, "\n") {
			wl = applyURLMarkers(wl, styles, &urlOpen)
			if !first {
				out.WriteString("\n")
			}
			if first {
				out.WriteString(prefix)
				first = false
			} else {
				out.WriteString(continuationIndent)
			}
			out.WriteString(wl)
		}
	}
	out.WriteString(suffix)
	return out.String()
}

func renderMessages(msgs []shared.Message, styles themeStyles, username string, users []string, width int, twentyFourHour bool, showMessageMetadata bool, reactions ...map[int64]map[string]map[string]bool) string {
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
			b.WriteString("\n")
			b.WriteString(styles.Info.Width(width).Align(lipgloss.Center).Render("─── " + dateStr + " ───"))
			b.WriteString("\n\n")
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
			content = prepareURLWrapping(content)
			content = markURLsForWrap(content)
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

		metaSuffix := ""
		if showMessageMetadata {
			var metadata []string
			if msg.MessageID > 0 {
				metadata = append(metadata, fmt.Sprintf("id:%d", msg.MessageID))
			}
			if msg.Encrypted {
				metadata = append(metadata, "encrypted")
			}
			if len(metadata) > 0 {
				metaSuffix = " " + styles.Timestamp.Render("["+strings.Join(metadata, ", ")+"]")
			}
		}

		headPrefix := timestamp + " " + prefix
		switch msg.Sender {
		case "System":
			styled := systemLineStyle(styles, content).Render(content)
			b.WriteString(wrapStyledBlock(headPrefix, styled, metaSuffix, width, styles))
			b.WriteString("\n")
		case username:
			head := headPrefix + styles.Me.Render(msg.Sender) + ": "
			b.WriteString(wrapStyledBlock(head, content, metaSuffix, width, styles))
			b.WriteString("\n")
		default:
			head := headPrefix + styles.Other.Render(msg.Sender) + ": "
			b.WriteString(wrapStyledBlock(head, content, metaSuffix, width, styles))
			b.WriteString("\n")
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
					b.WriteString(reactionLine)
					b.WriteString("\n")
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
	"+1":         "👍",
	"-1":         "👎",
	"thumbsup":   "👍",
	"thumbsdown": "👎",
	"heart":      "❤️",
	"laugh":      "😂",
	"fire":       "🔥",
	"party":      "🎉",
	"eyes":       "👀",
	"check":      "✅",
	"x":          "❌",
	"think":      "🤔",
	"clap":       "👏",
	"rocket":     "🚀",
	"wave":       "👋",
	"100":        "💯",
	"sad":        "😢",
	"wow":        "😮",
	"angry":      "😡",
	"skull":      "💀",
	"pray":       "🙏",
	"star":       "⭐",
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

type dmSidebarEntry struct {
	User   string
	Unread int
	Active bool
}

func renderUserList(users []string, me string, styles themeStyles, width int, isAdmin bool, selectedUserIndex int, dmThreads []dmSidebarEntry) string {
	var b strings.Builder
	title := " Users "
	b.WriteString(styles.UserList.Width(width).Render(title))
	b.WriteString("\n")
	max := maxUsersDisplay
	for i, u := range users {
		if i >= max {
			b.WriteString(lipgloss.NewStyle().Italic(true).Faint(true).Width(width).Render(fmt.Sprintf("+%d more", len(users)-max)))
			b.WriteString("\n")
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

		b.WriteString(userStyle.Render(prefix + u))
		b.WriteString("\n")
	}

	if len(dmThreads) > 0 {
		b.WriteString("\n")
		b.WriteString(styles.User.Render(" DMs "))
		b.WriteString("\n")
		for _, dm := range dmThreads {
			label := "• " + dm.User
			if dm.Active {
				label = "► " + dm.User
			}
			if dm.Unread > 0 {
				label += fmt.Sprintf(" (%d)", dm.Unread)
			}
			b.WriteString(styles.Other.Render(label))
			b.WriteString("\n")
		}
	}
	return b.String()
}
