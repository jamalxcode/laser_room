# Laser Room: specification

**Laser Room** is a private work table for invited groups: a whiteboard of signed, timestamped post-it notes,
connections, 👍/👎 and question cards, with any attachment. An invite link from the room's owner is all anyone needs to join; a PIN code protects their
place on their device. The owner is responsible for inviting the right people. When the work is done, the owner clears the table.

Status: **specification agreed, not built yet** (2026-11-22). It replaces the Laser Email and Laser Messenger
plans (their history is in `jamalxcode/lasermail`, `docs/DECISIONS.md`).

## 0. Principle: a voluntary sharing board

Laser Room helps a group put different ideas in front of everyone. **Every action is voluntary**: nobody is
required to answer, vote, react or read anything, and the app never enforces, chases or penalises
participation. If an owner wants something to be compulsory, that is between them and their people, inside
their own organisation, not something the app enforces (owner, 2026-11-22).

## 1. Joining and identity: invite link + PIN (owner, 2026-11-22)

| # | Rule |
|---|---|
| I0 | **Custody:** Apple, Google or anyone else never hold, store or sync any Laser Room key, password or code; the Laser Room app creates and keeps them on the device. |
| I1 | **The invite link is enough to join.** Opening it adds the person to the room. There is no approval step: **the owner is responsible for inviting the right people.** No biometrics are used. |
| I2 | **PIN code:** on first open the person sets a PIN (at least 6 digits). It locks the app's vault on that device, so someone else picking up the device can't post as them. The app asks for it when opened and after 15 idle minutes. (Assumed reading; see open question J1.) |
| I3 | **Identity:** the app creates the member's signing key for each room automatically and keeps it in the vault; its one-way hash is the member's **ID number** in that room. Another room → an unrelated key and number. Nothing identifies the person beyond the alias they choose. |
| I4 | **Aliases:** chosen per room, unique within it (the app asks for another if taken). An alias can be a real name or a made-up one (a fun one is suggested). People reveal who they are, if they want to, outside Laser Room. |
| I5 | **Forgotten PIN:** after 10 wrong tries the vault is wiped on that device; the person re-joins with the invite link under the same alias, and their earlier posts are shown as theirs again once a co-member confirms (or as a new member if not). |
| I6 | **New or second device:** open the invite link there and set a PIN; link it to the first device by scanning a QR code between them so it is the same member, or join as a new member. |
| I7 | **Lost device:** the PIN protects it; the person re-joins on a new device (I6). |

## 2. Workspaces (the work table)

| # | Rule |
|---|---|
| W1 | Workspaces are **private**. The creator (owner) invites people; there is no public listing or sign-up. A collaboration tool, not a public forum. |
| W2 | Joining is **by invite link** (or its QR code), shared by the owner through any channel. The link carries the room key, so whoever has it can join and read: share it with the right people only. |
| W3 | **No removal** of members (owner: 11b). The owner controls membership by choosing carefully whom to invite; each member is responsible for their own content. The owner can clear the whole table (W8) and start a new one without someone. |
| W4 | **Ownership can be handed over** to another member. |
| W5 | **Everything is encrypted** with the room key carried in the invite link; anyone carrying data for the room without the link sees locked data only. Anyone who obtains the invite link can join and read, so it should be shared privately. |
| W6 | **Sync:** whoever opens the board is brought up to date with the latest version and saves it on their device. Members' copies sync with each other whenever they are online together (owner: 9). |
| W7 | **No host server:** rooms sync peer to peer (section 4a). The owner's computer and volunteering members' computers stay online as permanent peers (S6), keeping an encrypted copy; they can't read anything they don't have the board password for. |
| W8 | **Clearing the table:** the owner can delete the whole board when the work is done ("files shared, decisions made, table cleared"). The cloud copy is deleted and every member's copy is wiped the next time it syncs. A modified app could ignore the wipe; the app says so. |

## 3. The whiteboard: notes, connections, questions

