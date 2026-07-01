<template>
  <view class="verification-page">
    <view class="status-card">
      <view class="status-copy">
        <text class="section-title">认证状态</text>
        <text class="status-text">{{ statusLabel(latestVerification.status) }}</text>
        <text class="status-meta" v-if="latestVerification.verificationType">
          {{ typeLabel(latestVerification.verificationType) }} · {{ latestVerification.reviewedAt || '等待审核' }}
        </text>
        <text class="status-meta">{{ billingSummary }}</text>
      </view>
      <button v-if="canPayVerification" class="primary-button" :loading="paying" @click="payVerification">支付认证费</button>
    </view>

    <view class="form-card">
      <view class="form-section">
        <view class="section-heading">
          <view class="section-title-row">
            <text class="section-title">主体资料</text>
            <text class="required-badge">必填</text>
          </view>
        </view>
        <view class="section-body">
          <view class="form-field">
            <text class="field-label">营业主体名称</text>
            <input v-model="form.businessName" class="field" placeholder="请按营业执照填写" />
          </view>
          <view class="form-field">
            <text class="field-label">统一社会信用代码</text>
            <input v-model="form.socialCreditCode" class="field" maxlength="64" placeholder="营业执照上的统一社会信用代码" />
          </view>
          <view class="form-field">
            <text class="field-label">营业执照</text>
            <view class="proof-grid">
              <button v-for="item in proofItems('subject')" :key="item.kind" class="proof-tile" @click="uploadProof(item)">
                <image v-if="item.url" class="proof-image" :src="item.url" mode="aspectFill" />
                <view v-else class="proof-empty">
                  <view class="proof-plus">
                    <view class="proof-plus-icon" />
                  </view>
                </view>
                <text class="proof-label">{{ item.label }}</text>
                <text v-if="item.required" class="proof-required">必填</text>
                <text v-if="item.url" class="proof-change">更换</text>
              </button>
            </view>
          </view>
        </view>
      </view>

      <view class="form-section">
        <view class="section-heading">
          <view class="section-title-row">
            <text class="section-title">联系人和地址</text>
            <text class="required-badge">必填</text>
          </view>
        </view>
        <view class="section-body">
          <view class="form-field">
            <text class="field-label">联系人姓名</text>
            <input v-model="form.applicantName" class="field" placeholder="负责认证资料的联系人" />
          </view>
          <view class="form-field">
            <text class="field-label">联系电话</text>
            <input v-model="form.contactPhone" class="field" type="number" maxlength="20" placeholder="6-20位联系电话" @input="sanitizeContactPhone" />
          </view>
          <view class="form-field">
            <text class="field-label">联系微信</text>
            <input v-model="form.contactWechat" class="field" placeholder="选填" />
          </view>
          <view class="form-field">
            <text class="field-label">经营地址</text>
            <input v-model="form.addressText" class="field" placeholder="市场、档口号、工厂或仓库地址" />
          </view>
        </view>
      </view>

      <view class="form-section">
        <view class="section-heading">
          <view class="section-title-row">
            <text class="section-title">经营实拍</text>
          </view>
        </view>
        <view class="section-body">
          <view class="form-field">
            <text class="field-label">经营照片</text>
            <view class="proof-grid">
              <button v-for="item in proofItems('scene')" :key="item.kind" class="proof-tile" @click="uploadProof(item)">
                <image v-if="item.url" class="proof-image" :src="item.url" mode="aspectFill" />
                <view v-else class="proof-empty">
                  <view class="proof-plus">
                    <view class="proof-plus-icon" />
                  </view>
                </view>
                <text class="proof-label">{{ item.label }}</text>
                <text v-if="item.required" class="proof-required">必填</text>
                <text v-if="item.url" class="proof-change">更换</text>
              </button>
            </view>
          </view>
        </view>
      </view>

      <view class="form-section">
        <view class="section-heading">
          <text class="section-title">补充证明</text>
        </view>
        <view class="section-body">
          <view class="form-field">
            <view class="proof-grid">
              <button v-for="item in proofItems('optional')" :key="item.kind" class="proof-tile optional-proof" @click="uploadProof(item)">
                <image v-if="item.url" class="proof-image" :src="item.url" mode="aspectFill" />
                <view v-else class="proof-empty">
                  <view class="proof-plus">
                    <view class="proof-plus-icon" />
                  </view>
                </view>
                <text class="proof-label">{{ item.label }}</text>
                <text v-if="item.required" class="proof-required">必填</text>
                <text v-if="item.url" class="proof-change">更换</text>
              </button>
            </view>
          </view>
          <checkbox-group @change="changeCommitment">
            <label class="commitment-row">
              <checkbox value="accepted" :checked="form.commitmentAccepted" />
              <text>我承诺资料真实有效。</text>
            </label>
          </checkbox-group>
        </view>
      </view>
    </view>

    <view class="fixed-save-spacer" />
    <view class="fixed-save-bar">
      <button class="primary-button" @click="submit">提交认证</button>
    </view>
  </view>
