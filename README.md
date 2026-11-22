# Laser Room

A private sharing board for groups, passed around like a document. Everyone in a room can read it; anyone can
work on their own draft; changes are **committed like Git**: a commit locks the room for a moment, lands on top
of the latest version and becomes the new version, which spreads to everyone over **BitTorrent**. Every version
is kept and can be reverted to. The owner sends invite links; each person picks a PIN and a name. No servers,
no accounts. Taking part is always voluntary. When the work is done, the owner clears the table.

## Status

- **Engine (Go): versions, commits and the PIN vault are built and tested.** Signed versions checked on every
  device, drafts re-played on top of the latest version, "your edit was replaced" notices, reverts (whole board
  or one note), mute, remove (with a key change), co-owners, handover, clearing, unique names with look-alike
  protection, and the PIN vault (10 wrong tries wipe).
- **Next:** BitTorrent sync (versions and attachments as torrents, a DHT "latest version" pointer with
  compare-and-swap), invite links and joining, then the Mac and iPhone app (Flutter).

## Layout

```
docs/SPEC.md           what Laser Room does (rules, numbered) and the build status
docs/DECISIONS.md      why, and what was dropped along the way
docs/CONVERSATION.md   how the design came about, in order
engine/                the engine (Go): board rules, version chain, PIN vault
```

## Run the tests

```bash
cd engine
go test -race ./...
```
