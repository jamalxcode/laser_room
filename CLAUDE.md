# CLAUDE.md

Guidance for Claude Code in this repository.

## Project

**Laser Room**: a private, voluntary sharing board for invited groups. An owner sends invite links; each person
picks a PIN and a name; everyone posts post-it notes with 👍/👎, replies and attachments on a shared whiteboard.
No servers, no accounts: rooms sync peer to peer (Hypercore), encrypted. The owner can mute, remove and finally
clear the table.

## Key documents (read before changing behaviour)

- `docs/SPEC.md`: every rule, numbered (R rooms, J joining, B board, S sync, T platforms), plus the build
  status (section 9). A behaviour change updates it in the same commit.
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
5. **Simple first:** the first version is notes, 👍/👎, replies and attachments (SPEC section 8). Don't add
   later features (connections, question cards, outcomes, templates…) unless asked.

## Layout

```
engine/lib/room.js    Room: Autobase (one signed log per member) + Hyperbee view + Hyperswarm;
                      invites via blind-pairing (modelled on Holepunch's Autopass)
engine/lib/board.js   the rules, applied in Autobase's apply() identically on every device
engine/lib/vault.js   PIN vault: Argon2id + secretbox; wipe after 10 wrong PINs
engine/test/          brittle tests: several members on a private hyperdht testnet
.github/workflows/ci.yml   engine tests on Linux and macOS
```

The Flutter app (Mac and iPhone first) will embed the engine with Holepunch's Bare runtime (bare-kit) and talk
to it over a local message channel. It isn't built yet.

## Commands

```bash
cd engine && npm install && npm test
node node_modules/brittle/cmd.js test/room.js   # one file (brittle's default bin wants the Bare runtime)
```

## Engine conventions

- Every change is an operation `{ type, ..., at }` appended with `Room._append`; its author is the writer key of
  the log it came from (never trust an author field in the operation).
- All permission checks live in `board.js` `applyOne`. An operation that breaks a rule is ignored (no throw), so
  every device ends up with the same board.
- View keys: `room`, `member/<writerKeyHex>`, `name/<lowercased>`, `invite/<idHex>`, `note/<id>`,
  `reply/<noteId>/<id>`, `vote/<noteId>/<writerKeyHex>`.
- Deletes keep a marker (author and time) and drop the content.
- Tests use `hyperdht/testnet` with **10** nodes (3 is too few for blind-pairing), and close stores after rooms
  (`t.teardown(..., { order: 100 })`).
- Operations sent at the same moment without seeing each other can be ordered either way (the same on every
  device); tests that depend on order must wait until the other side has received the earlier operation.

## Related repositories

- `jamalxcode/lasermail`: the earlier Laser Email (macOS email client, Go + SwiftUI, released to 0.1.6) and the
  dropped Laser Messenger design. Its CI, release and updater patterns can be reused for the app.
- `jamalxcode/laserdot-go`: the owner's other app; source of the release-proof and updater design.