</template>

<script setup>
import { computed, reactive, ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { getMerchantId, getUserId } from '../../store/session'
import { getMerchant } from '../../api/merchant'
import { createVerificationPayment, getLatestVerification, getVerificationBillingConfig, submitVerification } from '../../api/verification'
import { chooseAndUploadImage } from '../../common/upload'

const merchantIdentityOptions = [
  { label: '源头工厂', value: 'factory' },
  { label: '现货档口', value: 'stall' },
  { label: '库存货源', value: 'stockist' },
  { label: '配套服务', value: 'service_provider' },
]

const form = reactive({
  merchantId: '',
  verificationType: 'factory',
  businessName: '',
  socialCreditCode: '',
  licenseUrl: '',
  storefrontUrl: '',
  applicantName: '',
  contactPhone: '',
  contactWechat: '',
  addressText: '',
  sceneUrl: '',
  authorizationUrl: '',
  qualificationUrl: '',
  commitmentAccepted: false,
})

const proofUploadItems = computed(() => [
  { kind: 'license', group: 'subject', label: '营业执照', required: true, url: form.licenseUrl },
  { kind: 'storefront', group: 'scene', label: '门头/场地', required: true, url: form.storefrontUrl },
  { kind: 'scene', group: 'scene', label: '经营实拍', required: false, url: form.sceneUrl },
  { kind: 'authorization', group: 'optional', label: '授权证明', required: false, url: form.authorizationUrl },
  { kind: 'qualification', group: 'optional', label: '其他证明', required: false, url: form.qualificationUrl },
])
const latestVerification = ref({ status: 'none' })
const billingConfig = ref({ chargeEnabled: false, feeAmount: 0, currency: 'CNY', freeEnabled: false })
const paying = ref(false)
const billingSummary = computed(() => {
  if (!billingConfig.value.chargeEnabled) return '当前认证免费，审核通过后直接生效'
  const feeText = `认证费 ¥${(Number(billingConfig.value.feeAmount || 0) / 100).toFixed(2)}`
  if (isFreeWindowActive()) return `${feeText}，当前限时免费`
  if (latestVerification.value.status === 'payment_pending') return `${feeText}，支付成功后认证生效`
  return `${feeText}，资料审核通过后在线支付`
})
const canPayVerification = computed(() => {
  return latestVerification.value.status === 'payment_pending' && billingConfig.value.chargeEnabled && !isFreeWindowActive() && latestVerification.value.id
})

onLoad(async (options) => {
  form.merchantId = options.merchantId || getMerchantId()
  await Promise.all([loadMerchantProfile(), loadLatestVerification(), loadBillingConfig()])
})

async function loadMerchantProfile() {
  if (!form.merchantId) return
  try {
    const detail = await getMerchant(form.merchantId, { suppressErrorToast: true })
    form.verificationType = detail.merchantType || form.verificationType
    if (!form.businessName.trim()) form.businessName = detail.name || ''
    if (!form.addressText.trim()) form.addressText = detail.addressText || ''
  } catch (err) {
    // 认证资料仍可手动填写；商家身份加载失败时保留默认身份，避免阻断提交。
  }
}

async function loadLatestVerification() {
  if (!form.merchantId) {
    latestVerification.value = { status: 'none' }
    return
  }
  try {
    latestVerification.value = await getLatestVerification(form.merchantId)
  } catch (err) {
    latestVerification.value = { status: 'none' }
  }
}

async function loadBillingConfig() {
  try {
    billingConfig.value = await getVerificationBillingConfig('zhili')
  } catch (err) {
    billingConfig.value = { chargeEnabled: false, feeAmount: 0, currency: 'CNY', freeEnabled: false }
  }
}

function typeLabel(type) {
  const matched = merchantIdentityOptions.find((item) => item.value === type) || {}
  return matched.label || type || '-'
}

function statusLabel(status) {
  const statusText = {
    none: '未提交认证',
    pending: '审核中',
    payment_pending: '待支付',
    verified: '已认证',
    rejected: '未通过',
  }
  return statusText[status] || status || '未提交认证'
}

async function uploadLicense() {
  await uploadVerificationImage('license')
}

async function uploadStorefront() {
  await uploadVerificationImage('storefront')
}

async function uploadProof(item) {
  if (!item?.kind) return
  await uploadVerificationImage(item.kind)
}

async function uploadVerificationImage(kind) {
  try {
    // 认证材料上传成功后只提交 CDN URL，后台审核无需接收二进制文件。
    const url = await chooseAndUploadImage(`verification-${kind}`)
    form[imageFieldName(kind)] = url
    uni.showToast({ title: '图片已上传', icon: 'none' })
  } catch (err) {
    uni.showToast({ title: err.message || '图片上传失败，请重试', icon: 'none' })
  }
}

function imageFieldName(kind) {
  const fields = {
    license: 'licenseUrl',
    storefront: 'storefrontUrl',
    scene: 'sceneUrl',
    authorization: 'authorizationUrl',
    qualification: 'qualificationUrl',
  }
  return fields[kind] || 'storefrontUrl'
}

function proofItems(group) {
  return proofUploadItems.value.filter((item) => item.group === group)
}

async function submit() {
  const userId = getUserId()
  if (!userId) {
    uni.showToast({ title: '请先在我的页保存用户 ID', icon: 'none' })
    return
  }
  form.contactPhone = sanitizeContactPhoneValue(form.contactPhone)
  const validationMessage = validateForm()
  if (validationMessage) {
    uni.showToast({ title: validationMessage, icon: 'none' })
    return
  }
  const verificationMaterials = buildVerificationMaterials()
  // 认证申请需要记录提交人，便于后台审核留痕和后续消息通知。
  await submitVerification(form.merchantId.trim(), {
    applicantUserId: userId,
    verificationType: form.verificationType,
    businessName: form.businessName.trim(),
    licenseUrl: form.licenseUrl.trim(),
    storefrontUrl: form.storefrontUrl.trim(),
    materials: verificationMaterials,
  })
  uni.showToast({ title: '认证已提交', icon: 'none' })
  await loadLatestVerification()
}

function validateForm() {
  if (!form.merchantId.trim()) return '请先完成商家入驻'
  if (!form.businessName.trim()) return '请填写营业主体名称'
  if (!form.socialCreditCode.trim()) return '请填写统一社会信用代码'
  if (!form.licenseUrl.trim()) return '请上传营业执照'
  if (!form.applicantName.trim()) return '请填写联系人姓名'
  if (!form.contactPhone.trim() || !isValidContactPhone(form.contactPhone)) return '请填写6-20位联系电话'
  if (!form.addressText.trim()) return '请填写经营地址'
  if (!form.storefrontUrl.trim()) return '请上传门头或场地照片'
  if (!form.commitmentAccepted) return '请勾选资料真实性承诺'
  return ''
}

function buildVerificationMaterials() {
  return compactMaterials({
    socialCreditCode: form.socialCreditCode.trim(),
    applicantName: form.applicantName.trim(),
    contactPhone: form.contactPhone.trim(),
    contactWechat: form.contactWechat.trim(),
    addressText: form.addressText.trim(),
    sceneUrl: form.sceneUrl.trim(),
    authorizationUrl: form.authorizationUrl.trim(),
    qualificationUrl: form.qualificationUrl.trim(),
    commitmentAccepted: form.commitmentAccepted,
  })
}

function compactMaterials(materials) {
  return Object.entries(materials).reduce((result, [key, value]) => {
    if (value !== '' && value !== null && value !== undefined) {
      result[key] = value
    }
    return result
  }, {})
}

function sanitizeContactPhone(event) {
  form.contactPhone = sanitizeContactPhoneValue(event?.detail?.value ?? form.contactPhone)
}

function sanitizeContactPhoneValue(value) {
  return String(value || '').replace(/\D/g, '').slice(0, 20)
}

function isValidContactPhone(value) {
  return /^\d{6,20}$/.test(value)
}

function changeCommitment(event) {
  form.commitmentAccepted = (event?.detail?.value || []).includes('accepted')
}

async function payVerification() {
  const userId = getUserId()
  if (!userId) {
    uni.showToast({ title: '请先登录后支付', icon: 'none' })
    return
  }
  paying.value = true
  try {
    // 小程序端只负责调起收银台，认证生效必须以后端收到微信支付成功通知为准。
    const resp = await createVerificationPayment(form.merchantId, latestVerification.value.id, { userId })
    const payment = resp.payment || {}
    await requestWechatPayment(payment)
    uni.showToast({ title: '支付成功，正在更新认证状态', icon: 'none' })
    await loadLatestVerification()
  } catch (err) {
    uni.showToast({ title: err.message || '支付未完成，请稍后重试', icon: 'none' })
  } finally {
    paying.value = false
  }
}

function requestWechatPayment(payment) {
  return new Promise((resolve, reject) => {
    uni.requestPayment({
      timeStamp: payment.timeStamp,
      nonceStr: payment.nonceStr,
      package: payment.package,
      signType: payment.signType || 'RSA',
      paySign: payment.paySign,
      success: resolve,
      fail: reject,
    })
  })
}

function isFreeWindowActive() {
  const config = billingConfig.value || {}
  if (!config.freeEnabled) return false
  const now = Date.now()
  const start = config.freeStartAt ? Date.parse(config.freeStartAt) : null
  const end = config.freeEndAt ? Date.parse(config.freeEndAt) : null
  if (start && now < start) return false
  if (end && now > end) return false
  return true
}
</script>

<style lang="scss" scoped>
.verification-page {
  min-height: 100vh;
  padding: 24rpx 24rpx 0;
  background: $wplink-bg;
}

.form-card {
  display: grid;
  gap: 20rpx;
}

.status-card {
  display: grid;
  gap: 18rpx;
  margin-bottom: 20rpx;
  padding: 24rpx;
  border-radius: 12rpx;
  background: $wplink-card;
}

.status-copy {
  display: grid;
  gap: 10rpx;
}

.form-section {
  display: grid;
  gap: 18rpx;
  padding: 24rpx;
  border-radius: 12rpx;
  background: $wplink-card;
}

.section-heading {
  display: grid;
  gap: 8rpx;
}

.section-title-row {
  display: flex;
  gap: 10rpx;
  align-items: center;
}

.section-title {
  color: $wplink-primary;
  font-size: 30rpx;
  font-weight: 700;
}

.required-badge {
  padding: 4rpx 12rpx;
  border-radius: 999rpx;
  background: $wplink-warning-soft;
  color: $wplink-warning;
  font-size: 22rpx;
  line-height: 1.2;
}

.status-meta {
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 1.45;
}

.section-body,
.form-field {
  display: grid;
  gap: 14rpx;
}

.section-body {
  gap: 18rpx;
}

.field-label {
  color: $wplink-primary;
  font-size: 26rpx;
  font-weight: 600;
}

.field {
  width: 100%;
  min-height: 80rpx;
  padding: 0 20rpx;
  border: 1rpx solid $wplink-line;
  border-radius: 10rpx;
  box-sizing: border-box;
}

.status-text {
  color: $wplink-primary;
  font-size: 34rpx;
  font-weight: 700;
}

.picker-field {
  display: flex;
  align-items: center;
  justify-content: space-between;
  color: $wplink-primary;
}

.primary-button {
  height: 84rpx;
  border-radius: 12rpx;
  background: $wplink-primary;
  color: $wplink-card;
}

.secondary-button {
  height: 84rpx;
  border: 1rpx solid $wplink-primary;
  border-radius: 12rpx;
  background: $wplink-card;
  color: $wplink-primary;
}

.proof-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16rpx;
}

