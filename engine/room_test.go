package engine

import (
	"path/filepath"
	"testing"
)

func TestInviteLinks(t *testing.T) {
	owner := NewKeyPair()
	r, err := CreateRoom(owner, "Launch", "Jamal", clock())
	if err != nil {
		t.Fatal(err)
	}
	link := r.Link(owner, true)
	inv, err := ParseLink(link)
	if err != nil || inv.Room != r.ID || string(inv.Secret) != string(r.Secret) || inv.Inviter != owner.Key() || !inv.Reuse {
		t.Fatalf("link: %v %+v", err, inv)
	}
	if _, err := ParseLink("laser-room://join/garbage!"); err == nil {
		t.Fatal("damaged link accepted")
	}
	if _, err := ParseLink("https://example.org"); err == nil {
		t.Fatal("not-a-link accepted")
	}
}

func TestRemovalMovesTheRoom(t *testing.T) {
	owner, ann, bob := NewKeyPair(), NewKeyPair(), NewKeyPair()
	r, _ := CreateRoom(owner, "Launch", "Jamal", clock())
	mustCommit(t, r.Chain, ann, Draft{Steps: []Step{Join(ann, "Ann")}})
	mustCommit(t, r.Chain, bob, Draft{Steps: []Step{Join(bob, "Bob")}})
	note := NewID()
	mustCommit(t, r.Chain, ann, Draft{Steps: []Step{{Type: StepNoteAdd, ID: note, Text: Text("keep me")}}})

	// Bob's device has a copy of the room too.
	bobsCopy := cloneChain(t, r.Chain)

	if _, _, err := r.Remove(ann, bob.Key()); err == nil {
		t.Fatal("a plain member removed someone")
	}
	next, v, err := r.Remove(owner, bob.Key())
	if err != nil {
		t.Fatal(err)
	}
	old := r.Chain.Latest()
	if old.MovedTo != next.ID || !old.Members[bob.Key()].Removed {
		t.Fatal("old room didn't record the move")
	}
	// The old room is frozen.
	if _, _, err := r.Chain.Commit(ann, Draft{Steps: []Step{{Type: StepNoteAdd, ID: NewID(), Text: Text("late")}}}, ""); err == nil {
		t.Fatal("committed to a moved room")
	}

	// Ann's device opens her invite and starts the new room from its first
	// version, received over the network.
	inv, ok := r.MoveInvite(ann)
	if !ok || inv.Room != next.ID || string(inv.Secret) != string(next.Secret) {
		t.Fatal("Ann can't open her invite")
	}
	first := next.Chain.History()[0]
	annChain, err := ChainFrom(first, clock())
	if err != nil {
		t.Fatal(err)
	}
	nb := annChain.Latest()
	if nb.Notes[note].Text != "keep me" || nb.Members[bob.Key()] != nil || nb.Members[ann.Key()] == nil || nb.From != r.ID {
		t.Fatalf("new room's starting document: %+v", nb)
	}
	// The new room works.
	mustCommit(t, annChain, ann, Draft{Steps: []Step{{Type: StepVote, ID: note, Value: "up"}}})

	// Bob gets the old room's last version but no invite, and can't post in
	// the new room even if he learned its keys.
	if err := bobsCopy.Add(v); err != nil {
		t.Fatal(err)
	}
	if _, ok := (&Room{Chain: bobsCopy}).MoveInvite(bob); ok {
		t.Fatal("the removed member got an invite")
	}
}

func TestSnapshotJoin(t *testing.T) {
	owner, ann, cara := NewKeyPair(), NewKeyPair(), NewKeyPair()
	r, _ := CreateRoom(owner, "Launch", "Jamal", clock())
	mustCommit(t, r.Chain, ann, Draft{Steps: []Step{Join(ann, "Ann")}})
	for i := 0; i < 5; i++ {
		mustCommit(t, r.Chain, ann, Draft{Steps: []Step{{Type: StepNoteAdd, ID: NewID(), Text: Text("idea")}}})
	}
	inv, _ := ParseLink(r.Link(owner, false))
	snap := r.Snapshot(owner)

	c, err := JoinFromSnapshot(inv, snap, clock())
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Chain.Latest().Notes) != 5 {
		t.Fatal("snapshot board")
	}
	// Later versions land on top of the snapshot.
	v := mustCommit(t, r.Chain, ann, Draft{Steps: []Step{{Type: StepNoteAdd, ID: NewID(), Text: Text("later")}}})
	if err := c.Chain.Add(v); err != nil {
		t.Fatal(err)
	}
	mustCommit(t, c.Chain, cara, Draft{Steps: []Step{Join(cara, "Cara")}})
	if len(c.Chain.Latest().Notes) != 6 || c.Chain.Latest().Members[cara.Key()] == nil {
		t.Fatal("after snapshot")
	}

	// A snapshot signed by someone other than the inviter is refused, and so is a doctored board.
	other := r.Snapshot(ann)
	if _, err := JoinFromSnapshot(inv, other, clock()); err == nil {
		t.Fatal("snapshot from a non-inviter accepted")
	}
	bad := r.Snapshot(owner)
	bad.Board.Title = "doctored"
	if _, err := JoinFromSnapshot(inv, bad, clock()); err == nil {
		t.Fatal("doctored snapshot accepted")
	}
}

func TestDraftsSaved(t *testing.T) {
	path := filepath.Join(t.TempDir(), "drafts")
	key := make([]byte, 32)
	key[0] = 7
	d := map[string]Draft{"room1": {Base: "abc", Steps: []Step{{Type: StepNoteAdd, ID: "n1", Text: Text("half-written")}}}}
	if err := SaveDrafts(path, key, d); err != nil {
		t.Fatal(err)
	}
	got, err := LoadDrafts(path, key)
	if err != nil || got["room1"].Count() != 1 || *got["room1"].Steps[0].Text != "half-written" {
		t.Fatalf("drafts: %v %+v", err, got)
	}
	wrong := make([]byte, 32)
	if _, err := LoadDrafts(path, wrong); err == nil {
		t.Fatal("drafts opened with the wrong key")
	}
	none, err := LoadDrafts(filepath.Join(t.TempDir(), "missing"), key)
	if err != nil || len(none) != 0 {
		t.Fatal("missing drafts file")
	}
}
