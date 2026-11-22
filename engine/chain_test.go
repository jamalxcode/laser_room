package engine

import (
	"strings"
	"sync/atomic"
	"testing"
)

// clock gives each call a later time, like real devices (but deterministic).
func clock() func() int64 {
	var t int64 = 1_700_000_000_000
	return func() int64 { return atomic.AddInt64(&t, 1000) }
}

// copyChain simulates another device receiving every version.
func syncTo(t *testing.T, from, to *Chain) {
	t.Helper()
	hist := from.History()
	for i := len(hist) - 1; i >= 0; i-- {
		if err := to.Add(hist[i]); err != nil {
			t.Fatalf("add version %d: %v", hist[i].Seq, err)
		}
	}
}

func cloneChain(t *testing.T, c *Chain) *Chain {
	t.Helper()
	hist := c.History()
	first := hist[len(hist)-1]
	n := newChain(clock())
	board, _ := apply(newBoard(), first.Author, first.At, first.Seq, "", first.Steps, n)
	h := first.Hash()
	n.versions[h], n.boards[h], n.head = first, board, h
	syncTo(t, c, n)
	return n
}

func mustCommit(t *testing.T, c *Chain, kp KeyPair, d Draft) *Version {
	t.Helper()
	v, out, err := c.Commit(kp, d, "")
	if err != nil {
		t.Fatalf("commit: %v (%+v)", err, out)
	}
	return v
}

func TestRoomCommitsAndChecks(t *testing.T) {
	owner, ann, bob := NewKeyPair(), NewKeyPair(), NewKeyPair()
	room, err := NewRoom(owner, "Spring launch", "Jamal", clock())
	if err != nil {
		t.Fatal(err)
	}
	b := room.Latest()
	if b.Title != "Spring launch" || b.Members[owner.Key()].Role != Owner || b.Members[owner.Key()].Name != "Jamal" {
		t.Fatalf("room: %+v", b)
	}

	// Ann and Bob join (they hold the invite link's keys) and pick names.
	mustCommit(t, room, ann, Draft{Base: room.Head(), Steps: []Step{Join("Ann")}})
	if _, _, err := room.Commit(bob, Draft{Steps: []Step{Join("ann")}}, ""); err == nil || !strings.Contains(err.Error(), "nothing") {
		t.Fatal("a taken name was accepted")
	}
	if _, _, err := room.Commit(bob, Draft{Steps: []Step{Join("Аnn")}}, ""); err == nil { // Cyrillic А
		t.Fatal("a look-alike name was accepted")
	}
	mustCommit(t, room, bob, Draft{Steps: []Step{Join("Bob")}})

	// A note with an attachment, a vote and a reply, each a commit.
	note := NewID()
	mustCommit(t, room, owner, Draft{Base: room.Head(), Steps: []Step{{Type: StepNoteAdd, ID: note, Text: Text("Launch in April?"), Color: "pink",
		Attachments: []Attachment{{Kind: "image", Name: "hero.png", Torrent: "abcd", Size: 1234}}}}})
	mustCommit(t, room, ann, Draft{Steps: []Step{{Type: StepVote, ID: note, Value: "up"}}})
	mustCommit(t, room, bob, Draft{Steps: []Step{{Type: StepReplyAdd, ID: note, Reply: NewID(), Text: Text("Mid-April")}}})

	n := room.Latest().Notes[note]
	if n.Votes[ann.Key()] != "up" || len(n.Replies) != 1 || n.Attachments[0].Torrent != "abcd" {
		t.Fatalf("note: %+v", n)
	}

	// Another device receives the whole chain and checks it.
	other := cloneChain(t, room)
	if other.Head() != room.Head() || boardHash(other.Latest()) != boardHash(room.Latest()) {
		t.Fatal("devices disagree")
	}

	// Tampering is caught: a changed step breaks the signature; a re-signed
	// version whose board doesn't match its steps is rejected too.
	v, _, _ := room.Commit(ann, Draft{Steps: []Step{{Type: StepVote, ID: note, Value: "down"}}}, "")
	bad := *v
	bad.Steps = []Step{{Type: StepVote, ID: note, Value: "up"}}
	if err := other.Add(&bad); err == nil || !strings.Contains(err.Error(), "signature") {
		t.Fatalf("tampered version accepted: %v", err)
	}
	forged := *v
	forged.BoardHash = strings.Repeat("0", 64)
	ann.sign(&forged)
	if err := other.Add(&forged); err == nil || !strings.Contains(err.Error(), "match") {
		t.Fatalf("forged board accepted: %v", err)
	}
	if err := other.Add(v); err != nil {
		t.Fatalf("real version rejected: %v", err)
	}
}

