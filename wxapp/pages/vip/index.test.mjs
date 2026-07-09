import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('../..', import.meta.url).pathname)

test('vip page is registered and shows launch quota benefits', () => {
  const pagesConfig = JSON.parse(fs.readFileSync(path.join(root, 'pages.json'), 'utf8'))
  const pagePaths = pagesConfig.pages.map((item) => item.path)
  const source = fs.readFileSync(path.join(root, 'pages/vip/index.vue'), 'utf8')
  const apiSource = fs.readFileSync(path.join(root, 'api/vip.js'), 'utf8')

  assert.ok(pagePaths.includes('pages/vip/index'))
  assert.match(apiSource, /listVIPPlans/)
  assert.match(apiSource, /\/api\/v1\/vip\/plans/)
  assert.match(apiSource, /getMerchantVIP/)
  assert.match(apiSource, /createVIPOrder/)
  assert.match(apiSource, /createVIPPayment/)
  assert.match(source, /年卡限时 ¥299/)
  assert.match(source, /80 条发布额度/)
  assert.match(source, /30 次刷新/)
  assert.match(source, /3 张置顶券/)
  assert.match(source, /await requestWechatPayment\(payment\)/)
  assert.match(source, /resp\.status === 'paid'/)
})
