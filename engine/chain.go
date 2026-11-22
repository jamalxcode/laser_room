package engine

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
)

// Version is one commit: its parent, the author's steps, and the hash of the
// board they produce, signed by the author (V1).
type Version struct {
	Parent    string `json:"parent"`         // "" for the room's first version
	Seq       uint64 `json:"seq"`            // parent's seq + 1
	Author    string `json:"author"`         // hex Ed25519 public key (per room)
	At        int64  `json:"at"`             // author's clock (unix ms), labelled as theirs
	Base      string `json:"base,omitempty"` // the version the author's draft started from
	Message   string `json:"message,omitempty"`
	Steps     []Step `json:"steps"`
	BoardHash string `json:"board"`
	Sig       string `json:"sig"`
}

// Hash identifies a version: SHA-256 of everything but the signature.
func (v *Version) Hash() string {
	c := *v
	c.Sig = ""
	b, _ := json.Marshal(c)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func boardHash(b *Board) string {
	data, _ := json.Marshal(b) // maps marshal in key order: deterministic
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// KeyPair is a member's per-room signing key.
type KeyPair struct {
	Public  ed25519.PublicKey
	Private ed25519.PrivateKey
}

// NewKeyPair makes a fresh per-room key pair.
func NewKeyPair() KeyPair {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	return KeyPair{pub, priv}
}

// Key is the member key as stored on the board.
func (k KeyPair) Key() string { return hex.EncodeToString(k.Public) }

func (k KeyPair) sign(v *Version) {
	v.Sig = hex.EncodeToString(ed25519.Sign(k.Private, []byte(v.Hash())))
}

func verify(v *Version) error {
	pub, err := hex.DecodeString(v.Author)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return errors.New("bad author key")
	}
	sig, err := hex.DecodeString(v.Sig)
	if err != nil || !ed25519.Verify(pub, []byte(v.Hash()), sig) {
		return errors.New("bad signature")
	}
	return nil
}

// Draft is a member's uncommitted work: steps recorded on top of Base (V2).
type Draft struct {
	Base  string
	Steps []Step
}

// Chain is a room's version history as one device knows it.
type Chain struct {
	mu       sync.Mutex
	versions map[string]*Version
	boards   map[string]*Board
	head     string
	Clock    func() int64
}

// ErrUnknownParent: a version arrived before its parent; fetch the parent first.
var ErrUnknownParent = errors.New("parent version not known yet")

func newChain(clock func() int64) *Chain {
	return &Chain{versions: map[string]*Version{}, boards: map[string]*Board{}, Clock: clock}
}

// NewRoom creates a room's first version: the owner creates it with a title
// and their name (R1).
func NewRoom(owner KeyPair, title, name string, clock func() int64) (*Chain, error) {
	c := newChain(clock)
	v := &Version{Seq: 1, Author: owner.Key(), At: clock(), Steps: []Step{{Type: StepCreate, Title: title, Name: name}}}
	board, out := apply(newBoard(), v.Author, v.At, v.Seq, "", v.Steps, c)
	if !out[0].Applied {
		return nil, errors.New(out[0].Reason)
	}
	v.BoardHash = boardHash(board)
	owner.sign(v)
	h := v.Hash()
	c.versions[h], c.boards[h], c.head = v, board, h
	return c, nil
}

// Head is the latest version this device knows (by the V8 rule).
func (c *Chain) Head() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.head
}

// HeadVersion returns the head version.
func (c *Chain) HeadVersion() *Version {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.versions[c.head]
}

// Board returns the board at a version (implements history).
func (c *Chain) Board(hash string) (*Board, error) {
	b, ok := c.boards[hash]
	if !ok {
		return nil, fmt.Errorf("version %.12s not available", hash)
	}
	return b, nil
}

// Latest returns the board at the head.
func (c *Chain) Latest() *Board {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.boards[c.head]
}

// Version returns a known version.
func (c *Chain) Version(hash string) *Version {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.versions[hash]
}

