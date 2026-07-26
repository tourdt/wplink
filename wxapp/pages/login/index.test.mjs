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

test('login uses stable local dev code when connected to local api', () => {
  const source = fs.readFileSync(path.join(root, 'pages/login/index.vue'), 'utf8')

  assert.match(source, /API_BASE_URL,[\s\S]*DEFAULT_CITY_CODE,[\s\S]*from '\.\.\/\.\.\/common\/constants'/)
  assert.match(source, /if \(shouldUseLocalDevLoginCode\(\)\) \{[\s\S]*resolve\(localDevLoginCode\(\)\)/)
  assert.match(source, /function shouldUseLocalDevLoginCode\(\)/)
  assert.match(source, /127\\\.0\\\.0\\\.1/)
  assert.match(source, /localhost/)
})
