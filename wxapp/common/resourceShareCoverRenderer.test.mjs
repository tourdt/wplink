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
