package doctor

import (
	"os"
	"sort"
	"strings"
)

// osEnviron is the source of environment pairs; tests may replace it.
var osEnviron = os.Environ

type envLine struct {
	Key     string `json:"key"`
	Display string `json:"display"`
}

func collectMarchatEnviron() map[string]string {
	out := make(map[string]string)
	for _, e := range osEnviron() {
		key, val, ok := strings.Cut(e, "=")
		if !ok {
			continue
		}
		if strings.HasPrefix(key, "MARCHAT_") {
			out[key] = val
		}
	}
	return out
}

func marchatEnvKeyOrder(role string) []string {
	keys := make([]string, 0, len(KnownMarchatEnvKeys)+len(ClientHookMarchatEnvKeys))
	keys = append(keys, KnownMarchatEnvKeys...)
	if role == "client" {
		keys = append(keys, ClientHookMarchatEnvKeys...)
	}
	return keys
}

func isClientOnlyHookEnvKey(key string) bool {
	for _, k := range ClientHookMarchatEnvKeys {
		if k == key {
			return true
		}
	}
	return false
}

// buildEnvLines builds the ordered environment section for doctor. Role must be "client" or "server".
func buildEnvLines(role string) []envLine {
	fromEnv := collectMarchatEnviron()
	seen := make(map[string]bool)
	var lines []envLine
	for _, k := range marchatEnvKeyOrder(role) {
		seen[k] = true
		v := fromEnv[k]
		lines = append(lines, envLine{Key: k, Display: FormatEnvValue(k, v)})
	}
	var extra []string
	for k := range fromEnv {
		if seen[k] {
			continue
		}
		// Client hook vars are often exported in the same shell as the server; the server never reads them.
		if role == "server" && isClientOnlyHookEnvKey(k) {
			continue
		}
		extra = append(extra, k)
	}
	sort.Strings(extra)
	for _, k := range extra {
		lines = append(lines, envLine{Key: k, Display: FormatEnvValue(k, fromEnv[k])})
	}
	return lines
}
