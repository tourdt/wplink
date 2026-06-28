import assert from 'node:assert/strict'
import test from 'node:test'

import { buildApiUrl, normalizeApiBaseUrl } from './url.js'

test('builds absolute local API URL when base url is empty', () => {
  assert.equal(buildApiUrl('', '/api/v1/home/banners'), 'http://127.0.0.1:4000/api/v1/home/banners')
})

test('joins configured API base URL without duplicate slashes', () => {
  assert.equal(buildApiUrl('https://api.example.com/', '/api/v1/home/banners'), 'https://api.example.com/api/v1/home/banners')
})

test('normalizes configured API base URL', () => {
  assert.equal(normalizeApiBaseUrl(' https://api.example.com/ '), 'https://api.example.com')
})
