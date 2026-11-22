// The board's rules, applied identically on every member's device.
//
// Every change is an operation appended to its author's own signed log;
// Autobase orders everyone's operations the same way and runs them through
// apply(). The author of an operation is the writer key of the log it came
// from, so nobody can act in someone else's name. Operations that break a
// rule (a muted member posting, editing someone else's note) are ignored on
// every device alike.

const b4a = require('b4a')
const crypto = require('hypercore-crypto')

const MAX_NAME = 40
const MAX_TEXT = 10000

// Member ID number: a one-way hash of the member's writer key in this room.
function memberId(key) {
  return b4a.toString(crypto.hash(key), 'hex').slice(0, 16)
}

function hex(key) {
  return b4a.isBuffer(key) ? b4a.toString(key, 'hex') : key
}

async function get(view, key) {
  const node = await view.get(key)
  return node ? node.value : null
}

async function member(view, key) {
  return get(view, 'member/' + hex(key))
}

function canManage(m) {
  return m && !m.removed && (m.role === 'owner' || m.role === 'coowner')
}

function canPost(m) {
  return m && !m.removed && !m.muted && m.name
}

function cleanText(s, max = MAX_TEXT) {
  return typeof s === 'string' ? s.slice(0, max) : ''
}

function cleanAttachments(list) {
  if (!Array.isArray(list)) return []
  return list.slice(0, 50).map((a) => ({
    kind: ['file', 'image', 'video', 'link'].includes(a.kind) ? a.kind : 'file',
    name: cleanText(a.name, 300),
    url: a.kind === 'link' ? cleanText(a.url, 2000) : null,
    blob: a.blob || null, // reference to the encrypted blob (files, images, videos)
    size: Number.isFinite(a.size) ? a.size : null,
    type: cleanText(a.type, 100)
  }))
}

// apply runs on every device for every operation, in the same order.
async function apply(nodes, view, host) {
  for (const node of nodes) {
    const op = node.value
    if (!op || typeof op !== 'object') continue
    const author = hex(node.from.key)
    const at = typeof op.at === 'number' ? op.at : 0 // author's clock, shown as-is
    await applyOne(op, author, at, view, host)
  }
}

