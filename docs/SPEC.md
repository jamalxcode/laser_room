# Laser Room: specification

**Laser Room** is a private sharing board for groups. The board is a **document passed around**: everyone in the
room can read it; anyone can edit their own draft; changes are **committed** like Git: a commit locks the room
for a moment, is placed on top of the latest version, and becomes the new version, which spreads to everyone
over **BitTorrent**. Every version is kept and can be reverted to. The owner of a room sends invite links;
each person picks a PIN and a name. No servers, no accounts. Taking part is always voluntary. When the work is
done, the owner clears the table.

Status (2026-11-22): **redesigned** from a Hypercore multi-writer room to a versioned document over BitTorrent.
The Go engine is being built (section 9). Why: `docs/DECISIONS.md`. How the design came about:
`docs/CONVERSATION.md`.

## 1. Principles

1. **Voluntary.** Nobody is required to read, vote, reply or post; the app never chases, reminds or penalises.
2. **The owner is responsible for who is in the room.** An invite link is all anyone needs to join; the owner
   shares links only with the right people, and can mute or remove people afterwards.
3. **Nobody else holds anything.** No servers, no accounts, no Apple or Google services: the app creates and keeps
   every key on the person's device; versions travel between members' devices over BitTorrent, encrypted.
4. **Simple first.** The first version does a few things well (section 8).

## 2. Rooms and invitations

| # | Rule |
|---|---|
| R1 | Anyone with the app can **create a room** and becomes its **owner**. |
| R2 | The owner **invites** people with an **invite link** (also a QR code) sent through any channel. Per invite: **reusable** or **single-use**. |
| R3 | **Opening an invite link is enough to join**: it carries the room's keys, so the app fetches the latest version from whoever has it. No approval and nobody has to be online to let you in (someone holding the room's versions must be reachable to download from). |
| R4 | Anyone who obtains a link can join and read; it should be shared privately. |
| R5 | The owner can name **co-owners**, who can do everything the owner can except remove the owner. |
| R6 | The owner can **hand over** ownership to another member. |
| R7 | **Mute:** the owner or a co-owner can mute a member; commits from a muted member are rejected by every app. |
| R8 | **Remove:** the owner or a co-owner removes a member with a commit that also **changes the room keys**; the new keys are locked separately for each remaining member, so the removed person can't read any later version. Outstanding invite links stop working; new ones are needed. |
| R9 | **Clearing the table:** the owner clears a room with a final commit; every app then deletes all the room's versions and attachments from the device. Devices that never come online again keep their copy; the app says so. |
| R10 | **No push notifications.** Members see what's new when they open the app; the owner tells people through the invite channel when they should look. |

## 3. Joining and identity

| # | Rule |
|---|---|
| J1 | First open of an invite link: install the app (if needed), pick a **PIN** (6+ digits), pick a **name** for this room. |
| J2 | **The PIN** locks the app's vault, which holds every key (room keys and the person's own keys). Asked when the app opens and after 15 idle minutes; on devices with a security chip the vault key is kept there too. Nothing secret is stored outside the vault. |
| J3 | **10 wrong PINs in a row** wipe the person's rooms and keys from the device; they need a new invite and join as a new member. |
| J4 | **Forgotten PIN:** the same. |
| J5 | **Identity:** for each room the app creates a personal key pair; its one-way hash is the member's **ID number** in that room. A different key, network identity and number per room, so rooms can't be linked by key. |
| J6 | **Names** are unique within a room, can be real or made up, and can't imitate another member with look-alike characters. Roles show as badges (owner, co-owner) that a name can't fake. |
| J7 | **Second device:** link it from the first by QR code (later); otherwise a new device joins as a new member. |
| J8 | **Lost device:** the PIN and J3 protect it; ask the owner for a new invite. |

## 4. The board

| # | Rule |
|---|---|
| B1 | A **whiteboard canvas** (zoom and pan) of **post-it notes** placed anywhere, in a few colours. |
| B2 | **A note** holds text and optional **attachments**: files, images, videos, links. Images and videos preview; files open in the device's apps; links open in the browser (no automatic previews). No size limit; phones download attachments when opened. |
| B3 | **👍 / 👎** on every note: one per member, changeable; counts and names shown. |
| B4 | **Replies:** one-level thread per note; replies can carry attachments. |
| B5 | **Editing is a draft until committed** (section 5); others see changes once committed. |
| B6 | Every change is part of a **signed commit** carrying its author's name, ID number and time (the author's own clock, labelled as such; order follows the version chain, not the clock). |
| B7 | **Editing:** only the author edits their note or reply. |
| B8 | **Moving:** any member can move notes. |
| B9 | **Deleting:** only the author deletes their note or reply; it disappears from the board with a marker (who, when). Earlier versions still hold it until the table is cleared or old versions are dropped; the app says "removed from the board". |
| B10 | **Since you were last here:** what's new since the version you last saw. |
| B11 | **Seen by N:** each note shows how many members' apps have downloaded the version containing it (counted from torrent peers, nothing extra shared). |
| B12 | Screenshots and copying can't be prevented; the app says so. |

