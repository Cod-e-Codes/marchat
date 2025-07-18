package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"marchat/client/config"
	"marchat/shared"

	"os/signal"
	"syscall"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gorilla/websocket"
)

var (
	configPath = flag.String("config", "config.json", "Path to config file")
	serverURL  = flag.String("server", "", "Server URL (overrides config)")
	username   = flag.String("username", "", "Username (overrides config)")
	theme      = flag.String("theme", "", "Theme (overrides config)")
)

type model struct {
	cfg       config.Config
	textarea  textarea.Model
	viewport  viewport.Model
	messages  []shared.Message
	styles    themeStyles
	banner    string
	connected bool

	users []string // NEW: user list

	width  int // NEW: track window width
	height int // NEW: track window height

	userListViewport viewport.Model // NEW: scrollable user list

	twentyFourHour bool // NEW: timestamp format toggle

	sending bool // NEW: sending message feedback

	conn     *websocket.Conn // persistent WebSocket connection
	msgChan  chan tea.Msg    // channel for incoming messages from WS goroutine
	quitChan chan struct{}   // signal for shutdown
}

type themeStyles struct {
	User    lipgloss.Style
	Time    lipgloss.Style
	Msg     lipgloss.Style
	Banner  lipgloss.Style
	Box     lipgloss.Style // frame color
	Mention lipgloss.Style // mention highlighting

	UserList lipgloss.Style // NEW: user list panel
	Me       lipgloss.Style // NEW: current user style
	Other    lipgloss.Style // NEW: other user style
}

func getThemeStyles(theme string) themeStyles {
	switch theme {
	case "slack":
		return themeStyles{
			User:     lipgloss.NewStyle().Foreground(lipgloss.Color("#36C5F0")).Bold(true),
			Time:     lipgloss.NewStyle().Foreground(lipgloss.Color("#999999")),
			Msg:      lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")),
			Banner:   lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F5F")).Bold(true),
			Box:      lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#36C5F0")),
			Mention:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF00FF")),
			UserList: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#36C5F0")).Padding(0, 1),
			Me:       lipgloss.NewStyle().Foreground(lipgloss.Color("#36C5F0")).Bold(true),
			Other:    lipgloss.NewStyle().Foreground(lipgloss.Color("#AAAAAA")),
		}
	case "discord":
		return themeStyles{
			User:     lipgloss.NewStyle().Foreground(lipgloss.Color("#7289DA")).Bold(true),
			Time:     lipgloss.NewStyle().Foreground(lipgloss.Color("#99AAB5")),
			Msg:      lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")),
			Banner:   lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F5F")).Bold(true),
			Box:      lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#7289DA")),
			Mention:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFD700")),
			UserList: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#7289DA")).Padding(0, 1),
			Me:       lipgloss.NewStyle().Foreground(lipgloss.Color("#7289DA")).Bold(true),
			Other:    lipgloss.NewStyle().Foreground(lipgloss.Color("#AAAAAA")),
		}
	case "aim":
		return themeStyles{
			User:     lipgloss.NewStyle().Foreground(lipgloss.Color("#FFCC00")).Bold(true),
			Time:     lipgloss.NewStyle().Foreground(lipgloss.Color("#00AEEF")),
			Msg:      lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")),
			Banner:   lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F5F")).Bold(true),
			Box:      lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#FFCC00")),
			Mention:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFD700")),
			UserList: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#FFCC00")).Padding(0, 1),
			Me:       lipgloss.NewStyle().Foreground(lipgloss.Color("#FFCC00")).Bold(true),
			Other:    lipgloss.NewStyle().Foreground(lipgloss.Color("#AAAAAA")),
		}
	default:
		return themeStyles{
			User:     lipgloss.NewStyle().Bold(true),
			Time:     lipgloss.NewStyle().Faint(true),
			Msg:      lipgloss.NewStyle(),
			Banner:   lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F5F")).Bold(true),
			Box:      lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#AAAAAA")),
			Mention:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFD700")),
			UserList: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#AAAAAA")).Padding(0, 1),
			Me:       lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700")).Bold(true),
			Other:    lipgloss.NewStyle().Foreground(lipgloss.Color("#AAAAAA")),
		}
	}
}

