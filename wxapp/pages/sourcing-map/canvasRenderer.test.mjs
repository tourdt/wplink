import assert from 'node:assert/strict'
import test from 'node:test'
import { createSourcingMapRenderer } from './canvasRenderer.js'

function createMockContext(operations, options = {}) {
  const record = (type, args) => operations.push({ type, args: Array.from(args) })
  return {
    clearRect() {
      record('clearRect', arguments)
    },
    drawImage() {
      if (options.throwOnDrawImage) {
        throw new Error('drawImage failed')
      }
      record('drawImage', arguments)
    },
    fillRect() {
      record('fillRect', arguments)
    },
    beginPath() {
      record('beginPath', arguments)
    },
    arc() {
      record('arc', arguments)
    },
    closePath() {
      record('closePath', arguments)
    },
    fill() {
      record('fill', arguments)
    },
    stroke() {
      record('stroke', arguments)
    },
    setFillStyle() {},
    setStrokeStyle() {},
    setLineWidth() {},
    setFontSize() {},
    setTextAlign() {},
    setTextBaseline() {},
    fillText() {
      record('fillText', arguments)
    },
    draw() {
      record('draw', arguments)
    },
  }
}

test('renderer draws the base map and points on the same canvas after the base image is ready', () => {
  const operations = []
  const originalUni = globalThis.uni
  let pendingImageInfo = null
  globalThis.uni = {
    createCanvasContext() {
      return createMockContext(operations)
    },
    getImageInfo(request) {
      pendingImageInfo = request
    },
  }

  try {
    const renderer = createSourcingMapRenderer()
    assert.equal(renderer.init({ canvasId: 'sourcingMapCanvas', width: 400, height: 200 }), true)
    renderer.setScene({ width: 1000, height: 500, backgroundUrl: 'https://cdn.test/base.png' })
    renderer.setObjects([{ id: 'poi-1', geometryType: 'point', geometry: { x: 100, y: 50 } }])

    renderer.render({ scale: 1, offsetX: 0, offsetY: 0 }, {
      width: 400,
      height: 200,
      baseScale: 0.4,
      drawBackground: 'whenReady',
    })

    assert.equal(operations.some((entry) => entry.type === 'drawImage'), false)
    assert.equal(renderer.hasRenderedBackground(), false)

    pendingImageInfo.success({ path: '/tmp/base.png' })
    pendingImageInfo.complete()
    operations.length = 0
    renderer.render({ scale: 1, offsetX: 0, offsetY: 0 }, {
      width: 400,
      height: 200,
      baseScale: 0.4,
      drawBackground: 'whenReady',
    })

    const baseImage = operations.find((entry) => entry.type === 'drawImage')
    const pointArc = operations.find((entry) => entry.type === 'arc')
    assert.equal(renderer.hasRenderedBackground(), true)
    assert.deepEqual(baseImage.args, ['/tmp/base.png', 0, 0, 400, 200])
    assert.equal(pointArc.args[0], 40)
    assert.equal(pointArc.args[1], 20)
  } finally {
    globalThis.uni = originalUni
  }
})

test('renderer keeps the background unready when canvas image drawing fails', () => {
  const operations = []
  const originalUni = globalThis.uni
  globalThis.uni = {
    createCanvasContext() {
      return createMockContext(operations, { throwOnDrawImage: true })
    },
    getImageInfo(request) {
      request.success({ path: '/tmp/base.png' })
      request.complete()
    },
  }

  try {
    const renderer = createSourcingMapRenderer()
    assert.equal(renderer.init({ canvasId: 'sourcingMapCanvas', width: 400, height: 200 }), true)
    renderer.setScene({ width: 1000, height: 500, backgroundUrl: 'https://cdn.test/base.png' })
    renderer.render({ scale: 1, offsetX: 0, offsetY: 0 }, {
      width: 400,
      height: 200,
      baseScale: 0.4,
      drawBackground: 'whenReady',
    })

    assert.equal(renderer.hasBackgroundImage(), true)
    assert.equal(renderer.hasRenderedBackground(), false)
    assert.equal(operations.some((entry) => entry.type === 'drawImage'), false)
  } finally {
    globalThis.uni = originalUni
  }
})

test('renderer notifies asset readiness when native image info fails so overlays can redraw over fallback image', () => {
  const operations = []
  const originalUni = globalThis.uni
  let assetReadyCount = 0
  globalThis.uni = {
    createCanvasContext() {
      return createMockContext(operations)
    },
    getImageInfo(request) {
      request.fail()
      request.complete()
    },
  }

  try {
    const renderer = createSourcingMapRenderer({
      onAssetReady() {
        assetReadyCount += 1
      },
    })
    assert.equal(renderer.init({ canvasId: 'sourcingMapCanvas', width: 400, height: 200 }), true)
    renderer.setScene({ width: 1000, height: 500, backgroundUrl: 'https://cdn.test/base.png' })
    renderer.setObjects([{ id: 'poi-1', geometryType: 'point', geometry: { x: 100, y: 50 } }])
    renderer.render({ scale: 1, offsetX: 0, offsetY: 0 }, {
      width: 400,
      height: 200,
      baseScale: 0.4,
      drawBackground: 'whenReady',
    })

    assert.equal(assetReadyCount, 1)
    assert.equal(renderer.hasRenderedBackground(), false)
    assert.equal(operations.some((entry) => entry.type === 'arc'), true)
  } finally {
    globalThis.uni = originalUni
  }
})

test('renderer reports background readiness after the canvas draw callback completes', () => {
  const operations = []
  const originalUni = globalThis.uni
  let drawCallback = null
  globalThis.uni = {
    createCanvasContext() {
      return {
        ...createMockContext(operations),
        draw(reserve, callback) {
          operations.push({ type: 'draw', args: [reserve] })
          drawCallback = callback
        },
      }
    },
    getImageInfo(request) {
      request.success({ path: '/tmp/base.png' })
      request.complete()
    },
  }

  try {
    const renderer = createSourcingMapRenderer()
    assert.equal(renderer.init({ canvasId: 'sourcingMapCanvas', width: 400, height: 200 }), true)
    renderer.setScene({ width: 1000, height: 500, backgroundUrl: 'https://cdn.test/base.png' })
    let drawCompletePayload = null
    renderer.render({ scale: 1, offsetX: 0, offsetY: 0 }, {
      width: 400,
      height: 200,
      baseScale: 0.4,
      drawBackground: 'whenReady',
      onDrawComplete(payload) {
        drawCompletePayload = payload
      },
    })

    assert.equal(drawCompletePayload, null)
    assert.equal(typeof drawCallback, 'function')
    drawCallback()
    assert.deepEqual(drawCompletePayload, { backgroundDrawn: true })
  } finally {
    globalThis.uni = originalUni
  }
})
