package main

import "github.com/charmbracelet/bubbles/key"

// keyMap defines all keybindings for the help system
type keyMap struct {
	Send        key.Binding
	ScrollUp    key.Binding
	ScrollDown  key.Binding
	PageUp      key.Binding
	PageDown    key.Binding
	Copy        key.Binding
	Paste       key.Binding
	Cut         key.Binding
	SelectAll   key.Binding
	Help        key.Binding
	Quit        key.Binding
	TimeFormat  key.Binding
	Clear       key.Binding
	QuitCommand key.Binding
	// Commands with both text commands and hotkey alternatives
	SendFile    key.Binding
	SaveFile    key.Binding
	Theme       key.Binding
	CodeSnippet key.Binding
	// Hotkey alternatives for commands (work even in encrypted sessions)
	SendFileHotkey    key.Binding
	ThemeHotkey       key.Binding
	TimeFormatHotkey  key.Binding
	MessageInfoHotkey key.Binding
	ClearHotkey       key.Binding
	CodeSnippetHotkey key.Binding
	// Notification controls
	NotifyDesktop key.Binding
	// Admin UI commands
	DatabaseMenu key.Binding
	SelectUser   key.Binding
	CloseMenu    key.Binding
	// Admin action hotkeys
	BanUser             key.Binding
	KickUser            key.Binding
	UnbanUser           key.Binding
	AllowUser           key.Binding
	ForceDisconnectUser key.Binding
	// Plugin management hotkeys (admin only)
	PluginList      key.Binding
	PluginStore     key.Binding
	PluginRefresh   key.Binding
	PluginInstall   key.Binding
	PluginUninstall key.Binding
	PluginEnable    key.Binding
	PluginDisable   key.Binding
	// Legacy admin commands (for help display only)
	ClearDB key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// FullHelp returns keybindings for the expanded help view
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Send, k.ScrollUp, k.ScrollDown, k.PageUp, k.PageDown},
		{k.Copy, k.Paste, k.Cut, k.SelectAll},
		{k.TimeFormat, k.Clear, k.Help, k.Quit},
	}
}

// GetCommandHelp returns command-specific help based on user permissions
func (k keyMap) GetCommandHelp(isAdmin, useE2E bool) [][]key.Binding {
	commands := [][]key.Binding{
		{k.SendFile, k.SaveFile, k.Theme, k.CodeSnippet},
		{k.SendFileHotkey, k.ThemeHotkey, k.TimeFormatHotkey, k.MessageInfoHotkey, k.ClearHotkey, k.CodeSnippetHotkey},
	}

	if isAdmin {
		commands = append(commands, []key.Binding{k.DatabaseMenu, k.SelectUser, k.BanUser, k.KickUser, k.UnbanUser, k.AllowUser, k.ForceDisconnectUser})
	}

	return commands
}

