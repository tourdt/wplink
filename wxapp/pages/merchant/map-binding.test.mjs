import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'
import { parse, compileTemplate } from '@vue/compiler-sfc'
import { createSSRApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const source = fs.readFileSync(path.join(root, 'pages/merchant/map-binding.vue'), 'utf8')

async function renderSubmitButton(submitting) {
  const { descriptor } = parse(source, { filename: 'pages/merchant/map-binding.vue' })
  const button = (descriptor.template.content.match(/<button\b[^>]*>[\s\S]*?<\/button>/g) || [])
    .find((item) => item.includes('@click="submitBindingRequest"'))
  assert.ok(button, '绑定提交按钮应存在')
  const compiled = compileTemplate({
    source: button,
    filename: 'pages/merchant/map-binding.vue',
    id: 'merchant-map-binding-submit-button',
    compilerOptions: { mode: 'function' },
  })
  assert.equal(compiled.errors.length, 0, compiled.errors.join('\n'))
  const render = new Function('Vue', compiled.code)(await import('vue'))
  const buttonComponent = {
    props: ['submitting', 'selectedObjectId', 'submitBindingRequest'],
    setup(props) {
      return () => render(props, [])
    },
  }
  return renderToString(createSSRApp({
    render: () => h(buttonComponent, { submitting, selectedObjectId: 'object-1', submitBindingRequest: () => {} }),
  }))
}

test('merchant map binding prompts for merchant profile before loading map data', () => {
  assert.match(source, /import \{ ensureMerchantProfileReady \} from '\.\.\/\.\.\/common\/merchantProfileGuard'/)
  assert.match(source, /onLoad\(async \(options\) => \{[\s\S]*merchantId\.value = options\.merchantId \|\| getMerchantId\(\)[\s\S]*if \(!\(await ensureMerchantProfileReady\(merchantId\.value\)\)\) return[\s\S]*loadInitialData\(\)[\s\S]*\}\)/)
  assert.doesNotMatch(source, /uni\.showToast\(\{ title: '请先完善发布者资料'/)
})

test('merchant map binding shows the current submission state without manual review copy', async () => {
  const idleHtml = await renderSubmitButton(false)
  const busyHtml = await renderSubmitButton(true)

  assert.match(idleHtml, />确认绑定<\/button>/)
  assert.match(busyHtml, /<button\b(?=[^>]*\bdisabled(?:=|\s|>))(?=[^>]*\bloading(?:=|\s|>))[^>]*>绑定中<\/button>/)
  assert.match(source, /档口已绑定/)
  assert.match(source, /每个商家暂时只能绑定一个主档口/)
  assert.doesNotMatch(source, /提交绑定申请/)
  assert.doesNotMatch(source, /平台正在核对档口信息/)
})

test('merchant map binding preselects the directory booth from route object id', () => {
  assert.match(source, /const routeObjectId = ref\(''\)/)
  assert.match(source, /routeObjectId\.value = String\(options\.objectId \|\| ''\)\.trim\(\)/)
  assert.match(source, /function applyRouteObjectSelection\(\)/)
  assert.match(source, /candidates\.value\.find\(\(item\) => item\.objectId === routeObjectId\.value && !item\.isBound\)/)
  assert.match(source, /selectedObjectId\.value = candidate\.objectId/)
})
