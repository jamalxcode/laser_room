# Laser Room: specification

**Laser Room** is a private sharing board for groups. The owner of a room sends invite links to the people
they choose; each person picks a PIN and a name, and everyone puts ideas on a shared whiteboard of post-it notes
with 👍/👎, replies and attachments. There are no servers and no accounts: rooms sync peer to peer between the
members' installed apps. Taking part is always voluntary. When the work is done, the owner clears the table.

Status: **specification; not built yet.** Version of this document: 2026-11-22. Why each rule is the way it is:
`docs/DECISIONS.md`.

## 1. Principles

1. **Voluntary.** Nobody is required to read, vote, reply or post; the app never chases, reminds or penalises.
   If an owner wants something to be compulsory, that is between them and their people, outside the app.
2. **The owner is responsible for who is in the room.** An invite link is all anyone needs to join, so the owner
   shares links only with the right people, and can mute or remove people afterwards.
3. **Nobody else holds anything.** No servers, no accounts, no Apple or Google services: the app creates and keeps
   every key and code on the person's device, and rooms sync directly between members' devices, encrypted.
4. **Simple first.** The first version does a few things well (section 8).

## 2. Rooms and invitations

| # | Rule |
|---|---|
| R1 | Anyone with the app can **create a room** and becomes its **owner**. |
| R2 | The owner **invites** people with an **invite link** (also shown as a QR code) sent through any channel they like (message, email, in person). Per invite, the owner chooses a **reusable** link or a **single-use** link. |
| R3 | **Opening an invite link is enough to join.** There is no approval step. One existing member's app (the owner's always-on computer, or anyone online) has to be online at that moment to let the newcomer's app in, automatically; if nobody is online, joining waits until someone is. |
| R4 | The link carries the room's key; anyone who obtains it can join and read. It should be shared privately. |
| R5 | The owner can name **co-owners**, who can do everything the owner can except remove the owner. |
| R6 | The owner can **hand over** ownership to another member. |
| R7 | **Mute:** the owner or a co-owner can mute a member; they can still read but can't post, vote or reply. Unmute restores them. |
| R8 | **Remove:** the owner or a co-owner can remove a member. The app creates a new room key and shares it with everyone else, so the removed person gets nothing new (they keep what they had already seen). Outstanding reusable invite links stop working; the owner sends new ones as needed. |
| R9 | **Clearing the table:** the owner can clear a room when the work is done. Every member's copy is wiped the next time their app syncs, and the room ends. Copies on devices that never come back online can't be wiped; the app says so. |
| R10 | **No push notifications.** Members see what's new when they open the app. When the owner wants people to look, they tell them through the same channel they used for the invite. |

## 3. Joining and identity

| # | Rule |
|---|---|
| J1 | **First open of an invite link:** install the app (if needed), pick a **PIN** (6 digits or more), pick a **name** for this room. Done. |
| J2 | **The PIN** locks the app on that device. It is asked when the app opens and after 15 idle minutes. Where the device has a security chip (Secure Enclave, Android Keystore, TPM), the vault key is kept there so the PIN can't be guessed by copying the app's data. |
| J3 | **10 wrong PINs in a row:** the app wipes that person's rooms and keys from the device. To come back they need a new invite from the room's owner (or a co-owner), and they join as a new member. |
| J4 | **Forgotten PIN:** the same: a new invite, a new member. |
| J5 | **Identity:** for each room the app creates a signing key automatically and keeps it in its vault. Its one-way hash is the member's **ID number** in that room; another room gets an unrelated key and number. Nobody, including the owner, learns anything about a person beyond the name they chose. |
| J6 | **Names** are chosen per room and must be unique within it (the app asks for another if one is taken). A name can be a real name or a made-up one. People reveal who they are, if they want to, outside Laser Room. |
| J7 | **A second device** (phone + laptop): link it from the first device by scanning a QR code shown there, so both are the same member. Otherwise, joining from a new device with an invite link makes a new member. |
| J8 | **Lost device:** the PIN and J3 protect it; the person asks the owner for a new invite. |

