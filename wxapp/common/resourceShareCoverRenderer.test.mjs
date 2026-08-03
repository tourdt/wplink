import assert from 'node:assert/strict'
import test from 'node:test'

import { createResourceShareCoverRenderer } from './resourceShareCoverRenderer.js'

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

test('share cover renderer runs latest B after A releases the render lock', async () => {
  let currentContext
  const exports = new Map()
  const renderCalls = []
  const imageUpdates = []
  const renderer = createResourceShareCoverRenderer({
    getFallbackCover: (context) => `fallback-${context.resourceId}`,
    isCurrent: (context) => context === currentContext,
    renderCover(context) {
      renderCalls.push(context.resourceId)
      const request = createDeferred()
      exports.set(context.resourceId, request)
      return request.promise
    },
    setShareImageUrl(imageUrl) {
      imageUpdates.push(imageUrl)
    },
  })
  const loadA = { generation: 1, resourceId: 'resource-a' }
  const loadB = { generation: 2, resourceId: 'resource-b' }

  currentContext = loadA
  renderer.request(loadA)
  await flushTasks()
  currentContext = loadB
  renderer.request(loadB)

  exports.get('resource-a').resolve('cover-a')
  await flushTasks()
  assert.deepEqual(renderCalls, ['resource-a', 'resource-b'])
  assert.deepEqual(imageUpdates, [])

  exports.get('resource-b').resolve('cover-b')
  await flushTasks()

  assert.deepEqual(imageUpdates, ['cover-b'])
})

test('share cover renderer runs B full profile snapshot after B brief releases the lock', async () => {
  const briefExport = createDeferred()
  const renderCalls = []
  const imageUpdates = []
  const renderer = createResourceShareCoverRenderer({
    getFallbackCover: () => '',
    isCurrent: (context) => context.generation === 2 && context.resourceId === 'resource-b',
    renderCover(context) {
      renderCalls.push(context.merchant.name)
      if (context.merchant.name === 'B brief') return briefExport.promise
      return Promise.resolve('cover-full')
    },
    setShareImageUrl(imageUrl) {
      imageUpdates.push(imageUrl)
    },
  })
  const briefContext = { generation: 2, resourceId: 'resource-b', merchant: { name: 'B brief' } }
  const fullContext = { generation: 2, resourceId: 'resource-b', merchant: { name: 'B full' } }

  renderer.request(briefContext)
  await flushTasks()
  renderer.request(fullContext)
  briefExport.resolve('cover-brief')
  await flushTasks()

  assert.deepEqual(renderCalls, ['B brief', 'B full'])
  assert.deepEqual(imageUpdates, ['cover-brief', 'cover-full'])
  assert.equal(imageUpdates.at(-1), 'cover-full')
})

test('share cover renderer keeps the latest same-resource snapshot and deduplicates its identical request', async () => {
  const briefExport = createDeferred()
  const renderCalls = []
  const imageUpdates = []
  const renderer = createResourceShareCoverRenderer({
    getFallbackCover: () => '',
    isCurrent: (context) => context.generation === 2 && context.resourceId === 'resource-b',
    renderCover(context) {
      renderCalls.push(context.version)
      if (context.version === 'brief') return briefExport.promise
      return Promise.resolve(`cover-${context.version}`)
    },
    setShareImageUrl(imageUrl) {
      imageUpdates.push(imageUrl)
    },
  })
  const briefContext = { generation: 2, resourceId: 'resource-b', version: 'brief' }
  const fullV1Context = { generation: 2, resourceId: 'resource-b', version: 'full-v1' }
  const fullV2Context = { generation: 2, resourceId: 'resource-b', version: 'full-v2' }

  renderer.request(briefContext)
  await flushTasks()
  renderer.request(fullV1Context)
  renderer.request(fullV1Context)
  renderer.request(fullV2Context)
  renderer.request(fullV2Context)
  briefExport.resolve('cover-brief')
  await flushTasks()

  assert.deepEqual(renderCalls, ['brief', 'full-v2'])
  assert.equal(imageUpdates.at(-1), 'cover-full-v2')
})

test('share cover renderer does not requeue the identical active snapshot', async () => {
  const exportRequest = createDeferred()
  const renderCalls = []
  const renderer = createResourceShareCoverRenderer({
    getFallbackCover: () => '',
    isCurrent: () => true,
    renderCover(context) {
      renderCalls.push(context)
      return exportRequest.promise
    },
    setShareImageUrl() {},
  })
  const context = { generation: 2, resourceId: 'resource-b' }

  renderer.request(context)
  await flushTasks()
  renderer.request(context)
  renderer.request(context)
  exportRequest.resolve('cover-b')
  await flushTasks()

  assert.deepEqual(renderCalls, [context])
})

test('share cover renderer keeps only the latest pending context while busy', async () => {
  let currentContext
  const exports = new Map()
  const renderCalls = []
  const renderer = createResourceShareCoverRenderer({
    getFallbackCover: () => '',
    isCurrent: (context) => context === currentContext,
    renderCover(context) {
      renderCalls.push(context.resourceId)
      const request = createDeferred()
      exports.set(context.resourceId, request)
      return request.promise
    },
    setShareImageUrl() {},
  })
  const loadA = { generation: 1, resourceId: 'resource-a' }
  const loadB = { generation: 2, resourceId: 'resource-b' }
  const loadC = { generation: 3, resourceId: 'resource-c' }

  currentContext = loadA
  renderer.request(loadA)
  await flushTasks()
  currentContext = loadB
  renderer.request(loadB)
  currentContext = loadC
  renderer.request(loadC)

  exports.get('resource-a').resolve('cover-a')
  await flushTasks()

  assert.deepEqual(renderCalls, ['resource-a', 'resource-c'])
})

test('share cover renderer never writes a stale fallback after export failure', async () => {
  let currentContext
  const exportA = createDeferred()
  const imageUpdates = []
  const renderer = createResourceShareCoverRenderer({
    getFallbackCover: (context) => `fallback-${context.resourceId}`,
    isCurrent: (context) => context === currentContext,
    renderCover: () => exportA.promise.then(() => { throw new Error('export failed') }),
    setShareImageUrl(imageUrl) {
      imageUpdates.push(imageUrl)
    },
  })
  const loadA = { generation: 1, resourceId: 'resource-a' }
  const loadB = { generation: 2, resourceId: 'resource-b' }

  currentContext = loadA
  renderer.request(loadA)
  await flushTasks()
  currentContext = loadB
  exportA.resolve()
  await flushTasks()

  assert.deepEqual(imageUpdates, [])
})
