package host

import (
	"testing"

	"github.com/Cod-e-Codes/marchat/plugin/sdk"
)

func TestEnqueuePluginChatMessageDropsWhenBufferFull(t *testing.T) {
	h := NewPluginHost(t.TempDir(), t.TempDir())
	inst := &PluginInstance{
		Name:     "p",
		msgQueue: make(chan sdk.Message, 2),
	}
	inst.msgQueue <- sdk.Message{Content: "a"}
	inst.msgQueue <- sdk.Message{Content: "b"}

	h.enqueuePluginChatMessage(inst, "p", sdk.Message{Content: "c"})

	if n := len(inst.msgQueue); n != 2 {
		t.Fatalf("expected buffer to stay at 2 after drop, got %d", n)
	}
}

func TestEnqueuePluginChatMessageNilQueueNoop(t *testing.T) {
	h := NewPluginHost(t.TempDir(), t.TempDir())
	inst := &PluginInstance{Name: "p", msgQueue: nil}
	h.enqueuePluginChatMessage(inst, "p", sdk.Message{Content: "x"})
}

func TestDrainAndWaitPluginOutboundNoQueue(t *testing.T) {
	h := NewPluginHost(t.TempDir(), t.TempDir())
	inst := &PluginInstance{Name: "p"}
	h.drainAndWaitPluginOutbound(inst)
}

func TestDrainAndWaitPluginOutboundClosesWriter(t *testing.T) {
	h := NewPluginHost(t.TempDir(), t.TempDir())
	inst := &PluginInstance{Name: "p"}
	ch := make(chan sdk.Message, 1)
	inst.msgQueue = ch
	inst.outboundDone.Add(1)
	go func() {
		defer inst.outboundDone.Done()
		h.runPluginOutboundWriter(ch, inst)
	}()

	h.drainAndWaitPluginOutbound(inst)
}
