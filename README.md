# Laser Room

A private sharing board for groups. The room's owner sends invite links to the people they choose; each person
picks a PIN and a name, and everyone puts ideas on a shared whiteboard of post-it notes with 👍/👎, replies and
attachments (files, images, videos, links). No servers, no accounts: rooms sync peer to peer between members'
installed apps, encrypted. Taking part is always voluntary. When the work is done, the owner clears the table.

## Status

- **Engine: built and tested.** Rooms, reusable and single-use invite links, unique names and ID numbers, notes,
  👍/👎, replies, edit/move/delete markers, mute, remove, co-owners, handover, clearing the table, and the PIN
  vault, on Holepunch's Hypercore stack. Tests run several simulated members on a private network (CI on Linux
  and macOS).
- **Next:** re-keying a room when someone is removed, attachment files, "since you were last here", then the
  Mac and iPhone app (Flutter).

## Layout

```
docs/SPEC.md           what Laser Room does (rules, numbered) and the build status
docs/DECISIONS.md      why, and what was dropped along the way
docs/CONVERSATION.md   how the design came about, in order
engine/                the engine (JavaScript, Hypercore): rooms, invites, PIN vault, board rules, sync
```

## Run the tests

```bash
cd engine
npm install
npm test
```
