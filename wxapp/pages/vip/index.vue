<template>
  <view class="vip-page">
    <view class="vip-tabs">
      <button :class="['vip-tab', activeTab === 'vip' ? 'active' : '']" @click="switchTab('vip')">VIP权益包</button>
      <button :class="['vip-tab', activeTab === 'addons' ? 'active' : '']" @click="switchTab('addons')">购买次数</button>
    </view>

    <view v-if="activeTab === 'vip'" class="tab-panel">
      <view class="plan-list">
        <view
          v-for="plan in displayPlans"
          :key="plan.code"
          :class="['plan-card', selectedPlanCode === plan.code ? 'selected' : '']"
          @click="selectPlan(plan.code)"
        >
          <view class="plan-main">
            <text class="plan-name">{{ planDisplayName(plan) }}</text>
            <text class="plan-benefits">{{ planBenefitText(plan) }}</text>
          </view>
          <view class="plan-price">
            <text v-if="planSaleLabel(plan)" class="sale-label">{{ planSaleLabel(plan) }}</text>
            <text class="sale-price">{{ formatPrice(plan.salePriceCent || plan.standardPriceCent) }}</text>
            <text v-if="plan.salePriceCent" class="standard-price">{{ formatPrice(plan.standardPriceCent) }}</text>
          </view>
        </view>
      </view>

      <button class="primary-button" :disabled="paying || !selectedPlanCode" @click="openSelectedPlan">
        {{ paying ? '正在开通' : '立即开通 VIP' }}
      </button>
    </view>

    <view v-else-if="activeTab === 'addons'" class="tab-panel">
      <view class="quota-pack-list">
        <view v-for="item in displayAddOnPacks" :key="item.code" class="quota-pack-card">
          <view class="pack-main">
            <text class="pack-title">{{ item.name }}</text>
            <text class="pack-meta">{{ packMetaText(item) }}</text>
          </view>
          <view class="pack-action">
            <view class="pack-price-line">
              <text v-if="packSaleLabel(item)" class="sale-label pack-sale-label">{{ packSaleLabel(item) }}</text>
              <text class="pack-price">{{ packPriceText(item) }}</text>
            </view>
            <text v-if="isPackDiscounted(item)" class="standard-price pack-standard-price">{{ formatPrice(item.standardPriceCent) }}</text>
            <button class="pack-button" :disabled="payingPackCode === item.code" @click="openQuotaPack(item)">
              {{ payingPackCode === item.code ? '购买中' : packActionText(item) }}
            </button>
          </view>
        </view>
      </view>
    </view>

    <view class="rules-panel">
      <text class="rules-title">权益有效期说明</text>
      <text class="rules-text">VIP 赠送的发布额度、刷新次数和置顶券均在发放后 30 天内有效，未使用完不结转。</text>
      <text class="rules-text">单独购买置顶时，需要先选择具体已发布资源，支付成功后置顶服务直接生效。</text>
      <text class="rules-text">单独购买的次数包有效期 180 天，到期未使用自动失效。</text>
    </view>
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'

import { createQuotaPackOrder, createVIPOrder, createVIPPayment, listQuotaPacks, listVIPPlans } from '../../api/vip'
import { requireLogin } from '../../common/auth'
import { getSession } from '../../store/session'

const merchantId = ref('')
const plans = ref([])
const quotaPacks = ref([])
const selectedPlanCode = ref('yearly')
const selectedQuotaType = ref('')
const activeTab = ref('vip')
const paying = ref(false)
const payingPackCode = ref('')
const fallbackQuotaPacks = [
  { code: 'publish_5', name: '发布次数包', standardPriceCent: 2500, salePriceCent: 2500, actionText: '¥25 购买', description: '临时多发供需', saleLabel: '限时特价', benefits: { publishQuota: 5 } },
  { code: 'refresh_10', name: '刷新次数包', standardPriceCent: 1900, salePriceCent: 1900, actionText: '¥19 购买', description: '让信息回到前面', saleLabel: '限时特价', benefits: { refreshQuota: 10 } },
]

