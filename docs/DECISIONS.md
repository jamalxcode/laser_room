# Decisions and how we got here

The owner's decisions behind `SPEC.md`, newest first, with the reasons and what was dropped. The conversation
itself, in order, is in `CONVERSATION.md`.

## Engine built on Holepunch's Autopass pattern

- **Decision:** the engine follows Holepunch's Autopass (a shared store joined by invite): an Autobase per room
  (one signed log per member, merged into one view), a Hyperbee view, Hyperswarm for connections, and
  blind-pairing for invite links. Proven parts rather than new protocols.
- **Authorship:** every operation's author is the writer key of the log it came from, so signatures are
  Hypercore's own; the ID number is a one-way hash of that key.
- **Rules run everywhere:** permission checks (who may edit, move, mute, remove, clear) run inside Autobase's
  apply, identically on every device, so a modified app can't break them for others.
- **Concurrent actions:** operations made at the same moment without seeing each other are put in one agreed
  order; e.g. a muted member's post and the owner's unmute sent at the same time may land either way, the same
  on every device.
- **Learned:** blind-pairing needs enough DHT nodes; tests use a 10-node private test network (3 was too few).
- **Joining needs one member online** (R3): pairing is completed by a member's app, automatically.
- **Open:** re-keying on removal; attachment files (Hyperblobs).

## Technology and platforms

- **Decision:** Hypercore engine in JavaScript (Bare runtime, embedded with bare-kit) + Flutter screens; Mac and
  iPhone first, then Android and Windows.
- **Considered:** all JavaScript with React Native screens (one language, less native feel); a Go core (no solid
  Hypercore implementation in Go).

## 2026-11-22: spec rewritten cleanly

After many design rounds the spec had piled up contradictions (host servers, biometrics, approvals that no
longer existed), so it was rewritten from the current decisions. This file keeps the history.

## Joining: invite link + PIN, no biometrics

- **Decision:** an invite link is enough to join; the invited person picks their own PIN; the owner is
  responsible for inviting the right people. 10 wrong PINs in a row wipe the person's rooms from the device and
  they need a new invite (as a new member). A forgotten PIN means a new invite too.
- **Dropped:** biometric identity (Face ID / fingerprint producing an ID number). It required installed apps,
  careful handling so Apple and Google never held any secret, and fallbacks for devices without sensors. The
  owner chose simplicity: the owner's choice of invitees is the trust.
- **Dropped:** owner approval of each newcomer; aliases confirmed by the owner.

## Removing and muting

- **Decision:** the owner (and co-owners) can mute or remove members. Removal re-keys the room so the removed
  person gets nothing new.
- **Earlier:** no removal at all, with care in inviting as the only control. With approval gone, a forwarded
  link would have meant permanent access, so removal came back.

## Sync: Hypercore only

- **Decision:** Hypercore (Autobase multi-writer, Hyperswarm/HyperDHT) for sync and files; everyone installs the
  app.
- **Considered:** BitTorrent updatable torrents (BEP 46/44) as primary or backup (dropped: two networking stacks
  double the work); an owner-hosted relay/host server (dropped: home hosting is hard to reach, and the owner
  wanted no servers); a browser version through a relay (dropped: everyone installs the app).
- **Accepted:** members of a room see each other's IP addresses.

## No push notifications

- **Decision:** no push. Waking a sleeping iPhone needs Apple's push service and our own developer credentials
  on some machine (a server in all but name). Instead the owner tells people about updates through the same
  channel they used for the invite.

## First version

- **Decision:** post-it notes with 👍/👎, replies, and attachments (files, images, videos, links), plus the rooms,
  invitations, PINs, mute/remove, co-owners, clearing and sync needed to use them.
- **Later:** connections (mind map), question cards, outcomes, design-review pins and versions, templates,
  close-out pack, Arabic, fun touches. No tasks (dropped by the owner).

## Principle: voluntary sharing board

- Every action is voluntary; the app never enforces or chases participation. Anything compulsory is between the
  owner and their people.

## Before Laser Room

The project began as **Laser Email** (a private macOS email client, `jamalxcode/lasermail`, released up to
0.1.6), then briefly **Laser Messenger** (owner-less peer-to-peer messenger with a credit economy). Email was
dropped because of Gmail/iCloud sign-in hurdles and home mail servers being blocked; the messenger because its
economy and ledger were too complex. Both designs are in `jamalxcode/lasermail`'s history.
