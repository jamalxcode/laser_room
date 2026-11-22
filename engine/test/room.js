const test = require('brittle')
const Corestore = require('corestore')
const createTestnet = require('hyperdht/testnet')
const fs = require('fs')
const os = require('os')
const path = require('path')
const b4a = require('b4a')
const { Room } = require('..')

function store(t) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'lr-room-'))
  const s = new Corestore(dir)
  t.teardown(() => s.close().catch(() => {}), { order: 100 }) // after the rooms
  return s
}

async function until(t, what, fn, ms = 20000) {
  const end = Date.now() + ms
  while (Date.now() < end) {
    if (await fn()) return
    await new Promise((r) => setTimeout(r, 100))
  }
  t.fail('timed out waiting for ' + what)
}

const hex = (k) => b4a.toString(k, 'hex')

test('a room end to end: invite, names, notes, votes, replies, edits, moves, deletes, mute, remove', async (t) => {
  const testnet = await createTestnet(10, t.teardown)
  const bootstrap = testnet.bootstrap

  // Owner creates a room and posts a note.
  const owner = new Room(store(t), { bootstrap, title: 'Spring launch' })
  t.teardown(() => owner.close())
  await owner.ready()
  await owner.setName('Jamal')
  t.is((await owner.info()).title, 'Spring launch')
  const n1 = await owner.addNote({ text: 'Launch in April?', x: 10, y: 20, color: 'pink' })

  // A reusable invite link; two people join with it.
  const link = await owner.createInvite({ reusable: true })
  t.ok(link.startsWith('laser-room://join/'), 'invite link')

  const ann = await Room.join(store(t), link, { name: 'Ann', bootstrap })
  t.teardown(() => ann.close())
  const bob = await Room.join(store(t), link, { name: 'Bob', bootstrap })
  t.teardown(() => bob.close())
  t.ok(ann.writable && bob.writable, 'both can post')

  await until(t, 'Ann sees the note', async () => (await ann.notes()).some((n) => n.id === n1))

  // Names are unique within the room.
  await t.exception(() => bob.setName('ann'), /taken/)

  // Votes and replies, with attachments.
  await ann.vote(n1, 'up')
  await bob.vote(n1, 'down')
  await bob.vote(n1, 'up') // changeable
  const r1 = await ann.reply(n1, { text: 'Yes, mid-April', attachments: [{ kind: 'link', name: 'calendar', url: 'https://example.org/cal' }] })
  const n2 = await bob.addNote({ text: 'Mockups', attachments: [{ kind: 'image', name: 'hero.png', blob: 'b1', size: 1234, type: 'image/png' }] })

  await until(t, 'owner sees everything', async () => {
    const notes = await owner.notes()
    const a = notes.find((n) => n.id === n1)
    return a && a.votes.up.length === 2 && a.replies.length === 1 && notes.some((n) => n.id === n2)
  })
  const seen = (await owner.notes()).find((n) => n.id === n1)
  t.is(seen.votes.down.length, 0, 'Bob changed his vote')
  t.is(seen.replies[0].attachments[0].url, 'https://example.org/cal')

  // Only the author edits or deletes; anyone moves.
  await ann.editNote(n1, { text: 'hijacked' }) // not hers: ignored everywhere
  await ann.moveNote(n1, 300, 400) // moving is allowed
  await owner.editNote(n1, { text: 'Launch on 15 April' })
  await ann.deleteReply(n1, r1)
  await until(t, 'Bob sees edits, move and delete marker', async () => {
    const n = (await bob.notes()).find((x) => x.id === n1)
    return n && n.text === 'Launch on 15 April' && n.x === 300 && n.replies[0] && n.replies[0].deleted
  })
  const bn = (await bob.notes()).find((x) => x.id === n1)
  t.ok(bn.edited, 'edit time shown')
  t.is(bn.replies[0].text, '', 'deleted reply content removed')
  t.is(bn.replies[0].author, hex(ann.writerKey), 'marker keeps who deleted it')

  // Mute: Bob can read but not post.
  await owner.mute(bob.writerKey)
  await until(t, 'Bob is muted on his own device', async () => (await bob.members()).find((m) => m.key === hex(bob.writerKey)).muted)
  await bob.addNote({ text: 'spam' })
  // Wait until the owner has Bob's post before unmuting, so the unmute comes after it.
  const bobLog = owner.store.get({ key: bob.writerKey })
  await bobLog.ready()
  await until(t, 'owner has Bob\'s post', async () => { await owner.base.update(); return bobLog.length >= bob.base.local.length })
  await owner.unmute(bob.writerKey)
  await owner.addNote({ text: 'marker' })
  await until(t, 'Ann sees the marker note', async () => (await ann.notes()).some((n) => n.text === 'marker'))
  t.absent((await ann.notes()).some((n) => n.text === 'spam'), 'muted post ignored everywhere')

  // Remove: Ann is out; her later posts are ignored.
  await owner.remove(ann.writerKey)
  await until(t, 'Bob sees Ann removed', async () => (await bob.members()).find((m) => m.key === hex(ann.writerKey)).removed)
  const members = await owner.members()
  t.is(members.find((m) => m.role === 'owner').name, 'Jamal')
  t.is(members.length, 3)
  t.ok(members.every((m) => /^[0-9a-f]{16}$/.test(m.id)), 'every member has an ID number')
})

test('single-use invites work once; co-owners manage; clearing the table', async (t) => {
  const testnet = await createTestnet(10, t.teardown)
  const bootstrap = testnet.bootstrap
  const owner = new Room(store(t), { bootstrap, title: 'Design review' })
  t.teardown(() => owner.close())
  await owner.ready()
  await owner.setName('Owner')

  const once = await owner.createInvite({ reusable: false })
  const cara = await Room.join(store(t), once, { name: 'Cara', bootstrap })
  t.teardown(() => cara.close())
  await until(t, 'invite used up', async () => {
    let n = 0
    for await (const _ of owner.view.createReadStream({ gte: 'invite/', lt: 'invite0' })) n++ // eslint-disable-line
    return n === 0
  })

  // Co-owner can mute; a plain member can't.
  const link = await owner.createInvite({ reusable: true })
  const dan = await Room.join(store(t), link, { name: 'Dan', bootstrap })
  t.teardown(() => dan.close())
  await dan.mute(cara.writerKey) // not allowed
  await owner.setCoOwner(cara.writerKey)
  await until(t, 'Cara is co-owner', async () => (await cara.members()).find((m) => m.key === hex(cara.writerKey)).role === 'coowner')
  await cara.mute(dan.writerKey)
  await until(t, 'Dan muted by the co-owner', async () => (await owner.members()).find((m) => m.key === hex(dan.writerKey)).muted)

  await cara.addNote({ text: 'final logo' })
  await until(t, 'owner sees the note', async () => (await owner.notes()).length === 1)
  await cara.clear() // only the owner can clear
  await owner.clear()
  await until(t, 'table cleared on Cara\'s copy', async () => (await cara.info()).cleared)
  t.is((await cara.notes()).length, 0, 'notes gone')
})
