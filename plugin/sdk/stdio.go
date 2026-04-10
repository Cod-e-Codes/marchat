package sdk

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
)

// InitRequestData is the JSON object the host sends as PluginRequest.Data for type "init".
// If "config" is absent or JSON null, HandlePluginRequest skips calling Init (legacy host compatibility).
type InitRequestData struct {
	Config *Config `json:"config,omitempty"`
}

// CommandFunc handles chat commands from the host (PluginRequest with Type "command").
// Return the full PluginResponse; for chat output the Type is typically "message" with Data
// holding a marshaled Message, matching existing plugin examples.
// If onCommand is nil, command requests receive an "unknown command" error response.
type CommandFunc func(command string, args []string) PluginResponse

// HandlePluginRequest dispatches one host request to a Plugin implementation.
// init, message, and shutdown are handled internally; command is delegated to onCommand.
func HandlePluginRequest(p Plugin, req PluginRequest, onCommand CommandFunc) PluginResponse {
	switch req.Type {
	case "init":
		if len(req.Data) == 0 {
			return PluginResponse{Type: "init", Success: true}
		}
		var envelope InitRequestData
		if err := json.Unmarshal(req.Data, &envelope); err != nil {
			return PluginResponse{
				Type:    "init",
				Success: false,
				Error:   fmt.Sprintf("failed to parse init data: %v", err),
			}
		}
		if envelope.Config != nil {
			if err := p.Init(*envelope.Config); err != nil {
				return PluginResponse{
					Type:    "init",
					Success: false,
					Error:   fmt.Sprintf("failed to initialize plugin: %v", err),
				}
			}
		}
		return PluginResponse{Type: "init", Success: true}

	case "message":
		var msg Message
		if err := json.Unmarshal(req.Data, &msg); err != nil {
			return PluginResponse{
				Type:    "message",
				Success: false,
				Error:   fmt.Sprintf("failed to parse message: %v", err),
			}
		}
		responses, err := p.OnMessage(msg)
		if err != nil {
			return PluginResponse{
				Type:    "message",
				Success: false,
				Error:   fmt.Sprintf("failed to process message: %v", err),
			}
		}
		if len(responses) > 0 {
			responseData, err := json.Marshal(responses[0])
			if err != nil {
				return PluginResponse{
					Type:    "message",
					Success: false,
					Error:   fmt.Sprintf("failed to marshal response message: %v", err),
				}
			}
			return PluginResponse{
				Type:    "message",
				Success: true,
				Data:    responseData,
			}
		}
		return PluginResponse{Type: "message", Success: true}

	case "command":
		var args []string
		if err := json.Unmarshal(req.Data, &args); err != nil {
			return PluginResponse{
				Type:    "command",
				Success: false,
				Error:   fmt.Sprintf("failed to parse command args: %v", err),
			}
		}
		if onCommand == nil {
			return PluginResponse{
				Type:    "command",
				Success: false,
				Error:   "unknown command",
			}
		}
		return onCommand(req.Command, args)

	case "shutdown":
		return PluginResponse{Type: "shutdown", Success: true}

	default:
		return PluginResponse{
			Type:    req.Type,
			Success: false,
			Error:   "unknown request type",
		}
	}
}

// RunStdio runs the plugin protocol on os.Stdin / os.Stdout, logging to os.Stderr.
func RunStdio(p Plugin, onCommand CommandFunc) error {
	return RunIO(os.Stdin, os.Stdout, os.Stderr, p, onCommand)
}

// RunIO runs the plugin JSON line protocol on the given streams.
// logOut receives the standard library logger output; if nil, os.Stderr is used.
func RunIO(in io.Reader, out io.Writer, logOut io.Writer, p Plugin, onCommand CommandFunc) error {
	if logOut == nil {
		logOut = os.Stderr
	}
	log.SetOutput(logOut)
	log.SetFlags(log.LstdFlags)

	dec := json.NewDecoder(in)
	enc := json.NewEncoder(out)

	for {
		var req PluginRequest
		if err := dec.Decode(&req); err != nil {
			if err == io.EOF {
				return nil
			}
			log.Printf("Failed to decode request: %v", err)
			return fmt.Errorf("decode request: %w", err)
		}

		resp := HandlePluginRequest(p, req, onCommand)
		if err := enc.Encode(resp); err != nil {
			log.Printf("Failed to encode response: %v", err)
			return fmt.Errorf("encode response: %w", err)
		}
	}
}
