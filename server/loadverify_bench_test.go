// Optional hub load benchmarks for maintainers (not run by plain go test ./...).
// They use in-memory SQLite, a real Hub with plugin manager, synthetic Clients
// (nil conn, large send buffers) and TypingMessage to avoid plugin IPC on broadcast.
//
// Run (from repo root):
//
//	go test ./server -run=^$ -bench=Loadverify -benchmem -count=5 | tee loadverify-bench.txt
//
// CPU profile (GC / hot paths). The -bench regexp must match the full sub-benchmark
// name (underscores): BenchmarkLoadverify_HubBroadcast_ChannelMessage/all_in_channel_128.
//
// PowerShell: quote -cpuprofile or ".pprof" is parsed incorrectly and you get a wrong filename.
//
// From repo root, -cpuprofile="name.pprof" is usually created in that same directory (your shell cwd).
// If pprof cannot find it, also try .\server\name.pprof (behavior can depend on Go version).
//
//	go test ./server -run=^$ -bench=Loadverify_HubBroadcast_ChannelMessage/all_in_channel_128 -cpuprofile="loadverify-cpu.pprof"
//	go tool pprof -top .\loadverify-cpu.pprof
//
// Code reality check: channel-scoped broadcasts still range over every entry in
// h.clients (see hub.go broadcast case); recipients are filtered by channel
// membership. Benchmarks vary both total clients and in-channel count.
//
// Compare ChannelMessage/all_in_channel_* vs ChannelMessage_fixedChannel8:
// cost grows with total registered clients, not only #bench population.

package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/Cod-e-Codes/marchat/shared"
	_ "modernc.org/sqlite"
)

func loadverifyDrain(ch <-chan interface{}) {
	go func() {
		for range ch {
		}
	}()
}

// setupLoadverifyHub registers total clients; the first inChannel join "bench",
// the rest join "lobby" only. Starts hub.Run in the background.
func setupLoadverifyHub(b *testing.B, total, inChannel int) *Hub {
	b.Helper()
	if inChannel > total {
		b.Fatal("inChannel > total")
	}
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		b.Fatalf("sqlite: %v", err)
	}
	b.Cleanup(func() { db.Close() })
	CreateSchema(db)
	hub := NewHub("", "", "", db)
	go hub.Run()
	time.Sleep(20 * time.Millisecond)

	// Large buffer avoids hub drop-on-full during fast benchmarks; production uses 256
	// (handlers.go) and then conn.Close() runs. These Clients have conn == nil.
	const loadverifySendBuf = 65536
	for i := 0; i < total; i++ {
		c := &Client{
			username: fmt.Sprintf("loadverify-%d", i),
			send:     make(chan interface{}, loadverifySendBuf),
		}
		loadverifyDrain(c.send)
		hub.clientsMutex.Lock()
		hub.clients[c] = true
		hub.clientsMutex.Unlock()
		if i < inChannel {
			hub.joinChannel(c, "bench")
		} else {
			hub.joinChannel(c, "lobby")
		}
	}

	return hub
}

func BenchmarkLoadverify_HubBroadcast_ChannelMessage(b *testing.B) {
	for _, n := range []int{8, 32, 64, 128} {
		b.Run(fmt.Sprintf("all_in_channel_%d", n), func(b *testing.B) {
			hub := setupLoadverifyHub(b, n, n)

			// TypingMessage avoids plugin IPC in hub Run (TextMessage triggers SendMessageToPlugins).
			msg := shared.Message{
				Sender:    "loadverify",
				Channel:   "bench",
				Type:      shared.TypingMessage,
				CreatedAt: time.Now(),
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				hub.broadcast <- msg
			}
		})
	}
}

func BenchmarkLoadverify_HubBroadcast_ChannelMessage_fixedChannel8(b *testing.B) {
	// 8 recipients in #bench, many extra clients only in #lobby; highlights
	// iteration over all registered clients vs channel population.
	for _, total := range []int{16, 64, 128} {
		b.Run(fmt.Sprintf("in_bench_8_total_%d", total), func(b *testing.B) {
			hub := setupLoadverifyHub(b, total, 8)

			// TypingMessage avoids plugin IPC in hub Run (TextMessage triggers SendMessageToPlugins).
			msg := shared.Message{
				Sender:    "loadverify",
				Channel:   "bench",
				Type:      shared.TypingMessage,
				CreatedAt: time.Now(),
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				hub.broadcast <- msg
			}
		})
	}
}

func BenchmarkLoadverify_HubBroadcast_SystemWide(b *testing.B) {
	for _, n := range []int{8, 32, 64} {
		b.Run(fmt.Sprintf("clients_%d", n), func(b *testing.B) {
			hub := setupLoadverifyHub(b, n, n)

			msg := shared.Message{
				Sender:    "System",
				Channel:   "bench",
				Content:   "announce",
				Type:      shared.TypingMessage,
				CreatedAt: time.Now(),
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				hub.broadcast <- msg
			}
		})
	}
}

func BenchmarkLoadverify_HubBroadcast_ParallelSenders(b *testing.B) {
	const clients = 32
	hub := setupLoadverifyHub(b, clients, clients)

	msg := shared.Message{
		Sender:    "loadverify",
		Channel:   "bench",
		Type:      shared.TypingMessage,
		CreatedAt: time.Now(),
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			hub.broadcast <- msg
		}
	})
}

func BenchmarkLoadverify_JSONMarshal_TextMessage(b *testing.B) {
	msg := shared.Message{
		Sender:    "user",
		Channel:   "general",
		Content:   "hello world loadverify",
		Type:      shared.TextMessage,
		CreatedAt: time.Now(),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(&msg)
	}
}
