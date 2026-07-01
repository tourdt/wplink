import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const sourcePath = path.join(root, 'pages/verification/index.vue')

test('verification page collects recommended merchant certification materials', () => {
  const source = fs.readFileSync(sourcePath, 'utf8')

  assert.match(source, /<text class="section-title">主体资料<\/text>/)
  assert.match(source, /<text class="section-title">联系人和地址<\/text>/)
  assert.match(source, /<text class="section-title">经营实拍<\/text>/)
  assert.match(source, /<text class="section-title">补充证明<\/text>/)
  assert.match(source, /v-model="form\.socialCreditCode"/)
  assert.match(source, /v-model="form\.applicantName"/)
  assert.match(source, /<text class="field-label">联系人姓名<\/text>/)
  assert.match(source, /v-model="form\.contactPhone"/)
  assert.match(source, /v-model="form\.addressText"/)
  assert.match(source, /sceneUrl: form\.sceneUrl\.trim\(\)/)
  assert.match(source, /authorizationUrl: form\.authorizationUrl\.trim\(\)/)
  assert.match(source, /const verificationMaterials = buildVerificationMaterials\(\)/)
  assert.match(source, /materials: verificationMaterials/)
})

test('verification page keeps merchant id as hidden context', () => {
  const source = fs.readFileSync(sourcePath, 'utf8')

  assert.match(source, /form\.merchantId = options\.merchantId \|\| getMerchantId\(\)/)
  assert.doesNotMatch(source, /<text class="field-label">商家 ID<\/text>/)
  assert.doesNotMatch(source, /<input v-model="form\.merchantId"/)
  assert.match(source, /请先完成商家入驻/)
})

test('verification page reuses merchant profile identity instead of asking users to choose', () => {
  const source = fs.readFileSync(sourcePath, 'utf8')

  assert.match(source, /import \{ getMerchant \} from '\.\.\/\.\.\/api\/merchant'/)
  assert.match(source, /loadMerchantProfile\(\)/)
  assert.match(source, /form\.verificationType = detail\.merchantType \|\| form\.verificationType/)
  assert.doesNotMatch(source, /<text class="field-label">认证类型<\/text>/)
  assert.doesNotMatch(source, /<picker :range="typeOptions"/)
  assert.doesNotMatch(source, /工厂认证/)
  assert.doesNotMatch(source, /档口认证/)
  assert.doesNotMatch(source, /库存商认证/)
  assert.doesNotMatch(source, /服务商认证/)
})

test('verification page does not ask users to choose applicant role', () => {
  const source = fs.readFileSync(sourcePath, 'utf8')

  assert.doesNotMatch(source, /<text class="field-label">申请人身份<\/text>/)
  assert.doesNotMatch(source, /applicantRoleOptions/)
  assert.doesNotMatch(source, /changeApplicantRole/)
  assert.doesNotMatch(source, /currentApplicantRoleLabel/)
  assert.doesNotMatch(source, /applicantRole:/)
  assert.doesNotMatch(source, /applicantRoleLabel:/)
})

test('verification image uploads use simple tiles instead of manual url inputs', () => {
  const source = fs.readFileSync(sourcePath, 'utf8')

  assert.match(source, /class="proof-grid"/)
  assert.match(source, /class="proof-tile"/)
  assert.match(source, /proofUploadItems/)
  assert.match(source, /uploadProof\(item\)/)
  assert.match(source, /proof-plus-icon/)
  assert.doesNotMatch(source, /营业执照图片 URL/)
  assert.doesNotMatch(source, /门头或场地图片 URL/)
  assert.doesNotMatch(source, /车间、仓库、货架或设备图片 URL/)
  assert.doesNotMatch(source, /授权书或经办人证明 URL/)
  assert.doesNotMatch(source, /品牌授权、案例或服务资质 URL/)
  assert.doesNotMatch(source, /typeEvidenceHint/)
  assert.doesNotMatch(source, /sceneEvidenceHint/)
  assert.doesNotMatch(source, /section-summary/)
})

test('verification required markers are attached to required proof tiles only', () => {
  const source = fs.readFileSync(sourcePath, 'utf8')

  assert.doesNotMatch(source, /<text class="section-title">经营实拍<\/text>\s*<text class="required-badge">必填<\/text>/)
  assert.match(source, /<text v-if="item\.required" class="proof-required">必填<\/text>/)
  assert.match(source, /kind: 'license'[\s\S]*?label: '营业执照'[\s\S]*?required: true/)
  assert.match(source, /kind: 'storefront'[\s\S]*?label: '门头\/场地'[\s\S]*?required: true/)
  assert.match(source, /kind: 'scene'[\s\S]*?label: '经营实拍'[\s\S]*?required: false/)
})

test('verification page validates required certification evidence before submitting', () => {
  const source = fs.readFileSync(sourcePath, 'utf8')

  assert.match(source, /请填写统一社会信用代码/)
  assert.match(source, /请上传营业执照/)
  assert.match(source, /请填写联系人姓名/)
  assert.match(source, /请填写经营地址/)
  assert.match(source, /请上传门头或场地照片/)
  assert.match(source, /请勾选资料真实性承诺/)
  assert.match(source, /if \(!form\.businessName\.trim\(\)\) return '请填写营业主体名称'/)
  assert.match(source, /function sanitizeContactPhone/)
})