func TestRulesOnlyAuthorsEditAnyoneMoves(t *testing.T) {
	owner, ann := NewKeyPair(), NewKeyPair()
	room, _ := NewRoom(owner, "Board", "Owner", clock())
	mustCommit(t, room, ann, Draft{Steps: []Step{Join("Ann")}})
	note := NewID()
	mustCommit(t, room, owner, Draft{Steps: []Step{{Type: StepNoteAdd, ID: note, Text: Text("mine")}}})

	_, out, err := room.Commit(ann, Draft{Steps: []Step{{Type: StepNoteEdit, ID: note, Text: Text("hijack")}}}, "")
	if err == nil || out[0].Reason != "only the note's author can edit it" {
		t.Fatalf("edit by non-author: %v %+v", err, out)
	}
	mustCommit(t, room, ann, Draft{Steps: []Step{{Type: StepNoteMove, ID: note, X: 300, Y: 40}}})
	if n := room.Latest().Notes[note]; n.X != 300 || n.Text != "mine" {
		t.Fatalf("move: %+v", n)
	}
	// Delete leaves a marker; it's gone from the board.
	mustCommit(t, room, owner, Draft{Steps: []Step{{Type: StepNoteDelete, ID: note}}})
	if n := room.Latest().Notes[note]; n.Deleted == 0 || n.Text != "" || n.Author != owner.Key() {
		t.Fatalf("delete marker: %+v", n)
	}
}

func TestDraftsReplayOnTopAndReplacedEdits(t *testing.T) {
	owner, ann := NewKeyPair(), NewKeyPair()
	room, _ := NewRoom(owner, "Board", "Owner", clock())
	mustCommit(t, room, ann, Draft{Steps: []Step{Join("Ann")}})
	note := NewID()
	mustCommit(t, room, owner, Draft{Steps: []Step{{Type: StepNoteAdd, ID: note, Text: Text("v1")}}})

	// Two drafts start from the same version.
	base := room.Head()
	d1 := Draft{Base: base, Steps: []Step{{Type: StepNoteEdit, ID: note, Text: Text("owner's v2")}}}
	d2 := Draft{Base: base, Steps: []Step{{Type: StepNoteEdit, ID: note, Text: Text("owner's v2b")}, {Type: StepVote, ID: note, Value: "up"}}}
	// Ann adds a note meanwhile.
	mustCommit(t, room, ann, Draft{Base: base, Steps: []Step{{Type: StepNoteAdd, ID: NewID(), Text: Text("ann's")}}})
	// The first draft lands on top of Ann's commit (replayed, V3/V4).
	mustCommit(t, room, owner, d1)
	if len(room.Latest().Notes) != 2 || room.Latest().Notes[note].Text != "owner's v2" {
		t.Fatal("draft not replayed on top")
	}
	// The second draft edits the same note: it wins, the earlier text is kept
	// in history and the author is told (V5).
	_, out, err := room.Commit(owner, d2, "")
	if err != nil || out[0].Replaced == "" || !out[1].Applied {
		t.Fatalf("replaced edit: %v %+v", err, out)
	}
	n := room.Latest().Notes[note]
	if n.Text != "owner's v2b" || len(n.History) != 2 || n.History[1].Text != "owner's v2" {
		t.Fatalf("history: %+v", n)
	}
	// A step that no longer applies is skipped with a reason.
	gone := NewID()
	mustCommit(t, room, ann, Draft{Steps: []Step{{Type: StepNoteAdd, ID: gone, Text: Text("x")}}})
	mustCommit(t, room, ann, Draft{Steps: []Step{{Type: StepNoteDelete, ID: gone}}})
	_, out, _ = room.Commit(ann, Draft{Steps: []Step{{Type: StepNoteEdit, ID: gone, Text: Text("y")}, {Type: StepVote, ID: note, Value: "down"}}}, "")
	if out[0].Applied || out[0].Reason != "that note no longer exists" || !out[1].Applied {
		t.Fatalf("skipped step: %+v", out)
	}
}

