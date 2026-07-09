<template>
  <view class="vip-page">
    <view class="status-band">
      <view class="status-copy">
        <text class="status-title">{{ statusTitle }}</text>
        <text class="status-subtitle">{{ statusSubtitle }}</text>
      </view>
      <view class="status-pill" :class="{ active: isVIPActive }">{{ isVIPActive ? 'VIP' : '限时优惠' }}</view>
    </view>

    <view class="benefit-strip">
      <view class="benefit-item">
        <text class="benefit-value">80</text>
        <text class="benefit-label">条发布额度</text>
      </view>
      <view class="benefit-item">
        <text class="benefit-value">30</text>
        <text class="benefit-label">次刷新</text>
      </view>
      <view class="benefit-item">
        <text class="benefit-value">3</text>
        <text class="benefit-label">张置顶券</text>
      </view>
    </view>

    <view class="launch-offer">
      <text class="offer-title">年卡限时 ¥299</text>
      <text class="offer-meta">80 条发布额度 · 30 次刷新 · 3 张置顶券</text>
    </view>

    <view class="plan-list">
      <view
        v-for="plan in displayPlans"
        :key="plan.code"
        :class="['plan-card', selectedPlanCode === plan.code ? 'selected' : '']"
        @click="selectPlan(plan.code)"
      >
        <view class="plan-main">
          <text class="plan-name">{{ plan.name }}</text>
          <text class="plan-benefits">{{ planBenefitText(plan) }}</text>
        </view>
        <view class="plan-price">
          <text v-if="plan.saleLabel" class="sale-label">{{ plan.saleLabel }}</text>
          <text class="sale-price">{{ formatPrice(plan.salePriceCent || plan.standardPriceCent) }}</text>
          <text v-if="plan.salePriceCent" class="standard-price">{{ formatPrice(plan.standardPriceCent) }}</text>
        </view>
      </view>
    </view>

    <button class="primary-button" :disabled="paying || !selectedPlanCode" @click="openSelectedPlan">
      {{ paying ? '正在开通' : '立即开通 VIP' }}
    </button>
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'

import { createVIPOrder, createVIPPayment, getMerchantVIP, listVIPPlans } from '../../api/vip'
import { requireLogin } from '../../common/auth'
import { getSession } from '../../store/session'

const merchantId = ref('')
const plans = ref([])
const vipStatus = ref({ status: 'none' })
const selectedPlanCode = ref('yearly')
const paying = ref(false)

const isVIPActive = computed(() => vipStatus.value.status === 'active')
const statusTitle = computed(() => (isVIPActive.value ? 'VIP 权益已生效' : '开通 VIP 权益'))
const statusSubtitle = computed(() => {
  if (isVIPActive.value) {
    return `剩余发布 ${vipStatus.value.publishQuotaRemaining || 0} 条，刷新 ${vipStatus.value.refreshQuotaRemaining || 0} 次`
  }
  return '冷启动限时优惠，赠送发布次数，不做平台认证背书'
})
const displayPlans = computed(() => {
  if (plans.value.length > 0) return plans.value
  return [
    fallbackPlan('monthly', 'VIP 月卡', 1, 4900, 1990, '首月优惠'),
    fallbackPlan('half_year', 'VIP 半年卡', 6, 29900, 19900, '限时优惠'),
    fallbackPlan('yearly', 'VIP 年卡', 12, 49900, 29900, '限时优惠'),
  ]
})

onLoad((options = {}) => {
  merchantId.value = options.merchantId || getSession().merchantId || ''
  loadVIPData()
})

onShow(() => {
  const sessionMerchantId = getSession().merchantId
  if (!merchantId.value && sessionMerchantId) merchantId.value = sessionMerchantId
  loadVIPStatus()
})

async function loadVIPData() {
  await Promise.all([loadPlans(), loadVIPStatus()])
}

async function loadPlans() {
  try {
    const resp = await listVIPPlans()
    plans.value = resp.items || []
    if (!plans.value.some((plan) => plan.code === selectedPlanCode.value) && plans.value[0]) {
      selectedPlanCode.value = plans.value[0].code
    }
  } catch (err) {
    plans.value = []
  }
}

async function loadVIPStatus() {
  if (!merchantId.value) return
  try {
    vipStatus.value = await getMerchantVIP(merchantId.value, { suppressErrorToast: true })
  } catch (err) {
    vipStatus.value = { status: 'none' }
  }
}

function selectPlan(code) {
  selectedPlanCode.value = code
}