const displayPlans = computed(() => {
  if (plans.value.length > 0) return plans.value
  return [
    fallbackPlan('monthly', 'VIP 月卡', 1, 4900, 1990, '限时特价'),
    fallbackPlan('half_year', 'VIP 半年卡', 6, 29900, 19900, '限时特价'),
    fallbackPlan('yearly', 'VIP 年卡', 12, 49900, 29900, '限时特价'),
  ]
})

const displayAddOnPacks = computed(() => {
  const packs = quotaPacks.value.filter((item) => !isTopVoucherPack(item))
  return sortQuotaPacksBySelectedType(packs.length > 0 ? packs : fallbackQuotaPacks)
})

onLoad((options = {}) => {
  merchantId.value = options.merchantId || getSession().merchantId || ''
  selectedQuotaType.value = options.quotaType || ''
  if (options.tab === 'addons') activeTab.value = 'addons'
  if (options.quotaType) activeTab.value = 'addons'
  loadVIPData()
})

onShow(() => {
  const sessionMerchantId = getSession().merchantId
  if (!merchantId.value && sessionMerchantId) merchantId.value = sessionMerchantId
})

async function loadVIPData() {
  await Promise.all([loadPlans(), loadQuotaPacks()])
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

async function loadQuotaPacks() {
  try {
    const resp = await listQuotaPacks()
    quotaPacks.value = resp.items || []
  } catch (err) {
    quotaPacks.value = []
  }
}

function selectPlan(code) {
  selectedPlanCode.value = code
}

function switchTab(tab) {
  activeTab.value = tab
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
    await payOrder(order)
  } catch (err) {
    uni.showToast({ title: err.message || '支付未完成，请稍后重试', icon: 'none' })
  } finally {
    paying.value = false
  }
}

async function openQuotaPack(item) {
  if (!requireLogin()) return
  if (!merchantId.value) {
    uni.showToast({ title: '请先登录后再购买次数包', icon: 'none' })
    return
  }
  payingPackCode.value = item.code
  try {
    const order = await createQuotaPackOrder(merchantId.value, item.code)
    await payOrder(order)
  } catch (err) {
    uni.showToast({ title: err.message || '支付未完成，请稍后重试', icon: 'none' })
  } finally {
    payingPackCode.value = ''
  }
}

