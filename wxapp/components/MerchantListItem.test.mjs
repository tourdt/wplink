import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const componentPath = path.resolve(new URL('.', import.meta.url).pathname, 'MerchantListItem.vue')
const source = fs.existsSync(componentPath) ? fs.readFileSync(componentPath, 'utf8') : ''

test('merchant list item provides shared content regions without scene-specific copy', () => {
  assert.equal(fs.existsSync(componentPath), true)
  for (const slot of ['leading', 'badge', 'meta', 'actions']) {
    assert.match(source, new RegExp(`name="${slot}"`))
  }
  assert.match(source, /defineEmits\(\['activate'\]\)/)
  assert.match(source, /title:\s*\{\s*type:\s*String,\s*required:\s*true/)
  assert.match(source, /subtitle:\s*\{\s*type:\s*String,\s*default:\s*''/)
  assert.match(source, /tags:\s*\{\s*type:\s*Array,\s*default:\s*\(\)\s*=>\s*\[\]/)
  assert.match(source, /selected:\s*\{\s*type:\s*Boolean,\s*default:\s*false/)
  assert.doesNotMatch(source, /新入驻|待认领|导航|入驻日期/)
})
