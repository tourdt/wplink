import assert from 'node:assert/strict'
import test from 'node:test'

import { createResourceDetailAuxiliaryLoader } from './resourceDetailAuxiliary.js'

function createDeferred() {
  let reject
  let resolve
  const promise = new Promise((resolvePromise, rejectPromise) => {
    resolve = resolvePromise
    reject = rejectPromise
  })
  return { promise, reject, resolve }
}

async function flushTasks() {
  await Promise.resolve()
  await Promise.resolve()
}

test('detail auxiliary loader does not clear B recommendations when stale A starts after B succeeds', async () => {
  const relatedRequests = new Map()
  const relatedUpdates = []
  const loader = createResourceDetailAuxiliaryLoader({
    listRelatedResources(resourceId) {
      const request = createDeferred()
      relatedRequests.set(resourceId, request)
      return request.promise
    },
    setRelatedResources(items) {
      relatedUpdates.push(items)
    },
  })
  const auxiliaryDeps = {
    initializeSharing() {},
    loadFavoriteState: async () => {},
    loadMerchantProfile: async () => {},
    recordResourceDetailView: async () => {},
  }

  const loadA = loader.begin('resource-a')
  const loadB = loader.begin('resource-b')
  const runB = loader.run(loadB, { ...auxiliaryDeps, isOwnResource: false })
  await flushTasks()
  relatedRequests.get('resource-b').resolve({ items: [{ id: 'related-b' }] })
  await runB

  await loader.run(loadA, { ...auxiliaryDeps, isOwnResource: false })

  assert.deepEqual(relatedUpdates.at(-1), [{ id: 'related-b' }])
  assert.equal(relatedRequests.has('resource-a'), false)
})

test('detail auxiliary loader ignores A recommendation response that arrives after B', async () => {
  const relatedRequests = new Map()
  const relatedUpdates = []
  const loader = createResourceDetailAuxiliaryLoader({
    listRelatedResources(resourceId) {
      const request = createDeferred()
      relatedRequests.set(resourceId, request)
      return request.promise
    },
    setRelatedResources(items) {
      relatedUpdates.push(items)
    },
  })
  const auxiliaryDeps = {
    initializeSharing() {},
    loadFavoriteState: async () => {},
    loadMerchantProfile: async () => {},
    recordResourceDetailView: async () => {},
  }

  const loadA = loader.begin('resource-a')
  const runA = loader.run(loadA, { ...auxiliaryDeps, isOwnResource: false })
  await flushTasks()
  const loadB = loader.begin('resource-b')
  const runB = loader.run(loadB, { ...auxiliaryDeps, isOwnResource: false })
  await flushTasks()

  relatedRequests.get('resource-b').resolve({ items: [{ id: 'related-b' }] })
  await runB
  relatedRequests.get('resource-a').resolve({ items: [{ id: 'related-a' }] })
  await runA

  assert.deepEqual(relatedUpdates.at(-1), [{ id: 'related-b' }])
})

test('public detail keeps sharing independent while all auxiliary requests settle', async () => {
  const merchant = createDeferred()
  const related = createDeferred()
  const view = createDeferred()
  const favorite = createDeferred()
  const relatedCalls = []
  const calls = []
  const relatedUpdates = []
  const loader = createResourceDetailAuxiliaryLoader({
    listRelatedResources(resourceId, params, options) {
      relatedCalls.push({ options, params, resourceId })
      return related.promise
    },
    setRelatedResources(items) {
      relatedUpdates.push(items)
    },
  })

  const context = loader.begin('public-resource')
  const run = loader.run(context, {
    initializeSharing() {
      calls.push('share')
    },
    isOwnResource: false,
    loadFavoriteState() {
      calls.push('favorite')
      return favorite.promise
    },
    loadMerchantProfile() {
      calls.push('merchant')
      return merchant.promise
    },
    recordResourceDetailView() {
      calls.push('view')
      return view.promise
    },
  })
  await flushTasks()

  assert.deepEqual(relatedCalls, [{
    options: { requireAuth: false, suppressErrorToast: true },
    params: { pageSize: 3 },
    resourceId: 'public-resource',
  }])
  assert.deepEqual(calls, ['share', 'merchant', 'view', 'favorite'])

  merchant.reject(new Error('merchant unavailable'))
  related.resolve({ items: [{ id: 'related-public' }] })
  view.resolve()
  favorite.resolve()
  const results = await run

  assert.equal(results.filter((result) => result.status === 'rejected').length, 1)
  assert.deepEqual(relatedUpdates.at(-1), [{ id: 'related-public' }])
})

test('own detail requires authentication for related resources without public-only tasks', async () => {
  const related = createDeferred()
  const relatedCalls = []
  const calls = []
  const loader = createResourceDetailAuxiliaryLoader({
    listRelatedResources(resourceId, params, options) {
      relatedCalls.push({ options, params, resourceId })
      return related.promise
    },
    setRelatedResources() {},
  })

  const context = loader.begin('own-resource')
  const run = loader.run(context, {
    initializeSharing() {
      calls.push('share')
    },
    isOwnResource: true,
    loadFavoriteState() {
      calls.push('favorite')
    },
    loadMerchantProfile() {
      calls.push('merchant')
    },
    recordResourceDetailView() {
      calls.push('view')
    },
  })
  await flushTasks()
  related.resolve({ items: [] })
  await run

  assert.deepEqual(relatedCalls, [{
    options: { requireAuth: true, suppressErrorToast: true },
    params: { pageSize: 3 },
    resourceId: 'own-resource',
  }])
  assert.deepEqual(calls, ['share', 'merchant'])
})
