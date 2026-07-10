import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(import.meta.dirname, '..')

test('growth campaign admin page exposes campaign rule and grant controls', () => {
  const view = fs.readFileSync(path.join(root, 'src/views/GrowthCampaignView.vue'), 'utf8')
  for (const token of ['增长活动', '暂停', '停用', '规则配置', '奖励数量', '有效期', '每日上限', '发放记录']) {
    assert.match(view, new RegExp(token))
  }
  assert.doesNotMatch(view, /top_voucher/)
})

test('growth campaign route and menu are registered', () => {
  const router = fs.readFileSync(path.join(root, 'src/router/index.js'), 'utf8')
  const layout = fs.readFileSync(path.join(root, 'src/layouts/AdminLayout.vue'), 'utf8')
  assert.match(router, /growth-campaigns/)
  assert.match(layout, /增长活动/)
})
