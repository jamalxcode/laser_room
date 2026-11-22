// Package engine is Laser Room's engine: the room is a document passed
// around as a chain of signed versions (like Git), shared over BitTorrent.
//
// This file holds the board and its rules: applying a member's steps to a
// board, the same way on every device.
package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"unicode"
)

// Roles.
const (
	Owner   = "owner"
	CoOwner = "coowner"
	Member  = "member"
)

// Attachment is a file, image, video or link on a note or reply. Files are
// separate torrents (Torrent is their infohash), shared across versions.
type Attachment struct {
	Kind    string `json:"kind"` // file, image, video, link
	Name    string `json:"name"`
	URL     string `json:"url,omitempty"`
	Torrent string `json:"torrent,omitempty"`
	Size    int64  `json:"size,omitempty"`
	Type    string `json:"type,omitempty"`
}

// Revision is an earlier state of a note's content, kept for history and
// reverting one note (V7).
type Revision struct {
	Seq         uint64       `json:"seq"` // the version number that produced it
	Text        string       `json:"text"`
	Attachments []Attachment `json:"attachments,omitempty"`
	By          string       `json:"by"`
	At          int64        `json:"at"`
}

// Reply is one entry in a note's thread.
type Reply struct {
	ID          string       `json:"id"`
	Author      string       `json:"author"`
	Text        string       `json:"text"`
	Attachments []Attachment `json:"attachments,omitempty"`
	Created     int64        `json:"created"`
	Edited      int64        `json:"edited,omitempty"`
	Deleted     int64        `json:"deleted,omitempty"`
}

// Note is a post-it on the board.
type Note struct {
	ID          string            `json:"id"`
	Author      string            `json:"author"`
	Text        string            `json:"text"`
	Attachments []Attachment      `json:"attachments,omitempty"`
	X           float64           `json:"x"`
	Y           float64           `json:"y"`
	Color       string            `json:"color"`
	Created     int64             `json:"created"`
	Edited      int64             `json:"edited,omitempty"`
	EditedSeq   uint64            `json:"edited_seq,omitempty"` // version number of the last content edit
	Deleted     int64             `json:"deleted,omitempty"`
	Votes       map[string]string `json:"votes,omitempty"` // member key → up/down
	Replies     []Reply           `json:"replies,omitempty"`
	History     []Revision        `json:"history,omitempty"`
}

// MemberInfo is one person in the room.
type MemberInfo struct {
	Key     string `json:"key"`               // hex public key (per room)
	ID      string `json:"id"`                // one-way hash of the key
	BoxKey  string `json:"box_key,omitempty"` // hex X25519 key for receiving locked invites (room moves)
	Name    string `json:"name"`
	Role    string `json:"role"`
	Muted   bool   `json:"muted,omitempty"`
	Removed bool   `json:"removed,omitempty"`
}

// Board is the room's content at one version.
type Board struct {
	Title    string                 `json:"title"`
	Owner    string                 `json:"owner"`
	Members  map[string]*MemberInfo `json:"members"`
	Notes    map[string]*Note       `json:"notes"`
	Cleared  bool                   `json:"cleared,omitempty"`
	KeyEpoch int                    `json:"key_epoch"` // bumped when someone is removed (R8)
	// After someone is removed the room moves (R8): MovedTo names the new room
	// and Invites holds, for each remaining member, the new room's invite
	// locked with their box key. The old room takes no more commits.
	MovedTo string            `json:"moved_to,omitempty"`
	Invites map[string]string `json:"invites,omitempty"`
	From    string            `json:"from,omitempty"` // the room this one continues (after a move)
}

// MemberID is the one-way ID number of a member key.
func MemberID(key string) string {
	sum := sha256.Sum256([]byte("laser-room/member/" + key))
	return hex.EncodeToString(sum[:8])
}

func newBoard() *Board {
	return &Board{Members: map[string]*MemberInfo{}, Notes: map[string]*Note{}}
}

