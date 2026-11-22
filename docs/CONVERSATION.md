# How the design came about

The owner's requests and answers, in order, as condensed notes (2026-11-21 to 2026-11-22). Decisions and
reasons are in `DECISIONS.md`; the resulting rules in `SPEC.md`.

## 1. Laser Email (2026-11-21)

- **Ask:** "an email client for macOS using Go, focused on privacy, handling many emails from different Gmail
  and Apple accounts, called Laser Email", then "follow the style and GUI of an Apple macOS email client".
- **Built** in `jamalxcode/lasermail`: a Go engine with a SwiftUI shell in a native macOS mail layout. It had a
  unified inbox, IMAP IDLE, local search, tracker blocking, Keychain storage, a signed DMG and Check for Updates.
  Released 0.1.0 to 0.1.6.
- **Friction:** Gmail refused normal passwords (needs an app password or OAuth); iCloud needs app-specific
  passwords; Google OAuth setup in Cloud Console was long ("I don't want extra steps").
- **Explored:** own mail server on a Mac mini at home. Residential IPs are blocklisted, port 25 is often
  blocked and reverse DNS isn't available, so it would need a small relay server.

## 2. Laser Messenger (2026-11-21)

- **Ask:** "make it an instant messaging app and ignore all this email nonsense."
- **Owner's answers:** anyone can use it, Bitcoin-style keys as identity, peer to peer with no servers, super
  nodes, torrent tech for large files, text first. Each person has a permissioned private address plus a public
  name that anyone can message (accepted contacts only for the private one). Credits earned by running a super
  node, relaying, storing and seeding, spent on names, first messages and storage.
- **Critique given:** the owner-less ledger amounts to a new cryptocurrency. Phone-only users would be locked
  out, and phones need push and super nodes anyway.
- **Dropped** for a new design.

## 3. Laser Room (2026-11-22)

- **Ask:** "a whiteboard where different groups work on projects." People identified by a biometric-derived
  number (hashed, one-way), shown by an alias, identity hidden unless they reveal it.
- **Biometrics, iterated:**
  1. Biometrics can't be hashed like passwords; options were device unlock, a fuzzy extractor, or uniqueness.
  2. Owner: use what the device offers and extract an ID; unique within the group only; no Apple or Google
     involvement.
  3. Owner: biometric input only, then the app holds every password (no Apple/Google custody). That required
     an installed app.
  4. Owner, finally: "**forget biometrics**": an invite link is enough, plus a PIN the invited user picks.
     The owner is responsible for inviting the right people.
- **Board:** posts and replies first, then "remove tasks". Post-it notes with 👍/👎, mind-map connections,
  multiple-choice questions, any attachment (links, images, PDFs, torrents).
- **Principle:** "the action the user is taking is voluntary, not mandatory… a sharing board that helps convey
  different ideas for everyone."
- **Sync:** owner asked whether the room could be a torrent link that syncs on update. Answer: yes, with
  BitTorrent's updatable torrents. Owner chose **Hypercore** first, torrents as backup, then **Hypercore only**.
  Members see each other's IP addresses (accepted). The owner and volunteers keep rooms online.
- **Hosting:** each owner hosts their own room. Later superseded by pure peer to peer (no servers).
- **Critique rounds** (owner answered each as multiple choice):
  - workspaces private and invite-only
  - no live typing
  - only the author edits or deletes; deletes leave a marker
  - mute or remove members; co-owners; owner can hand over
  - 10 wrong PINs: out, needs a new invite
  - forgotten PIN: new member
  - no push: the owner tells people through the invite channel
  - everyone installs the app (no browser version)
- **First version:** post-it notes with 👍/👎, replies, and attachments (files, images, videos, URLs).
- **Technology:** JavaScript Hypercore engine on Bare, with Flutter screens. Mac and iPhone first.
- **Repo:** owner created `jamalxcode/laser_room`. The spec moved here and was rewritten cleanly.

## 4. Engine built (2026-11-22)

- Built `engine/` on the Autopass pattern: Autobase, Hyperbee, Hyperswarm and blind-pairing.
  - rooms, reusable and single-use invite links
  - unique names and ID numbers
  - notes, 👍/👎, replies, edit/move/delete markers
  - mute, remove, co-owners, handover, clearing
  - the PIN vault
- Tests simulate several members on a private network. CI passes on Linux and macOS.
- Told the owner: joining needs one member's app online. Removal doesn't re-key the room yet.
- **Next:** re-keying on removal, attachment files, "since you were last here", then the Flutter app.
