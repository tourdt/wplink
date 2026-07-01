import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const sourcePath = path.join(root, 'pages/verification/index.vue')

test('verification page collects recommended merchant certification materials', () => {
  const source = fs.readFileSync(sourcePath, 'utf8')

  assert.match(source, /<text class="section-title">主体资料<\/text>/)
  assert.match(source, /<text class="section-title">联系信息<\/text>/)
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

test('verification page highlights limited free billing window', () => {
  const source = fs.readFileSync(sourcePath, 'utf8')

  assert.match(source, /<text class="status-meta" v-if="showBillingSummary">\{\{ billingSummary \}\}<\/text>/)
  assert.match(source, /const showBillingSummary = computed\(\(\) => !isLimitedFreeActive\.value && !isVerificationVerified\.value\)/)
  assert.match(source, /v-if="isLimitedFreeActive"/)
  assert.match(source, /限时免费/)
  assert.match(source, /原认证费/)
  assert.match(source, /审核通过后免费生效/)
  assert.match(source, /class="primary-button submit-button"/)
  assert.match(source, /class="submit-button-main">\{\{ submitButtonMainText \}\}<\/text>/)
  assert.match(source, /<text v-if="submitButtonSubText" class="submit-button-sub">\{\{ submitButtonSubText \}\}<\/text>/)
  assert.doesNotMatch(source, /submitButtonText/)
})

test('verification status reviewed time displays yyyy-mm-dd only', () => {
  const source = fs.readFileSync(sourcePath, 'utf8')

  assert.match(source, /import \{ formatDateToDay \} from '\.\.\/\.\.\/common\/date'/)
  assert.match(source, /const verificationReviewedDate = computed\(\(\) => formatDateToDay\(latestVerification\.value\.reviewedAt, '等待审核'\)\)/)
  assert.match(source, /\{\{ typeLabel\(latestVerification\.verificationType\) \}\} · \{\{ verificationReviewedDate \}\}/)
  assert.doesNotMatch(source, /latestVerification\.reviewedAt \|\| '等待审核'/)
})

test('verification commitment is a separate submit confirmation row', () => {
  const source = fs.readFileSync(sourcePath, 'utf8')

  assert.match(source, /class="form-section commitment-section"/)
  assert.match(source, /class="commitment-checkbox"/)
  assert.match(source, /class="commitment-text">我承诺资料真实有效。<\/text>/)
  assert.match(source, /<text class="section-title">补充证明<\/text>[\s\S]*?<\/view>\s*<\/view>\s*<view class="form-section commitment-section">/)
})

test('verification pending state shows review progress instead of editable form', () => {
  const source = fs.readFileSync(sourcePath, 'utf8')

  assert.match(source, /const isVerificationPending = computed\(\(\) => latestVerification\.value\.status === 'pending'\)/)
  assert.match(source, /const showSubmitBar = computed\(\(\) => showVerificationForm\.value\)/)
  assert.match(source, /<view v-if="isVerificationPending" class="review-progress-card">/)
  assert.match(source, /资料已提交/)
  assert.match(source, /平台正在审核/)
  assert.match(source, /审核结果会通过站内消息通知/)
  assert.match(source, /const showVerificationForm = computed\(\(\) => !isVerificationPending\.value && \(!isVerificationVerified\.value \|\| changingVerifiedCertification\.value\)/)
  assert.match(source, /<view v-if="showVerificationForm" class="form-card">/)
  assert.match(source, /<view v-if="showSubmitBar" class="fixed-save-spacer" \/>/)
  assert.match(source, /<view v-if="showSubmitBar" class="fixed-save-bar">/)
})

test('verification rejected state shows the review note to the merchant', () => {
  const source = fs.readFileSync(sourcePath, 'utf8')

  assert.match(source, /const isVerificationRejected = computed\(\(\) => latestVerification\.value\.status === 'rejected'\)/)
  assert.match(source, /const verificationRejectReason = computed/)
  assert.match(source, /latestVerification\.value\.reviewNote/)
  assert.match(source, /v-if="isVerificationRejected"/)
  assert.match(source, /认证未通过/)
  assert.match(source, /驳回原因/)
  assert.match(source, /请按原因修改资料后重新提交/)
})

test('verification verified state asks merchants to start a change request before editing', () => {
  const source = fs.readFileSync(sourcePath, 'utf8')

  assert.match(source, /const changingVerifiedCertification = ref\(false\)/)
  assert.match(source, /const isVerificationVerified = computed\(\(\) => latestVerification\.value\.status === 'verified'\)/)
  assert.match(source, /const showVerifiedSummary = computed\(\(\) => isVerificationVerified\.value && !changingVerifiedCertification\.value\)/)
  assert.match(source, /<view v-if="showVerifiedSummary" class="verified-summary-card">/)
  assert.match(source, /认证已通过/)
  assert.match(source, /认证资料已生效/)
  assert.match(source, /变更认证资料/)
  assert.match(source, /function startCertificationChange\(\)/)
  assert.match(source, /changingVerifiedCertification\.value = true/)
  assert.match(source, /提交变更审核/)
})

test('verification change request prefills the last certified materials', () => {
  const source = fs.readFileSync(sourcePath, 'utf8')

  assert.match(source, /function applyVerificationFormDefaults\(verification\)/)
  assert.match(source, /const materials = verification\.materials \|\| \{\}/)
  assert.match(source, /form\.businessName = verification\.businessName \|\| form\.businessName/)
  assert.match(source, /form\.licenseUrl = verification\.licenseUrl \|\| form\.licenseUrl/)
  assert.match(source, /form\.storefrontUrl = verification\.storefrontUrl \|\| form\.storefrontUrl/)
  assert.match(source, /form\.socialCreditCode = String\(materials\.socialCreditCode \|\| form\.socialCreditCode \|\| ''\)/)
  assert.match(source, /form\.applicantName = String\(materials\.applicantName \|\| form\.applicantName \|\| ''\)/)
  assert.match(source, /form\.contactPhone = sanitizeContactPhoneValue\(materials\.contactPhone \|\| form\.contactPhone\)/)
  assert.match(source, /form\.addressText = String\(materials\.addressText \|\| form\.addressText \|\| ''\)/)
  assert.match(source, /if \(latest\.status === 'verified'\) applyVerificationFormDefaults\(latest\)/)
  assert.match(source, /applyVerificationFormDefaults\(latestVerification\.value\)[\s\S]*changingVerifiedCertification\.value = true/)
})

test('verification submit button shows loading feedback while submitting', () => {
  const source = fs.readFileSync(sourcePath, 'utf8')

  assert.match(source, /const submitting = ref\(false\)/)
  assert.match(source, /:disabled="submitting"/)
  assert.match(source, /submit-button-loading/)
  assert.match(source, /<view v-if="submitting" class="submit-spinner" \/>/)
  assert.match(source, /submitButtonMainText/)
  assert.match(source, /正在提交/)
  assert.match(source, /submitButtonSubText/)
  assert.match(source, /if \(submitting\.value\) return/)
  assert.match(source, /submitting\.value = true/)
  assert.match(source, /finally \{[\s\S]*?submitting\.value = false[\s\S]*?\}/)
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

test('verification images are uploaded only when submitting certification', () => {
  const source = fs.readFileSync(sourcePath, 'utf8')

  assert.match(source, /import \{ chooseImageFile, uploadSelectedImage \} from '\.\.\/\.\.\/common\/upload'/)
  assert.doesNotMatch(source, /chooseAndUploadImage/)
  assert.match(source, /const pendingVerificationFiles = reactive\(\{\}\)/)
  assert.match(source, /const file = await chooseImageFile\(\)/)
  assert.match(source, /pendingVerificationFiles\[kind\] = file/)
  assert.match(source, /await uploadPendingVerificationImages\(\)/)
  assert.match(source, /await uploadSelectedImage\(file, `verification-\$\{kind\}`\)/)
  assert.doesNotMatch(source, /图片已选择/)
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
