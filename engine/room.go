package engine

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	"golang.org/x/crypto/nacl/box"
)

// Room is one room as a device knows it: its id (used to find it on the
// network), its secret (encrypts everything shared), and its version chain.
type Room struct {
	ID     string
	Secret []byte
	Chain  *Chain
}

// Invite is what an invite link carries: the room, its secret, and who made
// the link (the newcomer trusts that member's snapshot to start from).
type Invite struct {
	Room    string `json:"room"`
	Secret  []byte `json:"secret"`
	Inviter string `json:"inviter"`
	Reuse   bool   `json:"reuse,omitempty"`
}

const linkPrefix = "laser-room://join/"

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func randomSecret() []byte {
	b := make([]byte, 32)
	rand.Read(b)
	return b
}

// CreateRoom makes a new room owned by kp (R1).
func CreateRoom(kp KeyPair, title, name string, clock func() int64) (*Room, error) {
	c, err := NewRoom(kp, title, name, clock)
	if err != nil {
		return nil, err
	}
	return &Room{ID: randomHex(32), Secret: randomSecret(), Chain: c}, nil
}

// Link makes an invite link for the room (R2). reusable is recorded for the
// app; single-use links are enforced by the inviter's app.
func (r *Room) Link(inviter KeyPair, reusable bool) string {
	inv := Invite{Room: r.ID, Secret: r.Secret, Inviter: inviter.Key(), Reuse: reusable}
	data, _ := json.Marshal(inv)
	return linkPrefix + base64.RawURLEncoding.EncodeToString(data)
}

// ParseLink reads an invite link.
func ParseLink(link string) (Invite, error) {
	var inv Invite
	s := strings.TrimSpace(link)
	if !strings.HasPrefix(s, linkPrefix) {
		return inv, errors.New("not a Laser Room invite link")
	}
	data, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(s, linkPrefix))
	if err != nil {
		return inv, errors.New("the invite link is damaged")
	}
	if err := json.Unmarshal(data, &inv); err != nil || len(inv.Secret) != 32 || inv.Room == "" {
		return inv, errors.New("the invite link is damaged")
	}
	return inv, nil
}

// Remove removes a member by moving the room (R8): a new room starts from the
// current document without them, and the old room's last version carries an
// invite to it, locked separately for each remaining member. It returns the
// new room; the caller publishes both the old room's version and the new
// room's first version.
func (r *Room) Remove(kp KeyPair, member string) (*Room, *Version, error) {
	cur := r.Chain.Latest()
	t := cur.Members[member]
	if t == nil || t.Removed {
		return nil, nil, errors.New("not a member")
	}
	if !canManage(cur.Members[kp.Key()]) {
		return nil, nil, errors.New("only the owner or a co-owner can remove members")
	}
	next := &Room{ID: randomHex(32), Secret: randomSecret()}
	snap := cur.Clone()
	snap.Members[member].Removed = true
	first := &Version{Seq: 1, Author: kp.Key(), At: r.Chain.Clock(), Steps: []Step{{Type: StepContinue, To: r.ID, Board: snap}}}
	c, err := startChain(kp, first, r.Chain.Clock)
	if err != nil {
		return nil, nil, err
	}
	next.Chain = c

	invites := map[string]string{}
	for k, m := range cur.Members {
		if k == member || m.Removed {
			continue
		}
		pub, err := hex.DecodeString(m.BoxKey)
		if err != nil || len(pub) != 32 {
			return nil, nil, errors.New("member " + m.Name + " has no receiving key")
		}
		var to [32]byte
		copy(to[:], pub)
		data, _ := json.Marshal(Invite{Room: next.ID, Secret: next.Secret, Inviter: kp.Key()})
		sealed, err := box.SealAnonymous(nil, data, &to, rand.Reader)
		if err != nil {
			return nil, nil, err
		}
		invites[k] = base64.StdEncoding.EncodeToString(sealed)
	}
	v, out, err := r.Chain.Commit(kp, Draft{Steps: []Step{{Type: StepRemove, Member: member, To: next.ID, Invites: invites}}}, "")
	if err != nil {
		return nil, nil, errors.New(out[0].Reason)
	}
	return next, v, nil
}

// MoveInvite opens this member's invite to the room's new home after a
// removal; ok is false if the room hasn't moved or there's no invite for them.
func (r *Room) MoveInvite(kp KeyPair) (Invite, bool) {
	b := r.Chain.Latest()
	sealed, ok := b.Invites[kp.Key()]
	if b.MovedTo == "" || !ok {
		return Invite{}, false
	}
	data, err := base64.StdEncoding.DecodeString(sealed)
	if err != nil {
		return Invite{}, false
	}
	plain, ok := box.OpenAnonymous(nil, data, &kp.BoxPublic, &kp.BoxPrivate)
	if !ok {
		return Invite{}, false
	}
	var inv Invite
	if json.Unmarshal(plain, &inv) != nil || inv.Room != b.MovedTo {
		return Invite{}, false
	}
	return inv, true
}

// Snapshot lets a newcomer start from the latest version instead of
// replaying all of history (S1): the version and its board, signed by the
// member who made it.
type Snapshot struct {
	Room    string   `json:"room"`
	Version *Version `json:"version"`
	Board   *Board   `json:"board"`
	Signer  string   `json:"signer"`
	Sig     string   `json:"sig"`
}

func (s *Snapshot) digest() []byte {
	sum := sha256.Sum256([]byte("laser-room/snapshot/" + s.Room + "/" + s.Version.Hash() + "/" + boardHash(s.Board)))
	return sum[:]
}

// Snapshot of the room at its head, signed by kp.
func (r *Room) Snapshot(kp KeyPair) *Snapshot {
	r.Chain.mu.Lock()
	v, b := r.Chain.versions[r.Chain.head], r.Chain.boards[r.Chain.head]
	r.Chain.mu.Unlock()
	s := &Snapshot{Room: r.ID, Version: v, Board: b.Clone(), Signer: kp.Key()}
	s.Sig = hex.EncodeToString(ed25519.Sign(kp.Private, s.digest()))
	return s
}

// JoinFromSnapshot starts a newcomer's copy of a room from a snapshot made
// by the member who invited them (the invite names them), then later
// versions are added on top as usual.
func JoinFromSnapshot(inv Invite, s *Snapshot, clock func() int64) (*Room, error) {
	if s.Room != inv.Room || s.Signer != inv.Inviter {
		return nil, errors.New("the snapshot isn't from the person who invited you")
	}
	pub, err := hex.DecodeString(s.Signer)
	sig, err2 := hex.DecodeString(s.Sig)
	if err != nil || err2 != nil || len(pub) != ed25519.PublicKeySize || !ed25519.Verify(pub, s.digest(), sig) {
		return nil, errors.New("the snapshot's signature doesn't check out")
	}
	if err := verify(s.Version); err != nil {
		return nil, err
	}
	if boardHash(s.Board) != s.Version.BoardHash {
		return nil, errors.New("the snapshot's board doesn't match its version")
	}
	c := newChain(clock)
	h := s.Version.Hash()
	c.versions[h], c.boards[h], c.head = s.Version, s.Board.Clone(), h
	return &Room{ID: inv.Room, Secret: inv.Secret, Chain: c}, nil
}
