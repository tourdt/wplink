import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const source = fs.readFileSync(path.join(root, 'pages/resource/detail.vue'), 'utf8')

test('resource detail gallery uses banner swiper and full screen preview', () => {
  assert.match(source, /const selectedGalleryIndex = ref\(0\)/)
  assert.match(source, /<swiper[\s\S]*v-if="galleryImages\.length > 1"[\s\S]*:current="selectedGalleryIndex"[\s\S]*@change="handleGalleryChange"/)
  assert.match(source, /<swiper-item[\s\S]*v-for="\(\s*url,\s*index\s*\) in galleryImages"/)
  assert.match(source, /@click="previewGalleryImage\(index\)"/)
  assert.match(source, /v-else-if="mainImage"[\s\S]*@click="previewGalleryImage\(0\)"/)
  assert.match(source, /function handleGalleryChange\(event\) \{[\s\S]*selectedGalleryIndex\.value = current[\s\S]*\}/)
  assert.match(source, /function previewGalleryImage\(index = selectedGalleryIndex\.value\) \{[\s\S]*uni\.previewImage\(\{[\s\S]*current,[\s\S]*urls: galleryImages\.value[\s\S]*\}\)/)
  assert.equal(source.includes('gallery-strip'), false)
  assert.equal(source.includes('gallery-thumb'), false)
  assert.equal(source.includes('selectGalleryImage'), false)
})