func TestMuteRemoveCoOwnerHandoverClear(t *testing.T) {
	owner, ann, bob := NewKeyPair(), NewKeyPair(), NewKeyPair()
	room, _ := NewRoom(owner, "Board", "Owner", clock())
	mustCommit(t, room, ann, Draft{Steps: []Step{Join("Ann")}})
	mustCommit(t, room, bob, Draft{Steps: []Step{Join("Bob")}})

	if _, _, err := room.Commit(bob, Draft{Steps: []Step{{Type: StepMute, Member: ann.Key()}}}, ""); err == nil {
		t.Fatal("a member muted someone")
	}
	mustCommit(t, room, owner, Draft{Steps: []Step{{Type: StepCoOwner, Member: ann.Key(), On: true}}})
	mustCommit(t, room, ann, Draft{Steps: []Step{{Type: StepMute, Member: bob.Key()}}})
	if _, _, err := room.Commit(bob, Draft{Steps: []Step{{Type: StepNoteAdd, ID: NewID(), Text: Text("spam")}}}, ""); err == nil {
		t.Fatal("muted member posted")
	}
	epoch := room.Latest().KeyEpoch
	mustCommit(t, room, ann, Draft{Steps: []Step{{Type: StepRemove, Member: bob.Key()}}})
	if !room.Latest().Members[bob.Key()].Removed || room.Latest().KeyEpoch != epoch+1 {
		t.Fatal("remove didn't take or keys didn't change")
	}
	if _, _, err := room.Commit(bob, Draft{Steps: []Step{Join("Bob2")}}, ""); err == nil {
		t.Fatal("a removed member re-joined with the old key")
	}
	if _, _, err := room.Commit(ann, Draft{Steps: []Step{{Type: StepRemove, Member: owner.Key()}}}, ""); err == nil {
		t.Fatal("co-owner removed the owner")
	}
	mustCommit(t, room, owner, Draft{Steps: []Step{{Type: StepHandover, Member: ann.Key()}}})
	if room.Latest().Owner != ann.Key() || room.Latest().Members[owner.Key()].Role != CoOwner {
		t.Fatal("handover")
	}
	mustCommit(t, room, ann, Draft{Steps: []Step{{Type: StepNoteAdd, ID: NewID(), Text: Text("final")}}})
	if _, _, err := room.Commit(owner, Draft{Steps: []Step{{Type: StepClear}}}, ""); err == nil {
		t.Fatal("non-owner cleared")
	}
	mustCommit(t, room, ann, Draft{Steps: []Step{{Type: StepClear}}})
	if !room.Latest().Cleared || len(room.Latest().Notes) != 0 {
		t.Fatal("clear")
	}
	if _, _, err := room.Commit(ann, Draft{Steps: []Step{{Type: StepNoteAdd, ID: NewID(), Text: Text("after")}}}, ""); err == nil {
		t.Fatal("posted after clearing")
	}
}

func TestRevertWholeBoardAndOneNote(t *testing.T) {
	owner, ann := NewKeyPair(), NewKeyPair()
	room, _ := NewRoom(owner, "Board", "Owner", clock())
	mustCommit(t, room, ann, Draft{Steps: []Step{Join("Ann")}})
	note := NewID()
	mustCommit(t, room, ann, Draft{Steps: []Step{{Type: StepNoteAdd, ID: note, Text: Text("first")}}})
	good := room.Head()
	mustCommit(t, room, ann, Draft{Base: good, Steps: []Step{{Type: StepNoteEdit, ID: note, Text: Text("second")}}})
	mustCommit(t, room, ann, Draft{Steps: []Step{{Type: StepNoteAdd, ID: NewID(), Text: Text("noise")}}})

	// Ann reverts her own note to its first text (a new version).
	mustCommit(t, room, ann, Draft{Steps: []Step{{Type: StepNoteRevert, ID: note, Rev: 0}}})
	if room.Latest().Notes[note].Text != "first" {
		t.Fatal("note revert")
	}
	// Only owners revert the whole board.
	if _, _, err := room.Commit(ann, Draft{Steps: []Step{{Type: StepRevert, To: good}}}, ""); err == nil {
		t.Fatal("member reverted the board")
	}
	before := len(room.History())
	mustCommit(t, room, owner, Draft{Steps: []Step{{Type: StepRevert, To: good}}})
	if len(room.Latest().Notes) != 1 || room.Latest().Notes[note].Text != "first" {
		t.Fatalf("board revert: %+v", room.Latest().Notes)
	}
	if len(room.History()) != before+1 {
		t.Fatal("a revert must be a new version (nothing lost)")
	}
	if room.Latest().Members[ann.Key()] == nil {
		t.Fatal("revert must not undo membership")
	}
}

func TestDoublePublishConverges(t *testing.T) {
	owner, ann := NewKeyPair(), NewKeyPair()
	a, _ := NewRoom(owner, "Board", "Owner", clock())
	mustCommit(t, a, ann, Draft{Steps: []Step{Join("Ann")}})
	b := cloneChain(t, a)

	// Both commit on the same parent without seeing each other (V8).
	va := mustCommit(t, a, owner, Draft{Steps: []Step{{Type: StepNoteAdd, ID: NewID(), Text: Text("from owner")}}})
	vb := mustCommit(t, b, ann, Draft{Steps: []Step{{Type: StepNoteAdd, ID: NewID(), Text: Text("from ann")}}})
	if err := a.Add(vb); err != nil {
		t.Fatal(err)
	}
	if err := b.Add(va); err != nil {
		t.Fatal(err)
	}
	if a.Head() != b.Head() {
		t.Fatal("devices picked different heads")
	}
	// The loser's app re-commits its steps on top; then everyone has both notes.
	loser, lc := ann, b
	if a.Head() == vb.Hash() {
		loser, lc = owner, a
	}
	orphans := lc.Orphans(loser.Key())
	if len(orphans) != 1 {
		t.Fatalf("orphans: %d", len(orphans))
	}
	v := mustCommit(t, lc, loser, Draft{Base: orphans[0].Parent, Steps: orphans[0].Steps})
	other := a
	if lc == a {
		other = b
	}
	if err := other.Add(v); err != nil {
		t.Fatal(err)
	}
	if a.Head() != b.Head() || len(a.Latest().Notes) != 2 {
		t.Fatalf("not converged: %d notes", len(a.Latest().Notes))
	}
}