## 5. Versions and commits (like Git)

| # | Rule |
|---|---|
| V1 | The room's content is a **chain of versions**. Each version names its parent, lists the **changes** in it, carries the resulting board, and is **signed by its author**. Every app checks the whole chain. |
| V2 | **Drafts:** a member edits freely on their own copy; nothing is locked while they work. Changes are recorded as steps (add, edit, move, delete note; reply; 👍/👎; mute; remove…). |
| V3 | **Commit:** the member presses **Commit** (with a summary of their changes and an optional short note). The app pulls the latest version, re-plays the steps on top of it, and publishes the result as the new version **only if the latest is still the one it pulled**. The room is locked only for that moment. |
| V4 | **Someone committed first:** the publish is rejected; the app pulls the newer version, re-plays the steps on top and tries again, automatically. |
| V5 | **Same note edited in two drafts:** the later commit wins; the earlier text stays in that note's history and its author is told their edit was replaced. A step that no longer applies (e.g. editing a note that was deleted) is skipped and its author is told. |
| V6 | **👍/👎 and replies** are commits too, made instantly when tapped. |
| V7 | **Every version is kept.** **Revert:** the owner and co-owners can revert the whole board to an earlier version; any author can revert one of their own notes to an earlier version of itself. A revert is a new version: nothing is lost, and a revert can be reverted. |
| V8 | **Rare double publish:** with no server, two commits can very occasionally both be published on the same parent. Every app picks the same one (deterministic rule) and the other author's app re-plays theirs on top, so all copies converge. |
| V9 | Only members (not muted, not removed) can commit; owner-only steps (mute, remove, co-owners, handover, clear, whole-board revert) are accepted only from the owner or co-owners. |

## 6. Sync over BitTorrent

| # | Rule |
|---|---|
| S1 | **Each version is a torrent:** the encrypted version package; **attachments are separate torrents**, shared across versions so they're never sent twice. |
| S2 | **The "latest version" pointer** is a BitTorrent DHT mutable item (BEP 44/46) signed with the room's pointer key, with a sequence number; publishing uses compare-and-swap on that number (V3). |
| S3 | The pointer is found by a value derived from the room key (BEP 46 salt), so people without the link can't find the room. |
| S4 | **Encrypted:** versions and attachments are encrypted with the room key before sharing; anyone outside the room sees only encrypted pieces. |
| S5 | Members of a room can see each other's IP addresses; outsiders can't. |
| S6 | **Always-on members:** the owner's computer and any volunteer can stay online seeding the room's versions and attachments. Each sets a storage limit; when full, the biggest and oldest attachments are dropped first (they stay available from others). |
| S7 | New versions reach online members within seconds to a minute; phones catch up when the app opens. |

## 7. Platforms and technology

| # | Rule |
|---|---|
| T1 | Installed apps for Mac, Windows, iPhone and Android (no browser version). **Mac and iPhone first.** iPhone through the App Store / TestFlight. |
| T2 | **Engine in Go** (versions, commits, signatures, vault, BitTorrent via anacrolix/torrent and anacrolix/dht for BEP 44/46); **screens in Flutter**, calling the engine through a native bridge. |
| T3 | English first; Arabic (right-to-left) planned. |

## 8. What the app never does

- Collect biometrics, phone numbers, emails or contacts.
- Use servers, accounts, analytics, push services or Apple/Google keychains.
- Read rooms on anyone's behalf: only members with the room keys can.
- Force anyone to answer, vote or read.

## 9. First version and later

**First version:** rooms, invite links and PINs; post-it notes with 👍/👎, replies and attachments; drafts and
commits; version history and revert; mute, remove (with new keys), co-owners, handover, clearing; "since you
were last here"; sync over BitTorrent with always-on members.

**Later:** connections between notes (mind map); multiple-choice question cards; outcomes; design-review pins;
templates; close-out pack; second-device linking; Arabic; fun touches.

## 10. Build status

| Area | Status |
|---|---|
| Versions, signed commits, chain checks, replay on top, conflicts, reverts (V1–V9) | In progress (Go, `engine/`) |
| PIN vault (J2–J4) | Next (port from the Hypercore engine) |
| BitTorrent: version and attachment torrents, DHT pointer with compare-and-swap (S1–S3) | Next |
| Invite links, joining, key change on removal (R2–R8) | Next |
| Flutter app for Mac and iPhone | After the engine |
| Hypercore engine | Retired (in git history before this change) |
