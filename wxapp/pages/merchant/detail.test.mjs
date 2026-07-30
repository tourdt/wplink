import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const sourcePath = path.join(root, 'pages/merchant/detail.vue')

test('merchant detail page does not show verification wording', () => {
  const source = fs.readFileSync(sourcePath, 'utf8')

  assert.doesNotMatch(source, />认证</)
  assert.doesNotMatch(source, /已认证/)
  assert.doesNotMatch(source, /已核实/)
  assert.doesNotMatch(source, /核验项/)
  assert.doesNotMatch(source, /主体资质、经营场地/)
  assert.doesNotMatch(source, /有效期/)
  assert.doesNotMatch(source, /showVerificationInfo/)
  assert.doesNotMatch(source, /merchantVerification/)
  assert.doesNotMatch(source, /verification-info/)
  assert.doesNotMatch(source, /formatDateToDay/)
  assert.doesNotMatch(source, /licenseUrl/)
  assert.doesNotMatch(source, /socialCreditCode/)
  assert.doesNotMatch(source, /businessName/)
})

test('merchant detail page keeps full resources at the bottom without overview card', () => {
  const source = fs.readFileSync(sourcePath, 'utf8')

  assert.doesNotMatch(source, /class="supply-overview-panel"/)
  assert.doesNotMatch(source, />供应概览</)
  assert.doesNotMatch(source, /scrollToResourceList/)
  assert.doesNotMatch(source, /uni\.pageScrollTo/)
  assert.match(source, /label: '公开供应'/)
  assert.doesNotMatch(source, /label: '在售供应'/)
  assert.match(source, /class="section resource-list-section"/)
  assert.match(source, />公开供应</)
  assert.doesNotMatch(source, />全部公开供应</)
  assert.match(source, /:empty-text="merchantResourcesEmptyText"/)

  const profileIndex = source.indexOf('class="profile-panel"')
  const trustNoteIndex = source.indexOf('class="section trust-note-section"')
  const resourceListIndex = source.indexOf('class="section resource-list-section"')

  assert.ok(profileIndex > -1)
  assert.ok(trustNoteIndex > profileIndex)
  assert.ok(resourceListIndex > trustNoteIndex)
})

test('merchant detail page renders address as a lightweight single-address block', () => {
  const source = fs.readFileSync(sourcePath, 'utf8')

  assert.match(source, /<view class="section merchant-address-section" v-if="merchantAddressLocation">/)
  assert.doesNotMatch(source, /merchant-address-card/)
  assert.doesNotMatch(source, /merchant-address-head/)
  assert.doesNotMatch(source, />经营地址</)
  assert.doesNotMatch(source, /merchant-address-title/)
  assert.match(source, /<view class="section-head">[\s\S]*<text class="section-title">地址<\/text>[\s\S]*<button/)
  assert.match(source, /<button v-if="merchantAddressLocation\.hasGps" class="address-action" @click="openMerchantLocation">导航<\/button>/)
  assert.match(source, /<button v-else class="address-action secondary" @click="copyMerchantAddress\(\)">复制<\/button>/)
  assert.match(source, /<text class="merchant-address-text">\{\{ merchantAddressLocation\.address \}\}<\/text>/)
  assert.match(source, /<map[\s\S]*v-if="merchantAddressLocation\.hasGps"[\s\S]*class="merchant-address-map"[\s\S]*:latitude="merchantAddressLocation\.latitude"[\s\S]*:longitude="merchantAddressLocation\.longitude"[\s\S]*@tap="openMerchantLocation"/)
  assert.match(source, /const merchantAddressLocation = computed\(\(\) => buildMerchantAddressLocation\(\)\)/)
  assert.match(source, /function buildMerchantAddressLocation\(\) \{[\s\S]*const hasGps = Number\.isFinite\(latitude\) && Number\.isFinite\(longitude\)[\s\S]*markers: \[\{/)
  assert.match(source, /function openMerchantLocation\(\) \{[\s\S]*uni\.openLocation\(\{[\s\S]*scale: 18[\s\S]*导航打开失败，已复制地址/)
  assert.match(source, /function copyMerchantAddress\(title = '地址已复制'\) \{[\s\S]*uni\.setClipboardData\(\{ data: location\.address \}\)/)
  assert.match(source, /\.merchant-address-section \{[\s\S]*display: grid;[\s\S]*gap: 14rpx;/)
  assert.match(source, /\.merchant-address-section \.section-head \{[\s\S]*margin-bottom: 0;/)
  assert.match(source, /\.merchant-address-text \{[\s\S]*display: block;[\s\S]*font-size: 30rpx;[\s\S]*line-height: 1\.55;[\s\S]*word-break: break-word;/)
})

test('merchant detail page only directs users to contact details in published supply and demand', () => {
  const source = fs.readFileSync(sourcePath, 'utf8')

  assert.doesNotMatch(source, /merchant\.contact/)
  assert.match(source, /联系方式仅随有效供需信息展示/)
})
