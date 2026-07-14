import assert from 'node:assert/strict'
import path from 'node:path'
import test from 'node:test'
import { pathToFileURL } from 'node:url'

const root = path.resolve(new URL('..', import.meta.url).pathname)
const sessionUrl = pathToFileURL(path.join(root, 'store/session.js')).href
let importSerial = 0

async function loadSessionModule() {
  importSerial += 1
  return import(`${sessionUrl}?case=${importSerial}`)
}

function installUniStorage(initial = {}) {
  const storage = new Map(Object.entries(initial))
  globalThis.uni = {
    getStorageSync(key) {
      return storage.get(key) || ''
    },
    setStorageSync(key, value) {
      storage.set(key, value)
    },
    removeStorageSync(key) {
      storage.delete(key)
    },
  }
  return storage
}

test('session stores merchant id as trimmed string and clears empty values', async () => {
  const storage = installUniStorage({ wplink_merchant_id: 70088564334919780 })
  const { getMerchantId, getSession, saveMerchantId } = await loadSessionModule()

  assert.equal(getMerchantId(), '70088564334919780')
  assert.equal(getSession().merchantId, '70088564334919780')

  saveMerchantId(' 70088564334919781 ')
  assert.equal(storage.get('wplink_merchant_id'), '70088564334919781')

  saveMerchantId('')
  assert.equal(storage.has('wplink_merchant_id'), false)
})
