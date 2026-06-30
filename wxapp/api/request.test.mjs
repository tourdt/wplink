import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'
import { fileURLToPath, pathToFileURL } from 'node:url'
import vm from 'node:vm'

const root = path.resolve(new URL('..', import.meta.url).pathname)

test('redirects to login and clears session when API reports unauthorized session', async () => {
  const storage = new Map([
    ['wplink_token', 'expired-token'],
    ['wplink_user_id', 'user-1'],
    ['wplink_merchant_id', 'merchant-1'],
  ])
  const removedKeys = []
  const toasts = []
  const navigations = []
  let requestOptions

  globalThis.uni = {
    getStorageSync(key) {
      return storage.get(key) || ''
    },
    removeStorageSync(key) {
      removedKeys.push(key)
      storage.delete(key)
    },
    request(options) {
      requestOptions = options
      options.success({
        statusCode: 401,
        data: {
          errorCode: 'UNAUTHORIZED',
          msg: '登录已过期，请重新登录',
        },
      })
    },
    showToast(options) {
      toasts.push(options)
    },
    navigateTo(options) {
      navigations.push(options)
    },
  }
  globalThis.getCurrentPages = () => [{ route: 'pages/my-resources/index', options: { merchantId: 'merchant-1' } }]

  const requestModule = await loadWxappModule('api/request.js')
  const request = requestModule.namespace.default

  await assert.rejects(() => request({ url: '/api/v1/me' }), /登录已过期/)

  assert.equal(requestOptions.header.Authorization, 'Bearer expired-token')
  assert.deepEqual(removedKeys, ['wplink_token', 'wplink_user_id', 'wplink_merchant_id'])
  assert.deepEqual(toasts, [{ title: '登录已过期，请重新登录', icon: 'none' }])
  assert.deepEqual(navigations, [
    { url: '/pages/login/index?redirect=%2Fpages%2Fmy-resources%2Findex%3FmerchantId%3Dmerchant-1' },
  ])
})

async function loadWxappModule(relativePath) {
  const cache = new Map()
  return loadModule(path.join(root, relativePath), cache)
}

async function loadModule(filename, cache) {
  const resolvedFilename = resolveModuleFilename(filename)
  if (cache.has(resolvedFilename)) return cache.get(resolvedFilename)

  const modulePromise = createModule(resolvedFilename, cache)
  cache.set(resolvedFilename, modulePromise)
  return modulePromise
}

async function createModule(resolvedFilename, cache) {
  const source = fs.readFileSync(resolvedFilename, 'utf8')
  const module = new vm.SourceTextModule(source, {
    identifier: pathToFileURL(resolvedFilename).href,
    initializeImportMeta(meta) {
      meta.url = pathToFileURL(resolvedFilename).href
      meta.env = { VITE_API_BASE_URL: '' }
    },
  })
  await module.link((specifier, referencingModule) => {
    if (!specifier.startsWith('.')) {
      throw new Error(`unsupported import in test: ${specifier}`)
    }
    const referencingFilename = fileURLToPath(referencingModule.identifier)
    return loadModule(path.resolve(path.dirname(referencingFilename), specifier), cache)
  })
  await module.evaluate()
  return module
}

function resolveModuleFilename(filename) {
  if (fs.existsSync(filename)) return filename
  for (const extension of ['.js', '.mjs']) {
    const candidate = `${filename}${extension}`
    if (fs.existsSync(candidate)) return candidate
  }
  throw new Error(`module not found: ${filename}`)
}
