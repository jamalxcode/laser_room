# CLAUDE.md

Guidance for Claude Code in this repository.

## Project

**Laser Room**: a private, voluntary sharing board for invited groups. The room is a document passed around: everyone reads it;
anyone edits their own draft; commits work like Git (pull the latest, re-play the draft on top, publish; a brief
lock only while committing); every version is kept and can be reverted to; versions spread over BitTorrent. An
owner sends invite links; each person picks a PIN and a name. No servers, no accounts. The owner can mute,
remove and finally clear the table.

## Key documents (read before changing behaviour)

- `docs/SPEC.md`: every rule, numbered (R rooms, J joining, B board, V versions and commits, S sync,
  T platforms), plus the build status (section 10). A behaviour change updates it in the same commit.
- `docs/DECISIONS.md`: why each rule is the way it is, and what was dropped. Add an entry for each significant
  decision.
- `docs/CONVERSATION.md`: the owner's requests and answers in order. Append to it when the owner decides
  something new.

When the spec and the owner's latest instruction disagree, follow the owner and update the spec; when two
documents disagree, flag it rather than silently picking one.

## Owner's principles (don't break these)

1. **Voluntary:** the app never requires, chases or reminds anyone to read, vote, reply or post.
2. **The owner is responsible for who's in the room:** an invite link is enough to join; no approval step.
3. **Nobody else holds anything:** no servers, accounts, analytics, push services or Apple/Google keychains.
   The app creates and keeps every key on the device. (Apple's App Store distributes the iPhone app; that's
   accepted.)
4. **No biometrics.** A PIN the person picks protects their place; 10 wrong in a row wipe the device's rooms.
5. **Simple first:** the first version is notes, 👍/👎, replies and attachments, with commits and versions
   (SPEC section 9). Don't add
   later features (connections, question cards, outcomes, templates…) unless asked.

## Layout

```
engine/board.go       the board and its rules: applying a member's steps, identically on every device
engine/chain.go       the version chain: signed versions, commit (re-play a draft on the head), Add (check a
                      received version: signature, parent, steps produce the claimed board), head choice
engine/room.go        Room (id, secret, chain): invite links, removal = move to a new room with invites locked
                      per remaining member (sealed boxes), snapshots for newcomers
engine/drafts.go      drafts saved on the device, encrypted
engine/vault.go       PIN vault: Argon2id + secretbox; wipe after 10 wrong PINs
engine/*_test.go      tests
.github/workflows/ci.yml   gofmt, vet, tests with the race detector, on Linux and macOS
```

Next: BitTorrent (anacrolix/torrent + anacrolix/dht): one swarm per room carrying change sets as an extension
message; snapshots for newcomers; attachments as torrents; each member's own "my latest version" DHT entry
(BEP 44, their own key; no shared pointer key); always-on members republish. A minimal TestFlight build to test
App Store acceptance. Then the Flutter app (Mac and iPhone first) calling the Go engine through a native bridge.

## Commands

```bash
cd engine && go test -race ./...
gofmt -l . && go vet ./...
```

## Engine conventions

- A **version** = parent hash, seq (parent's + 1), author (hex Ed25519 per-room key), the author's clock, the
  draft's base, a message, the **steps**, the resulting **board hash**, and a signature over the rest.
- **All rules live in `board.go` `applyStep`.** A step that breaks a rule is skipped with a reason (never an
  error), so the author's app and every receiving app produce the same board and the same hash.
- Commits always re-play on the current head; if publishing fails (someone committed first), add their version
  and commit the same draft again.
- Head choice (V8): longest chain, then lowest hash; losing versions' authors re-commit on top (`Orphans`).
- "Replaced edit" (V5) compares the note's `EditedSeq` with the draft base's seq.
- Whole-board revert restores notes and title only; membership and keys stay as they are now.
- Removing a member **moves the room** (`Room.Remove`): a `room/continue` first version in a new room carries
  the document without them; the old room's `member/remove` step must carry a sealed invite for every remaining
  member (checked in `applyStep`), and a moved room takes no more commits.
- Every member's join/create step carries their receiving key (`BoxKey`); use `Join(kp, name)`.
- Newcomers start from a snapshot signed by their inviter (`JoinFromSnapshot`).

## Related repositories

- The retired JavaScript Hypercore engine is in this repo's history (commit 944f3eb).
- `jamalxcode/lasermail`: the earlier Laser Email (macOS email client, Go + SwiftUI, released to 0.1.6) and the
  dropped Laser Messenger design. Its CI, release and updater patterns can be reused for the app.
- `jamalxcode/laserdot-go`: the owner's other app; source of the release-proof and updater design.
