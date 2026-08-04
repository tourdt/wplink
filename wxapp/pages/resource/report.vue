<template>
  <view class="report-page">
    <view class="summary-card">
      <text class="summary-title">举报供需信息</text>
      <text v-if="resourceTitle" class="summary-resource">{{ resourceTitle }}</text>
    </view>

    <view class="form-card">
      <view class="form-section">
        <view class="section-head">
          <text class="section-title">举报原因</text>
          <text class="required-badge">必填</text>
        </view>
        <view class="reason-grid">
          <button
            v-for="option in reasonOptions"
            :key="option.code"
            :class="['reason-option', form.reasonCode === option.code ? 'active' : '']"
            @click="selectReason(option.code)"
          >
            {{ option.label }}
          </button>
        </view>
      </view>

      <view class="form-section">
        <view class="section-head">
          <text class="section-title">补充说明</text>
          <text :class="['required-badge', isOtherReason ? '' : 'muted']">{{ isOtherReason ? '必填' : '选填' }}</text>
        </view>
        <textarea
          v-model="form.reasonText"
          class="textarea"
          maxlength="200"
          :placeholder="reasonTextPlaceholder"
        />
        <text class="field-tip">{{ reasonTextLength }}/200</text>
      </view>

      <view class="form-section">
        <view class="section-head">
          <text class="section-title">联系方式</text>
          <text class="required-badge muted">选填</text>
        </view>
        <input
          v-model="form.reporterContact"
          class="field"
          maxlength="80"
          placeholder="手机号或微信，方便平台回访"
        />
      </view>
    </view>

    <view class="submit-spacer" />
    <view class="submit-bar">
      <button
        class="submit-button"
        :class="{ disabled: submitting }"
        :disabled="submitting"
        :loading="submitting"
        @click="submitReport"
      >
        {{ submitting ? '提交中' : '提交举报' }}
      </button>
    </view>
  </view>
</template>

<script setup>
import { computed, reactive, ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { reportResource } from '../../api/resource'
import { requireLogin } from '../../common/auth'

const maxReasonTextLength = 200
const maxReportContactLength = 80
const reasonOptions = [
  { code: 'fake_info', label: '虚假信息' },
  { code: 'unreachable', label: '联系不上' },
  { code: 'inaccurate_price_quantity', label: '价格或数量不实' },
  { code: 'image_infringement', label: '图片侵权或盗图' },
  { code: 'illegal_content', label: '违法违规内容' },
  { code: 'malicious_redirect', label: '恶意导流' },
  { code: 'other', label: '其他' },
]

const form = reactive({
  resourceId: '',
  reasonCode: '',
  reasonText: '',
  reporterContact: '',
})
const resourceTitle = ref('')
const submitting = ref(false)

const selectedReason = computed(() => reasonOptions.find((item) => item.code === form.reasonCode) || null)
const isOtherReason = computed(() => form.reasonCode === 'other')
const reasonTextLength = computed(() => Array.from(form.reasonText || '').length)
const reasonTextPlaceholder = computed(() => (
  isOtherReason.value ? '请简单说明举报问题' : '可补充说明具体情况'
))

onLoad((options = {}) => {
  form.resourceId = decodeRouteText(options.resourceId)
  resourceTitle.value = decodeRouteText(options.title)
})

function selectReason(code) {
  form.reasonCode = code
}

async function submitReport() {
  if (submitting.value) return
  if (!requireLogin()) return
  if (!validateReportForm()) return
  submitting.value = true
  try {
    const resp = await reportResource(form.resourceId, {
      reasonCode: form.reasonCode,
      reasonText: buildReportReasonText(),
      evidence: buildReportEvidence(),
    })
    uni.showToast({ title: resp?.message || '举报已提交', icon: 'none' })
    setTimeout(() => {
      uni.navigateBack({ delta: 1 })
    }, 600)
  } catch (err) {
    uni.showToast({ title: err?.message || '举报提交失败，请稍后重试', icon: 'none' })
  } finally {
    submitting.value = false
  }
}

function validateReportForm() {
  if (!form.resourceId) {
    uni.showToast({ title: '资源不存在或已下架', icon: 'none' })
    return false
  }
  if (!selectedReason.value) {
    uni.showToast({ title: '请选择举报原因', icon: 'none' })
    return false
  }
  if (isOtherReason.value && !trimText(form.reasonText)) {
    uni.showToast({ title: '请填写举报说明', icon: 'none' })
    return false
  }
  if (reasonTextLength.value > maxReasonTextLength) {
    uni.showToast({ title: '举报说明请控制在 200 字以内', icon: 'none' })
    return false
  }
  if (Array.from(trimText(form.reporterContact)).length > maxReportContactLength) {
    uni.showToast({ title: '联系方式请控制在 80 字以内', icon: 'none' })
    return false
  }
  return true
}

function buildReportReasonText() {
  return trimText(form.reasonText)
}

function buildReportEvidence() {
  const evidence = { source: 'resource_report_page' }
  const reporterContact = trimText(form.reporterContact)
  if (reporterContact) {
    evidence.reporterContact = reporterContact
  }
  if (resourceTitle.value) {
    evidence.resourceTitle = resourceTitle.value
  }
  return evidence
}

function trimText(value) {
  return String(value || '').trim()
}

function decodeRouteText(value) {
  if (!value) return ''
  try {
    return decodeURIComponent(value)
  } catch {
    return String(value)
  }
}
</script>

<style lang="scss" scoped>
.report-page {
  min-height: 100vh;
  padding: 24rpx 24rpx 0;
  background: $wplink-bg;
}

.summary-card,
.form-card {
  display: grid;
  gap: 14rpx;
  margin-bottom: 20rpx;
  padding: 24rpx;
  border-radius: 12rpx;
  background: $wplink-card;
  box-shadow: 0 8rpx 24rpx rgba(15, 23, 42, 0.04);
}

.summary-title {
  color: $wplink-primary;
  font-size: 36rpx;
  font-weight: 700;
  line-height: 1.3;
}

.summary-resource {
  color: $wplink-primary;
  font-size: 28rpx;
  font-weight: 700;
  line-height: 1.45;
  word-break: break-word;
}

.field-tip {
  color: $wplink-muted;
  font-size: 25rpx;
  line-height: 1.5;
}

.form-section {
  display: grid;
  gap: 14rpx;
}

.form-section + .form-section {
  padding-top: 22rpx;
  border-top: 1rpx solid $wplink-line;
}

.section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16rpx;
}

