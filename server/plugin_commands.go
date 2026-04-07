package server

import (
	"fmt"
	"strings"

	"github.com/Cod-e-Codes/marchat/plugin/manager"
	"github.com/Cod-e-Codes/marchat/plugin/sdk"
	"github.com/Cod-e-Codes/marchat/shared"
)

// PluginCommandHandler handles plugin-related commands
type PluginCommandHandler struct {
	manager *manager.PluginManager
}

// NewPluginCommandHandler creates a new plugin command handler
func NewPluginCommandHandler(pluginManager *manager.PluginManager) *PluginCommandHandler {
	return &PluginCommandHandler{
		manager: pluginManager,
	}
}

// HandlePluginCommand handles plugin-related commands
func (h *PluginCommandHandler) HandlePluginCommand(cmd string, args []string, isAdmin bool) (string, error) {
	switch cmd {
	case "plugin":
		return h.handlePluginSubcommand(args, isAdmin)
	case "install":
		return h.handleInstall(args, isAdmin)
	case "uninstall":
		return h.handleUninstall(args, isAdmin)
	case "enable":
		return h.handleEnable(args, isAdmin)
	case "disable":
		return h.handleDisable(args, isAdmin)
	case "list":
		return h.handleList()
	case "store":
		return h.handleStore()
	case "refresh":
		return h.handleRefresh()
	default:
		// Check if it's a plugin command
		return h.handlePluginCommand(cmd, args, isAdmin)
	}
}

// handlePluginSubcommand handles the :plugin command
func (h *PluginCommandHandler) handlePluginSubcommand(args []string, isAdmin bool) (string, error) {
	if len(args) == 0 {
		return "Usage: :plugin <subcommand> [args...]", nil
	}

	subcmd := args[0]
	subargs := args[1:]

	switch subcmd {
	case "list":
		return h.handleList()
	case "install":
		return h.handleInstall(subargs, isAdmin)
	case "uninstall":
		return h.handleUninstall(subargs, isAdmin)
	case "enable":
		return h.handleEnable(subargs, isAdmin)
	case "disable":
		return h.handleDisable(subargs, isAdmin)
	case "store":
		return h.handleStore()
	case "refresh":
		return h.handleRefresh()
	default:
		return fmt.Sprintf("Unknown plugin subcommand: %s", subcmd), nil
	}
}

// handleInstall handles plugin installation
func (h *PluginCommandHandler) handleInstall(args []string, isAdmin bool) (string, error) {
	if !isAdmin {
		return "Plugin installation requires admin privileges", nil
	}

	if len(args) == 0 {
		return "Usage: :install <plugin-name> [--os <goos>] [--arch <goarch>]", nil
	}

	pluginName := args[0]
	var osName, arch string

	// Simple flag parsing for --os and --arch
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--os":
			if i+1 < len(args) {
				osName = args[i+1]
				i++
			}
		case "--arch":
			if i+1 < len(args) {
				arch = args[i+1]
				i++
			}
		}
	}

	if err := h.manager.InstallPluginWithPlatform(pluginName, osName, arch); err != nil {
		return fmt.Sprintf("Failed to install plugin %s: %v", pluginName, err), nil
	}

	return fmt.Sprintf("Plugin %s installed successfully", pluginName), nil
}

// handleUninstall handles plugin uninstallation
func (h *PluginCommandHandler) handleUninstall(args []string, isAdmin bool) (string, error) {
	if !isAdmin {
		return "Plugin uninstallation requires admin privileges", nil
	}

	if len(args) == 0 {
		return "Usage: :uninstall <plugin-name>", nil
	}

	pluginName := args[0]

	if err := h.manager.UninstallPlugin(pluginName); err != nil {
		return fmt.Sprintf("Failed to uninstall plugin %s: %v", pluginName, err), nil
	}

	return fmt.Sprintf("Plugin %s uninstalled successfully", pluginName), nil
}

// handleEnable handles plugin enabling
func (h *PluginCommandHandler) handleEnable(args []string, isAdmin bool) (string, error) {
	if !isAdmin {
		return "Plugin enabling requires admin privileges", nil
	}

	if len(args) == 0 {
		return "Usage: :enable <plugin-name>", nil
	}

	pluginName := args[0]

	if err := h.manager.EnablePlugin(pluginName); err != nil {
		return fmt.Sprintf("Failed to enable plugin %s: %v", pluginName, err), nil
	}

	return fmt.Sprintf("Plugin %s enabled successfully", pluginName), nil
}

// handleDisable handles plugin disabling
func (h *PluginCommandHandler) handleDisable(args []string, isAdmin bool) (string, error) {
	if !isAdmin {
		return "Plugin disabling requires admin privileges", nil
	}

	if len(args) == 0 {
		return "Usage: :disable <plugin-name>", nil
	}

	pluginName := args[0]

	if err := h.manager.DisablePlugin(pluginName); err != nil {
		return fmt.Sprintf("Failed to disable plugin %s: %v", pluginName, err), nil
	}

	return fmt.Sprintf("Plugin %s disabled successfully", pluginName), nil
}

