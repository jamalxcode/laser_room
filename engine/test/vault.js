const test = require('brittle')
const fs = require('fs')
const os = require('os')
const path = require('path')
const { Vault } = require('..')

function tmp() { return fs.mkdtempSync(path.join(os.tmpdir(), 'lr-vault-')) }

test('vault opens with the right PIN only', (t) => {
  const dir = tmp()
  const v = new Vault(path.join(dir, 'vault.json'))
  t.exception(() => v.create('123', {}), /at least 6 digits/)
  v.create('482913', { primaryKey: 'abc', rooms: [] })
  t.alike(v.open('482913'), { primaryKey: 'abc', rooms: [] })
  t.exception(() => v.open('000000'), /Wrong PIN \(9 tries left\)/)
  t.alike(v.open('482913').primaryKey, 'abc', 'a right PIN resets the count')
  t.absent(fs.readFileSync(v.file, 'utf8').includes('abc'), 'secrets are not stored in the clear')
})

test('ten wrong PINs in a row wipe the vault and the room data', (t) => {
  const dir = tmp()
  const data = path.join(dir, 'rooms')
  fs.mkdirSync(data)
  fs.writeFileSync(path.join(data, 'x'), 'room data')
  const v = new Vault(path.join(dir, 'vault.json'), { wipe: [data] })
  v.create('482913', { primaryKey: 'abc' })
  for (let i = 1; i <= 9; i++) t.exception(() => v.open('111111'), /tries left/)
  t.exception(() => v.open('111111'), /wiped/)
  t.absent(fs.existsSync(v.file), 'vault gone')
  t.absent(fs.existsSync(data), 'room data gone')
})
