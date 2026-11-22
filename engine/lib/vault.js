// The PIN vault: everything secret the app keeps (the Corestore primary key
// that all signing keys derive from, and each room's key) is stored only in
// this file, encrypted with a key derived from the person's PIN (Argon2id).
// After 10 wrong PINs in a row the vault and the person's room data are
// wiped; they need a new invite to come back (spec J3).
//
// The Mac/iPhone app additionally wraps the vault key with the Secure
// Enclave (spec J2); this module is the portable part.

const fs = require('fs')
const path = require('path')
const sodium = require('sodium-universal')
const b4a = require('b4a')

const MAX_TRIES = 10
const MIN_PIN = 6

class WrongPin extends Error {
  constructor(left) {
    super(left > 0 ? `Wrong PIN (${left} tries left)` : 'Too many wrong PINs: this device was wiped')
    this.code = left > 0 ? 'WRONG_PIN' : 'WIPED'
    this.triesLeft = left
  }
}

function deriveKey(pin, salt) {
  const key = b4a.allocUnsafe(sodium.crypto_secretbox_KEYBYTES)
  sodium.crypto_pwhash(key, b4a.from(String(pin)), salt,
    sodium.crypto_pwhash_OPSLIMIT_MODERATE, sodium.crypto_pwhash_MEMLIMIT_MODERATE, sodium.crypto_pwhash_ALG_ARGON2ID13)
  return key
}

function seal(pin, secrets) {
  const salt = b4a.allocUnsafe(sodium.crypto_pwhash_SALTBYTES)
  const nonce = b4a.allocUnsafe(sodium.crypto_secretbox_NONCEBYTES)
  sodium.randombytes_buf(salt)
  sodium.randombytes_buf(nonce)
  const msg = b4a.from(JSON.stringify(secrets))
  const box = b4a.allocUnsafe(msg.length + sodium.crypto_secretbox_MACBYTES)
  sodium.crypto_secretbox_easy(box, msg, nonce, deriveKey(pin, salt))
  return { v: 1, salt: b4a.toString(salt, 'hex'), nonce: b4a.toString(nonce, 'hex'), box: b4a.toString(box, 'hex'), failures: 0 }
}

class Vault {
  // file: the vault file; wipe: paths removed after too many wrong PINs.
  constructor(file, { wipe = [] } = {}) {
    this.file = file
    this.wipePaths = wipe
  }

  exists() { return fs.existsSync(this.file) }

  create(pin, secrets) {
    if (!/^\d+$/.test(String(pin)) || String(pin).length < MIN_PIN) throw new Error(`The PIN must be at least ${MIN_PIN} digits`)
    fs.mkdirSync(path.dirname(this.file), { recursive: true })
    this._write(seal(pin, secrets))
    return secrets
  }

  open(pin) {
    const data = JSON.parse(fs.readFileSync(this.file, 'utf8'))
    const box = b4a.from(data.box, 'hex')
    const msg = b4a.allocUnsafe(box.length - sodium.crypto_secretbox_MACBYTES)
    const ok = sodium.crypto_secretbox_open_easy(msg, box, b4a.from(data.nonce, 'hex'), deriveKey(pin, b4a.from(data.salt, 'hex')))
    if (!ok) {
      data.failures = (data.failures || 0) + 1
      const left = MAX_TRIES - data.failures
      if (left <= 0) {
        this.wipe()
        throw new WrongPin(0)
      }
      this._write(data)
      throw new WrongPin(left)
    }
    if (data.failures) { data.failures = 0; this._write(data) }
    return JSON.parse(b4a.toString(msg))
  }

  // Saves new secrets under the same PIN (e.g. after joining a room).
  update(pin, secrets) {
    this.open(pin) // checks the PIN
    this._write(seal(pin, secrets))
  }

  wipe() {
    fs.rmSync(this.file, { force: true })
    for (const p of this.wipePaths) fs.rmSync(p, { recursive: true, force: true })
  }

  _write(data) {
    const tmp = this.file + '.tmp'
    fs.writeFileSync(tmp, JSON.stringify(data), { mode: 0o600 })
    fs.renameSync(tmp, this.file)
  }
}

module.exports = { Vault, WrongPin, MAX_TRIES }
