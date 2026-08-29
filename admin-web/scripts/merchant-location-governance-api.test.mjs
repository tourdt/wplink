import assert from 'node:assert/strict'
import test from 'node:test'

import http, { classifyHttpError } from '../src/api/http.js'
import * as merchantApi from '../src/api/merchant.js'
import { listMerchants, merchantLocationGovernanceApi } from '../src/api/merchant.js'

globalThis.localStorage = {
  getItem: () => '',
  removeItem: () => {},
}

test('http error policy keeps ordinary errors global, governance errors local, and 401 unified', () => {
  assert.equal(classifyHttpError({ config: {}, response: { status: 500, data: {} } }), 'global')
  assert.equal(classifyHttpError({ config: { skipGlobalErrorMessage: true }, response: { status: 500, data: {} } }), 'page')
  assert.equal(classifyHttpError({ config: { skipGlobalErrorMessage: true }, response: { status: 401, data: {} } }), 'unauthorized')
})

test('merchant location governance API marks only its own requests as page-handled', async () => {
  const requests = []
  const originalAdapter = http.defaults.adapter
  http.defaults.adapter = async (config) => {
    requests.push(config)
    return { data: { data: { items: [], total: 0 } }, status: 200, statusText: 'OK', headers: {}, config }
  }

  try {
    await listMerchants({ cityCode: 'zhili' })
    await merchantLocationGovernanceApi.listMerchants({ cityCode: 'zhili' })
    await merchantLocationGovernanceApi.listMerchantLocationReports({ status: 'pending' })
    await merchantLocationGovernanceApi.reviewMerchantLocationReport('r-1', { status: 'valid' })
    await merchantLocationGovernanceApi.updateMerchantLocationStatus('m-1', { action: 'hide' })
  } finally {
    http.defaults.adapter = originalAdapter
  }

  assert.equal(requests[0].skipGlobalErrorMessage, undefined)
  assert.deepEqual(requests.slice(1).map((config) => config.skipGlobalErrorMessage), [true, true, true, true])
  assert.deepEqual(requests.map((config) => config.url), [
    '/api/v1/admin/merchants',
    '/api/v1/admin/merchants',
    '/api/v1/admin/merchants/location-reports',
    '/api/v1/admin/merchants/location-reports/r-1/review',
    '/api/v1/admin/merchants/m-1/location-status',
  ])
})

test('merchant admin API exposes governance endpoints instead of public merchant writes', () => {
  assert.equal('createMerchant' in merchantApi, false)
  assert.equal('updateMerchant' in merchantApi, false)
  assert.equal(typeof merchantApi.listMerchants, 'function')
  assert.equal(typeof merchantApi.merchantLocationGovernanceApi.updateMerchantLocationStatus, 'function')
})
