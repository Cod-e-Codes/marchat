package main

import (
	"fmt"

	"github.com/Cod-e-Codes/marchat/client/config"
	"github.com/Cod-e-Codes/marchat/shared"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) generateHelpContent() string {
	title := m.styles.HelpTitle.Render("marchat help")

	var sessionInfo string
	if m.useE2E {
		sessionInfo = "Session: E2E Encrypted (messages are encrypted for privacy)\n"
	} else {
		sessionInfo = "Session: Unencrypted (messages are sent in plain text)\n"
	}

	shortcuts := "\nKeyboard Shortcuts:\n"
	shortcuts += "  Ctrl+H               Toggle this help\n"
	shortcuts += "  :q                    Quit client\n"
	shortcuts += "  Esc                  Close menus\n"
	shortcuts += "  Enter                Send message\n"
	shortcuts += "  ↑/↓                  Scroll chat\n"
	shortcuts += "  PgUp/PgDn            Page through chat\n"
	shortcuts += "  Ctrl+C/V/X/A         Copy/Paste/Cut/Select all\n"
	shortcuts += "  Alt+F                Send file (file picker)\n"
	shortcuts += "  Alt+C                Create code snippet\n"
	shortcuts += "  Ctrl+T               Cycle themes\n"
	shortcuts += "  Alt+T                Toggle 12/24h time\n"
	shortcuts += "  Alt+M                Toggle message metadata (id/encrypted)\n"
	shortcuts += "  Alt+N                Toggle desktop notifications\n"
	shortcuts += "  Ctrl+L               Clear chat history\n"

	commands := "\nText Commands:\n"
	commands += "  :q                   Quit client\n"
	commands += "  :sendfile [path]     Send a file (or Alt+F)\n"
	commands += "  :savefile <name>     Save received file\n"
	commands += "  :theme <name>        Change theme (or Ctrl+T to cycle)\n"
	commands += "  :themes              List all available themes\n"
	commands += "  :time                Toggle 12/24h time (or Alt+T)\n"
	commands += "  :msginfo             Toggle message metadata (or Alt+M)\n"
	commands += "  :clear               Clear chat history (or Ctrl+L)\n"
	commands += "  :code                Create code snippet (or Alt+C)\n"
	commands += "  :edit <id> <text>    Edit a message by ID\n"
	commands += "  :delete <id>         Delete a message by ID\n"
	commands += "  :dm [user] [msg]     Send DM or toggle DM mode\n"
	commands += "  :search <query>      Search message history\n"
	commands += "  :react <id> <emoji>  React to a message (+1, heart, fire, party, etc.)\n"
	commands += "  :pin <id>            Toggle pin on a message\n"
	commands += "  :pinned              Show pinned messages\n"
	commands += "  :join <channel>      Join a channel (default: general)\n"
	commands += "  :leave               Leave current channel (back to #general)\n"
	commands += "  :channels            List active channels\n"
	commands += "  :export [file]       Export chat history to file\n"
	commands += "\nNotifications:\n"
	commands += "  :bell                Toggle message bell\n"
	commands += "  :bell-mention        Bell on mentions only\n"
	commands += "  :notify-mode <mode>  Set notification mode (none/bell/desktop/both)\n"
	commands += "  :notify-desktop      Toggle desktop notifications\n"
	commands += "  :notify-status       Show notification settings\n"
	commands += "  :quiet <start> <end> Enable quiet hours (e.g., :quiet 22 8)\n"
	commands += "  :quiet-off           Disable quiet hours\n"
	commands += "  :focus [duration]    Enable focus mode (e.g., :focus 30m)\n"
	commands += "  :focus-off           Disable focus mode\n"

	var adminSection string
	if *isAdmin {
		adminSection = "\nAdmin Features:\n"
		adminSection += "\n  User Management:\n"
		adminSection += "    Ctrl+U             Select/cycle user\n"
		adminSection += "    Ctrl+K             Kick selected user (or :kick <user>)\n"
		adminSection += "    Ctrl+B             Ban selected user (or :ban <user>)\n"
		adminSection += "    Ctrl+F             Force disconnect (or :forcedisconnect <user>)\n"
		adminSection += "    Ctrl+Shift+B       Unban user (or :unban <user>)\n"
		adminSection += "    Ctrl+Shift+A       Allow user (or :allow <user>)\n"
		adminSection += "    :cleanup           Clean stale connections\n"
		adminSection += "\n  Plugin Management:\n"
		adminSection += "    Alt+P              List plugins (or :list)\n"
		adminSection += "    Alt+S              Plugin store (or :store)\n"
		adminSection += "    Alt+R              Refresh plugins (or :refresh)\n"
		adminSection += "    Alt+I              Install plugin (or :install <name>)\n"
		adminSection += "    Alt+U              Uninstall plugin (or :uninstall <name>)\n"
		adminSection += "    Alt+E              Enable plugin (or :enable <name>)\n"
		adminSection += "    Alt+D              Disable plugin (or :disable <name>)\n"
		adminSection += "\n  Database:\n"
		adminSection += "    Ctrl+D             Database menu (or :cleardb, :backup, :stats)\n"
		adminSection += "\n  Note: Both hotkeys and text commands work in encrypted sessions.\n"
	}

	return title + "\n\n" + sessionInfo + shortcuts + commands + adminSection
}

func (m *model) generateDBMenuContent() string {
	title := m.styles.HelpTitle.Render("Database Operations")

	content := "\nAvailable Operations:\n\n"
	content += "  1. Clear Database (delete all messages)\n"
	content += "  2. Backup Database (save current state)\n"
	content += "  3. Show Database Stats\n\n"
	content += "Press 1-3 to select operation, Esc to cancel"

	return title + content
}