func renderMessages(msgs []shared.Message, styles themeStyles, username string, width int, twentyFourHour bool) string {
	const max = 100
	if len(msgs) > max {
		msgs = msgs[len(msgs)-max:]
	}
	var b strings.Builder
	var prevDate string
	for _, msg := range msgs {
		sender := msg.Sender
		align := lipgloss.Left
		msgBoxStyle := lipgloss.NewStyle().Width(width - 4)
		if sender == username {
			align = lipgloss.Right
			msgBoxStyle = msgBoxStyle.Background(lipgloss.Color("#222244")).Foreground(lipgloss.Color("#FFFFFF"))
		} else {
			msgBoxStyle = msgBoxStyle.Background(lipgloss.Color("#222222")).Foreground(lipgloss.Color("#AAAAAA"))
		}
		// Date header if date changes
		dateStr := msg.CreatedAt.Format("2006-01-02")
		if dateStr != prevDate {
			b.WriteString(styles.Time.Render(dateStr) + "\n")
			prevDate = dateStr
		}
		// Time format
		timeFmt := "15:04:05"
		if !twentyFourHour {
			timeFmt = "03:04:05 PM"
		}
		timestamp := styles.Time.Render(msg.CreatedAt.Format(timeFmt))
		content := renderEmojis(msg.Content)
		if strings.Contains(msg.Content, "@"+username) {
			content = styles.Mention.Render(content)
		} else {
			content = styles.Msg.Render(content)
		}
		meta := styles.User.Render(sender) + " " + timestamp
		wrapped := msgBoxStyle.Render(content)
		msgBlock := lipgloss.JoinVertical(lipgloss.Left, meta, wrapped)
		b.WriteString(msgBoxStyle.Align(align).Render(msgBlock) + "\n\n")
	}
	return b.String()
}

type wsMsg shared.Message

type wsErr error

type wsConnected bool

func (m *model) connectWebSocket(serverURL string) error {
	conn, _, err := websocket.DefaultDialer.Dial(serverURL, nil)
	if err != nil {
		return err
	}
	m.conn = conn
	m.connected = true
	m.banner = "✅ Connected to server!"
	m.msgChan = make(chan tea.Msg)
	m.quitChan = make(chan struct{})
	// Start goroutine to read messages
	go func() {
		for {
			var msg shared.Message
			err := conn.ReadJSON(&msg)
			if err != nil {
				m.msgChan <- wsErr(err)
				return
			}
			m.msgChan <- wsMsg(msg)
		}
	}()
	return nil
}

func (m *model) closeWebSocket() {
	if m.conn != nil {
		m.conn.Close()
	}
	if m.quitChan != nil {
		close(m.quitChan)
	}
}

func (m model) Init() tea.Cmd {
	return func() tea.Msg {
		err := m.connectWebSocket(m.cfg.ServerURL)
		if err != nil {
			return wsErr(err)
		}
		// Handle Ctrl+C/interrupt for clean shutdown
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-c
			m.closeWebSocket()
			os.Exit(0)
		}()
		return wsConnected(true)
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case wsConnected:
		m.connected = true
		m.banner = "✅ Connected to server!"
		return m, m.listenWebSocket()
	case wsMsg:
		m.messages = append(m.messages, shared.Message(msg))
		m.viewport.SetContent(renderMessages(m.messages, m.styles, m.cfg.Username, m.viewport.Width, m.twentyFourHour))
		m.viewport.GotoBottom()
		return m, m.listenWebSocket()
	case wsErr:
		m.connected = false
		m.banner = "🚫 Connection lost. Reconnecting..."
		m.closeWebSocket()
		return m, tea.Tick(time.Second*2, func(time.Time) tea.Msg {
			return m.Init()()
		})
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.closeWebSocket()
			return m, tea.Quit
		case "up":
			if m.textarea.Focused() {
				m.viewport.ScrollUp(1)
			} else {
				m.userListViewport.ScrollUp(1)
			}
			return m, nil
		case "down":
			if m.textarea.Focused() {
				m.viewport.ScrollDown(1)
			} else {
				m.userListViewport.ScrollDown(1)
			}
			return m, nil
		case "enter":
			text := m.textarea.Value()
			if strings.HasPrefix(text, ":theme ") {
				parts := strings.SplitN(text, " ", 2)
				if len(parts) == 2 {
					m.cfg.Theme = parts[1]
					m.styles = getThemeStyles(m.cfg.Theme)
					m.banner = "Theme changed to " + m.cfg.Theme
				}
				m.textarea.SetValue("")
				return m, nil
			}
			if text == ":clear" {
				m.messages = nil
				m.viewport.SetContent("")
				m.banner = "Chat cleared."
				m.textarea.SetValue("")
				return m, nil
			}
			if text == ":cleardb" {
				err := sendClearDB(m.cfg.ServerURL)
				if err != nil {
					m.banner = "Failed to clear DB: " + err.Error()
				} else {
					m.messages = nil
					m.viewport.SetContent("")
					m.banner = "Database cleared."
				}
				m.textarea.SetValue("")
				return m, nil
			}
			if text == ":time" {
				m.twentyFourHour = !m.twentyFourHour
				m.banner = "Timestamp format: " + map[bool]string{true: "24h", false: "12h"}[m.twentyFourHour]
				return m, nil
			}
			if text != "" {
				m.sending = true
				if m.conn != nil {
					msg := shared.Message{Sender: m.cfg.Username, Content: text}
					err := m.conn.WriteJSON(msg)
					if err != nil {
						m.banner = "❌ Failed to send (connection lost)"
						m.sending = false
						return m, m.listenWebSocket()
					}
					m.banner = ""
				}
				m.sending = false
				m.textarea.SetValue("")
				return m, m.listenWebSocket()
			}
			return m, nil
		default:
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			return m, cmd
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		userListWidth := 18
		chatWidth := m.width - userListWidth - 4
		if chatWidth < 20 {
			chatWidth = 20
		}
		m.viewport.Width = chatWidth
		m.viewport.Height = m.height - m.textarea.Height() - 6
		m.textarea.SetWidth(chatWidth)
		m.viewport.SetContent(renderMessages(m.messages, m.styles, m.cfg.Username, chatWidth, m.twentyFourHour))
		m.viewport.GotoBottom()
		m.userListViewport.Height = m.height - m.textarea.Height() - 6
		m.userListViewport.SetContent(renderUserList(m.users, m.cfg.Username, m.styles, userListWidth))
		return m, nil
	default:
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd
	}
}