// Clone deep-copies a board so applying steps never changes a stored version.
func (b *Board) Clone() *Board {
	c := &Board{Title: b.Title, Owner: b.Owner, Cleared: b.Cleared, KeyEpoch: b.KeyEpoch, MovedTo: b.MovedTo, From: b.From,
		Members: make(map[string]*MemberInfo, len(b.Members)), Notes: make(map[string]*Note, len(b.Notes))}
	for k, m := range b.Members {
		mm := *m
		c.Members[k] = &mm
	}
	if b.Invites != nil {
		c.Invites = make(map[string]string, len(b.Invites))
		for k, v := range b.Invites {
			c.Invites[k] = v
		}
	}
	for k, n := range b.Notes {
		nn := *n
		nn.Attachments = append([]Attachment(nil), n.Attachments...)
		nn.Replies = append([]Reply(nil), n.Replies...)
		nn.History = append([]Revision(nil), n.History...)
		if n.Votes != nil {
			nn.Votes = make(map[string]string, len(n.Votes))
			for vk, vv := range n.Votes {
				nn.Votes[vk] = vv
			}
		}
		c.Notes[k] = &nn
	}
	return c
}

// SortedNotes returns the notes in creation order.
func (b *Board) SortedNotes() []*Note {
	out := make([]*Note, 0, len(b.Notes))
	for _, n := range b.Notes {
		out = append(out, n)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Created != out[j].Created {
			return out[i].Created < out[j].Created
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// Step is one change in a commit.
type Step struct {
	Type        string            `json:"type"`
	ID          string            `json:"id,omitempty"`    // note id
	Reply       string            `json:"reply,omitempty"` // reply id
	Text        *string           `json:"text,omitempty"`
	Attachments []Attachment      `json:"attachments,omitempty"`
	X           float64           `json:"x,omitempty"`
	Y           float64           `json:"y,omitempty"`
	Color       string            `json:"color,omitempty"`
	Value       string            `json:"value,omitempty"` // vote: up, down, ""
	Member      string            `json:"member,omitempty"`
	Name        string            `json:"name,omitempty"`
	Title       string            `json:"title,omitempty"`
	On          bool              `json:"on,omitempty"`
	To          string            `json:"to,omitempty"`      // revert target version; new room on a move
	Rev         int               `json:"rev,omitempty"`     // note revert: index into History
	BoxKey      string            `json:"box_key,omitempty"` // create/join: the member's receiving key
	Invites     map[string]string `json:"invites,omitempty"` // remove: locked invites per remaining member
	Board       *Board            `json:"board,omitempty"`   // continue: the document the new room starts from
}

// Step types.
const (
	StepCreate      = "room/create"
	StepJoin        = "member/join" // a new member introduces themselves with a name
	StepName        = "member/name"
	StepMute        = "member/mute"
	StepUnmute      = "member/unmute"
	StepRemove      = "member/remove"
	StepCoOwner     = "member/coowner"
	StepHandover    = "room/handover"
	StepClear       = "room/clear"
	StepNoteAdd     = "note/add"
	StepNoteEdit    = "note/edit"
	StepNoteMove    = "note/move"
	StepNoteDelete  = "note/delete"
	StepNoteRevert  = "note/revert"
	StepVote        = "vote"
	StepReplyAdd    = "reply/add"
	StepReplyEdit   = "reply/edit"
	StepReplyDelete = "reply/delete"
	StepRevert      = "room/revert"
	StepContinue    = "room/continue" // first step of a room that a removal moved to
)

// Outcome says what happened to one step when it was applied.
type Outcome struct {
	Step     int    `json:"step"`
	Applied  bool   `json:"applied"`
	Reason   string `json:"reason,omitempty"`   // why a step was skipped
	Replaced string `json:"replaced,omitempty"` // V5: an edit that replaced someone's newer edit
}

const maxText = 10000
const maxName = 40

func clip(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

// nameKey folds look-alike characters so "Jamal", "JAMAL" and "Jаmal" (Cyrillic
// а) count as the same name (J6).
func nameKey(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		if c, ok := confusables[r]; ok {
			r = c
		}
		if unicode.IsSpace(r) || unicode.Is(unicode.Mn, r) || r == '.' || r == '_' || r == '-' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

var confusables = map[rune]rune{
	'а': 'a', 'е': 'e', 'о': 'o', 'р': 'p', 'с': 'c', 'х': 'x', 'у': 'y', 'і': 'i', 'ј': 'j', 'ѕ': 's',
	'0': 'o', '1': 'l', 'ı': 'i', '|': 'l', 'α': 'a', 'ο': 'o', 'ν': 'v', 'κ': 'k', 'τ': 't',
}

func (b *Board) nameTaken(name, except string) bool {
	k := nameKey(name)
	for key, m := range b.Members {
		if key != except && !m.Removed && m.Name != "" && nameKey(m.Name) == k {
			return true
		}
	}
	return false
}

func canManage(m *MemberInfo) bool {
	return m != nil && !m.Removed && (m.Role == Owner || m.Role == CoOwner)
}

func canPost(m *MemberInfo) bool {
	return m != nil && !m.Removed && !m.Muted && m.Name != ""
}

// history gives the rules what they need from the version graph: old boards
// (for whole-board reverts) and version numbers (to spot replaced edits, V5).
type history interface {
	Board(hash string) (*Board, error)
	SeqOf(hash string) uint64
}

// apply applies one author's steps to a copy of the board. version is the
// hash of the version being built (recorded in note history). lookup finds
// old boards for reverts. It never fails as a whole: steps that break a rule
// are skipped, with the reason, identically on every device.
func apply(b *Board, author string, at int64, seq uint64, base string, steps []Step, h history) (*Board, []Outcome) {
	nb := b.Clone()
	out := make([]Outcome, len(steps))
	for i, s := range steps {
		out[i] = Outcome{Step: i}
		reason, replaced := applyStep(nb, author, at, seq, base, s, h)
		out[i].Applied = reason == ""
		out[i].Reason = reason
		out[i].Replaced = replaced
	}
	return nb, out
}

func applyStep(b *Board, author string, at int64, seq uint64, base string, s Step, h history) (reason, replaced string) {
	if b.Cleared {
		return "the table has been cleared", ""
	}
	if b.MovedTo != "" {
		return "this room has moved", ""
	}
	me := b.Members[author]
	switch s.Type {
	case StepCreate:
		if b.Owner != "" {
			return "room already created", ""
		}
		b.Owner = author
		b.Title = clip(s.Title, 200)
		b.Members[author] = &MemberInfo{Key: author, ID: MemberID(author), BoxKey: s.BoxKey, Name: clip(strings.TrimSpace(s.Name), maxName), Role: Owner}
		return "", ""

	case StepContinue:
		// The new room after a removal starts from the old room's last
		// document; only its owner or a co-owner can start it.
		if b.Owner != "" || s.Board == nil {
			return "room already started", ""
		}
		m := s.Board.Members[author]
		if !canManage(m) {
			return "only the owner or a co-owner can move the room", ""
		}
		nb := s.Board.Clone()
		nb.MovedTo, nb.Invites, nb.Cleared = "", nil, false
		nb.From = s.To
		for k, mm := range nb.Members {
			if mm.Removed {
				delete(nb.Members, k)
			}
		}
		*b = *nb
		return "", ""

	case StepJoin:
		// Anyone holding the invite link's keys can join (the owner chose to
		// share it, R3); removed people can't come back with an old key.
		if me != nil {
			if me.Removed {
				return "removed from this room", ""
			}
			return "already a member", ""
		}
		name := clip(strings.TrimSpace(s.Name), maxName)
		if name == "" || b.nameTaken(name, author) {
			return "that name is taken in this room", ""
		}
		b.Members[author] = &MemberInfo{Key: author, ID: MemberID(author), BoxKey: s.BoxKey, Name: name, Role: Member}
		return "", ""

	case StepName:
		if me == nil || me.Removed {
			return "not a member", ""
		}
		name := clip(strings.TrimSpace(s.Name), maxName)
		if name == "" || b.nameTaken(name, author) {
			return "that name is taken in this room", ""
		}
		me.Name = name
		return "", ""

	case StepMute, StepUnmute:
		if !canManage(me) {
			return "only the owner or a co-owner can do that", ""
		}
		t := b.Members[s.Member]
		if t == nil || t.Role == Owner {
			return "can't mute that member", ""
		}
		t.Muted = s.Type == StepMute
		return "", ""

	case StepRemove:
		if !canManage(me) {
			return "only the owner or a co-owner can do that", ""
		}
		t := b.Members[s.Member]
		if t == nil || t.Removed || t.Role == Owner {
			return "can't remove that member", ""
		}
		// Everyone who stays must get an invite to the new room (R8).
		if s.To == "" {
			return "a removal must move the room", ""
		}
		for k, m := range b.Members {
			if k == s.Member || m.Removed {
				continue
			}
			if _, ok := s.Invites[k]; !ok {
				return "a remaining member has no invite to the new room", ""
			}
		}
		if _, ok := s.Invites[s.Member]; ok {
			return "the removed member can't be invited to the new room", ""
		}
		t.Removed = true
		b.KeyEpoch++
		b.MovedTo = s.To
		b.Invites = s.Invites
		return "", ""

	case StepCoOwner:
		if me == nil || me.Removed || me.Role != Owner {
			return "only the owner can do that", ""
		}
		t := b.Members[s.Member]
		if t == nil || t.Removed || t.Role == Owner {
			return "can't change that member", ""
		}
		if s.On {
			t.Role = CoOwner
		} else {
			t.Role = Member
		}
		return "", ""

	case StepHandover:
		if me == nil || me.Removed || me.Role != Owner {
			return "only the owner can do that", ""
		}
		t := b.Members[s.Member]
		if t == nil || t.Removed {
			return "can't hand over to that member", ""
		}
		t.Role, t.Muted = Owner, false
		me.Role = CoOwner
		b.Owner = s.Member
		return "", ""

	case StepClear:
		if me == nil || me.Removed || me.Role != Owner {
			return "only the owner can clear the table", ""
		}
		b.Notes = map[string]*Note{}
		b.Cleared = true
		return "", ""

	case StepRevert:
		if !canManage(me) {
			return "only the owner or a co-owner can revert the board", ""
		}
		old, err := h.Board(s.To)
		if err != nil || old == nil {
			return "that version isn't available", ""
		}
		// The board's content goes back; membership and keys stay as they are now.
		b.Notes = old.Clone().Notes
		b.Title = old.Title
		return "", ""

	case StepNoteAdd:
		if !canPost(me) {
			return "you can't post in this room", ""
		}
		if s.ID == "" || b.Notes[s.ID] != nil {
			return "bad note id", ""
		}
		text := ""
		if s.Text != nil {
			text = clip(*s.Text, maxText)
		}
		color := s.Color
		if color == "" {
			color = "yellow"
		}
		b.Notes[s.ID] = &Note{ID: s.ID, Author: author, Text: text, Attachments: cleanAttachments(s.Attachments),
			X: s.X, Y: s.Y, Color: clip(color, 20), Created: at, EditedSeq: seq}
		return "", ""

	case StepNoteEdit:
		n := b.Notes[s.ID]
		if n == nil || n.Deleted != 0 {
			return "that note no longer exists", ""
		}
		if n.Author != author {
			return "only the note's author can edit it", ""
		}
		if !canPost(me) {
			return "you can't post in this room", ""
		}
		// V5: the note changed after this draft started: this later commit
		// wins; the replaced text stays in the note's history.
		if base != "" && n.EditedSeq > h.SeqOf(base) {
			replaced = fmt.Sprintf("version %d", n.EditedSeq)
		}
		n.History = append(n.History, Revision{Seq: n.EditedSeq, Text: n.Text, Attachments: n.Attachments, By: n.Author, At: n.Edited})
		if s.Text != nil {
			n.Text = clip(*s.Text, maxText)
		}
		if s.Attachments != nil {
			n.Attachments = cleanAttachments(s.Attachments)
		}
		if s.Color != "" {
			n.Color = clip(s.Color, 20)
		}
		n.Edited, n.EditedSeq = at, seq
		return "", replaced

	case StepNoteRevert:
		n := b.Notes[s.ID]
		if n == nil || n.Deleted != 0 {
			return "that note no longer exists", ""
		}
		if n.Author != author && !canManage(me) {
			return "only the note's author can revert it", ""
		}
		if s.Rev < 0 || s.Rev >= len(n.History) {
			return "that revision doesn't exist", ""
		}
		r := n.History[s.Rev]
		n.History = append(n.History, Revision{Seq: n.EditedSeq, Text: n.Text, Attachments: n.Attachments, By: n.Author, At: n.Edited})
		n.Text, n.Attachments = r.Text, r.Attachments
		n.Edited, n.EditedSeq = at, seq
		return "", ""

	case StepNoteMove:
		n := b.Notes[s.ID]
		if n == nil || n.Deleted != 0 {
			return "that note no longer exists", ""
		}
		if !canPost(me) {
			return "you can't post in this room", ""
		}
		n.X, n.Y = s.X, s.Y
		return "", ""

	case StepNoteDelete:
		n := b.Notes[s.ID]
		if n == nil || n.Deleted != 0 {
			return "that note no longer exists", ""
		}
		if n.Author != author || me == nil || me.Removed {
			return "only the note's author can delete it", ""
		}
		n.Text, n.Attachments, n.Votes, n.Deleted = "", nil, nil, at
		return "", ""

	case StepVote:
		n := b.Notes[s.ID]
		if n == nil || n.Deleted != 0 {
			return "that note no longer exists", ""
		}
		if !canPost(me) {
			return "you can't post in this room", ""
		}
		if n.Votes == nil {
			n.Votes = map[string]string{}
		}
		switch s.Value {
		case "up", "down":
			n.Votes[author] = s.Value
		default:
			delete(n.Votes, author)
		}
		return "", ""

	case StepReplyAdd:
		n := b.Notes[s.ID]
		if n == nil || n.Deleted != 0 {
			return "that note no longer exists", ""
		}
		if !canPost(me) {
			return "you can't post in this room", ""
		}
		for _, r := range n.Replies {
			if r.ID == s.Reply {
				return "bad reply id", ""
			}
		}
		text := ""
		if s.Text != nil {
			text = clip(*s.Text, maxText)
		}
		n.Replies = append(n.Replies, Reply{ID: s.Reply, Author: author, Text: text, Attachments: cleanAttachments(s.Attachments), Created: at})
		return "", ""

	case StepReplyEdit, StepReplyDelete:
		n := b.Notes[s.ID]
		if n == nil {
			return "that note no longer exists", ""
		}
		for i := range n.Replies {
			r := &n.Replies[i]
			if r.ID != s.Reply {
				continue
			}
			if r.Deleted != 0 {
				return "that reply was deleted", ""
			}
			if r.Author != author || me == nil || me.Removed {
				return "only the reply's author can change it", ""
			}
			if s.Type == StepReplyDelete {
				r.Text, r.Attachments, r.Deleted = "", nil, at
			} else {
				if !canPost(me) {
					return "you can't post in this room", ""
				}
				if s.Text != nil {
					r.Text = clip(*s.Text, maxText)
				}
				r.Edited = at
			}
			return "", ""
		}
		return "that reply no longer exists", ""
	}
	return fmt.Sprintf("unknown step %q", s.Type), ""
}

func cleanAttachments(in []Attachment) []Attachment {
	if len(in) == 0 {
		return nil
	}
	if len(in) > 50 {
		in = in[:50]
	}
	out := make([]Attachment, 0, len(in))
	for _, a := range in {
		switch a.Kind {
		case "file", "image", "video", "link":
		default:
			a.Kind = "file"
		}
		a.Name = clip(a.Name, 300)
		a.Type = clip(a.Type, 100)
		if a.Kind == "link" {
			a.URL = clip(a.URL, 2000)
			a.Torrent = ""
		} else {
			a.URL = ""
		}
		out = append(out, a)
	}
	return out
}