func (m *model) executeAdminAction(action, targetUser string) (tea.Model, tea.Cmd) {
	if !*isAdmin || targetUser == "" {
		return m, nil
	}

	var command string
	switch action {
	case "kick":
		command = fmt.Sprintf(":kick %s", targetUser)
	case "ban":
		command = fmt.Sprintf(":ban %s", targetUser)
	case "unban":
		command = fmt.Sprintf(":unban %s", targetUser)
	case "allow":
		command = fmt.Sprintf(":allow %s", targetUser)
	case "forcedisconnect":
		command = fmt.Sprintf(":forcedisconnect %s", targetUser)
	default:
		return m, nil
	}

	if m.conn != nil {
		msg := shared.Message{
			Sender:  m.cfg.Username,
			Content: command,
			Type:    shared.AdminCommandType,
		}
		err := m.conn.WriteJSON(msg)
		if err != nil {
			m.banner = "[ERROR] Failed to send admin command"
		} else {
			m.banner = fmt.Sprintf("[OK] %s action sent for %s", action, targetUser)
			if action == "kick" || action == "ban" || action == "forcedisconnect" {
				m.selectedUserIndex = -1
				m.selectedUser = ""
			}
		}
	}

	return m, m.listenWebSocket()
}

func (m *model) promptForUsername(action string) (tea.Model, tea.Cmd) {
	switch action {
	case "unban":
		m.banner = "Type username to unban in chat and press Enter (prefix with :unban)"
	case "allow":
		m.banner = "Type username to allow in chat and press Enter (prefix with :allow)"
	}
	return m, nil
}

func (m *model) promptForPluginName(action string) (tea.Model, tea.Cmd) {
	m.pendingPluginAction = action
	switch action {
	case "install":
		m.banner = "Enter plugin name to install (press Enter to confirm, Esc to cancel)"
	case "uninstall":
		m.banner = "Enter plugin name to uninstall (press Enter to confirm, Esc to cancel)"
	case "enable":
		m.banner = "Enter plugin name to enable (press Enter to confirm, Esc to cancel)"
	case "disable":
		m.banner = "Enter plugin name to disable (press Enter to confirm, Esc to cancel)"
	}
	m.textarea.Focus()
	return m, nil
}

func (m *model) executePluginCommand(command string) (tea.Model, tea.Cmd) {
	if !*isAdmin {
		return m, nil
	}

	if m.conn != nil {
		msg := shared.Message{
			Sender:  m.cfg.Username,
			Content: command,
			Type:    shared.AdminCommandType,
		}
		err := m.conn.WriteJSON(msg)
		if err != nil {
			m.banner = "[ERROR] Failed to send plugin command (connection lost)"
		} else {
			m.banner = fmt.Sprintf("[OK] Sent: %s", command)
		}
	}

	return m, m.listenWebSocket()
}

func (m *model) executeDBAction(action string) (tea.Model, tea.Cmd) {
	if !*isAdmin {
		m.showDBMenu = false
		return m, nil
	}

	switch action {
	case "cleardb":
		if m.conn != nil {
			msg := shared.Message{
				Sender:  m.cfg.Username,
				Content: ":cleardb",
				Type:    shared.AdminCommandType,
			}
			err := m.conn.WriteJSON(msg)
			if err != nil {
				m.banner = "[ERROR] Failed to send cleardb command"
			} else {
				m.banner = "[OK] Database clear command sent"
			}
		}
	case "backup":
		if m.conn != nil {
			msg := shared.Message{
				Sender:  m.cfg.Username,
				Content: ":backup",
				Type:    shared.AdminCommandType,
			}
			err := m.conn.WriteJSON(msg)
			if err != nil {
				m.banner = "[ERROR] Failed to send backup command"
			} else {
				m.banner = "[OK] Database backup command sent"
			}
		}
	case "stats":
		if m.conn != nil {
			msg := shared.Message{
				Sender:  m.cfg.Username,
				Content: ":stats",
				Type:    shared.AdminCommandType,
			}
			err := m.conn.WriteJSON(msg)
			if err != nil {
				m.banner = "[ERROR] Failed to send stats command"
			} else {
				m.banner = "[OK] Database stats command sent"
			}
		}
	}

	m.showDBMenu = false
	return m, m.listenWebSocket()
}

func getCurrentProfileName(cfg *config.Config) string {
	loader, err := config.NewInteractiveConfigLoader()
	if err != nil {
		return ""
	}

	profiles, err := loader.LoadProfiles()
	if err != nil {
		return ""
	}

	for _, profile := range profiles.Profiles {
		if profile.Username == cfg.Username &&
			profile.ServerURL == cfg.ServerURL &&
			profile.IsAdmin == cfg.IsAdmin &&
			profile.UseE2E == cfg.UseE2E {
			return profile.Name
		}
	}

	return ""
}

func (m *model) updateProfileTheme(newTheme string) error {
	if m.profileName == "" {
		return nil
	}

	loader, err := config.NewInteractiveConfigLoader()
	if err != nil {
		return err
	}

	profiles, err := loader.LoadProfiles()
	if err != nil {
		return err
	}

	for i, profile := range profiles.Profiles {
		if profile.Name == m.profileName {
			profiles.Profiles[i].Theme = newTheme
			return loader.SaveProfiles(profiles)
		}
	}

	return nil
}