func (m model) listenWebSocket() tea.Cmd {
	return func() tea.Msg {
		return <-m.msgChan
	}
}

func (m model) View() string {
	// Header
	headerStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#36C5F0")).
		Foreground(lipgloss.Color("230")).
		Bold(true).
		Padding(0, 1)
	footerStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#222222")).
		Foreground(lipgloss.Color("#36C5F0")).
		Padding(0, 1)

	totalWidth := m.viewport.Width + 18 + 4 // chat + userlist + borders
	header := headerStyle.Width(totalWidth).Render(" marchat ")
	footer := footerStyle.Width(totalWidth).Render(
		"[Enter] Send  [Mouse Scroll] Scroll  [Esc/Ctrl+C] Quit  Commands: :clear :cleardb :theme NAME :time",
	)

	// Banner
	var bannerBox string
	if m.banner != "" || m.sending {
		bannerText := m.banner
		if m.sending {
			if bannerText != "" {
				bannerText += " ⏳ Sending..."
			} else {
				bannerText = "⏳ Sending..."
			}
		}
		bannerBox = lipgloss.NewStyle().
			Width(m.viewport.Width).
			PaddingLeft(1).
			Background(lipgloss.Color("#FF5F5F")).
			Foreground(lipgloss.Color("#000000")).
			Bold(true).
			Render(bannerText)
	}

	// Chat and user list layout
	chatBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#36C5F0")).
		Padding(0, 1)
	chatPanel := chatBoxStyle.Width(m.viewport.Width).Render(m.viewport.View())
	userPanel := m.userListViewport.View()
	row := lipgloss.JoinHorizontal(lipgloss.Top, userPanel, chatPanel)

	// Input
	inputPanel := chatBoxStyle.Width(m.viewport.Width).Render(m.textarea.View())

	// Compose layout
	ui := lipgloss.JoinVertical(lipgloss.Left,
		header,
		bannerBox,
		row,
		inputPanel,
		footer,
	)
	return ui
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

func sendClearDB(serverURL string) error {
	req, err := http.NewRequest("POST", serverURL+"/clear", nil)
	if err != nil {
		fmt.Println("sendClearDB request error:", err)
		return err
	}
	fmt.Println("sendClearDB: sending POST to", serverURL+"/clear")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("sendClearDB error:", err)
		return err
	}
	defer resp.Body.Close()
	fmt.Println("sendClearDB response status:", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %d", resp.StatusCode)
	}
	return nil
}

func renderUserList(users []string, me string, styles themeStyles, width int) string {
	var b strings.Builder
	b.WriteString(styles.UserList.Width(width).Render(" Users ") + "\n")
	for _, u := range users {
		if u == me {
			b.WriteString(styles.Me.Render("• "+u) + "\n")
		} else {
			b.WriteString(styles.Other.Render("• "+u) + "\n")
		}
	}
	return styles.UserList.Width(width).Render(b.String())
}

func main() {
	flag.Parse()
	cfg, _ := config.LoadConfig(*configPath)
	if *serverURL != "" {
		cfg.ServerURL = *serverURL
	}
	if *username != "" {
		cfg.Username = *username
	}
	if *theme != "" {
		cfg.Theme = *theme
	}
	if cfg.Username == "" {
		fmt.Println("Username required. Use --username or set in config file.")
		return
	}
	if cfg.ServerURL == "" {
		cfg.ServerURL = "http://localhost:9090"
	}

	ta := textarea.New()
	ta.Placeholder = "Type your message..."
	ta.Focus()
	ta.Prompt = "┃ "
	ta.CharLimit = 280
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false)

	vp := viewport.New(80, 20)

	userListVp := viewport.New(18, 10) // height will be set on resize
	userListVp.SetContent(renderUserList([]string{cfg.Username}, cfg.Username, getThemeStyles(cfg.Theme), 18))

	m := model{
		cfg:              cfg,
		textarea:         ta,
		viewport:         vp,
		styles:           getThemeStyles(cfg.Theme),
		users:            []string{cfg.Username},
		userListViewport: userListVp,
		twentyFourHour:   true,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
