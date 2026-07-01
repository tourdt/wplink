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

test('resource detail keeps contact reminder friendly and visually quiet', () => {
  assert.match(source, /<text class="section-title">友情提示<\/text>/)
  assert.match(source, /<text class="section-content contact-tip-content">联系商家前，建议先确认实物、价格、数量和交付方式。<\/text>/)
  assert.match(source, /\.contact-tip-content \{[\s\S]*font-size: 26rpx;[\s\S]*line-height: 1\.5;[\s\S]*\}/)
  assert.equal(source.includes('平台已记录联系行为'), false)
  assert.equal(source.includes('<text class="section-title">联系提示</text>'), false)
})

test('own resource detail keeps share and management actions in the bottom bar', () => {
  assert.match(source, /<view v-if="isOwnResource" class="owner-action-bar">/)
  assert.match(source, /<button class="share-button" @click="shareOwnResource" :open-type="canShareOwnResource \? 'share' : ''">分享<\/button>/)
  assert.match(source, /const canShareOwnResource = computed\(\(\) => resource\.value\.status === 'published' && !isExpiredResource\.value && !resource\.value\.dealtAt\)/)
  assert.match(source, /<button class="primary-button" @click="openManagementSheet">管理<\/button>/)
  assert.match(source, /<view v-else class="contact-bar">/)
  assert.doesNotMatch(source, /这是你发布的资源，可在我的发布中管理/)
})

test('pending own resource management sheet only explains review state', () => {
  assert.match(source, /const managementNotice = computed\(\(\) => \{[\s\S]*resource\.value\.status === 'pending'[\s\S]*资源正在审核，审核通过后会公开展示。当前暂不能刷新、置顶、下架或分享。[\s\S]*\}\)/)
  assert.match(source, /<view v-if="showManagementSheet" class="sheet-mask" @click="closeManagementSheet">/)
  assert.match(source, /<text class="sheet-title">\{\{ managementTitle \}\}<\/text>/)
  assert.match(source, /<text v-if="!managementActions\.length" class="sheet-desc">\{\{ managementNotice \}\}<\/text>/)
  assert.match(source, /<view v-if="managementActions\.length" class="management-actions">/)
  assert.match(source, /<text v-else class="empty-management">暂无可操作功能<\/text>/)
  assert.doesNotMatch(source, /<text class="sheet-desc">\{\{ managementNotice \}\}<\/text>/)
  assert.doesNotMatch(source, /查看我的发布/)
  assert.doesNotMatch(source, /返回列表/)
})

test('own resource management sheet does not expose deal action', () => {
  assert.doesNotMatch(source, /标记成交/)
  assert.doesNotMatch(source, /key: 'deal'/)
  assert.doesNotMatch(source, /markOwnResourceDealt/)
  assert.doesNotMatch(source, /markResourceDeal/)
})

test('resource detail unlocks contact through backend before copy or call', () => {
  assert.match(source, /import \{ requireLogin \} from '\.\.\/\.\.\/common\/auth'/)
  assert.match(source, /if \(isContactUnlockAction\(action\) && !requireLogin\(\)\) return false/)
  assert.match(source, /function isContactUnlockAction\(action\) \{[\s\S]*return action === 'phone' \|\| action === 'wechat'[\s\S]*\}/)
  assert.match(source, /const resp = await recordResourceContact\(resource\.value\.id, action\)/)
  assert.match(source, /return resp \|\| \{\}/)
  assert.match(source, /uni\.setClipboardData\(\{ data: resp\.wechat \}\)/)
  assert.match(source, /uni\.makePhoneCall\(\{ phoneNumber: resp\.phone \}\)/)
  assert.match(source, /微信号已复制/)
  assert.equal(source.includes('已记录联系，完整微信由平台保护'), false)
  assert.equal(source.includes('已记录联系，完整电话由平台保护'), false)
})