async function applyOne(op, author, at, view, host) {
  const me = await member(view, author)
  const room = (await get(view, 'room')) || {}
  if (room.cleared) return

  switch (op.type) {
    // --- room and membership ---
    case 'room/create': {
      if (room.owner) return // only once, by the first writer
      await view.put('room', { owner: author, title: cleanText(op.title, 200), created: at, cleared: false })
      await view.put('member/' + author, { key: author, id: memberId(b4a.from(author, 'hex')), name: null, role: 'owner', muted: false, removed: false, joined: at })
      return
    }
    case 'invite/add': {
      if (!canManage(me)) return
      await view.put('invite/' + op.id, { id: op.id, publicKey: op.publicKey, reusable: !!op.reusable, by: author, at })
      return
    }
    case 'invite/del': {
      if (!canManage(me) && !(me && !me.removed)) return
      await view.del('invite/' + op.id)
      return
    }
    case 'member/add': {
      // Any member's device can let a newcomer in, but only with a live invite.
      if (!me || me.removed) return
      const inv = await get(view, 'invite/' + op.invite)
      if (!inv) return
      const existing = await member(view, op.key)
      if (existing && existing.removed) return
      if (!existing) {
        await view.put('member/' + op.key, { key: op.key, id: memberId(b4a.from(op.key, 'hex')), name: null, role: 'member', muted: false, removed: false, joined: at, invitedBy: author })
      }
      await host.addWriter(b4a.from(op.key, 'hex'), { indexer: false })
      if (!inv.reusable) await view.del('invite/' + op.invite)
      return
    }
    case 'member/name': {
      if (!me || me.removed) return
      const name = cleanText(op.name, MAX_NAME).trim()
      if (!name) return
      const taken = await get(view, 'name/' + name.toLowerCase())
      if (taken && taken !== author) return // names are unique within the room
      if (me.name) await view.del('name/' + me.name.toLowerCase())
      await view.put('name/' + name.toLowerCase(), author)
      await view.put('member/' + author, { ...me, name })
      return
    }
    case 'member/mute':
    case 'member/unmute': {
      if (!canManage(me)) return
      const target = await member(view, op.key)
      if (!target || target.role === 'owner') return
      await view.put('member/' + op.key, { ...target, muted: op.type === 'member/mute' })
      return
    }
    case 'member/remove': {
      if (!canManage(me)) return
      const target = await member(view, op.key)
      if (!target || target.removed || target.role === 'owner') return
      await view.put('member/' + op.key, { ...target, removed: true, removedAt: at })
      if (target.name) await view.del('name/' + target.name.toLowerCase())
      await host.removeWriter(b4a.from(op.key, 'hex'))
      return
    }
    case 'member/coowner': {
      if (!me || me.removed || me.role !== 'owner') return
      const target = await member(view, op.key)
      if (!target || target.removed || target.role === 'owner') return
      await view.put('member/' + op.key, { ...target, role: op.on ? 'coowner' : 'member' })
      return
    }
    case 'room/handover': {
      if (!me || me.removed || me.role !== 'owner') return
      const target = await member(view, op.key)
      if (!target || target.removed) return
      await view.put('member/' + op.key, { ...target, role: 'owner', muted: false })
      await view.put('member/' + author, { ...me, role: 'coowner' })
      await view.put('room', { ...room, owner: op.key })
      return
    }
    case 'room/clear': {
      if (!me || me.removed || me.role !== 'owner') return
      for await (const { key } of view.createReadStream({ gte: 'note/', lt: 'note0' })) await view.del(key)
      for await (const { key } of view.createReadStream({ gte: 'reply/', lt: 'reply0' })) await view.del(key)
      for await (const { key } of view.createReadStream({ gte: 'vote/', lt: 'vote0' })) await view.del(key)
      await view.put('room', { ...room, cleared: true, clearedAt: at })
      return
    }

    // --- the board ---
    case 'note/add': {
      if (!canPost(me)) return
      if (await get(view, 'note/' + op.id)) return
      await view.put('note/' + op.id, {
        id: op.id, author, text: cleanText(op.text), attachments: cleanAttachments(op.attachments),
        x: Number(op.x) || 0, y: Number(op.y) || 0, color: cleanText(op.color, 20) || 'yellow',
        created: at, edited: null, deleted: null
      })
      return
    }
    case 'note/edit': {
      const note = await get(view, 'note/' + op.id)
      if (!note || note.deleted || note.author !== author || !canPost(me)) return
      await view.put('note/' + op.id, {
        ...note, text: op.text !== undefined ? cleanText(op.text) : note.text,
        attachments: op.attachments !== undefined ? cleanAttachments(op.attachments) : note.attachments,
        color: op.color ? cleanText(op.color, 20) : note.color, edited: at
      })
      return
    }
    case 'note/move': {
      // Anyone (not muted) can tidy the board; moves aren't edits.
      const note = await get(view, 'note/' + op.id)
      if (!note || note.deleted || !canPost(me)) return
      await view.put('note/' + op.id, { ...note, x: Number(op.x) || 0, y: Number(op.y) || 0, movedBy: author })
      return
    }
    case 'note/delete': {
      const note = await get(view, 'note/' + op.id)
      if (!note || note.deleted || note.author !== author || !me || me.removed) return
      // Content goes; a marker with who and when stays.
      await view.put('note/' + op.id, { ...note, text: '', attachments: [], deleted: at })
      return
    }
    case 'vote': {
      const note = await get(view, 'note/' + op.note)
      if (!note || note.deleted || !canPost(me)) return
      const key = 'vote/' + op.note + '/' + author
      if (op.value === 'up' || op.value === 'down') await view.put(key, { note: op.note, by: author, value: op.value, at })
      else await view.del(key)
      return
    }
    case 'reply/add': {
      const note = await get(view, 'note/' + op.note)
      if (!note || note.deleted || !canPost(me)) return
      if (await get(view, 'reply/' + op.note + '/' + op.id)) return
      await view.put('reply/' + op.note + '/' + op.id, {
        id: op.id, note: op.note, author, text: cleanText(op.text),
        attachments: cleanAttachments(op.attachments), created: at, edited: null, deleted: null
      })
      return
    }
    case 'reply/edit': {
      const key = 'reply/' + op.note + '/' + op.id
      const r = await get(view, key)
      if (!r || r.deleted || r.author !== author || !canPost(me)) return
      await view.put(key, { ...r, text: cleanText(op.text), edited: at })
      return
    }
    case 'reply/delete': {
      const key = 'reply/' + op.note + '/' + op.id
      const r = await get(view, key)
      if (!r || r.deleted || r.author !== author || !me || me.removed) return
      await view.put(key, { ...r, text: '', attachments: [], deleted: at })
      return
    }
  }
}

module.exports = { apply, memberId }
