package main

import (
	"encoding/base64"
	"strconv"
	"strings"
	"testing"
)

func TestBuildToastXML_EscapesQuotesAndSpecialChars(t *testing.T) {
	title := `say "hello" & <world>`
	body := `$(Write-Host MARCHAT_SUBEXPR)`
	got := buildToastXML(title, body)

	if !strings.Contains(got, "&amp;") {
		t.Fatalf("expected & escaped in XML: %q", got)
	}
	if !strings.Contains(got, "&lt;") {
		t.Fatalf("expected < escaped in XML: %q", got)
	}
	if !strings.Contains(got, "&#34;") && !strings.Contains(got, "&quot;") {
		t.Fatalf("expected quotes escaped in XML: %q", got)
	}
	if !strings.Contains(got, "$(Write-Host MARCHAT_SUBEXPR)") {
		t.Fatalf("payload should remain as literal XML text: %q", got)
	}
}

func TestBuildToastXML_RejectsHereStringBreakout(t *testing.T) {
	payload := "x\n\"@\nWrite-Host MARCHAT_PWNED\n\"@"
	got := buildToastXML("safe-title", payload)

	script := buildWindowsToastPowerShellScript(got, "marchat")
	if strings.Contains(script, `@"`) || strings.Contains(script, `"@`) {
		t.Fatalf("script must not use here-strings: %q", script)
	}
	if strings.Contains(script, "Write-Host MARCHAT_PWNED") {
		t.Fatalf("injected statement must not appear outside base64 blobs: %q", script)
	}

	encoded := encodePowerShellEncodedCommand(script)
	if strings.Contains(encoded, "$(") {
		t.Fatalf("encoded command must not contain subexpression syntax: %q", encoded)
	}
}

func TestWindowsToastEncodedCommand_NoSubexpressionInScript(t *testing.T) {
	payload := "$(Write-Host MARCHAT_SUBEXPR)"
	encoded := windowsToastEncodedCommand("safe-title", payload, "marchat")
	if encoded == "" {
		t.Fatal("expected non-empty encoded command")
	}

	script := buildWindowsToastPowerShellScript(buildToastXML("safe-title", payload), "marchat")
	if strings.Contains(script, "$(") {
		t.Fatalf("powershell script must not contain unencoded subexpressions: %q", script)
	}
}

func TestBuildDarwinNotificationScript_QuotesNewlines(t *testing.T) {
	payload := "hi\ndo shell script 'touch /tmp/pwned'"
	got := buildDarwinNotificationScript("marchat", payload)

	if strings.Contains(got, "\ndo shell script") {
		t.Fatalf("raw newline injection must not appear: %q", got)
	}
	if !strings.Contains(got, `\n`) {
		t.Fatalf("newlines should be escaped via strconv.Quote: %q", got)
	}
	if !strings.HasPrefix(got, "display notification ") {
		t.Fatalf("unexpected script prefix: %q", got)
	}
}

func TestBuildDarwinNotificationScript_EscapesQuotesAndBackslashes(t *testing.T) {
	title := `ti"tle`
	message := `he said "hi" and \ bye`
	got := buildDarwinNotificationScript(title, message)
	want := "display notification " + strconv.Quote(message) + " with title " + strconv.Quote(title)
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestEncodePowerShellEncodedCommand_NonEmpty(t *testing.T) {
	encoded := encodePowerShellEncodedCommand("Write-Host ok")
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(raw)%2 != 0 {
		t.Fatal("utf16le length should be even")
	}
}

func TestBuildToastXML_Structure(t *testing.T) {
	xmlText := buildToastXML("t", "b")
	if !strings.HasPrefix(xmlText, "<toast>") || !strings.HasSuffix(xmlText, "</toast>") {
		t.Fatalf("unexpected toast xml: %q", xmlText)
	}
}
