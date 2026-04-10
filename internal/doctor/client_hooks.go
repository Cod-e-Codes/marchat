package doctor

import (
	"fmt"
	"os"
	"strings"

	"github.com/Cod-e-Codes/marchat/client/exthook"
)

// appendClientHookChecks validates experimental client hook paths when the corresponding env vars are set.
func appendClientHookChecks(checks *[]Check) {
	recv := strings.TrimSpace(os.Getenv("MARCHAT_CLIENT_HOOK_RECEIVE"))
	send := strings.TrimSpace(os.Getenv("MARCHAT_CLIENT_HOOK_SEND"))
	if recv == "" && send == "" {
		return
	}
	if recv != "" {
		if _, err := exthook.ValidateHookExecutable(recv); err != nil {
			appendCheck(checks, "client_hook_receive", "warn", fmt.Sprintf("MARCHAT_CLIENT_HOOK_RECEIVE: %v", err))
		} else {
			appendCheck(checks, "client_hook_receive", "ok", fmt.Sprintf("MARCHAT_CLIENT_HOOK_RECEIVE executable OK: %s", recv))
		}
	}
	if send != "" {
		if _, err := exthook.ValidateHookExecutable(send); err != nil {
			appendCheck(checks, "client_hook_send", "warn", fmt.Sprintf("MARCHAT_CLIENT_HOOK_SEND: %v", err))
		} else {
			appendCheck(checks, "client_hook_send", "ok", fmt.Sprintf("MARCHAT_CLIENT_HOOK_SEND executable OK: %s", send))
		}
	}
}