// handleList lists installed plugins
func (h *PluginCommandHandler) handleList() (string, error) {
	plugins := h.manager.ListPlugins()

	if len(plugins) == 0 {
		return "No plugins installed", nil
	}

	var result strings.Builder
	result.WriteString("Installed plugins:\n")

	for name, instance := range plugins {
		status := "disabled"
		if instance.Enabled {
			status = "enabled"
		}

		version := "unknown"
		if instance.Manifest != nil {
			version = instance.Manifest.Version
		}

		result.WriteString(fmt.Sprintf("  %s (%s) - %s\n", name, version, status))
	}

	return result.String(), nil
}

// handleStore opens the plugin store
func (h *PluginCommandHandler) handleStore() (string, error) {
	// Get available plugins from store
	storePlugins := h.manager.GetStore().GetPluginsPreferredForPlatform("", "")

	if len(storePlugins) == 0 {
		return "No plugins available in store. Use :refresh to update plugin list.", nil
	}

	var result strings.Builder
	result.WriteString("Available plugins in store:\n")

	for _, plugin := range storePlugins {
		status := ""
		// Check if plugin is already installed
		installedPlugins := h.manager.ListPlugins()
		if instance, exists := installedPlugins[plugin.Name]; exists {
			if instance.Enabled {
				status = " [installed, enabled]"
			} else {
				status = " [installed, disabled]"
			}
		}

		platform := plugin.GoOS
		if platform == "" {
			platform = "any"
		}
		arch := plugin.GoArch
		if arch == "" {
			arch = "any"
		}

		result.WriteString(fmt.Sprintf("  %s (%s) - %s [%s/%s]%s\n",
			plugin.Name, plugin.Version, plugin.Description, platform, arch, status))
	}

	result.WriteString("\nUse :install <plugin-name> to install a plugin")

	return result.String(), nil
}

// handleRefresh refreshes the plugin store
func (h *PluginCommandHandler) handleRefresh() (string, error) {
	if err := h.manager.RefreshStore(); err != nil {
		return fmt.Sprintf("Failed to refresh plugin store: %v", err), nil
	}

	return "Plugin store refreshed successfully", nil
}

// handlePluginCommand handles commands from specific plugins
func (h *PluginCommandHandler) handlePluginCommand(cmd string, args []string, isAdmin bool) (string, error) {
	// Get all plugin commands
	pluginCommands := h.manager.GetPluginCommands()

	// Find which plugin provides this command
	var pluginName string
	var command *sdk.PluginCommand

	for name, commands := range pluginCommands {
		for _, cmdInfo := range commands {
			if cmdInfo.Name == cmd {
				pluginName = name
				command = &cmdInfo
				break
			}
		}
		if pluginName != "" {
			break
		}
	}

	if pluginName == "" {
		return "", fmt.Errorf("unknown command: %s", cmd)
	}

	// Check admin requirements
	if command.AdminOnly && !isAdmin {
		return "This command requires admin privileges", nil
	}

	// Execute the plugin command
	if err := h.manager.ExecuteCommand(pluginName, cmd, args); err != nil {
		return fmt.Sprintf("Failed to execute plugin command: %v", err), nil
	}

	return fmt.Sprintf("Command %s executed successfully", cmd), nil
}

// SendMessageToPlugins sends a message to all enabled plugins
func (h *PluginCommandHandler) SendMessageToPlugins(msg shared.Message) {
	pluginMsg := sdk.Message{
		Sender:    msg.Sender,
		Content:   msg.Content,
		CreatedAt: msg.CreatedAt,
		Type:      string(msg.Type),
		Channel:   msg.Channel,
		Encrypted: msg.Encrypted,
		MessageID: msg.MessageID,
		Recipient: msg.Recipient,
		Edited:    msg.Edited,
	}

	h.manager.SendMessage(pluginMsg)
}

// UpdateUserListForPlugins updates the user list for plugins
func (h *PluginCommandHandler) UpdateUserListForPlugins(users []string) {
	h.manager.UpdateUserList(users)
}

// GetPluginMessageChannel returns the channel for receiving messages from plugins
func (h *PluginCommandHandler) GetPluginMessageChannel() <-chan sdk.Message {
	return h.manager.GetMessageChannel()
}

// ConvertPluginMessage converts a plugin message to a shared message
func ConvertPluginMessage(pluginMsg sdk.Message) shared.Message {
	return shared.Message{
		Sender:    pluginMsg.Sender,
		Content:   pluginMsg.Content,
		CreatedAt: pluginMsg.CreatedAt,
		Type:      shared.MessageType(pluginMsg.Type),
		Channel:   pluginMsg.Channel,
		Encrypted: pluginMsg.Encrypted,
		MessageID: pluginMsg.MessageID,
		Recipient: pluginMsg.Recipient,
		Edited:    pluginMsg.Edited,
	}
}