## 4. The board (first version)

| # | Rule |
|---|---|
| B1 | Each room has a **whiteboard canvas** (zoom and pan) of **post-it notes** that members place anywhere, in a few colours. |
| B2 | **A note** holds text and, optionally, **attachments**: files, images, videos and links (URLs). Images and videos play or preview on the note; files show their name and size and open in the device's own apps; links show their address and open in the browser (no automatic previews, which would contact the site). No size limit; phones download attachments when opened. |
| B3 | **👍 / 👎** on every note: one per member, changeable; counts and names of who voted are shown. |
| B4 | **Replies:** each note has its own reply thread (one level). Replies can carry attachments too. |
| B5 | Others see a note or reply **once it is posted**; there's no live typing. |
| B6 | **Signed and timestamped:** every note, reply, vote, edit, move and deletion carries its author's name, ID number and time, and is signed by their key; every app checks the signatures. |
| B7 | **Editing:** only the author edits their note or reply; each edit shows the name and time. |
| B8 | **Moving:** any member can move notes to organise the board; moves are recorded but aren't shown as edits. |
| B9 | **Deleting:** only the author deletes their own note or reply. The content and attachments are removed; a marker stays with the name and time of the deletion. |
| B10 | **Since you were last here:** opening a room shows what's new and lets you jump to it. |
| B11 | **Read receipts** are shown on notes (information only). |
| B12 | Screenshots and copying can't be prevented; the app says so. |

## 5. Sync and availability

| # | Rule |
|---|---|
| S1 | **Hypercore** (Holepunch's open-source peer-to-peer stack) only. Each member writes to their own signed, append-only log; the room merges all members' logs into one board (Autobase, multi-writer), so every copy converges on the same board. |
| S2 | Members find each other and connect through home routers with **Hyperswarm / HyperDHT**, without servers. The room is found by a value derived from its key, so people without an invite link can't find it. |
| S3 | **Everything is encrypted** with the room key before it leaves a device; attachments too (shared peer to peer with Hyperblobs). |
| S4 | Members of a room can see each other's IP addresses; outsiders can't. |
| S5 | **Always-on members:** the owner's computer, and any member who volunteers theirs, can stay online to keep the room reachable when phones are asleep. Each sets a storage limit and sees what uses space. |
| S6 | Updates reach other online members within seconds; phones catch up when the app is opened. |

## 6. Platforms and technology

| # | Rule |
|---|---|
| T1 | **Installed apps** for Mac, Windows, iPhone and Android (everyone installs the app; there is no browser version). **Mac and iPhone first**, then Android and Windows. The iPhone app is distributed through the App Store / TestFlight (Apple distributes it but never holds any key or code). |
| T2 | **The engine** (rooms, invitations, PIN vault, signing, the board, sync) is JavaScript on Hypercore, running in Holepunch's Bare runtime embedded in the app (bare-kit); **the screens** are Flutter, talking to the engine over a local message channel. |
| T3 | Languages: English first; Arabic (right-to-left) planned. |

## 7. What the app never does

- Collect biometrics, phone numbers, emails or contacts.
- Use servers, accounts, analytics, push services or Apple/Google keychains.
- Read rooms on anyone's behalf: only members with the room key can.
- Force anyone to answer, vote or read.

## 8. First version and later

**First version:** rooms with invite links (reusable or single-use) and PINs; post-it notes with 👍/👎, replies,
and attachments (files, images, videos, links); edit, move, delete markers; mute and remove; co-owners;
clearing the table; "since you were last here"; peer-to-peer sync with always-on members.

**Later, in order unless the owner says otherwise:** connections between notes (mind map); multiple-choice
question cards; outcomes and an outcome log; pinning comments on images and PDFs, with file versions; templates;
a close-out pack before clearing; Arabic; fun touches (suggested names, avatars, presence, tidy layout, sync
indicator).
