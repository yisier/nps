package main

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

//go:embed assets/npc/*
var npcAssets embed.FS

// findEmbeddedNpc scans embedded assets and returns the best candidate path and bytes.
func findEmbeddedNpc() (string, []byte, error) {
	entries, err := fs.ReadDir(npcAssets, "assets/npc")
	if err != nil {
		return "", nil, err
	}
	// prefer file matching GOOS and GOARCH
	var fallback string
	for _, e := range entries {
		name := e.Name()
		// skip placeholder or hidden files
		if name == ".keep" || name == ".gitkeep" || name[0] == '.' {
			continue
		}
		// skip directory entries
		if e.IsDir() {
			continue
		}
		if fallback == "" {
			fallback = name
		}
		if runtime.GOOS == "windows" {
			if filepath.Ext(name) == ".exe" && (contains(name, runtime.GOOS) || contains(name, runtime.GOARCH)) {
				b, err := npcAssets.ReadFile("assets/npc/" + name)
				return name, b, err
			}
		} else {
			if contains(name, runtime.GOOS) || contains(name, runtime.GOARCH) {
				b, err := npcAssets.ReadFile("assets/npc/" + name)
				return name, b, err
			}
		}
	}
	if fallback != "" && fallback != ".keep" && fallback != ".gitkeep" {
		b, err := npcAssets.ReadFile("assets/npc/" + fallback)
		return fallback, b, err
	}
	return "", nil, fs.ErrNotExist
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (filepath.Base(s) == sub || filepath.Ext(s) == sub || (s != "" && sub != "" && (stringIndex(s, sub) >= 0)))
}

// simple strings.Index replacement to avoid extra imports
func stringIndex(s, sep string) int {
	for i := 0; i+len(sep) <= len(s); i++ {
		if s[i:i+len(sep)] == sep {
			return i
		}
	}
	return -1
}

// extractEmbeddedNpc writes embedded bytes to a temp file and returns the path.
func extractEmbeddedNpc(name string, data []byte) (string, error) {
	tmpDir := os.TempDir()
	out := filepath.Join(tmpDir, name)
	if runtime.GOOS != "windows" {
		// ensure no .exe suffix on unix
	}
	if err := os.WriteFile(out, data, 0o755); err != nil {
		return "", err
	}
	// On unix ensure executable bit set
	if runtime.GOOS != "windows" {
		if err := os.Chmod(out, 0o755); err != nil {
			return "", err
		}
	}
	return out, nil
}
