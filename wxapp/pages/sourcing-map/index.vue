<template>
  <view class="sourcing-map-page">
    <view class="directory-toolbar">
      <view class="search-row">
        <view class="search-field">
          <view class="search-icon" />
          <input
            v-model="keyword"
            placeholder="搜索商家、市场、楼栋或档口号"
            confirm-type="search"
            @confirm="submitSearch"
          />
        </view>
        <button class="search-button" :disabled="loading" :loading="loading" @click="submitSearch">
          {{ loading ? '搜索中' : '搜索' }}
        </button>
      </view>

      <view class="category-row">
        <input
          v-model="categoriesText"
          class="category-input"
          placeholder="主营品类，多个用逗号分隔"
          confirm-type="search"
          @confirm="applyCategoryFilter"
        />
        <button v-if="categoriesText" class="category-clear" @click="clearCategories">清空</button>
        <button class="category-apply" :disabled="loading" @click="applyCategoryFilter">筛选</button>
      </view>
    </view>

    <view class="merchant-list">
      <view v-if="loading && !merchants.length" class="state-card">
        <text class="state-title">正在查找商家</text>
        <text class="state-desc">正在加载已入驻档口，请稍候。</text>
      </view>
      <view v-else-if="errorText && !merchants.length" class="state-card error">
        <text class="state-title">商家列表加载失败</text>
        <text class="state-desc">{{ errorText }}</text>
        <button class="state-button" :disabled="loading" :loading="loading" @click="loadMerchants({ reset: true })">
          {{ loading ? '加载中' : '重新加载' }}
        </button>
      </view>
      <view v-else-if="!merchants.length" class="state-card">
        <text class="state-title">暂无匹配商家</text>
        <text class="state-desc">换个商家名、市场、档口号或减少主营品类试试。</text>
        <button v-if="hasConditions" class="state-button secondary" @click="clearConditions">清除条件</button>
      </view>
      <template v-else>
        <MerchantPlaceCard
          v-for="merchant in merchants"
          :key="merchant.merchantId"
          :merchant="merchant"
          @detail="openMerchantDetail"
          @location="openMerchantLocation"
        />
        <text class="load-more-text">{{ loadMoreText }}</text>
      </template>
    </view>
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onLoad, onPullDownRefresh, onReachBottom, onShow } from '@dcloudio/uni-app'
import MerchantPlaceCard from '../../components/MerchantPlaceCard.vue'
import { listMerchantDirectory, submitMerchantLocationReport } from '../../api/merchantDirectory'
import { requireLogin } from '../../common/auth'
import { DEFAULT_CITY_CODE } from '../../common/constants'
import {
  claimNavigationFeedback,
  completeNavigationFeedback,
  releaseNavigationFeedback,
  trackMerchantDirectoryEvent,
} from '../../common/merchantDirectoryAnalytics'
import {
  buildMerchantDirectoryQuery,
  hasValidMerchantLocation,
  merchantDirectoryDetailPath,
  mergeMerchantDirectoryItems,
  normalizeMerchantDirectoryItem,
} from './merchantDirectoryState'

const LIST_PAGE_SIZE = 20

