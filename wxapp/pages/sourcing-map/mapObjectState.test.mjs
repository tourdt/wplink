import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

import { mapObjectIdentity } from './mapObjectState.js'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const pageSource = fs.readFileSync(path.join(root, 'pages/sourcing-map/index.vue'), 'utf8')
const rendererSource = fs.readFileSync(path.join(root, 'pages/sourcing-map/canvasRenderer.js'), 'utf8')
const hitTestSource = fs.readFileSync(path.join(root, 'pages/sourcing-map/mapHitTest.js'), 'utf8')

test('normalizes map object identity from backend id only', () => {
  assert.equal(mapObjectIdentity({ id: 'object-1', code: 'A01' }), 'object-1')
  assert.equal(mapObjectIdentity({ code: 'A01' }), '')
  assert.equal(mapObjectIdentity(null), '')
})

test('sourcing map modules share the same object identity helper', () => {
  assert.match(pageSource, /import \{[^}]*mapObjectIdentity[^}]*\} from '\.\/mapObjectState'/)
  assert.match(rendererSource, /import \{[^}]*mapObjectIdentity[^}]*\} from '\.\/mapObjectState\.js'/)
  assert.match(hitTestSource, /import \{[^}]*mapObjectIdentity[^}]*\} from '\.\/mapObjectState\.js'/)
  assert.doesNotMatch(pageSource, /function objectIdentity\(/)
  assert.doesNotMatch(rendererSource, /function objectIdentity\(/)
  assert.doesNotMatch(hitTestSource, /function objectIdentity\(/)
})
