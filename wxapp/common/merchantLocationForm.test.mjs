import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'
import vm from 'node:vm'

import {
  buildMerchantLocationPayload,
  buildMerchantLocationSubmission,
  clearMerchantLocationForm,
  createMerchantLocationSession,
  isMerchantLocationRequestCurrent,
  normalizeMerchantLocationForm,
  reduceMerchantLocationSession,
  resolveLatestMerchantLocationRequest,
} from './merchantLocationForm.js'

function loadMerchantApi(request) {
  const sourcePath = path.resolve(new URL('../api/merchant.js', import.meta.url).pathname)
  const source = fs.readFileSync(sourcePath, 'utf8')
    .replace(/import request from ['"].*?['"]\s*/, '')
    .replaceAll('export function ', 'function ')
  const sandbox = { request }
  vm.runInNewContext(`${source}\nglobalThis.merchantApi = { updateMerchant }`, sandbox)
  return sandbox.merchantApi
}

function deferred() {
  let resolve
  let reject
  const promise = new Promise((nextResolve, nextReject) => {
    resolve = nextResolve
    reject = nextReject
  })
  return { promise, resolve, reject }
}

const clearedForm = {
  marketName: '',
  buildingName: '',
  floorNo: '',
  boothNo: '',
  latitude: null,
  longitude: null,
  locationName: '',
}

const clearedPayload = {
  locationAction: 'clear',
  marketName: '',
  buildingName: '',
  floorNo: '',
  boothNo: '',
  latitude: 0,
  longitude: 0,
  locationName: '',
}

test('buildMerchantLocationPayload sets a complete selected location', () => {
  assert.deepEqual(buildMerchantLocationPayload({
    marketName: '织里童装城',
    buildingName: '一期',
    floorNo: '1F',
    boothNo: 'A108',
    latitude: 30.8732,
    longitude: 120.2255,
    locationName: '织里童装城',
  }), {
    locationAction: 'set',
    marketName: '织里童装城',
    buildingName: '一期',
    floorNo: '1F',
    boothNo: 'A108',
    latitude: 30.8732,
    longitude: 120.2255,
    locationName: '织里童装城',
  })
})

test('buildMerchantLocationPayload clears incomplete coordinates', () => {
  assert.deepEqual(buildMerchantLocationPayload({ latitude: 30.8732 }), clearedPayload)
})

test('buildMerchantLocationPayload clears non-finite and out-of-range coordinate pairs', () => {
  for (const coordinates of [
    { latitude: Number.NaN, longitude: 120.2255 },
    { latitude: Number.POSITIVE_INFINITY, longitude: 120.2255 },
    { latitude: 90.0001, longitude: 120.2255 },
    { latitude: 30.8732, longitude: -180.0001 },
    { latitude: '  ', longitude: 120.2255 },
  ]) {
    assert.deepEqual(buildMerchantLocationPayload({
      marketName: '不能残留的市场',
      boothNo: 'A108',
      ...coordinates,
    }), clearedPayload)
  }
})

test('buildMerchantLocationPayload keeps valid zero coordinates and trims structure text', () => {
  assert.deepEqual(buildMerchantLocationPayload({
    marketName: ' 织里童装城 ',
    buildingName: ' 一期 ',
    floorNo: ' 1F ',
    boothNo: ' A108 ',
    latitude: 0,
    longitude: '0',
    locationName: ' 童装城北门 ',
  }), {
    locationAction: 'set',
    marketName: '织里童装城',
    buildingName: '一期',
    floorNo: '1F',
    boothNo: 'A108',
    latitude: 0,
    longitude: 0,
    locationName: '童装城北门',
  })
})

test('normalizeMerchantLocationForm reads only the structured location contract', () => {
  assert.deepEqual(normalizeMerchantLocationForm({
    location: {
      marketName: '织里童装城',
      buildingName: '一期',
      floorNo: '1F',
      boothNo: 'A108',
      latitude: 30.8732,
      longitude: 120.2255,
      locationName: '织里童装城',
      lat: 31,
      lng: 121,
      arbitraryKey: '不得读取',
    },
  }), {
    marketName: '织里童装城',
    buildingName: '一期',
    floorNo: '1F',
    boothNo: 'A108',
    latitude: 30.8732,
    longitude: 120.2255,
    locationName: '织里童装城',
  })

  assert.deepEqual(normalizeMerchantLocationForm({
    location: { lat: 30.8732, lng: 120.2255, name: '旧 JSON 位置' },
  }), clearedForm)
})

test('normalizeMerchantLocationForm and clearMerchantLocationForm clear every structure field as one unit', () => {
  const input = {
    marketName: '织里童装城',
    buildingName: '一期',
    floorNo: '1F',
    boothNo: 'A108',
    latitude: 30.8732,
    longitude: 120.2255,
    locationName: '织里童装城',
  }

  assert.deepEqual(clearMerchantLocationForm(input), clearedForm)
  assert.deepEqual(input, {
    marketName: '织里童装城',
    buildingName: '一期',
    floorNo: '1F',
    boothNo: 'A108',
    latitude: 30.8732,
    longitude: 120.2255,
    locationName: '织里童装城',
  })
  assert.deepEqual(normalizeMerchantLocationForm({
    location: { marketName: '织里童装城', latitude: 30.8732 },
  }), clearedForm)
})

test('updateMerchant sends only the precise profile and structured location contract', async () => {
  let requested
  const { updateMerchant } = loadMerchantApi((options) => {
    requested = options
    return Promise.resolve({ id: 'merchant-1' })
  })

  await updateMerchant('merchant-1', {
      name: '晨星童装',
      mainCategories: ['童装'],
      merchantType: 'factory',
      description: '源头童装',
      logoUrl: 'https://cdn.example.com/logo.jpg',
      images: ['https://cdn.example.com/shop.jpg'],
      contactName: '陈女士',
      contactPhone: '13800000000',
      contactWechat: 'chen_01',
      addressText: '织里童装城一期 A108',
      locationAction: 'set',
      marketName: '织里童装城',
      buildingName: '一期',
      floorNo: '1F',
      boothNo: 'A108',
      latitude: 30.8732,
      longitude: 120.2255,
      locationName: '织里童装城',
      location: { lat: 31, lng: 121 },
      arbitraryKey: '不得发送',
  })

  assert.deepEqual(JSON.parse(JSON.stringify(requested.data)), {
    name: '晨星童装',
    mainCategories: ['童装'],
    merchantType: 'factory',
    description: '源头童装',
    logoUrl: 'https://cdn.example.com/logo.jpg',
    images: ['https://cdn.example.com/shop.jpg'],
    contactName: '陈女士',
    contactPhone: '13800000000',
    contactWechat: 'chen_01',
    addressText: '织里童装城一期 A108',
    locationAction: 'set',
    marketName: '织里童装城',
    buildingName: '一期',
    floorNo: '1F',
    boothNo: 'A108',
    latitude: 30.8732,
    longitude: 120.2255,
    locationName: '织里童装城',
  })
})

test('risk-hidden retained coordinates require an explicit location decision before profile save', () => {
  const form = normalizeMerchantLocationForm({
    location: {
      status: 'risk_hidden',
      latitude: 30.8732,
      longitude: 120.2255,
      locationName: '织里童装城',
    },
  })
  const session = createMerchantLocationSession({ status: 'risk_hidden', form })

  assert.deepEqual(session, {
    intent: 'decision_required',
    resolving: false,
    requestToken: 0,
  })
  assert.deepEqual(buildMerchantLocationSubmission(session, form), {
    ok: false,
    message: '该位置暂未公开，请重新选择地图位置或清除地图位置后再保存',
  })
})

test('risk-hidden location submits set only after the latest selection succeeds', () => {
  const form = {
    marketName: '织里童装城',
    latitude: 30.8732,
    longitude: 120.2255,
    locationName: '织里童装城',
  }
  let session = createMerchantLocationSession({ status: 'risk_hidden', form })
  session = reduceMerchantLocationSession(session, { type: 'selection_started' })
  const token = session.requestToken
  session = reduceMerchantLocationSession(session, { type: 'selection_succeeded', token })

  assert.deepEqual(buildMerchantLocationSubmission(session, form), {
    ok: true,
    payload: {
      locationAction: 'set',
      marketName: '织里童装城',
      buildingName: '',
      floorNo: '',
      boothNo: '',
      latitude: 30.8732,
      longitude: 120.2255,
      locationName: '织里童装城',
    },
  })
})

test('clear and address edits invalidate pending resolution and submit a complete clear', () => {
  const form = {
    marketName: '织里童装城',
    latitude: 30.8732,
    longitude: 120.2255,
    locationName: '织里童装城',
  }
  for (const type of ['location_cleared', 'address_changed']) {
    let session = createMerchantLocationSession({ status: 'risk_hidden', form })
    session = reduceMerchantLocationSession(session, { type: 'selection_started' })
    const staleToken = session.requestToken
    session = reduceMerchantLocationSession(session, { type })

    assert.equal(session.intent, 'clear')
    assert.equal(session.resolving, false)
    assert.equal(isMerchantLocationRequestCurrent(session, staleToken), false)
    assert.deepEqual(buildMerchantLocationSubmission(session, form), {
      ok: true,
      payload: clearedPayload,
    })
  }
})

test('profile save is blocked while the latest map location is resolving', () => {
  const form = { latitude: 30.8732, longitude: 120.2255 }
  let session = createMerchantLocationSession({ status: 'visible', form })
  session = reduceMerchantLocationSession(session, { type: 'selection_started' })

  assert.deepEqual(buildMerchantLocationSubmission(session, form), {
    ok: false,
    message: '地图位置解析中，请稍候再保存',
  })
})

test('a cleared location ignores an older reverse-geocode success', async () => {
  const pending = deferred()
  let session = createMerchantLocationSession({ status: 'visible', form: {
    latitude: 30.8732,
    longitude: 120.2255,
  } })
  session = reduceMerchantLocationSession(session, { type: 'selection_started' })
  const token = session.requestToken
  const resultPromise = resolveLatestMerchantLocationRequest({
    token,
    task: () => pending.promise,
    getSession: () => session,
  })

  session = reduceMerchantLocationSession(session, { type: 'location_cleared' })
  pending.resolve({ address: '不应写回的旧地址' })

  assert.deepEqual(await resultPromise, { current: false })
  assert.equal(session.intent, 'clear')
})

test('a second selection wins when reverse-geocode promises settle out of order', async () => {
  const first = deferred()
  const second = deferred()
  let session = createMerchantLocationSession({ status: 'hidden', form: clearedForm })

  session = reduceMerchantLocationSession(session, { type: 'selection_started' })
  const firstToken = session.requestToken
  const firstResult = resolveLatestMerchantLocationRequest({
    token: firstToken,
    task: () => first.promise,
    getSession: () => session,
  })

  session = reduceMerchantLocationSession(session, { type: 'selection_started' })
  const secondToken = session.requestToken
  const secondResult = resolveLatestMerchantLocationRequest({
    token: secondToken,
    task: () => second.promise,
    getSession: () => session,
  })

  first.resolve({ address: '第一次旧地址' })
  assert.deepEqual(await firstResult, { current: false })
  assert.equal(session.requestToken, secondToken)
  assert.equal(session.resolving, true)

  second.resolve({ address: '第二次新地址' })
  assert.deepEqual(await secondResult, {
    current: true,
    value: { address: '第二次新地址' },
  })
  session = reduceMerchantLocationSession(session, { type: 'selection_succeeded', token: secondToken })
  assert.equal(session.intent, 'set')
  assert.equal(session.resolving, false)
})

test('a stale reverse-geocode failure cannot cancel a newer selection', async () => {
  const first = deferred()
  let session = createMerchantLocationSession({ status: 'hidden', form: clearedForm })
  session = reduceMerchantLocationSession(session, { type: 'selection_started' })
  const firstToken = session.requestToken
  const firstResult = resolveLatestMerchantLocationRequest({
    token: firstToken,
    task: () => first.promise,
    getSession: () => session,
  })

  session = reduceMerchantLocationSession(session, { type: 'selection_started' })
  const secondToken = session.requestToken
  first.reject(new Error('旧请求失败'))

  assert.deepEqual(await firstResult, { current: false })
  assert.equal(session.requestToken, secondToken)
  assert.equal(session.resolving, true)
})

test('cancelling location selection keeps the previous explicit intent and invalidates its token', () => {
  const form = { latitude: 30.8732, longitude: 120.2255 }
  let session = createMerchantLocationSession({ status: 'visible', form })
  session = reduceMerchantLocationSession(session, { type: 'selection_started' })
  const token = session.requestToken
  session = reduceMerchantLocationSession(session, { type: 'selection_cancelled', token })

  assert.equal(session.intent, 'set')
  assert.equal(session.resolving, false)
  assert.equal(isMerchantLocationRequestCurrent(session, token), false)
})

test('a failed reselection keeps a risk-hidden location undecided instead of clearing it implicitly', () => {
  const form = { latitude: 30.8732, longitude: 120.2255, locationName: '织里童装城' }
  let session = createMerchantLocationSession({ status: 'risk_hidden', form })
  session = reduceMerchantLocationSession(session, { type: 'selection_started' })
  const token = session.requestToken
  session = reduceMerchantLocationSession(session, { type: 'selection_failed', token })

  assert.equal(session.intent, 'decision_required')
  assert.equal(session.resolving, false)
  assert.deepEqual(buildMerchantLocationSubmission(session, form), {
    ok: false,
    message: '该位置暂未公开，请重新选择地图位置或清除地图位置后再保存',
  })
})
