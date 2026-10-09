import assert from 'node:assert/strict'
import fs from 'node:fs'
import test from 'node:test'

const pages = JSON.parse(fs.readFileSync(new URL('../pages.json', import.meta.url),'utf8'))
test('map launch opens directory and exposes only directory and account tabs', () => {
 assert.equal(pages.pages[0].path,'pages/sourcing-map/index')
 assert.deepEqual(pages.tabBar.list.map(x=>[x.pagePath,x.text]),[['pages/sourcing-map/index','找档口'],['pages/my/index','我的']])
})
test('resource code remains available but inactive business pages are not registered', () => {
 const paths=pages.pages.map(x=>x.path)
 for(const path of ['pages/publish/index','pages/resource/detail','pages/market/index','pages/vip/index','pages/my-resources/index']) {
  assert.ok(!paths.includes(path),path)
  assert.ok(fs.existsSync(new URL(`../${path}.vue`,import.meta.url)),`keep source: ${path}`)
 }
})

test('login redirects preserve merchant context and reject disabled or external destinations', async () => {
 const {resolveMapRedirect}=await import('./mapNavigation.js')
 assert.deepEqual(resolveMapRedirect('/pages/merchant/profile?merchantId=123'),{url:'/pages/merchant/profile?merchantId=123',tab:false})
 assert.deepEqual(resolveMapRedirect('/pages/my/index?a=1'),{url:'/pages/my/index',tab:true})
 for(const value of ['/pages/home/index','/pages/publish/index','https://example.com','/pages/login/index']) {
  assert.deepEqual(resolveMapRedirect(value),{url:'/pages/sourcing-map/index',tab:true})
 }
})
