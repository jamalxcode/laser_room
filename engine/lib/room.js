// A Laser Room: an Autobase (one signed log per member, merged into one
// board) with its view in a Hyperbee, synced over Hyperswarm, joined through
// blind-pairing invites. Modelled on Holepunch's Autopass.

const Autobase = require('autobase')
const Hyperbee = require('hyperbee')
const Hyperswarm = require('hyperswarm')
const BlindPairing = require('blind-pairing')
const ReadyResource = require('ready-resource')
const b4a = require('b4a')
const z32 = require('z32')
const crypto = require('hypercore-crypto')
const { apply, memberId } = require('./board')

const LINK_PREFIX = 'laser-room://join/'

function inviteLink(invite) {
  return LINK_PREFIX + z32.encode(invite)
}

function parseLink(link) {
  const s = String(link).trim()
  const code = s.startsWith(LINK_PREFIX) ? s.slice(LINK_PREFIX.length) : s
  return z32.decode(code.split(/[?#]/)[0])
}

function newId() {
  return b4a.toString(crypto.randomBytes(8), 'hex')
}

function json(value) {
  return value
}

class Room extends ReadyResource {
  // opts: { key, encryptionKey, bootstrap, swarm, title, replicate }
  constructor(store, opts = {}) {
    super()
    this.store = store
    this.bootstrap = opts.bootstrap || null
    this.swarm = opts.swarm || null
    this.ownsSwarm = !opts.swarm
    this.replicateOn = opts.replicate !== false
    this.title = opts.title || ''
    this.pairing = null
    this.member = null
    this.clock = opts.clock || (() => Date.now())

    this.base = new Autobase(store, opts.key || null, {
      encrypt: true,
      encryptionKey: opts.encryptionKey || null,
      valueEncoding: 'json',
      open: (s) => new Hyperbee(s.get('view'), { keyEncoding: 'utf-8', valueEncoding: 'json', extension: false }),
      apply
    })
    this.base.on('update', () => {
      if (!this.base._interrupting) this.emit('update')
    })
  }

  async _open() {
    await this.base.ready()
    // The first writer of a new room creates it and becomes its owner.
    if (this.base.writable && !(await this.view.get('room'))) {
      await this._append({ type: 'room/create', title: this.title })
    }
    if (this.replicateOn) await this._replicate()
  }

  async _close() {
    if (this.member) await this.member.close()
    if (this.pairing) await this.pairing.close()
    if (this.swarm && this.ownsSwarm) await this.swarm.destroy()
    await this.base.close()
  }

  get view() { return this.base.view }
  get key() { return this.base.key }
  get discoveryKey() { return this.base.discoveryKey }
  get encryptionKey() { return this.base.encryptionKey }
  get writerKey() { return this.base.local.key }
  get writable() { return this.base.writable }
  get myId() { return memberId(this.writerKey) }

  async _append(op) {
    await this.base.append({ ...op, at: this.clock() })
    await this.base.update()
  }

  async _replicate() {
    if (!this.swarm) {
      this.swarm = new Hyperswarm({ keyPair: await this.store.createKeyPair('hyperswarm'), bootstrap: this.bootstrap })
      this.ownsSwarm = true
    }
    this.swarm.on('connection', (conn) => this.store.replicate(conn))
    this.pairing = new BlindPairing(this.swarm)
    // Any member online lets newcomers in, if their invite is live.
    this.member = this.pairing.addMember({
      discoveryKey: this.base.discoveryKey,
      onadd: (candidate) => this._onCandidate(candidate)
    })
    this.swarm.join(this.base.discoveryKey)
  }

  async _onCandidate(candidate) {
    const id = b4a.toString(candidate.inviteId, 'hex')
    const inv = await this._get('invite/' + id)
    if (!inv || !this.writable) return
    candidate.open(b4a.from(inv.publicKey, 'hex'))
    const { key } = JSON.parse(b4a.toString(candidate.userData))
    await this._append({ type: 'member/add', key, invite: id })
    candidate.confirm({ key: this.base.key, encryptionKey: this.base.encryptionKey })
  }

  async _get(key) {
    const node = await this.view.get(key)
    return node ? node.value : null
  }

  // --- invitations ---

  // Returns a laser-room:// invite link. reusable: many people can use it;
  // otherwise it works once.
  async createInvite({ reusable = false } = {}) {
    const { id, invite, publicKey } = BlindPairing.createInvite(this.base.key)
    await this._append({ type: 'invite/add', id: b4a.toString(id, 'hex'), publicKey: b4a.toString(publicKey, 'hex'), reusable })
    if (this.member) await this.member.flushed()
    return inviteLink(invite)
  }

  async deleteInvite(id) { await this._append({ type: 'invite/del', id }) }

  // Joins a room with an invite link. Resolves with the Room once a member's
  // device has let us in and we can write. name: our name in the room.
  static async join(store, link, { name, bootstrap, swarm } = {}) {
    const invite = parseLink(link)
    const own = !swarm
    swarm = swarm || new Hyperswarm({ keyPair: await store.createKeyPair('hyperswarm'), bootstrap })
    swarm.on('connection', (conn) => store.replicate(conn))
    const pairing = new BlindPairing(swarm)
    const local = Autobase.getLocalCore(store)
    await local.ready()
    const key = b4a.toString(local.key, 'hex')
    await local.close()

    const result = await new Promise((resolve, reject) => {
      const candidate = pairing.addCandidate({
        invite,
        userData: b4a.from(JSON.stringify({ key })),
        onadd: (res) => resolve(res)
      })
      candidate.on && candidate.on('error', reject)
    })
    await pairing.close()
    const room = new Room(store, { key: result.key, encryptionKey: result.encryptionKey, bootstrap, swarm })
    room.ownsSwarm = own
    await room.ready()
    if (!room.writable) await new Promise((resolve) => {
      const check = () => { if (room.writable) { room.off('update', check); resolve() } }
      room.on('update', check)
      check()
    })
    if (name) await room.setName(name)
    return room
  }

  // --- membership ---
  async setName(name) {
    await this._append({ type: 'member/name', name })
    const me = await this._get('member/' + b4a.toString(this.writerKey, 'hex'))
    if (!me || me.name !== String(name).trim().slice(0, 40)) throw new Error('That name is taken in this room')
  }
  mute(key) { return this._append({ type: 'member/mute', key: hexKey(key) }) }
  unmute(key) { return this._append({ type: 'member/unmute', key: hexKey(key) }) }
  remove(key) { return this._append({ type: 'member/remove', key: hexKey(key) }) }
  setCoOwner(key, on = true) { return this._append({ type: 'member/coowner', key: hexKey(key), on }) }
  handover(key) { return this._append({ type: 'room/handover', key: hexKey(key) }) }
  clear() { return this._append({ type: 'room/clear' }) }

  // --- the board ---
  async addNote({ text = '', attachments = [], x = 0, y = 0, color = 'yellow' } = {}) {
    const id = newId()
    await this._append({ type: 'note/add', id, text, attachments, x, y, color })
    return id
  }
  editNote(id, changes) { return this._append({ type: 'note/edit', id, ...changes }) }
  moveNote(id, x, y) { return this._append({ type: 'note/move', id, x, y }) }
  deleteNote(id) { return this._append({ type: 'note/delete', id }) }
  vote(note, value) { return this._append({ type: 'vote', note, value }) } // 'up' | 'down' | null
  async reply(note, { text = '', attachments = [] } = {}) {
    const id = newId()
    await this._append({ type: 'reply/add', note, id, text, attachments })
    return id
  }
  editReply(note, id, text) { return this._append({ type: 'reply/edit', note, id, text }) }
  deleteReply(note, id) { return this._append({ type: 'reply/delete', note, id }) }

  // --- reading ---
  async info() { return this._get('room') }

  async members() {
    const out = []
    for await (const { value } of this.view.createReadStream({ gte: 'member/', lt: 'member0' })) out.push(value)
    return out
  }

  async notes() {
    const notes = []
    for await (const { value } of this.view.createReadStream({ gte: 'note/', lt: 'note0' })) notes.push(value)
    for (const n of notes) {
      n.votes = { up: [], down: [] }
      for await (const { value } of this.view.createReadStream({ gte: 'vote/' + n.id + '/', lt: 'vote/' + n.id + '0' })) n.votes[value.value].push(value.by)
      n.replies = []
      for await (const { value } of this.view.createReadStream({ gte: 'reply/' + n.id + '/', lt: 'reply/' + n.id + '0' })) n.replies.push(value)
      n.replies.sort((a, b) => a.created - b.created)
    }
    return notes.sort((a, b) => a.created - b.created)
  }
}

function hexKey(key) {
  return b4a.isBuffer(key) ? b4a.toString(key, 'hex') : key
}

module.exports = { Room, inviteLink, parseLink, json }
