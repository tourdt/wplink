import assert from 'node:assert/strict'
import test from 'node:test'

import { buildQiniuUploadErrorMessage, buildUploadedFileUrl, inferContentType } from './uploadUtils.js'

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

test('builds qiniu upload error message from response json', () => {
  assert.equal(
    buildQiniuUploadErrorMessage(631, '{"error":"bucket not found"}'),
    '封面上传失败（七牛 631：bucket not found）',
  )
})

test('builds qiniu upload error message from response text', () => {
  assert.equal(
    buildQiniuUploadErrorMessage(631, 'bucket not found'),
    '封面上传失败（七牛 631：bucket not found）',
  )
})
