package main

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/sgx-labs/statelessagent/internal/graph"
	"github.com/sgx-labs/statelessagent/internal/store"
)

func TestResolveGraphNode_ExactMatch(t *testing.T) {
	db, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	defer db.Close()

	gdb := graph.NewDB(db.Conn())
	if _, err := gdb.UpsertNode(&graph.Node{Type: graph.NodeNote, Name: "notes/design.md"}); err != nil {
		t.Fatalf("UpsertNode: %v", err)
	}

	node, err := resolveGraphNode(gdb, graph.NodeNote, "notes/design.md")
	if err != nil {
		t.Fatalf("resolveGraphNode exact: %v", err)
	}
	if node.Type != graph.NodeNote {
		t.Fatalf("node type = %q, want %q", node.Type, graph.NodeNote)
	}
}

func TestResolveGraphNode_NoteFileFallback(t *testing.T) {
	db, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	defer db.Close()

	gdb := graph.NewDB(db.Conn())
	if _, err := gdb.UpsertNode(&graph.Node{Type: graph.NodeNote, Name: "notes/roadmap.md"}); err != nil {
		t.Fatalf("UpsertNode: %v", err)
	}

	node, err := resolveGraphNode(gdb, graph.NodeFile, "notes/roadmap.md")
	if err != nil {
		t.Fatalf("resolveGraphNode fallback: %v", err)
	}
	if node.Type != graph.NodeNote {
		t.Fatalf("node type = %q, want %q", node.Type, graph.NodeNote)
	}
}

func TestResolveGraphNode_NoFallbackForOtherTypes(t *testing.T) {
	db, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	defer db.Close()

	gdb := graph.NewDB(db.Conn())
	_, err = resolveGraphNode(gdb, graph.NodeAgent, "missing-agent")
	if err == nil {
		t.Fatal("expected error for missing non-note/file node")
	}
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}
