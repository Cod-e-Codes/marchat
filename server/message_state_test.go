package server

import (
	"testing"
	"time"

	"github.com/Cod-e-Codes/marchat/shared"
)

func TestPersistReaction_UsesReactionTargetID(t *testing.T) {
	db, err := InitDB(":memory:")
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer db.Close()
	CreateSchema(db)

	id, err := InsertMessage(db, shared.Message{Sender: "alice", Content: "hello", CreatedAt: time.Now()})
	if err != nil {
		t.Fatalf("InsertMessage failed: %v", err)
	}

	PersistReaction(db, shared.Message{
		Sender: "bob",
		Type:   shared.ReactionMessage,
		Reaction: &shared.ReactionMeta{
			Emoji:    "👍",
			TargetID: id,
		},
	})

	replayed := LoadReactionsForMessages(db, []int64{id})
	if len(replayed) != 1 {
		t.Fatalf("expected 1 replayed reaction, got %d", len(replayed))
	}
	if replayed[0].Reaction == nil || replayed[0].Reaction.TargetID != id {
		t.Fatalf("expected replayed reaction target id %d, got %+v", id, replayed[0].Reaction)
	}
}

func TestPersistReaction_RemovalUsesReactionTargetID(t *testing.T) {
	db, err := InitDB(":memory:")
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer db.Close()
	CreateSchema(db)

	id, err := InsertMessage(db, shared.Message{Sender: "alice", Content: "hello", CreatedAt: time.Now()})
	if err != nil {
		t.Fatalf("InsertMessage failed: %v", err)
	}

	PersistReaction(db, shared.Message{
		Sender: "bob",
		Type:   shared.ReactionMessage,
		Reaction: &shared.ReactionMeta{
			Emoji:    "👍",
			TargetID: id,
		},
	})
	PersistReaction(db, shared.Message{
		Sender: "bob",
		Type:   shared.ReactionMessage,
		Reaction: &shared.ReactionMeta{
			Emoji:     "👍",
			TargetID:  id,
			IsRemoval: true,
		},
	})

	replayed := LoadReactionsForMessages(db, []int64{id})
	if len(replayed) != 0 {
		t.Fatalf("expected 0 replayed reactions after removal, got %d", len(replayed))
	}
}
