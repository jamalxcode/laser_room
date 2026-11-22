package engine

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type secrets struct {
	Rooms map[string]string `json:"rooms"`
}

func TestVault(t *testing.T) {
	dir := t.TempDir()
	rooms := filepath.Join(dir, "rooms")
	os.MkdirAll(rooms, 0o700)
	os.WriteFile(filepath.Join(rooms, "v1"), []byte("room data"), 0o600)
	v := &Vault{Path: filepath.Join(dir, "vault.json"), Wipe: []string{rooms}}

	if err := v.Create("123", secrets{}); err == nil {
		t.Fatal("short PIN accepted")
	}
	if err := v.Create("482913", secrets{Rooms: map[string]string{"r1": "topsecretkey"}}); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(v.Path)
	if strings.Contains(string(raw), "topsecretkey") {
		t.Fatal("secret stored in the clear")
	}
	var got secrets
	if err := v.Open("482913", &got); err != nil || got.Rooms["r1"] != "topsecretkey" {
		t.Fatalf("open: %v %+v", err, got)
	}
	var wp WrongPin
	if err := v.Open("000000", &got); !errors.As(err, &wp) || wp.Left != 9 {
		t.Fatalf("wrong PIN: %v", err)
	}
	if err := v.Open("482913", &got); err != nil {
		t.Fatal("right PIN after a wrong one")
	}
	for i := 1; i < MaxPinTries; i++ {
		if err := v.Open("111111", &got); !errors.As(err, &wp) {
			t.Fatalf("try %d: %v", i, err)
		}
	}
	if err := v.Open("111111", &got); !errors.Is(err, ErrWiped) {
		t.Fatalf("10th wrong PIN: %v", err)
	}
	if v.Exists() {
		t.Fatal("vault not wiped")
	}
	if _, err := os.Stat(rooms); !os.IsNotExist(err) {
		t.Fatal("room data not wiped")
	}
}
