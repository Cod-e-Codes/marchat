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

func buildEnvLines() []envLine {
	fromEnv := collectMarchatEnviron()
	seen := make(map[string]bool)
	var lines []envLine
	for _, k := range KnownMarchatEnvKeys {
		seen[k] = true
		v := fromEnv[k]
		lines = append(lines, envLine{Key: k, Display: FormatEnvValue(k, v)})
	}
	var extra []string
	for k := range fromEnv {
		if !seen[k] {
			extra = append(extra, k)
		}
	}
	sort.Strings(extra)
	for _, k := range extra {
		lines = append(lines, envLine{Key: k, Display: FormatEnvValue(k, fromEnv[k])})
	}
	return lines
}