// SeqOf is a version's number (0 if unknown); implements history.
func (c *Chain) SeqOf(hash string) uint64 {
	if v := c.versions[hash]; v != nil {
		return v.Seq
	}
	return 0
}

// IsAncestor reports whether ancestor is of itself or one of its ancestors.
func (c *Chain) IsAncestor(ancestor, of string) bool {
	for h := of; h != ""; {
		if h == ancestor {
			return true
		}
		v := c.versions[h]
		if v == nil {
			return false
		}
		h = v.Parent
	}
	return false
}

// History lists versions from the head back to the first.
func (c *Chain) History() []*Version {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []*Version
	for h := c.head; h != ""; {
		v := c.versions[h]
		if v == nil {
			break
		}
		out = append(out, v)
		h = v.Parent
	}
	return out
}

// Commit places a draft on top of the latest version (V3): the steps are
// re-played on the head, signed, and returned as the new version, with what
// happened to each step. It doesn't publish; the network layer publishes
// with compare-and-swap and, if someone else got there first, adds their
// version and commits the same draft again (V4).
func (c *Chain) Commit(kp KeyPair, d Draft, message string) (*Version, []Outcome, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	parent := c.versions[c.head]
	v := &Version{Parent: c.head, Seq: parent.Seq + 1, Author: kp.Key(), At: c.Clock(), Base: d.Base, Message: clip(message, 500), Steps: d.Steps}
	board, out := apply(c.boards[c.head], v.Author, v.At, v.Seq, d.Base, d.Steps, c)
	applied := false
	for _, o := range out {
		applied = applied || o.Applied
	}
	if !applied {
		return nil, out, errors.New("nothing in this draft can be applied")
	}
	v.BoardHash = boardHash(board)
	kp.sign(v)
	h := v.Hash()
	c.versions[h], c.boards[h] = v, board
	c.pickHead(h)
	return v, out, nil
}

// Add checks a version received from someone else and adds it: the
// signature, the parent, and that its steps really produce the board it
// claims (V1). Rule-breaking steps were skipped by the author's app too, so
// the board hash still matches.
func (c *Chain) Add(v *Version) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	h := v.Hash()
	if c.versions[h] != nil {
		return nil
	}
	if err := verify(v); err != nil {
		return err
	}
	parent, ok := c.versions[v.Parent]
	if !ok {
		return ErrUnknownParent
	}
	if v.Seq != parent.Seq+1 {
		return errors.New("bad sequence number")
	}
	pb := c.boards[v.Parent]
	if a := pb.Members[v.Author]; v.Steps != nil && (a == nil || a.Removed) && !(len(v.Steps) == 1 && v.Steps[0].Type == StepJoin) {
		return errors.New("author isn't a member")
	}
	board, _ := apply(pb, v.Author, v.At, v.Seq, v.Base, v.Steps, c)
	if boardHash(board) != v.BoardHash {
		return errors.New("version's board doesn't match its changes")
	}
	c.versions[h], c.boards[h] = v, board
	c.pickHead(h)
	return nil
}

// pickHead applies the V8 rule: the longest chain wins; between versions of
// the same length, the lowest hash. Every device picks the same head.
func (c *Chain) pickHead(candidate string) {
	cur := c.versions[c.head]
	v := c.versions[candidate]
	if cur == nil || v.Seq > cur.Seq || (v.Seq == cur.Seq && candidate < c.head) {
		c.head = candidate
	}
}

// Orphans returns versions that lost a V8 tie and aren't on the head's
// chain: their authors' apps re-commit them on top.
func (c *Chain) Orphans(author string) []*Version {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []*Version
	for h, v := range c.versions {
		if v.Author == author && !c.IsAncestor(h, c.head) {
			out = append(out, v)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Seq < out[j].Seq })
	return out
}

// Join is the step a new member commits after opening an invite link: they
// introduce themselves with a name (R3, J1).
func Join(name string) Step { return Step{Type: StepJoin, Name: name} }

// NewID makes a random id for a note or reply.
func NewID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Text makes a *string for steps.
func Text(s string) *string { return &s }
