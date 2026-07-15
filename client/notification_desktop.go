package main

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"strconv"
	"unicode/utf16"
)

// buildToastXML constructs Windows toast XML with escaped text nodes.
func buildToastXML(title, body string) string {
	return `<toast><visual><binding template="ToastText02"><text id="1">` +
		escapeXMLText(title) +
		`</text><text id="2">` +
		escapeXMLText(body) +
		`</text></binding></visual></toast>`
}

func escapeXMLText(s string) string {
	var buf bytes.Buffer
	if err := xml.EscapeText(&buf, []byte(s)); err != nil {
		return s
	}
	return buf.String()
}

// encodePowerShellEncodedCommand UTF-16LE-encodes script for powershell -EncodedCommand.
func encodePowerShellEncodedCommand(script string) string {
	u16 := utf16.Encode([]rune(script))
	b := make([]byte, len(u16)*2)
	for i, v := range u16 {
		b[i*2] = byte(v)
		b[i*2+1] = byte(v >> 8)
	}
	return base64.StdEncoding.EncodeToString(b)
}

// buildWindowsToastPowerShellScript returns a fixed PowerShell script; user data is base64 only.
func buildWindowsToastPowerShellScript(toastXML, appID string) string {
	xmlB64 := base64.StdEncoding.EncodeToString([]byte(toastXML))
	appB64 := base64.StdEncoding.EncodeToString([]byte(appID))
	return fmt.Sprintf(`[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null
$xmlText = [System.Text.Encoding]::UTF8.GetString([Convert]::FromBase64String('%s'))
$appId = [System.Text.Encoding]::UTF8.GetString([Convert]::FromBase64String('%s'))
$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
$xml.LoadXml($xmlText)
$toast = New-Object Windows.UI.Notifications.ToastNotification $xml
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier($appId).Show($toast)`, xmlB64, appB64)
}

// buildDarwinNotificationScript builds a single-line osascript -e script with quoted literals.
func buildDarwinNotificationScript(title, message string) string {
	return fmt.Sprintf("display notification %s with title %s", strconv.Quote(message), strconv.Quote(title))
}

// windowsToastEncodedCommand builds the full -EncodedCommand argument for a toast notification.
func windowsToastEncodedCommand(title, message, appID string) string {
	script := buildWindowsToastPowerShellScript(buildToastXML(title, message), appID)
	return encodePowerShellEncodedCommand(script)
}
