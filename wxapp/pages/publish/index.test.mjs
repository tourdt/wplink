import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const source = fs.readFileSync(path.join(root, 'pages/publish/index.vue'), 'utf8')

test('publish entry requires login but does not require merchant profile completion', () => {
  assert.match(source, /import \{ requireLogin \} from '\.\.\/\.\.\/common\/auth'/)
  assert.doesNotMatch(source, /ensureMerchantProfileReady/)
  assert.match(source, /async function applyPendingPublishType\(\) \{[\s\S]*if \(!requireLogin\(\)\) return[\s\S]*navigateToPublishForm/)
  assert.match(source, /function startPublish\(direction\) \{[\s\S]*if \(!requireLogin\(\)\) return[\s\S]*navigateToPublishForm/)
})
