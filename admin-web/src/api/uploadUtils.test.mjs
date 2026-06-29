import assert from 'node:assert/strict'
import test from 'node:test'

import { buildUploadedFileUrl, inferContentType } from './uploadUtils.js'

test('builds uploaded file url from public base url and object key', () => {
  assert.equal(
    buildUploadedFileUrl({ publicBaseUrl: 'https://cdn.example.com/', objectKey: '/uploads/banner/a.png' }),
    'https://cdn.example.com/uploads/banner/a.png',
  )
})

test('infers image content type from file name', () => {
  assert.equal(inferContentType({ name: 'banner.webp', type: '' }), 'image/webp')
  assert.equal(inferContentType({ name: 'banner.png', type: '' }), 'image/png')
  assert.equal(inferContentType({ name: 'banner.jpg', type: '' }), 'image/jpeg')
})
