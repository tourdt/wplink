import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('../..', import.meta.url).pathname)

test('login restores merchant id from managed merchants after token is saved', () => {
  const source = fs.readFileSync(path.join(root, 'pages/login/index.vue'), 'utf8')

  assert.match(source, /import \{ getMe, wechatLogin \} from '\.\.\/\.\.\/api\/auth'/)
  assert.match(source, /import \{ saveMerchantId, saveToken, saveUserId \} from '\.\.\/\.\.\/store\/session'/)
  assert.match(source, /await restoreManagedMerchantId\(resp\)/)
  assert.match(source, /async function restoreManagedMerchantId\(loginResp = \{\}\)/)
  assert.match(source, /let managedMerchants = loginResp\.managedMerchants/)
  assert.match(source, /if \(!Array\.isArray\(managedMerchants\)\) \{[\s\S]*const me = await getMe\(\)/)
  assert.match(source, /managedMerchants = me\.managedMerchants \|\| \[\]/)
  assert.match(source, /const managedMerchant = managedMerchants\[0\]/)
  assert.match(source, /if \(managedMerchant\?\.id\) \{[\s\S]*saveMerchantId\(managedMerchant\.id\)/)
  assert.match(source, /当前账号没有可管理商家时必须清理旧缓存/)
  assert.match(source, /saveMerchantId\(''\)/)
})

test('login always obtains a real WeChat code and rejects missing codes', () => {
  const source = fs.readFileSync(path.join(root, 'pages/login/index.vue'), 'utf8')

  assert.doesNotMatch(source, /local-dev-|localDevLoginCode|shouldUseLocalDevLoginCode|API_BASE_URL/)
  assert.match(source, /uni\.login\(\{[\s\S]*provider: 'weixin'/)
  assert.match(source, /success: \(res\) => \{[\s\S]*if \(res\.code\) \{[\s\S]*resolve\(res\.code\)[\s\S]*reject\(new Error\('未获取到微信登录凭证，请重试'\)\)/)
  assert.match(source, /fail: \(\) => \{[\s\S]*reject\(new Error\('微信登录暂不可用，请稍后重试'\)\)/)
})
