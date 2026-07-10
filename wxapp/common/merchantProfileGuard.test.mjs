import assert from 'node:assert/strict'
import path from 'node:path'
import test from 'node:test'
import { pathToFileURL } from 'node:url'

const root = path.resolve(new URL('..', import.meta.url).pathname)
const guardUrl = pathToFileURL(path.join(root, 'common/merchantProfileGuard.js')).href
let importSerial = 0

async function loadGuardModule() {
  importSerial += 1
  return import(`${guardUrl}?case=${importSerial}`)
}

function installUniMock({ merchantId = '', confirm = true } = {}) {
  let modalOptions = null
  let navigateOptions = null
  globalThis.uni = {
    getStorageSync(key) {
      return key === 'wplink_merchant_id' ? merchantId : ''
    },
    showModal(options) {
      modalOptions = options
      options.success?.({ confirm, cancel: !confirm })
    },
    navigateTo(options) {
      navigateOptions = options
    },
  }
  return {
    get modalOptions() {
      return modalOptions
    },
    get navigateOptions() {
      return navigateOptions
    },
  }
}

test('merchant profile guard allows existing merchant profile without modal', async () => {
  const uniMock = installUniMock({ merchantId: 'merchant-1' })
  const { ensureMerchantProfileReady } = await loadGuardModule()

  const ready = await ensureMerchantProfileReady()

  assert.equal(ready, true)
  assert.equal(uniMock.modalOptions, null)
  assert.equal(uniMock.navigateOptions, null)
})

test('merchant profile guard shows concise prompt and opens profile when confirmed', async () => {
  const uniMock = installUniMock({ merchantId: '', confirm: true })
  const { ensureMerchantProfileReady } = await loadGuardModule()

  const ready = await ensureMerchantProfileReady()

  assert.equal(ready, false)
  assert.equal(uniMock.modalOptions.title, '完善资料')
  assert.equal(uniMock.modalOptions.content, '需要先完善发布者资料。')
  assert.equal(uniMock.modalOptions.confirmText, '去完善')
  assert.equal(uniMock.modalOptions.cancelText, '取消')
  assert.deepEqual(uniMock.navigateOptions, { url: '/pages/merchant/profile' })
})

test('merchant profile guard keeps current page when prompt is canceled', async () => {
  const uniMock = installUniMock({ merchantId: '', confirm: false })
  const { ensureMerchantProfileReady } = await loadGuardModule()

  const ready = await ensureMerchantProfileReady()

  assert.equal(ready, false)
  assert.equal(uniMock.modalOptions.title, '完善资料')
  assert.equal(uniMock.navigateOptions, null)
})
