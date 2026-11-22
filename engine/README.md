# Laser Room engine

The part of Laser Room that does the work, in JavaScript on Holepunch's Hypercore stack. The Flutter app will
embed it (Bare runtime, via bare-kit) and talk to it over a local message channel.

- `lib/room.js`: a room = an Autobase (one signed log per member, merged into one board) with a Hyperbee view,
  synced with Hyperswarm; invite links via blind-pairing (modelled on Holepunch's Autopass).
- `lib/board.js`: the board's rules, applied the same way on every device (who may post, edit, move, delete,
  vote, reply, mute, remove, clear).
- `lib/vault.js`: the PIN vault (Argon2id + secretbox); 10 wrong PINs wipe it and the person's rooms.

```bash
npm install
npm test        # several simulated members on a private test network
```

Done: rooms, reusable and single-use invite links, unique names, ID numbers, notes with attachment records,
👍/👎, replies, edit/move/delete markers, mute, remove, co-owners, handover, clearing, the PIN vault.

Next: re-keying the room when someone is removed (today a removed member can no longer write, but still holds
the key); attachment files themselves (Hyperblobs); "since you were last here"; then the Flutter app.