func newKeyMap() keyMap {
	return keyMap{
		Send: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "send message"),
		),
		ScrollUp: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("↑", "scroll up"),
		),
		ScrollDown: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("↓", "scroll down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown"),
			key.WithHelp("pgdown", "page down"),
		),
		Copy: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "copy"),
		),
		Paste: key.NewBinding(
			key.WithKeys("ctrl+v"),
			key.WithHelp("ctrl+v", "paste"),
		),
		Cut: key.NewBinding(
			key.WithKeys("ctrl+x"),
			key.WithHelp("ctrl+x", "cut"),
		),
		SelectAll: key.NewBinding(
			key.WithKeys("ctrl+a"),
			key.WithHelp("ctrl+a", "select all"),
		),
		Help: key.NewBinding(
			key.WithKeys("ctrl+h"),
			key.WithHelp("ctrl+h", "toggle help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "close menus"),
		),
		TimeFormat: key.NewBinding(
			key.WithKeys(":time"),
			key.WithHelp(":time", "toggle 12/24h format"),
		),
		Clear: key.NewBinding(
			key.WithKeys(":clear"),
			key.WithHelp(":clear", "clear chat history"),
		),
		QuitCommand: key.NewBinding(
			key.WithKeys(":q"),
			key.WithHelp(":q", "quit client"),
		),
		SendFile: key.NewBinding(
			key.WithKeys(":sendfile"),
			key.WithHelp(":sendfile <path>", "send a file"),
		),
		SaveFile: key.NewBinding(
			key.WithKeys(":savefile"),
			key.WithHelp(":savefile <name>", "save received file"),
		),
		Theme: key.NewBinding(
			key.WithKeys(":theme"),
			key.WithHelp(":theme <name>", "change theme"),
		),
		CodeSnippet: key.NewBinding(
			key.WithKeys(":code"),
			key.WithHelp(":code", "create syntax highlighted code snippet"),
		),
		SendFileHotkey: key.NewBinding(
			key.WithKeys("alt+f"),
			key.WithHelp("alt+f", "send a file (file picker)"),
		),
		ThemeHotkey: key.NewBinding(
			key.WithKeys("ctrl+t"),
			key.WithHelp("ctrl+t", "cycle through themes"),
		),
		TimeFormatHotkey: key.NewBinding(
			key.WithKeys("alt+t"),
			key.WithHelp("alt+t", "toggle 12/24h time format"),
		),
		MessageInfoHotkey: key.NewBinding(
			key.WithKeys("alt+m"),
			key.WithHelp("alt+m", "toggle message metadata"),
		),
		ClearHotkey: key.NewBinding(
			key.WithKeys("ctrl+l"),
			key.WithHelp("ctrl+l", "clear chat history"),
		),
		CodeSnippetHotkey: key.NewBinding(
			key.WithKeys("alt+c"),
			key.WithHelp("alt+c", "create code snippet"),
		),
		NotifyDesktop: key.NewBinding(
			key.WithKeys("alt+n"),
			key.WithHelp("alt+n", "toggle desktop notifications"),
		),
		DatabaseMenu: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "database menu (admin)"),
		),
		SelectUser: key.NewBinding(
			key.WithKeys("ctrl+u"),
			key.WithHelp("ctrl+u", "select/cycle user (admin)"),
		),
		CloseMenu: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "close menu/clear selection"),
		),
		BanUser: key.NewBinding(
			key.WithKeys("ctrl+b"),
			key.WithHelp("ctrl+b", "ban selected user (admin)"),
		),
		KickUser: key.NewBinding(
			key.WithKeys("ctrl+k"),
			key.WithHelp("ctrl+k", "kick selected user (admin)"),
		),
		UnbanUser: key.NewBinding(
			key.WithKeys("ctrl+shift+b"),
			key.WithHelp("ctrl+shift+b", "unban user (admin)"),
		),
		AllowUser: key.NewBinding(
			key.WithKeys("ctrl+shift+a"),
			key.WithHelp("ctrl+shift+a", "allow user (admin)"),
		),
		ForceDisconnectUser: key.NewBinding(
			key.WithKeys("ctrl+f"),
			key.WithHelp("ctrl+f", "force disconnect selected user (admin)"),
		),
		PluginList: key.NewBinding(
			key.WithKeys("alt+p"),
			key.WithHelp("alt+p", "list plugins (admin)"),
		),
		PluginStore: key.NewBinding(
			key.WithKeys("alt+s"),
			key.WithHelp("alt+s", "plugin store (admin)"),
		),
		PluginRefresh: key.NewBinding(
			key.WithKeys("alt+r"),
			key.WithHelp("alt+r", "refresh plugins (admin)"),
		),
		PluginInstall: key.NewBinding(
			key.WithKeys("alt+i"),
			key.WithHelp("alt+i", "install plugin (admin)"),
		),
		PluginUninstall: key.NewBinding(
			key.WithKeys("alt+u"),
			key.WithHelp("alt+u", "uninstall plugin (admin)"),
		),
		PluginEnable: key.NewBinding(
			key.WithKeys("alt+e"),
			key.WithHelp("alt+e", "enable plugin (admin)"),
		),
		PluginDisable: key.NewBinding(
			key.WithKeys("alt+d"),
			key.WithHelp("alt+d", "disable plugin (admin)"),
		),
		ClearDB: key.NewBinding(
			key.WithKeys(""),
			key.WithHelp("", ""),
		),
	}
}
