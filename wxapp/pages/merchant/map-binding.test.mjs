import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const source = fs.readFileSync(path.join(root, 'pages/merchant/map-binding.vue'), 'utf8')

test('merchant map binding prompts for merchant profile before loading map data', () => {
  assert.match(source, /import \{ ensureMerchantProfileReady \} from '\.\.\/\.\.\/common\/merchantProfileGuard'/)
  assert.match(source, /onLoad\(async \(options\) => \{[\s\S]*merchantId\.value = options\.merchantId \|\| getMerchantId\(\)[\s\S]*if \(!\(await ensureMerchantProfileReady\(merchantId\.value\)\)\) return[\s\S]*loadInitialData\(\)[\s\S]*\}\)/)
  assert.doesNotMatch(source, /uni\.showToast\(\{ title: '请先完善发布者资料'/)
})

test('merchant map binding confirms the main booth without normal manual review copy', () => {
  assert.match(source, />确认绑定<\/button>/)
  assert.match(source, /档口已绑定/)
  assert.match(source, /每个商家暂时只能绑定一个主档口/)
  assert.doesNotMatch(source, /提交绑定申请/)
  assert.doesNotMatch(source, /平台正在核对档口信息/)
})
