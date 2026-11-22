package engine

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/nacl/secretbox"
)

// The PIN vault (J2–J4): every secret the app keeps (room keys, the
// member's per-room signing keys) is stored only in this file, encrypted
// with a key derived from the person's PIN (Argon2id). 10 wrong PINs in a
// row wipe the vault and the person's room data. On Mac and iPhone the app
// also wraps the vault key with the Secure Enclave.

// MaxPinTries is the number of wrong PINs in a row that wipe the device.
const MaxPinTries = 10

// MinPinLength is the shortest PIN accepted.
const MinPinLength = 6

// ErrWiped: too many wrong PINs; the vault and room data were deleted.
var ErrWiped = errors.New("too many wrong PINs: this device's rooms were wiped")

// WrongPin is returned for a wrong PIN, with the tries left.
type WrongPin struct{ Left int }

func (e WrongPin) Error() string { return fmt.Sprintf("wrong PIN (%d tries left)", e.Left) }

type vaultFile struct {
	V        int    `json:"v"`
	Salt     []byte `json:"salt"`
	Nonce    []byte `json:"nonce"`
	Box      []byte `json:"box"`
	Failures int    `json:"failures"`
}

// Vault is the encrypted secrets file.
type Vault struct {
	Path string
	Wipe []string // removed with the vault after too many wrong PINs
}

var digits = regexp.MustCompile(`^[0-9]+$`)

func vaultKey(pin string, salt []byte) *[32]byte {
	var k [32]byte
	copy(k[:], argon2.IDKey([]byte(pin), salt, 3, 64*1024, 4, 32))
	return &k
}

// Exists reports whether a vault has been created on this device.
func (v *Vault) Exists() bool { _, err := os.Stat(v.Path); return err == nil }

// Create makes a new vault with the given secrets.
func (v *Vault) Create(pin string, secrets any) error {
	if !digits.MatchString(pin) || len(pin) < MinPinLength {
		return fmt.Errorf("the PIN must be at least %d digits", MinPinLength)
	}
	if err := os.MkdirAll(filepath.Dir(v.Path), 0o700); err != nil {
		return err
	}
	return v.seal(pin, secrets)
}

func (v *Vault) seal(pin string, secrets any) error {
	msg, err := json.Marshal(secrets)
	if err != nil {
		return err
	}
	f := vaultFile{V: 1, Salt: make([]byte, 16)}
	var nonce [24]byte
	rand.Read(f.Salt)
	rand.Read(nonce[:])
	f.Nonce = nonce[:]
	f.Box = secretbox.Seal(nil, msg, &nonce, vaultKey(pin, f.Salt))
	return v.write(f)
}

// Open decrypts the secrets into out. A wrong PIN counts towards the limit;
// the 10th wrong PIN in a row wipes everything (ErrWiped).
func (v *Vault) Open(pin string, out any) error {
	data, err := os.ReadFile(v.Path)
	if err != nil {
		return err
	}
	var f vaultFile
	if err := json.Unmarshal(data, &f); err != nil {
		return err
	}
	var nonce [24]byte
	copy(nonce[:], f.Nonce)
	msg, ok := secretbox.Open(nil, f.Box, &nonce, vaultKey(pin, f.Salt))
	if !ok {
		f.Failures++
		if f.Failures >= MaxPinTries {
			v.WipeAll()
			return ErrWiped
		}
		if err := v.write(f); err != nil {
			return err
		}
		return WrongPin{Left: MaxPinTries - f.Failures}
	}
	if f.Failures != 0 {
		f.Failures = 0
		if err := v.write(f); err != nil {
			return err
		}
	}
	return json.Unmarshal(msg, out)
}

// Update saves new secrets under the same PIN (checked first).
func (v *Vault) Update(pin string, secrets any) error {
	var discard json.RawMessage
	if err := v.Open(pin, &discard); err != nil {
		return err
	}
	return v.seal(pin, secrets)
}

// WipeAll deletes the vault and the room data.
func (v *Vault) WipeAll() {
	os.Remove(v.Path)
	for _, p := range v.Wipe {
		os.RemoveAll(p)
	}
}

func (v *Vault) write(f vaultFile) error {
	data, _ := json.Marshal(f)
	tmp := v.Path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, v.Path)
}