.proof-tile {
  position: relative;
  display: grid;
  align-content: center;
  justify-items: center;
  width: 100%;
  min-height: 184rpx;
  padding: 18rpx 12rpx;
  margin: 0;
  border: 1rpx dashed $wplink-line;
  border-radius: 12rpx;
  box-sizing: border-box;
  background: #f8fafc;
  color: $wplink-primary;
  line-height: 1.3;
  text-align: center;
  overflow: hidden;
}

.proof-tile::after {
  border: 0;
}

.proof-image {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  background: #e3e8ef;
}

.proof-empty {
  display: grid;
  gap: 12rpx;
  justify-items: center;
}

.proof-plus {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44rpx;
  height: 44rpx;
}

.proof-plus-icon,
.proof-plus-icon::after {
  display: block;
  width: 30rpx;
  height: 4rpx;
  border-radius: 999rpx;
  background: $wplink-muted;
}

.proof-plus-icon {
  position: relative;
}

.proof-plus-icon::after {
  position: absolute;
  top: 0;
  left: 0;
  content: '';
  transform: rotate(90deg);
}

.proof-label {
  position: relative;
  z-index: 1;
  padding: 6rpx 12rpx;
  border-radius: 999rpx;
  background: rgba(255, 255, 255, 0.88);
  color: $wplink-primary;
  font-size: 24rpx;
  line-height: 1.3;
}

.proof-required {
  position: absolute;
  top: 12rpx;
  right: 12rpx;
  z-index: 2;
  padding: 4rpx 10rpx;
  border-radius: 999rpx;
  background: $wplink-warning-soft;
  color: $wplink-warning;
  font-size: 20rpx;
  line-height: 1.2;
  pointer-events: none;
}

.proof-change {
  position: absolute;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 2;
  height: 40rpx;
  background: rgba(0, 0, 0, 0.48);
  color: #fff;
  font-size: 22rpx;
  line-height: 40rpx;
  text-align: center;
}

.optional-proof {
  min-height: 156rpx;
}

.commitment-row {
  display: grid;
  grid-template-columns: 48rpx minmax(0, 1fr);
  gap: 12rpx;
  align-items: start;
  color: $wplink-primary;
  font-size: 25rpx;
  line-height: 1.5;
}

.fixed-save-bar {
  position: fixed;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 20;
  padding: 18rpx 24rpx calc(18rpx + env(safe-area-inset-bottom));
  border-top: 1rpx solid $wplink-line;
  background: rgba(255, 255, 255, 0.96);
}

.fixed-save-spacer {
  height: calc(156rpx + env(safe-area-inset-bottom));
}
</style>