async function openSelectedPlan() {
  if (!requireLogin()) return
  if (!merchantId.value) {
    uni.showToast({ title: '请先登录后再开通 VIP', icon: 'none' })
    return
  }
  paying.value = true
  try {
    const order = await createVIPOrder(merchantId.value, { planCode: selectedPlanCode.value })
    const resp = await createVIPPayment(merchantId.value, order.orderId)
    if (resp.status === 'paid') {
      uni.showToast({ title: 'VIP 已开通', icon: 'none' })
      await loadVIPStatus()
      return
    }
    const payment = resp.payment || {}
    if (!payment.timeStamp || !payment.nonceStr || !payment.package || !payment.paySign) {
      throw new Error('支付参数无效，请稍后重试')
    }
    await requestWechatPayment(payment)
    uni.showToast({ title: '支付成功，权益更新中', icon: 'none' })
    await loadVIPStatus()
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

function planBenefitText(plan) {
  const benefits = plan.benefits || {}
  return `${benefits.publishQuota || 80} 条发布额度 · ${benefits.refreshQuota || 30} 次刷新 · ${benefits.topVoucherCount || 3} 张置顶券`
}

function formatPrice(value) {
  const price = Number(value || 0) / 100
  return `¥${Number.isInteger(price) ? price.toFixed(0) : price.toFixed(1)}`
}

function fallbackPlan(code, name, durationMonths, standardPriceCent, salePriceCent, saleLabel) {
  return {
    code,
    name,
    durationMonths,
    standardPriceCent,
    salePriceCent,
    saleLabel,
    benefits: { publishQuota: 80, refreshQuota: 30, topVoucherCount: 3, topDurationHours: 24 },
  }
}
</script>

<style lang="scss" scoped>
.vip-page {
  min-height: 100vh;
  padding: 24rpx 24rpx 40rpx;
  background: $wplink-bg;
}

.status-band,
.benefit-strip,
.launch-offer,
.plan-card {
  margin-bottom: 20rpx;
  border-radius: 12rpx;
  background: $wplink-card;
  box-shadow: 0 8rpx 24rpx rgba(15, 23, 42, 0.04);
}

.status-band {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20rpx;
  padding: 28rpx;
}

.status-copy {
  display: grid;
  gap: 10rpx;
  min-width: 0;
}

.status-title {
  color: $wplink-primary;
  font-size: 38rpx;
  font-weight: 700;
  line-height: 1.25;
}

.status-subtitle,
.offer-meta,
.plan-benefits {
  color: $wplink-muted;
  font-size: 26rpx;
  line-height: 1.45;
}

.status-pill,
.sale-label {
  flex: 0 0 auto;
  padding: 6rpx 14rpx;
  border-radius: 999rpx;
  background: rgba(194, 58, 0, 0.1);
  color: $wplink-warning;
  font-size: 24rpx;
  font-weight: 700;
}

.status-pill.active {
  background: $wplink-primary-soft;
  color: $wplink-primary;
}

.benefit-strip {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12rpx;
  padding: 24rpx;
}

.benefit-item {
  display: grid;
  gap: 6rpx;
  text-align: center;
}

.benefit-value {
  color: $wplink-primary;
  font-size: 40rpx;
  font-weight: 700;
}

.benefit-label {
  color: $wplink-muted;
  font-size: 24rpx;
}

.launch-offer {
  display: grid;
  gap: 8rpx;
  padding: 24rpx;
}

.offer-title {
  color: $wplink-warning;
  font-size: 34rpx;
  font-weight: 700;
}

.plan-list {
  display: grid;
  gap: 16rpx;
}

.plan-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20rpx;
  padding: 24rpx;
}

.plan-card.selected {
  box-shadow: inset 0 0 0 2rpx $wplink-warning, 0 8rpx 24rpx rgba(15, 23, 42, 0.04);
}

.plan-main,
.plan-price {
  display: grid;
  gap: 8rpx;
  min-width: 0;
}

.plan-name {
  color: $wplink-primary;
  font-size: 30rpx;
  font-weight: 700;
}

.plan-price {
  justify-items: end;
  flex: 0 0 auto;
}

.sale-price {
  color: $wplink-primary;
  font-size: 34rpx;
  font-weight: 700;
}

.standard-price {
  color: $wplink-muted;
  font-size: 24rpx;
  text-decoration: line-through;
}

.primary-button {
  width: 100%;
  height: 88rpx;
  margin-top: 28rpx;
  border-radius: 12rpx;
  background: $wplink-primary;
  color: $wplink-card;
  font-size: 30rpx;
  font-weight: 700;
}
</style>