async function payOrder(order) {
  const resp = await createVIPPayment(merchantId.value, order.orderId)
  if (resp.status === 'paid') {
    await loadVIPData()
    uni.showToast({ title: '支付已完成，权益到账后会自动更新', icon: 'none' })
    return
  }
  const payment = resp.payment || {}
  if (!payment.timeStamp || !payment.nonceStr || !payment.package || !payment.paySign) {
    throw new Error('支付参数无效，请稍后重试')
  }
  await requestWechatPayment(payment)
  await loadVIPData()
  uni.showToast({ title: '支付已完成，权益到账后会自动更新', icon: 'none' })
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

function packMetaText(item) {
  const value = packValueText(item)
  const description = item.description || item.meta || ''
  return description ? `${value} · ${description}` : value
}

function packValueText(item) {
  const benefits = item.benefits || {}
  if (benefits.publishQuota) return `${benefits.publishQuota} 条发布额度`
  if (benefits.refreshQuota) return `${benefits.refreshQuota} 次刷新`
  if (benefits.topVoucherCount) return `${benefits.topVoucherCount} 张置顶券`
  return item.value || '权益次数包'
}

function isTopVoucherPack(item) {
  const benefits = item.benefits || {}
  return Boolean(benefits.topVoucherCount)
}

function quotaPackType(item) {
  const benefits = item.benefits || {}
  if (benefits.publishQuota) return 'publish_quota'
  if (benefits.refreshQuota) return 'refresh_quota'
  return ''
}

function sortQuotaPacksBySelectedType(items) {
  if (!selectedQuotaType.value) return items

  const matchingItems = []
  const remainingItems = []
  items.forEach((item) => {
    if (quotaPackType(item) === selectedQuotaType.value) matchingItems.push(item)
    else remainingItems.push(item)
  })
  return [...matchingItems, ...remainingItems]
}

function packPriceText(item) {
  return formatPrice(item.salePriceCent || item.standardPriceCent)
}

function packSaleLabel(item) {
  return shouldShowPlanSaleLabel(item) ? item.saleLabel : ''
}

function isPackDiscounted(item) {
  const salePriceCent = Number(item.salePriceCent || 0)
  const standardPriceCent = Number(item.standardPriceCent || 0)
  return salePriceCent > 0 && standardPriceCent > 0 && salePriceCent < standardPriceCent
}

function packActionText(item) {
  return item.actionText || `${packPriceText(item)} 购买`
}

function planBenefitText(plan) {
  const benefits = plan.benefits || {}
  const periodText = Number(plan.durationMonths || 1) > 1 ? '每 30 天到账' : '开通后到账'
  const parts = [
    `${benefits.publishQuota || 80} 条发布额度`,
    `${benefits.refreshQuota || 30} 次刷新`,
  ]
  if (benefits.topVoucherCount) parts.push(`${benefits.topVoucherCount} 张置顶券`)
  return `${periodText}：${parts.join(' · ')}`
}

function planDisplayName(plan) {
  const names = {
    monthly: 'VIP 月卡',
    half_year: 'VIP 半年卡',
    yearly: 'VIP 年卡',
  }
  return names[plan.code] || plan.name
}

function planSaleLabel(plan) {
  return shouldShowPlanSaleLabel(plan) ? plan.saleLabel : ''
}

function shouldShowPlanSaleLabel(plan) {
  return Boolean(plan.saleLabel) && isPackDiscounted(plan)
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

.plan-card,
.vip-tabs,
.quota-pack-card,
.rules-panel {
  margin-bottom: 20rpx;
  border-radius: 12rpx;
  background: $wplink-card;
  box-shadow: 0 8rpx 24rpx rgba(15, 23, 42, 0.04);
}

.plan-benefits {
  color: $wplink-muted;
  font-size: 26rpx;
  line-height: 1.45;
}

.sale-label {
  flex: 0 0 auto;
  padding: 6rpx 14rpx;
  border-radius: 999rpx;
  background: rgba(194, 58, 0, 0.1);
  color: $wplink-warning;
  font-size: 24rpx;
  font-weight: 700;
}

.vip-tabs {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8rpx;
  padding: 8rpx;
}

.vip-tab {
  height: 68rpx;
  margin: 0;
  border-radius: 8rpx;
  background: transparent;
  color: $wplink-muted;
  font-size: 28rpx;
  font-weight: 700;
  line-height: 68rpx;
}

.vip-tab::after,
.pack-button::after {
  border: 0;
}

.vip-tab.active {
  background: $wplink-primary;
  color: $wplink-card;
}

.tab-panel {
  min-width: 0;
}

.plan-list {
  display: grid;
  gap: 16rpx;
}

.quota-pack-list {
  display: grid;
  gap: 16rpx;
}

.rules-panel {
  display: grid;
  gap: 10rpx;
  margin-top: 24rpx;
  padding: 24rpx;
}

.plan-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20rpx;
  padding: 24rpx;
}

.quota-pack-card {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 20rpx;
  align-items: center;
  padding: 24rpx;
}

.plan-card.selected {
  box-shadow: inset 0 0 0 2rpx $wplink-warning, 0 8rpx 24rpx rgba(15, 23, 42, 0.04);
}

.plan-main,
.plan-price,
.pack-main,
.pack-action {
  display: grid;
  gap: 8rpx;
  min-width: 0;
}

.plan-name,
.pack-title,
.rules-title {
  color: $wplink-primary;
  font-size: 30rpx;
  font-weight: 700;
}

.rules-text {
  color: $wplink-muted;
  font-size: 25rpx;
  line-height: 1.5;
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

.pack-price {
  color: $wplink-primary;
  font-size: 32rpx;
  font-weight: 700;
  text-align: right;
}

.pack-price-line {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8rpx;
  min-width: 0;
}

.pack-sale-label {
  padding: 4rpx 10rpx;
  font-size: 22rpx;
}

.pack-standard-price {
  text-align: right;
}

.pack-meta {
  color: $wplink-muted;
  font-size: 26rpx;
  line-height: 1.45;
}

.pack-button {
  width: 160rpx;
  height: 56rpx;
  margin: 0;
  border-radius: 8rpx;
  background: $wplink-primary-soft;
  color: $wplink-primary;
  font-size: 24rpx;
  font-weight: 700;
  line-height: 56rpx;
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
