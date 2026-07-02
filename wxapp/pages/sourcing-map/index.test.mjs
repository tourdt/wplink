import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const pagesConfig = JSON.parse(fs.readFileSync(path.join(root, 'pages.json'), 'utf8'))
const homeSource = fs.readFileSync(path.join(root, 'pages/home/index.vue'), 'utf8')
const apiSource = readOptionalSource('api/sourcingMap.js')
const source = readOptionalSource('pages/sourcing-map/index.vue')

function readOptionalSource(file) {
  const fullPath = path.join(root, file)
  return fs.existsSync(fullPath) ? fs.readFileSync(fullPath, 'utf8') : ''
}

test('sourcing map page is reachable from wxapp home', () => {
  assert.ok(pagesConfig.pages.some((entry) => entry.path === 'pages/sourcing-map/index'))
  assert.match(homeSource, /拿货地图/)
  assert.match(homeSource, /openSourcingMap/)
  assert.match(homeSource, /\/pages\/sourcing-map\/index/)
})

test('sourcing map api uses public map endpoints', () => {
  for (const token of [
    'listMapScenes',
    'getMapScene',
    'listMapObjects',
    'searchMapObjects',
    'getMapObject',
    'listNearbyPois',
    '/api/v1/map/scenes',
    '/api/v1/map/objects/search',
  ]) {
    assert.match(apiSource, new RegExp(token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }
})
