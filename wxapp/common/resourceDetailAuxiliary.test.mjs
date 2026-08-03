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

function createLoader(overrides = {}) {
  return createResourceDetailAuxiliaryLoader({
    getFavoriteState: async () => ({ favorited: false }),
    hasAuthToken: () => false,
    listRelatedResources: async () => ({ items: [] }),
    setFavorited() {},
    setRelatedResources() {},
    setShareImageUrl() {},
    ...overrides,
  })
}

function createAuxiliaryDeps(overrides = {}) {
  return {
    initializeSharing() {},
    loadMerchantProfile: async () => {},
    recordResourceDetailView: async () => {},
    ...overrides,
  }
}

test('detail auxiliary loader does not clear B recommendations when stale A starts after B succeeds', async () => {
  const relatedRequests = new Map()
  const relatedUpdates = []
  const loader = createLoader({
    listRelatedResources(resourceId) {
      const request = createDeferred()
      relatedRequests.set(resourceId, request)
      return request.promise
    },
    setRelatedResources(items) {
      relatedUpdates.push(items)
    },
  })
  const auxiliaryDeps = createAuxiliaryDeps()

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
  const loader = createLoader({
    listRelatedResources(resourceId) {
      const request = createDeferred()
      relatedRequests.set(resourceId, request)
      return request.promise
    },
    setRelatedResources(items) {
      relatedUpdates.push(items)
    },
  })
  const auxiliaryDeps = createAuxiliaryDeps()

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
  const relatedCalls = []
  const calls = []
  const relatedUpdates = []
  const loader = createLoader({
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
  assert.deepEqual(calls, ['share', 'merchant', 'view'])

  merchant.reject(new Error('merchant unavailable'))
  related.resolve({ items: [{ id: 'related-public' }] })
  view.resolve()
  const results = await run

  assert.equal(results.filter((result) => result.status === 'rejected').length, 1)
  assert.deepEqual(relatedUpdates.at(-1), [{ id: 'related-public' }])
})

test('own detail requires authentication for related resources without public-only tasks', async () => {
  const related = createDeferred()
  const relatedCalls = []
  const calls = []
  const loader = createLoader({
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

test('favorite state from stale A cannot overwrite B after B finishes first', async () => {
  const favoriteRequests = new Map()
  const favoritedUpdates = []
  const loader = createLoader({
    getFavoriteState(resourceId) {
      const request = createDeferred()
      favoriteRequests.set(resourceId, request)
      return request.promise
    },
    hasAuthToken: () => true,
    setFavorited(favorited) {
      favoritedUpdates.push(favorited)
    },
  })
  const auxiliaryDeps = createAuxiliaryDeps()

  const loadA = loader.begin('resource-a')
  const runA = loader.run(loadA, { ...auxiliaryDeps, isOwnResource: false })
  await flushTasks()
  const loadB = loader.begin('resource-b')
  const runB = loader.run(loadB, { ...auxiliaryDeps, isOwnResource: false })
  await flushTasks()

  favoriteRequests.get('resource-b').resolve({ favorited: true })
  await runB
  favoriteRequests.get('resource-a').resolve({ favorited: false })
  await runA

  assert.equal(favoritedUpdates.at(-1), true)
})

test('new detail without a token clears previous favorite state without requesting it', async () => {
  const favoriteRequests = []
  const favoritedUpdates = []
  let hasToken = true
  const loader = createLoader({
    async getFavoriteState(resourceId) {
      favoriteRequests.push(resourceId)
      return { favorited: true }
    },
    hasAuthToken: () => hasToken,
    setFavorited(favorited) {
      favoritedUpdates.push(favorited)
    },
  })
  const auxiliaryDeps = createAuxiliaryDeps()

  const loadA = loader.begin('resource-a')
  await loader.run(loadA, { ...auxiliaryDeps, isOwnResource: false })
  assert.equal(favoritedUpdates.at(-1), true)

  hasToken = false
  const loadB = loader.begin('resource-b')
  await loader.run(loadB, { ...auxiliaryDeps, isOwnResource: false })

  assert.deepEqual(favoriteRequests, ['resource-a'])
  assert.equal(favoritedUpdates.at(-1), false)
})

test('new generation immediately clears the previous share image', () => {
  const shareImageUpdates = []
  const loader = createLoader({
    setShareImageUrl(imageUrl) {
      shareImageUpdates.push(imageUrl)
    },
  })

  loader.begin('resource-a')
  loader.begin('resource-b')

  assert.deepEqual(shareImageUpdates, ['', ''])
})
