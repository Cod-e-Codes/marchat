package main

import (
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/Cod-e-Codes/marchat/plugin/sdk"
)

// EchoPlugin is a simple echo plugin
type EchoPlugin struct {
	*sdk.BasePlugin
	config sdk.Config
}

// NewEchoPlugin creates a new echo plugin
func NewEchoPlugin() *EchoPlugin {
	return &EchoPlugin{
		BasePlugin: sdk.NewBasePlugin("echo"),
	}
}

// Init initializes the echo plugin
func (p *EchoPlugin) Init(config sdk.Config) error {
	p.config = config
	return nil
}

// OnMessage handles incoming messages
func (p *EchoPlugin) OnMessage(msg sdk.Message) ([]sdk.Message, error) {
	if len(msg.Content) > 5 && msg.Content[:5] == "echo:" {
		echoMsg := sdk.Message{
			Sender:    "EchoBot",
			Content:   msg.Content[5:],
			CreatedAt: time.Now(),
		}
		return []sdk.Message{echoMsg}, nil
	}
	return nil, nil
}

// Commands returns the commands this plugin provides
func (p *EchoPlugin) Commands() []sdk.PluginCommand {
	return []sdk.PluginCommand{
		{
			Name:        "echo",
			Description: "Echo a message",
			Usage:       ":echo <message>",
			AdminOnly:   false,
		},
		{
			Name:        "echo-admin",
			Description: "Echo a message (admin only)",
			Usage:       ":echo-admin <message>",
			AdminOnly:   true,
		},
	}
}

func main() {
	plugin := NewEchoPlugin()
	if err := sdk.RunStdio(plugin, plugin.handleCommand); err != nil {
		log.Fatalf("plugin exited: %v", err)
	}
}

func (p *EchoPlugin) handleCommand(command string, args []string) sdk.PluginResponse {
	switch command {
	case "echo", "echo-admin":
		if len(args) == 0 {
			return sdk.PluginResponse{
				Type:    "command",
				Success: false,
				Error:   "unknown command",
			}
		}
		var content string
		if len(args) == 1 && strings.HasPrefix(args[0], `"`) && strings.HasSuffix(args[0], `"`) {
			content = strings.Trim(args[0], `"`)
		} else {
			content = strings.Join(args, " ")
		}

		echoMsg := sdk.Message{
			Sender:    "EchoBot",
			Content:   content,
			CreatedAt: time.Now(),
		}
		responseData, err := json.Marshal(echoMsg)
		if err != nil {
			return sdk.PluginResponse{
				Type:    "command",
				Success: false,
				Error:   err.Error(),
			}
		}
		return sdk.PluginResponse{
			Type:    "message",
			Success: true,
			Data:    responseData,
		}
	default:
		return sdk.PluginResponse{
			Type:    "command",
			Success: false,
			Error:   "unknown command",
		}
	}
}
