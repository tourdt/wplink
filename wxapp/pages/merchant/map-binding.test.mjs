import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const source = fs.readFileSync(path.join(root, 'pages/merchant/map-binding.vue'), 'utf8')

test('merchant map binding prompts for merchant profile before loading map data', () => {
  assert.match(source, /import \{ ensureMerchantProfileReady \} from '\.\.\/\.\.\/common\/merchantProfileGuard'/)
  assert.match(source, /onLoad\(async \(options\) => \{[\s\S]*merchantId\.value = options\.merchantId \|\| getMerchantId\(\)[\s\S]*if \(!\(await ensureMerchantProfileReady\(merchantId\.value\)\)\) return[\s\S]*loadInitialData\(\)[\s\S]*\}\)/)
  assert.doesNotMatch(source, /uni\.showToast\(\{ title: '请先完善商家资料'/)
})
