package main

import (
	"context"
	"testing"

	"ehang.io/nps/client"
)

func resetShortcutState(t *testing.T) {
	t.Helper()

	shortcutsMu.Lock()
	shortcuts = nil
	shortcutsMu.Unlock()

	runningMu.Lock()
	running = make(map[string]context.CancelFunc)
	clients = make(map[string]*client.TRPClient)
	runningMu.Unlock()

	statusMu.Lock()
	clientErrors = make(map[string]string)
	clientConnected = make(map[string]bool)
	clientAttempted = make(map[string]bool)
	statusMu.Unlock()
}

func TestUpdateShortcut(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("APPDATA", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)

	resetShortcutState(t)
	t.Cleanup(func() { resetShortcutState(t) })

	addShortcut(ShortClient{Name: "one", Addr: "1.1.1.1:8000", Key: "k1"})
	addShortcut(ShortClient{Name: "two", Addr: "2.2.2.2:8000", Key: "k2"})

	a := &App{}
	if err := a.UpdateShortcut("one", "1.1.1.1:8000", "k1", ShortClient{
		Name: "one-new", Addr: "1.1.1.1:8001", Key: "k1-new", TLS: true,
	}); err != nil {
		t.Fatalf("UpdateShortcut: %v", err)
	}

	loadShortcuts()
	found := false
	for _, sc := range shortcuts {
		if sc.Addr == "1.1.1.1:8001" {
			found = true
			if sc.Name != "one-new" || sc.Key != "k1-new" || !sc.TLS {
				t.Fatalf("updated shortcut = %+v", sc)
			}
		}
	}
	if !found {
		t.Fatal("updated shortcut not saved")
	}

	// 改成另一个已存在的客户端应被拒绝。
	if err := a.UpdateShortcut("one-new", "1.1.1.1:8001", "k1-new", ShortClient{
		Addr: "2.2.2.2:8000", Key: "k2",
	}); err == nil {
		t.Fatal("duplicate update was allowed")
	}

	// 不存在的客户端应报错。
	if err := a.UpdateShortcut("missing", "9.9.9.9:1", "nope", ShortClient{
		Addr: "9.9.9.9:2", Key: "nope2",
	}); err == nil {
		t.Fatal("updating a missing shortcut succeeded")
	}
}
