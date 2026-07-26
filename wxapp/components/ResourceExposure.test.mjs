import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('..', import.meta.url).pathname)

test('resource exposure requires meaningful visibility and uses batched backend collection', () => {
  const component = fs.readFileSync(path.join(root, 'components/ResourceExposure.vue'), 'utf8')
  const collector = fs.readFileSync(path.join(root, 'common/resourceExposure.js'), 'utf8')
  const api = fs.readFileSync(path.join(root, 'api/metrics.js'), 'utf8')

  assert.match(component, /Number\(result\.intersectionRatio\) >= 0\.5/)
  assert.match(component, /setTimeout\(recordExposure, 800\)/)
  assert.match(component, /enqueueResourceExposure/)
  assert.match(collector, /recordResourceExposures/)
  assert.match(collector, /visitorKey: getVisitorKey\(\)/)
  assert.match(collector, /deduplicateItems\(items\)/)
  assert.match(api, /\/api\/v1\/metrics\/exposures\/batch/)
  assert.match(api, /suppressErrorToast: true/)
})
