import assert from 'node:assert/strict'
import test from 'node:test'
import {
  createInitialTransform,
  endGesture,
  mapToScreen,
  moveGesture,
  screenToMap,
  startGesture,
} from './mapGesture.js'

const options = {
  viewportWidth: 300,
  viewportHeight: 200,
  mapWidth: 600,
  mapHeight: 400,
  minScale: 1,
  maxScale: 3,
}

function touch(x, y, identifier = 1) {
  return { clientX: x, clientY: y, identifier }
}

test('createInitialTransform centers small maps', () => {
  const transform = createInitialTransform({
    viewportWidth: 300,
    viewportHeight: 240,
    mapWidth: 200,
    mapHeight: 120,
  })

  assert.equal(transform.scale, 1)
  assert.equal(transform.offsetX, 50)
  assert.equal(transform.offsetY, 60)
})

test('single finger drag updates offsets by touch movement', () => {
  const startTransform = { scale: 1, offsetX: -100, offsetY: -80 }
  const state = startGesture([touch(100, 100)], startTransform, options)
  const result = moveGesture(state, [touch(130, 150)], options)

  assert.equal(result.transform.offsetX, -70)
  assert.equal(result.transform.offsetY, -30)
  assert.equal(result.state.mode, 'pan')
})

test('pinch zoom keeps the center map coordinate stable', () => {
  const startTransform = { scale: 1, offsetX: -100, offsetY: -50 }
  const state = startGesture([touch(100, 100, 1), touch(200, 100, 2)], startTransform, options)
  const anchor = screenToMap({ x: 150, y: 100 }, startTransform)
  const result = moveGesture(state, [touch(75, 100, 1), touch(225, 100, 2)], options)
  const anchoredScreenPoint = mapToScreen(anchor, result.transform)

  assert.equal(result.transform.scale, 1.5)
  assert.ok(Math.abs(anchoredScreenPoint.x - 150) < 0.001)
  assert.ok(Math.abs(anchoredScreenPoint.y - 100) < 0.001)
})

test('dragging past the viewport edge is clamped', () => {
  const startTransform = { scale: 1, offsetX: -20, offsetY: -10 }
  const state = startGesture([touch(100, 100)], startTransform, options)
  const result = moveGesture(state, [touch(260, 220)], options)

  assert.equal(result.transform.offsetX, 0)
  assert.equal(result.transform.offsetY, 0)
})

test('dragging past an edge can use rubber band overflow during interaction', () => {
  const startTransform = { scale: 1, offsetX: -20, offsetY: -10 }
  const state = startGesture([touch(100, 100)], startTransform, options)
  const result = moveGesture(state, [touch(260, 220)], { ...options, allowOverflow: true, maxOverflow: 60 })
  const finalTransform = endGesture(result.state, options)

  assert.ok(result.transform.offsetX > 0)
  assert.ok(result.transform.offsetX <= 60)
  assert.ok(result.transform.offsetY > 0)
  assert.ok(result.transform.offsetY <= 60)
  assert.equal(finalTransform.offsetX, 0)
  assert.equal(finalTransform.offsetY, 0)
})

test('small maps still rubber band around their centered position', () => {
  const smallOptions = {
    viewportWidth: 300,
    viewportHeight: 240,
    mapWidth: 200,
    mapHeight: 120,
    minScale: 1,
    maxScale: 3,
  }
  const startTransform = createInitialTransform(smallOptions)
  const state = startGesture([touch(100, 100)], startTransform, smallOptions)
  const result = moveGesture(state, [touch(100, 190)], { ...smallOptions, allowOverflow: true, maxOverflow: 50 })
  const finalTransform = endGesture(result.state, smallOptions)

  assert.ok(result.transform.offsetY > startTransform.offsetY)
  assert.ok(result.transform.offsetY <= startTransform.offsetY + 50)
  assert.equal(finalTransform.offsetY, startTransform.offsetY)
})

test('screenToMap reverses the active transform', () => {
  const transform = { scale: 2, offsetX: -50, offsetY: -30 }
  assert.deepEqual(screenToMap({ x: 150, y: 130 }, transform), { x: 100, y: 80 })
})

test('endGesture clamps the final transform to map bounds', () => {
  const state = startGesture([touch(100, 100)], { scale: 1, offsetX: -100, offsetY: -80 }, options)
  const moved = moveGesture(state, [touch(10, 10)], options)
  const finalTransform = endGesture(moved.state, options)

  assert.ok(finalTransform.offsetX <= 0)
  assert.ok(finalTransform.offsetY <= 0)
  assert.ok(finalTransform.offsetX >= options.viewportWidth - options.mapWidth)
  assert.ok(finalTransform.offsetY >= options.viewportHeight - options.mapHeight)
})
