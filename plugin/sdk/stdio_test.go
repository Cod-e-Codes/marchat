package sdk

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"
)

type stubPlugin struct {
	*BasePlugin
	initCalls int
	lastMsg   Message
	onMsg     func(Message) ([]Message, error)
	initErr   error
}

func (s *stubPlugin) Init(config Config) error {
	s.initCalls++
	if s.initErr != nil {
		return s.initErr
	}
	return s.BasePlugin.Init(config)
}

func (s *stubPlugin) OnMessage(msg Message) ([]Message, error) {
	s.lastMsg = msg
	if s.onMsg != nil {
		return s.onMsg(msg)
	}
	return nil, nil
}

func TestHandlePluginRequest_init_missingConfigSkipped(t *testing.T) {
	p := &stubPlugin{BasePlugin: NewBasePlugin("stub")}
	data, err := json.Marshal(map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	resp := HandlePluginRequest(p, PluginRequest{Type: "init", Data: data}, nil)
	if !resp.Success || resp.Type != "init" {
		t.Fatalf("want success init, got %+v", resp)
	}
	if p.initCalls != 0 {
		t.Fatalf("Init should not run without config key, calls=%d", p.initCalls)
	}
}

func TestHandlePluginRequest_init_withConfig(t *testing.T) {
	p := &stubPlugin{BasePlugin: NewBasePlugin("stub")}
	initBody, err := json.Marshal(InitRequestData{
		Config: &Config{PluginDir: "/p", DataDir: "/d", Settings: map[string]string{"k": "v"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	resp := HandlePluginRequest(p, PluginRequest{Type: "init", Data: initBody}, nil)
	if !resp.Success {
		t.Fatalf("init failed: %+v", resp)
	}
	if p.initCalls != 1 {
		t.Fatalf("want 1 Init call, got %d", p.initCalls)
	}
	cfg := p.GetConfig()
	if cfg.PluginDir != "/p" || cfg.DataDir != "/d" || cfg.Settings["k"] != "v" {
		t.Fatalf("config not applied: %+v", cfg)
	}
}

func TestHandlePluginRequest_init_configNullSkipped(t *testing.T) {
	p := &stubPlugin{BasePlugin: NewBasePlugin("stub")}
	resp := HandlePluginRequest(p, PluginRequest{Type: "init", Data: json.RawMessage(`{"config":null}`)}, nil)
	if !resp.Success {
		t.Fatal(resp)
	}
	if p.initCalls != 0 {
		t.Fatalf("Init should not run for config:null, got %d calls", p.initCalls)
	}
}

func TestHandlePluginRequest_init_initError(t *testing.T) {
	p := &stubPlugin{BasePlugin: NewBasePlugin("stub"), initErr: io.EOF}
	initBody, err := json.Marshal(InitRequestData{Config: &Config{PluginDir: "x"}})
	if err != nil {
		t.Fatal(err)
	}
	resp := HandlePluginRequest(p, PluginRequest{Type: "init", Data: initBody}, nil)
	if resp.Success || !strings.Contains(resp.Error, "initialize") {
		t.Fatalf("want init failure, got %+v", resp)
	}
}

func TestHandlePluginRequest_message(t *testing.T) {
	p := &stubPlugin{BasePlugin: NewBasePlugin("stub")}
	msg := Message{Sender: "u", Content: "hi", CreatedAt: time.Now().Truncate(time.Second)}
	raw, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	resp := HandlePluginRequest(p, PluginRequest{Type: "message", Data: raw}, nil)
	if !resp.Success || resp.Type != "message" {
		t.Fatalf("unexpected response %+v", resp)
	}
	if p.lastMsg.Content != "hi" {
		t.Fatalf("lastMsg=%+v", p.lastMsg)
	}
}

func TestHandlePluginRequest_message_customReply(t *testing.T) {
	p := &stubPlugin{
		BasePlugin: NewBasePlugin("stub"),
		onMsg: func(Message) ([]Message, error) {
			return []Message{{Sender: "bot", Content: "ack", CreatedAt: time.Now()}}, nil
		},
	}
	raw, err := json.Marshal(Message{Sender: "u", Content: "x", CreatedAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	resp := HandlePluginRequest(p, PluginRequest{Type: "message", Data: raw}, nil)
	if !resp.Success || len(resp.Data) == 0 {
		t.Fatal(resp)
	}
	var out Message
	if err := json.Unmarshal(resp.Data, &out); err != nil {
		t.Fatal(err)
	}
	if out.Content != "ack" {
		t.Fatalf("want ack, got %+v", out)
	}
}

func TestHandlePluginRequest_command_nilHandler(t *testing.T) {
	p := &stubPlugin{BasePlugin: NewBasePlugin("stub")}
	args, _ := json.Marshal([]string{"a"})
	resp := HandlePluginRequest(p, PluginRequest{Type: "command", Command: "x", Data: args}, nil)
	if resp.Success || resp.Error != "unknown command" {
		t.Fatalf("got %+v", resp)
	}
}

func TestHandlePluginRequest_command_withHandler(t *testing.T) {
	p := &stubPlugin{BasePlugin: NewBasePlugin("stub")}
	args, _ := json.Marshal([]string{"line"})
	handler := func(cmd string, args []string) PluginResponse {
		if cmd != "ping" {
			t.Fatalf("cmd=%q", cmd)
		}
		if len(args) != 1 || args[0] != "line" {
			t.Fatalf("args=%v", args)
		}
		b, _ := json.Marshal(Message{Sender: "bot", Content: "pong"})
		return PluginResponse{Type: "message", Success: true, Data: b}
	}
	resp := HandlePluginRequest(p, PluginRequest{Type: "command", Command: "ping", Data: args}, handler)
	if !resp.Success || string(resp.Data) == "" {
		t.Fatalf("got %+v", resp)
	}
}

func TestHandlePluginRequest_shutdown(t *testing.T) {
	p := &stubPlugin{BasePlugin: NewBasePlugin("stub")}
	resp := HandlePluginRequest(p, PluginRequest{Type: "shutdown"}, nil)
	if !resp.Success || resp.Type != "shutdown" {
		t.Fatalf("got %+v", resp)
	}
}

func TestHandlePluginRequest_unknownType(t *testing.T) {
	p := &stubPlugin{BasePlugin: NewBasePlugin("stub")}
	resp := HandlePluginRequest(p, PluginRequest{Type: "nope"}, nil)
	if resp.Success || resp.Error != "unknown request type" {
		t.Fatalf("got %+v", resp)
	}
}

func TestRunIO_roundTrip(t *testing.T) {
	p := &stubPlugin{BasePlugin: NewBasePlugin("stub")}
	var inBuf bytes.Buffer
	var outBuf bytes.Buffer

	initData, _ := json.Marshal(InitRequestData{Config: &Config{PluginDir: "/p", DataDir: "/d"}})
	initReq, _ := json.Marshal(PluginRequest{Type: "init", Data: initData})
	_, _ = inBuf.Write(append(initReq, '\n'))

	msg, _ := json.Marshal(Message{Sender: "a", Content: "b", CreatedAt: time.Now()})
	reqMsg, _ := json.Marshal(PluginRequest{Type: "message", Data: msg})
	_, _ = inBuf.Write(append(reqMsg, '\n'))

	shut, _ := json.Marshal(PluginRequest{Type: "shutdown"})
	_, _ = inBuf.Write(append(shut, '\n'))

	err := RunIO(&inBuf, &outBuf, io.Discard, p, nil)
	if err != nil {
		t.Fatal(err)
	}

	dec := json.NewDecoder(&outBuf)
	var r1, r2, r3 PluginResponse
	if err := dec.Decode(&r1); err != nil || !r1.Success || r1.Type != "init" {
		t.Fatalf("r1 %+v err %v", r1, err)
	}
	if err := dec.Decode(&r2); err != nil || !r2.Success || r2.Type != "message" {
		t.Fatalf("r2 %+v err %v", r2, err)
	}
	if err := dec.Decode(&r3); err != nil || !r3.Success || r3.Type != "shutdown" {
		t.Fatalf("r3 %+v err %v", r3, err)
	}
	if p.initCalls != 1 {
		t.Fatalf("initCalls=%d", p.initCalls)
	}
}

func TestRunIO_EOFNoWrites(t *testing.T) {
	p := &stubPlugin{BasePlugin: NewBasePlugin("stub")}
	err := RunIO(strings.NewReader(""), io.Discard, io.Discard, p, nil)
	if err != nil {
		t.Fatal(err)
	}
	if p.initCalls != 0 {
		t.Fatalf("unexpected Init: %d", p.initCalls)
	}
}