| # | Rule |
|---|---|
| P1 | Each board is a **whiteboard canvas** (zoom and pan) holding **post-it notes** placed anywhere. Notes can be coloured, resized and grouped by placing them near each other (owner, 2026-11-22). |
| P1a | **Connections:** any member can draw a line between two notes, like a mind map (optionally with a short label and an arrow). Connections are signed and timestamped like notes; only their author removes them. |
| P1b | **👍 / 👎 on every note:** one per member per note, changeable; counts and who voted are shown (read receipts and votes are visible, G17). |
| P1c | **Question cards:** when input would help, a member posts a multiple-choice question (single or multiple answers). Answering is **optional**; there are no required answers and no reminders. Each member can answer once (changeable until closed); results show live; the author closes it whenever they like (an optional "closes on" date is only shown, never enforced), and a closed question is locked and signed. |
| P1d | **Replies:** each note has its own reply thread (one level). |
| P2 | Every note, connection, vote, answer, reply, edit, move and deletion is **timestamped and signed** by the author's alias/ID number (owner: 6b). Signatures are checked by every member's app. |
| P3 | Others see a note **only once it is submitted**; there's no live typing. |
| P4 | **Editing and moving:** only the author edits a note's content; each edit shows the alias and time. Any member can move notes on the canvas to organise it (moves are recorded but not shown as edits). |
| P5 | **Deleting:** only the author deletes their own note, connection or reply. The content is removed; a marker remains with the alias and time (owner: 7, 10). |
| P6 | **Attachments on notes:** anything, with no size limit: links, images, PDFs, files, torrent files and magnet links (owner: 11, 14b, 2026-11-22). Images and PDFs preview on the note; links show their address (no automatic previews that would contact the site). |
| P7 | **No tasks** (owner, 2026-11-22). |
| P8 | **Outcomes:** a note can be marked an Outcome: a record of what the group arrived at (often after a question card or 👍/👎), closed by its author or the owner, locked and signed, and superseded only by a newer outcome that links to it. An **outcome log** lists them. It records, it doesn't bind: what an outcome means for people is up to the group. |
| P9 | **Design review:** comments can be pinned to a spot on an image or PDF page; a new version of a file keeps earlier versions and their comments viewable. |
| P10 | **Discussion quality:** a note's author or the owner can mark its thread **resolved** with a one-line conclusion; a **"since you were last here"** view shows what's new; unsent notes are kept as **drafts** on the device; **templates** (meeting agenda, design brief, decision record, kickoff) start common boards. |
| P11 | **Clearing the table:** first the app produces a signed **close-out pack** (outcome log, closed questions with results, final file versions, the board as an image) that members keep; then the board is cleared and the workspace can be reused, empty, with the same members. |
| P12 | **Convenience:** joining by an invitation QR code or link carrying the board password (the newcomer scans, picks an alias, waits for the owner's tap); keyboard shortcuts and drag-and-drop of files onto the canvas; notifications by default only when someone @mentions you, the rest in an optional daily digest. No reminders, no "awaiting your answer", no overdue states. |

## 3a. Gap decisions (owner, 2026-11-22)

| # | Decision |
|---|---|
| G1 | ~~Owner approval to join~~: dropped (2026-11-22); the invite link is enough (I1). |
| G2 | The owner can **mute** a member (they can read, not post); no removal (W3). |
| G3 | If the owner disappears, the board **continues with no owner**: existing members read and post; owner-only actions (approvals, alias confirmations, re-admissions, clearing) stop. |
| G4 | ~~Biometric vs PIN~~: dropped; everyone uses an app PIN (I2). |
| G5 | The host also stamps each post with its own clock; both times are shown if they differ. |
| G6 | Aliases are unique within a room; the app asks for another if one is taken (I4). |
| G7 | One identity per device; shared devices aren't supported. |
| G8 | Any member can export a signed copy of posts that anyone can verify. |
| G9 | Phones download attachments when opened; desktops can keep everything. |
| G10 | The host keeps a board until it is cleared, or until 90 days without anyone opening it (members are warned first). |
| G11 | Before the table is cleared, members get a warning and a few days to save what they need. |
| G12 | Each owner hosts their own board (W7). |
| G13 | Phone notifications through Apple/Google push, carrying only an empty "wake up" (never content, passwords or codes). |
| G14 | A workspace can hold several boards (topics). |
| G15 | Local search of posts and attachment names. |
| G16 | @alias mentions and simple reactions. |
| G17 | Read receipts are shown (as information only; nobody is chased for not reading). |
| G18 | English and Arabic (right-to-left) from the start. |
| G19 | Lost or stolen device: the PIN protects it (I7). |
| G20 | Screenshots and copying can't be prevented; the app says so. |
| D1 | iPhone app distributed through the App Store / TestFlight (Apple distributes; it never holds passwords or codes). |
| D2 | App technology: open (Flutter screens over a JavaScript Hypercore core is the current proposal). |
| D3 | Hosting: the owner's own computer (host mode) or any independent host they choose (G12). |

## 4. Architecture

```
Laser Room installed app (Flutter: Mac, Windows, iPhone, Android; desktop can run host mode)
├─ Identity     invite link (room key) + app PIN → app's own vault on the device
│               → signing key and ID number per room
├─ Board data   an append-only log of signed events (post, reply, edit, delete, invite, handover,
│               clear) merged the same way on every device (CRDT), so copies always converge
├─ Encryption   room key from the invite link; every event and attachment encrypted before it
│               leaves the device; each event signed with the author's key
├─ Storage      boards and attachments in the app's own encrypted storage on each device
├─ Sync         Hypercore: per-member signed logs merged by Autobase, peers found via
│               Hyperswarm/HyperDHT; backup: BitTorrent updatable torrents (BEP 46/44)
└─ Permanent    the owner's and volunteers' computers stay online as peers, keep the encrypted
   peers        copy and stamp times (S6, G5, G10)
```

## 4a. Sync: Hypercore first, updatable torrents as backup (owner, 2026-11-22)

| # | Decision |
|---|---|
| S1 | **Primary: Hypercore** (Holepunch's open-source peer-to-peer stack). Each member writes to their own signed, append-only log (Hypercore); the room combines all approved members' logs into one board (Autobase, multi-writer); peers find each other and get through home routers with Hyperswarm/HyperDHT (hole punching), with no server. |
| S2 | **Backup: updatable torrents.** The same encrypted logs are also published as BitTorrent updatable torrents (BEP 46/44): each member's key announces "my latest batch is X" on the BitTorrent DHT, so the room still syncs if Hypercore's network can't be reached. |
| S3 | **Invite link:** a `laser-room:` link / QR code carrying the room key. Lookups on both networks use a value derived from it, so outsiders can't find the room. |
| S4 | **Files** are encrypted and shared peer to peer (Hyperdrive/Hyperblobs; BitTorrent for the backup path); the note carries the reference and key. |
| S5 | **IP addresses:** members of a room can see each other's; outsiders can't (owner: 2a). |
| S6 | **Availability:** the owner's computer and any member who volunteers theirs stay online as permanent peers, keeping the room reachable when phones sleep (owner: 3c). This replaces the host relay (W7); host-time stamps (G5) come from whichever permanent peer is online. |
| S7 | **Limits:** updates reach others in seconds; phones sync when the app is open; everything on the shared networks is encrypted and unlinkable to the room without its link. |

## 4b. Open questions

- **J1. Whose PIN?** (a) the person sets their own PIN to lock the app on their device (assumed in I2); (b) the
  owner gives a PIN with the invite, sent separately, so a forwarded link alone isn't enough; (c) both.

## 5. Settled questions

- **O1 → superseded:** no passkeys and no syncing through Apple or Google; the app holds its own secrets (I0, I1a).
- **O2 → superseded by G12:** each owner hosts their own board (W7).
- **O3 → resolved:** Laser Room is an installed app, so browser storage limits don't apply.
