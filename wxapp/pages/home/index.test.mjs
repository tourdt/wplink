import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const source = fs.readFileSync(path.join(root, 'pages/home/index.vue'), 'utf8')

test('home platform picks fall back to public resources when curated feed is empty', () => {
  assert.match(source, /import \{ listResources \} from '\.\.\/\.\.\/api\/resource'/)
  assert.match(source, /async function resolveHomeResourceItems\(resp\)/)
  assert.match(source, /if \(items\.length\) return items/)
  assert.match(source, /async function loadFallbackHomeResources\(\)/)
  assert.match(source, /listResources\([\s\S]*cityCode: DEFAULT_CITY_CODE[\s\S]*pageSize: 30[\s\S]*suppressErrorToast: true[\s\S]*\)/)
})

test('home cold-start entries exclude recruitment and job seeking', () => {
  assert.doesNotMatch(source, /招聘/)
  assert.doesNotMatch(source, /求职/)
  assert.doesNotMatch(source, /groupCode:\s*'jobs'/)
})
