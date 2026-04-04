package doctor

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Cod-e-Codes/marchat/shared"
)

func TestFormatEnvValue(t *testing.T) {
	t.Parallel()
	if got := FormatEnvValue("MARCHAT_PORT", ""); got != "(not set)" {
		t.Fatalf("empty: got %q", got)
	}
	if got := FormatEnvValue("MARCHAT_PORT", "8080"); got != "8080" {
		t.Fatalf("plain: got %q", got)
	}
	got := FormatEnvValue("MARCHAT_ADMIN_KEY", "supersecret")
	if !strings.Contains(got, "len=11") || !strings.Contains(got, "****cret") {
		t.Fatalf("secret mask unexpected: %q", got)
	}
}

func TestParseLatestReleaseTag(t *testing.T) {
	t.Parallel()
	body := []byte(`{"tag_name":"v1.2.3","draft":false}`)
	tag, err := ParseLatestReleaseTag(body)
	if err != nil || tag != "v1.2.3" {
		t.Fatalf("got %q err %v", tag, err)
	}
	_, err = ParseLatestReleaseTag([]byte(`{}`))
	if err == nil {
		t.Fatal("expected error for empty tag")
	}
}

func TestBuildEnvLines_orderAndMask(t *testing.T) {
	t.Parallel()
	old := osEnviron
	t.Cleanup(func() { osEnviron = old })
	osEnviron = func() []string {
		return []string{
			"MARCHAT_PORT=9090",
			"MARCHAT_ADMIN_KEY=topsecret",
			"MARCHAT_EXTRA_CUSTOM=bar",
		}
	}
	lines := buildEnvLines()
	foundPort := false
	foundExtra := false
	for _, e := range lines {
		if e.Key == "MARCHAT_PORT" && e.Display == "9090" {
			foundPort = true
		}
		if e.Key == "MARCHAT_EXTRA_CUSTOM" && e.Display == "bar" {
			foundExtra = true
		}
		if e.Key == "MARCHAT_ADMIN_KEY" && strings.Contains(e.Display, "topsecret") {
			t.Fatalf("admin key leaked: %q", e.Display)
		}
	}
	if !foundPort || !foundExtra {
		t.Fatalf("missing lines: port=%v extra=%v", foundPort, foundExtra)
	}
}

func TestCheckUpdate_fakeClient(t *testing.T) {
	t.Setenv("MARCHAT_DOCTOR_NO_NETWORK", "")
	client := &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			body := `{"tag_name":"v99.0.0"}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	o := Options{HTTPClient: client}
	info := checkUpdate(o, "v0.1.0")
	if info.Latest != "v99.0.0" || info.UpToDate {
		t.Fatalf("unexpected: %+v", info)
	}
}

func TestCheckUpdate_noNetworkEnv(t *testing.T) {
	t.Setenv("MARCHAT_DOCTOR_NO_NETWORK", "1")
	info := checkUpdate(Options{}, "v1.0.0")
	if !info.Skipped || info.SkipReason == "" {
		t.Fatalf("expected skip: %+v", info)
	}
}

func TestWriteTextReport_nonTTYWriterIsPlainASCII(t *testing.T) {
	t.Parallel()
	rep := Report{
		Role:          "client",
		VersionDetail: "v0.0.0-test",
		GoVersion:     "go1.test",
		GOOS:          "plan9",
		GOARCH:        "amd64",
		StdoutTTY:     false,
		Checks: []Check{
			{ID: "sample", Status: "ok", Message: "all good"},
		},
		Update: UpdateInfo{Skipped: true, SkipReason: "test"},
	}
	var buf bytes.Buffer
	writeTextReport(&buf, rep)
	out := buf.String()
	if strings.Contains(out, "\x1b[") {
		t.Fatalf("expected plain output without ANSI escapes, got %q", out)
	}
	if !strings.Contains(out, "marchat doctor (client)") || !strings.Contains(out, "[ok] sample: all good") {
		t.Fatalf("unexpected plain report: %s", out)
	}
}

func TestReportJSON_roundTrip(t *testing.T) {
	t.Parallel()
	rep := Report{
		Role:          "client",
		Version:       shared.ClientVersion,
		VersionDetail: shared.GetVersionInfo(),
		Checks: []Check{
			{ID: "x", Status: "ok", Message: "y"},
		},
		Update: UpdateInfo{Current: "v1", Skipped: true, SkipReason: "test"},
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(rep); err != nil {
		t.Fatal(err)
	}
	var dec Report
	if err := json.NewDecoder(&buf).Decode(&dec); err != nil {
		t.Fatal(err)
	}
	if dec.Role != "client" || len(dec.Checks) != 1 {
		t.Fatalf("decode mismatch: %+v", dec)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