const keyword = ref('')
const categoriesText = ref('')
const merchants = ref([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const errorText = ref('')
const reportSubmitting = ref(false)
let requestVersion = 0
let appliedDirectoryQuery = buildMerchantDirectoryQuery({
  cityCode: DEFAULT_CITY_CODE,
  keyword: '',
  categories: '',
  page: 1,
  pageSize: LIST_PAGE_SIZE,
})
const pendingDirectoryRequests = new Map()
const exposedMerchantIds = new Set()

const hasMore = computed(() => merchants.value.length < total.value)
const hasConditions = computed(() => Boolean(keyword.value.trim() || categoriesText.value.trim()))
const loadMoreText = computed(() => {
  if (loading.value) return '正在加载'
  if (errorText.value && merchants.value.length) return '加载失败，继续上拉重试'
  if (hasMore.value) return '继续上拉查看更多'
  return merchants.value.length ? '已展示全部商家' : ''
})

onLoad(() => loadMerchants({ reset: true }))

onShow(() => {
  promptNavigationFeedback()
})

onPullDownRefresh(async () => {
  await loadMerchants({ reset: true })
  uni.stopPullDownRefresh()
})

onReachBottom(() => {
  if (hasMore.value && !loading.value) loadMerchants({ reset: false })
})

function loadMerchants({ reset }) {
  const targetPage = reset ? 1 : page.value + 1
  // 上拉和刷新只沿用最近一次明确提交的条件，输入框中的未提交草稿不能污染下一页。
  const query = buildMerchantDirectoryQuery({
    ...appliedDirectoryQuery,
    page: targetPage,
  })
  // 同一页同一查询共享 Promise；筛选变化允许并发，并由请求代次保证迟到响应不能覆盖最新结果。
  const signature = JSON.stringify({ reset: Boolean(reset), query })
  let request = pendingDirectoryRequests.get(signature)
  if (!request) {
    request = Promise.resolve()
      .then(() => listMerchantDirectory(query))
      .finally(() => {
        if (pendingDirectoryRequests.get(signature) === request) pendingDirectoryRequests.delete(signature)
      })
    pendingDirectoryRequests.set(signature, request)
  }
  const version = ++requestVersion
  loading.value = true
  if (reset) errorText.value = ''

  return (async () => {
    try {
      const response = await request
      if (version !== requestVersion) return
      errorText.value = ''
      const nextItems = (Array.isArray(response?.items) ? response.items : [])
        .map((item) => normalizeMerchantDirectoryItem(item))
        .filter((item) => item.merchantId)
      merchants.value = reset ? mergeMerchantDirectoryItems([], nextItems) : mergeMerchantDirectoryItems(merchants.value, nextItems)
      total.value = Math.max(0, Number(response?.total) || 0)
      page.value = targetPage
      recordDirectoryExposures(nextItems)
    } catch (err) {
      if (version !== requestVersion) return
      console.error('加载拿货档口目录失败', { page: targetPage, reset: Boolean(reset), err })
      errorText.value = '商家列表加载失败，请稍后重试'
      if (reset) {
        merchants.value = []
        total.value = 0
        page.value = 1
      }
    } finally {
      if (version === requestVersion) loading.value = false
    }
  })()
}

function recordDirectoryExposures(items) {
  for (const item of items) {
    if (!item.merchantId || exposedMerchantIds.has(item.merchantId)) continue
    exposedMerchantIds.add(item.merchantId)
    trackMerchantDirectoryEvent({
      merchantId: item.merchantId,
      eventType: 'directory_exposure',
      source: 'directory',
    })
  }
}

function submitSearch() {
  appliedDirectoryQuery = buildMerchantDirectoryQuery({
    cityCode: DEFAULT_CITY_CODE,
    keyword: keyword.value,
    categories: categoriesText.value,
    page: 1,
    pageSize: LIST_PAGE_SIZE,
  })
  return loadMerchants({ reset: true })
}

function applyCategoryFilter() {
  return submitSearch()
}

function clearCategories() {
  categoriesText.value = ''
  return submitSearch()
}

function clearConditions() {
  keyword.value = ''
  categoriesText.value = ''
  return submitSearch()
}

function recordMerchantClick(merchantId) {
  trackMerchantDirectoryEvent({
    merchantId,
    eventType: 'directory_merchant_click',
    source: 'directory',
  })
}

function openMerchantDetail(merchant) {
  const url = merchantDirectoryDetailPath(merchant)
  if (!url) return
  recordMerchantClick(merchant.merchantId)
  uni.navigateTo({ url })
}

function openMerchantLocation(merchant) {
  const merchantId = String(merchant?.merchantId || '').trim()
  if (!merchantId || !hasValidMerchantLocation(merchant)) return
  recordMerchantClick(merchantId)
  uni.navigateTo({ url: `/pages/merchant/location?merchantId=${encodeURIComponent(merchantId)}` })
}

function promptNavigationFeedback() {
  const target = claimNavigationFeedback()
  if (!target) return
  uni.showModal({
    title: '位置是否准确？',
    content: `${target.name || '该商家'} 的导航位置是否准确？`,
    confirmText: '位置有误',
    cancelText: '位置准确',
    success: ({ confirm }) => {
      if (confirm) {
        submitNavigationFeedback(target)
        return
      }
      completeNavigationFeedback(target)
    },
    fail: () => releaseNavigationFeedback(target),
  })
}

async function submitNavigationFeedback(target) {
  if (!target?.merchantId || reportSubmitting.value) return
  if (!requireLogin()) {
    // 登录跳转前归还认领但保留缓存，用户登录返回后可再次从 onShow 提交。
    releaseNavigationFeedback(target)
    return
  }
  reportSubmitting.value = true
  uni.showLoading({ title: '提交中', mask: true })
  try {
    const response = await submitMerchantLocationReport(target.merchantId, {
      reasonCode: 'navigation_inaccurate',
      description: '用户在腾讯地图导航后反馈位置不准确',
    })
    completeNavigationFeedback(target)
    uni.showToast({ title: response?.idempotent ? '已收到你的反馈' : '反馈已提交', icon: 'none' })
    trackMerchantDirectoryEvent({
      merchantId: target.merchantId,
      eventType: 'location_report_submit',
      source: 'directory',
    })
  } catch (err) {
    releaseNavigationFeedback(target)
    console.error('提交目录导航位置反馈失败', { merchantId: target.merchantId, err })
    uni.showToast({ title: '位置反馈提交失败，请稍后重试', icon: 'none' })
  } finally {
    uni.hideLoading()
    reportSubmitting.value = false
  }
}
</script>

<style lang="scss" scoped>
.sourcing-map-page {
  min-height: 100vh;
  background: $wplink-bg;
  color: $wplink-text;
}

.directory-toolbar {
  position: sticky;
  top: 0;
  z-index: 10;
  display: grid;
  gap: 14rpx;
  padding: 20rpx 24rpx 16rpx;
  border-bottom: 1rpx solid rgba(148, 163, 184, 0.22);
  background: rgba(255, 255, 255, 0.97);
}

.search-row,
.category-row {
  display: flex;
  align-items: center;
  gap: 12rpx;
}

.search-field,
.category-input {
  flex: 1;
  min-width: 0;
  height: 76rpx;
  border: 1rpx solid $wplink-line;
  border-radius: 10rpx;
  background: #f8fafc;
  font-size: 26rpx;
}

.search-field {
  display: flex;
  align-items: center;
  gap: 14rpx;
  padding: 0 22rpx;
}

.search-field input {
  flex: 1;
  min-width: 0;
  font-size: 27rpx;
}

.category-input {
  box-sizing: border-box;
  padding: 0 22rpx;
}

.search-icon {
  position: relative;
  width: 24rpx;
  height: 24rpx;
  border: 3rpx solid $wplink-muted;
  border-radius: 50%;
}

.search-icon::after {
  position: absolute;
  right: -9rpx;
  bottom: -6rpx;
  width: 12rpx;
  height: 3rpx;
  background: $wplink-muted;
  content: '';
  transform: rotate(45deg);
}

.search-button,
.category-apply,
.category-clear,
.state-button {
  margin: 0;
  border-radius: 9rpx;
  font-size: 24rpx;
}

.search-button,
.category-apply,
.state-button {
  background: $wplink-primary;
  color: $wplink-card;
}

.search-button {
  width: 112rpx;
  line-height: 76rpx;
}

.category-apply,
.category-clear {
  padding: 0 16rpx;
  line-height: 64rpx;
}

.category-clear {
  border: 1rpx solid $wplink-line;
  background: $wplink-card;
  color: $wplink-muted;
}

.search-button::after,
.category-apply::after,
.category-clear::after,
.state-button::after {
  border: 0;
}

.merchant-list {
  display: grid;
  gap: 18rpx;
  padding: 20rpx 24rpx calc(40rpx + env(safe-area-inset-bottom));
}

.state-card {
  display: grid;
  justify-items: center;
  gap: 14rpx;
  margin-top: 120rpx;
  padding: 44rpx 28rpx;
  text-align: center;
}

.state-title {
  font-size: 30rpx;
  font-weight: 750;
}

.state-desc {
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 1.6;
}

.state-button {
  min-width: 180rpx;
  margin-top: 10rpx;
  line-height: 68rpx;
}

.state-button.secondary {
  border: 1rpx solid $wplink-line;
  background: $wplink-card;
  color: $wplink-primary;
}

.load-more-text {
  padding: 14rpx;
  color: $wplink-muted;
  font-size: 22rpx;
  text-align: center;
}
</style>
