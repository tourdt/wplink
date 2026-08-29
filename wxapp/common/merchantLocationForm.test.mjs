import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'
import vm from 'node:vm'

import {
  buildMerchantLocationPayload,
  clearMerchantLocationForm,
  normalizeMerchantLocationForm,
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
