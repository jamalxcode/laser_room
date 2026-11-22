package engine

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"os"

	"golang.org/x/crypto/nacl/secretbox"
)

// Drafts (B5, V2) survive the app closing: they're saved on the device,
// encrypted with a key kept in the PIN vault.

// Count is how many uncommitted changes a draft holds ("Commit 3 changes").
func (d Draft) Count() int { return len(d.Steps) }

// SaveDrafts writes every room's draft, encrypted with key (32 bytes).
func SaveDrafts(path string, key []byte, drafts map[string]Draft) error {
	if len(key) != 32 {
		return errors.New("draft key must be 32 bytes")
	}
	msg, err := json.Marshal(drafts)
	if err != nil {
		return err
	}
	var k [32]byte
	var nonce [24]byte
	copy(k[:], key)
	rand.Read(nonce[:])
	out := secretbox.Seal(nonce[:], msg, &nonce, &k)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, out, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// LoadDrafts reads the saved drafts (none if the file doesn't exist).
func LoadDrafts(path string, key []byte) (map[string]Draft, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]Draft{}, nil
	}
	if err != nil {
		return nil, err
	}
	if len(data) < 24 || len(key) != 32 {
		return nil, errors.New("drafts file is damaged")
	}
	var k [32]byte
	var nonce [24]byte
	copy(k[:], key)
	copy(nonce[:], data[:24])
	msg, ok := secretbox.Open(nil, data[24:], &nonce, &k)
	if !ok {
		return nil, errors.New("drafts can't be opened with this key")
	}
	out := map[string]Draft{}
	return out, json.Unmarshal(msg, &out)
}