.section-title {
  color: $wplink-primary;
  font-size: 30rpx;
  font-weight: 700;
  line-height: 1.35;
}

.required-badge {
  flex: 0 0 auto;
  padding: 4rpx 12rpx;
  border-radius: 999rpx;
  background: $wplink-warning-soft;
  color: $wplink-warning;
  font-size: 22rpx;
  line-height: 1.25;
}

.required-badge.muted {
  background: #f8fafc;
  color: $wplink-muted;
}

.reason-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12rpx;
}

.reason-option {
  min-height: 72rpx;
  padding: 0 16rpx;
  border: 1rpx solid $wplink-line;
  border-radius: 10rpx;
  background: #f8fafc;
  color: #364152;
  font-size: 25rpx;
  line-height: 1.25;
}

.reason-option.active {
  border-color: $wplink-primary;
  background: $wplink-primary;
  color: $wplink-card;
  font-weight: 700;
}

.field,
.textarea {
  width: 100%;
  border: 1rpx solid $wplink-line;
  border-radius: 10rpx;
  background: #ffffff;
  color: $wplink-primary;
  font-size: 26rpx;
}

.field {
  height: 82rpx;
  padding: 0 20rpx;
}

.textarea {
  min-height: 168rpx;
  padding: 18rpx 20rpx;
  line-height: 1.5;
}

.submit-spacer {
  height: calc(144rpx + env(safe-area-inset-bottom));
}

.submit-bar {
  position: fixed;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 20;
  padding: 18rpx 24rpx calc(18rpx + env(safe-area-inset-bottom));
  border-top: 1rpx solid $wplink-line;
  background: rgba(255, 255, 255, 0.96);
}

.submit-button {
  width: 100%;
  height: 88rpx;
  border-radius: 12rpx;
  background: $wplink-primary;
  color: $wplink-card;
  font-size: 28rpx;
  font-weight: 700;
}

.submit-button.disabled {
  opacity: 0.68;
}
</style>
