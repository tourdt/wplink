import assert from 'node:assert/strict'
import test from 'node:test'

import { createResourceDetailAuxiliaryLoader } from './resourceDetailAuxiliary.js'
import { buildResourceShareRenderContext } from './resourceShareCoverRenderer.js'

function createDeferred() {
  let resolve
  const promise = new Promise((resolvePromise) => {
    resolve = resolvePromise
  })
  return { promise, resolve }
}

async function flushTasks() {
  await Promise.resolve()
  await Promise.resolve()
}

function createHarness() {
  let merchantProfile = {}
  let resource = {}
  const merchantRequests = new Map()
  const snapshots = []
  const loader = createResourceDetailAuxiliaryLoader({
    getFavoriteState: async () => ({ favorited: false }),
    getMerchantProfile(merchantId) {
      const request = createDeferred()
      merchantRequests.set(merchantId, request)
      return request.promise
    },
    hasAuthToken: () => false,
    listRelatedResources: async () => ({ items: [] }),
    setFavorited() {},
    setMerchantProfile(profile) {
      merchantProfile = profile
    },
    setRelatedResources() {},
    setShareImageUrl() {},
  })

  function capture(context) {
    snapshots.push(buildResourceShareRenderContext(context, resource, merchantProfile))
  }

  function run(context, merchantId) {
    return loader.run(context, {
      initializeSharing: capture,
      isOwnResource: true,
      merchantId,
      onMerchantProfileLoaded: capture,
      recordResourceDetailView: async () => {},
    })
  }

  return {
    getMerchantProfile: () => merchantProfile,
    loader,
    merchantRequests,
    run,
    setResource(value) {
      resource = value
    },
    snapshots,
  }
}

test('B initial share snapshot uses its merchant brief after A profile was loaded', async () => {
  const harness = createHarness()
  harness.setResource({ id: 'resource-a', merchant: { id: 'merchant-a', name: 'A 简介' } })
  const loadA = harness.loader.begin('resource-a')
  const runA = harness.run(loadA, 'merchant-a')
  await flushTasks()
  harness.merchantRequests.get('merchant-a').resolve({ id: 'merchant-a', name: 'A 完整资料' })
  await runA

  harness.setResource({ id: 'resource-b', merchant: { id: 'merchant-b', name: 'B 简介' } })
  const loadB = harness.loader.begin('resource-b')
  const runB = harness.run(loadB, 'merchant-b')
  await flushTasks()

  const firstBSnapshot = harness.snapshots.find((item) => item.resourceId === 'resource-b')
  assert.deepEqual(firstBSnapshot.merchant, { id: 'merchant-b', name: 'B 简介' })

  harness.merchantRequests.get('merchant-b').resolve({ id: 'merchant-b', name: 'B 完整资料' })
  await runB
})

test('B merchant profile completion reschedules a snapshot using only B profile', async () => {
  const harness = createHarness()
  harness.setResource({ id: 'resource-b', merchant: { id: 'merchant-b', name: 'B 简介' } })
  const loadB = harness.loader.begin('resource-b')
  const runB = harness.run(loadB, 'merchant-b')
  await flushTasks()

  harness.merchantRequests.get('merchant-b').resolve({ id: 'merchant-b', name: 'B 完整资料' })
  await runB

  const bSnapshots = harness.snapshots.filter((item) => item.resourceId === 'resource-b')
  assert.deepEqual(bSnapshots.map((item) => item.merchant), [
    { id: 'merchant-b', name: 'B 简介' },
    { id: 'merchant-b', name: 'B 完整资料' },
  ])
})

test('stale A merchant profile neither updates nor reschedules B snapshot', async () => {
  const harness = createHarness()
  harness.setResource({ id: 'resource-a', merchant: { id: 'merchant-a', name: 'A 简介' } })
  const loadA = harness.loader.begin('resource-a')
  const runA = harness.run(loadA, 'merchant-a')
  await flushTasks()

  harness.setResource({ id: 'resource-b', merchant: { id: 'merchant-b', name: 'B 简介' } })
  const loadB = harness.loader.begin('resource-b')
  const runB = harness.run(loadB, 'merchant-b')
  await flushTasks()

  harness.merchantRequests.get('merchant-a').resolve({ id: 'merchant-a', name: 'A 完整资料' })
  await runA
  assert.deepEqual(harness.getMerchantProfile(), {})
  assert.deepEqual(
    harness.snapshots.filter((item) => item.resourceId === 'resource-b').map((item) => item.merchant),
    [{ id: 'merchant-b', name: 'B 简介' }],
  )

  harness.merchantRequests.get('merchant-b').resolve({ id: 'merchant-b', name: 'B 完整资料' })
  await runB
  assert.deepEqual(
    harness.snapshots.filter((item) => item.resourceId === 'resource-b').at(-1).merchant,
    { id: 'merchant-b', name: 'B 完整资料' },
  )
})
