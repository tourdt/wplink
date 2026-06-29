import assert from 'node:assert/strict'
import {
  appendMerchantImageFiles,
  createPendingMerchantImageEntry,
  createStoredMerchantImageEntry,
  getMerchantImagePreviewUrl,
  getMerchantImageUrlsForPreview,
  removeMerchantImageEntry,
  resolveImageCompressionOptions,
} from './merchantProfileImages.js'

const stored = createStoredMerchantImageEntry('https://cdn.example.com/a.jpg')
const pending = createPendingMerchantImageEntry({
  id: 'local-1',
  path: '/tmp/b.jpg',
  fileName: 'b.jpg',
  contentType: 'image/jpeg',
  size: 600_000,
})

assert.equal(stored.kind, 'stored')
assert.equal(stored.url, 'https://cdn.example.com/a.jpg')
assert.equal(pending.kind, 'pending')
assert.equal(pending.url, '/tmp/b.jpg')
assert.equal(getMerchantImagePreviewUrl(stored), 'https://cdn.example.com/a.jpg')
assert.equal(getMerchantImagePreviewUrl(pending), '/tmp/b.jpg')

assert.deepEqual(getMerchantImageUrlsForPreview([stored, pending]), [
  'https://cdn.example.com/a.jpg',
  '/tmp/b.jpg',
])

const appended = appendMerchantImageFiles([stored], [
  { id: 'local-2', path: '/tmp/c.jpg', size: 1 },
  { id: 'local-3', path: '/tmp/d.jpg', size: 1 },
], 2)
assert.equal(appended.length, 2)
assert.equal(appended[1].id, 'local-2')

assert.deepEqual(removeMerchantImageEntry([stored, pending], pending.id), [stored])

assert.deepEqual(resolveImageCompressionOptions({
  width: 1600,
  height: 900,
  size: 600_000,
}), {
  shouldCompress: true,
  compressedWidth: 1280,
  quality: 85,
})

assert.deepEqual(resolveImageCompressionOptions({
  width: 900,
  height: 1200,
  size: 300_000,
}), {
  shouldCompress: false,
  compressedWidth: undefined,
  quality: 90,
})
