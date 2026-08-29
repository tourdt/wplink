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

test('rejects unauthorized background request without clearing session, toast, or login redirect', async () => {
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
  globalThis.getCurrentPages = () => [{ route: 'pages/merchant/location', options: { merchantId: 'merchant-1' } }]

  const requestModule = await loadWxappModule('api/request.js')
  const request = requestModule.namespace.default

  await assert.rejects(() => request({
    url: '/api/v1/metrics/merchant-directory-events',
    method: 'POST',
    suppressErrorToast: true,
    suppressUnauthorizedSideEffects: true,
  }), /登录已过期/)

  assert.equal(requestOptions.header.Authorization, 'Bearer expired-token')
  assert.deepEqual(removedKeys, [])
  assert.equal(storage.get('wplink_token'), 'expired-token')
  assert.deepEqual(toasts, [])
  assert.deepEqual(navigations, [])
})

test('merchant directory API uses only the new directory location context and report routes', async () => {
  const source = fs.readFileSync(path.join(root, 'api/merchantDirectory.js'), 'utf8')
    .replace("import request from './request'", '')
    .replaceAll('export function ', 'function ')
  const calls = []
  const sandbox = { request: async (options) => { calls.push(options); return {} } }
  vm.runInNewContext(`${source}\nglobalThis.api = {
    listMerchantDirectory, getMerchantLocationContext, submitMerchantLocationReport,
  }`, sandbox)

  await sandbox.api.listMerchantDirectory({ cityCode: 'zhili', keyword: 'A108', categories: '童装', page: 2, pageSize: 20 })
  await sandbox.api.getMerchantLocationContext(' merchant/a?b#c% ')
  await sandbox.api.submitMerchantLocationReport(' merchant/a?b#c% ', {
    reasonCode: 'navigation_inaccurate', description: '导航后反馈位置不准确', reporterUserId: 'untrusted',
  })

  assert.deepEqual(JSON.parse(JSON.stringify(calls)), [
    {
      url: '/api/v1/merchant-directory', method: 'GET',
      data: { cityCode: 'zhili', keyword: 'A108', categories: '童装', page: 2, pageSize: 20 },
      suppressErrorToast: true,
    },
    {
      url: '/api/v1/merchants/merchant%2Fa%3Fb%23c%25/location-context', method: 'GET',
      suppressErrorToast: true,
    },
    {
      url: '/api/v1/merchants/merchant%2Fa%3Fb%23c%25/location-reports', method: 'POST',
      data: { reasonCode: 'navigation_inaccurate', description: '导航后反馈位置不准确' }, requireAuth: true,
    },
  ])
})

test('requires local login before sending protected API requests', async () => {
  const storage = new Map([['wplink_token', '   ']])
  const toasts = []
  const navigations = []
  let requestCalled = false

  globalThis.uni = {
    getStorageSync(key) {
      return storage.get(key) || ''
    },
    request() {
      requestCalled = true
    },
    showToast(options) {
      toasts.push(options)
    },
    navigateTo(options) {
      navigations.push(options)
    },
  }
  globalThis.getCurrentPages = () => [{ route: 'pages/resource/detail', options: { id: 'resource-1' } }]

  const requestModule = await loadWxappModule('api/request.js')
  const request = requestModule.namespace.default

  let error
  try {
    await request({ url: '/api/v1/me', requireAuth: true })
  } catch (err) {
    error = err
  }

  assert.equal(requestCalled, false)
  assert.equal(error?.message, '请先登录')
  assert.equal(error?.code, 'UNAUTHORIZED')
  assert.equal(error?.statusCode, 401)
  assert.deepEqual(toasts, [{ title: '请先登录', icon: 'none' }])
  assert.deepEqual(navigations, [
    { url: '/pages/login/index?redirect=%2Fpages%2Fresource%2Fdetail%3Fid%3Dresource-1' },
  ])
})

test('keeps public API requests anonymous without local token', async () => {
  const storage = new Map()
  const navigations = []
  let requestOptions

  globalThis.uni = {
    getStorageSync(key) {
      return storage.get(key) || ''
    },
    request(options) {
      requestOptions = options
      options.success({
        statusCode: 200,
        data: {
          data: { items: [{ id: 'resource-1' }] },
        },
      })
    },
    showToast() {},
    navigateTo(options) {
      navigations.push(options)
    },
  }
  globalThis.getCurrentPages = () => [{ route: 'pages/home/index', options: {} }]

  const requestModule = await loadWxappModule('api/request.js')
  const request = requestModule.namespace.default

  const resp = await request({ url: '/api/v1/home/resources', method: 'GET' })

  assert.deepEqual(resp, { items: [{ id: 'resource-1' }] })
  assert.equal(requestOptions.header.Authorization, '')
  assert.deepEqual(navigations, [])
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
