<template>
  <view class="sourcing-map-page">
    <view class="directory-intro">
      <view><text class="intro-label">织里 · 找店拿货</text><text class="intro-title">找到合适的档口</text><text class="intro-desc">看主营，认门头，导航到店。</text></view>
      <button class="onboard-button" @click="openOnboarding">免费入驻 <text>＋</text></button>
    </view>
    <view class="directory-toolbar">
      <view class="search-row">
        <view class="search-field">
          <view class="search-icon" />
          <input
            v-model="keyword"
            placeholder="搜名称、市场或档口号"
            confirm-type="search"
            @confirm="submitSearch"
          />
        </view>
        <button class="search-button" :disabled="loading" :loading="loading" @click="submitSearch">
          {{ loading ? '搜索中' : '搜索' }}
        </button>
      </view>

      <view class="category-row">
        <button class="category-picker" @click="openCategoryFilter">{{ categoriesText ? '主营 · ' + categoriesText.replaceAll(',', ' / ') : '全部主营' }} <text>⌄</text></button>
        <button v-if="hasConditions" class="category-clear" @click="clearConditions">清除条件</button>
        <text class="directory-scope">织里档口</text>
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
        <text class="state-title">{{ hasConditions ? '暂未找到合适的档口' : '首批档口入驻中' }}</text>
        <text class="state-desc">{{ hasConditions ? '换个关键词，或减少主营条件再试试。' : '档口、工厂和服务者均可建立业务主页。' }}</text>
        <button v-if="hasConditions" class="state-button secondary" @click="clearConditions">清除条件</button>
        <button v-else class="state-button" @click="openOnboarding">免费入驻</button>
      </view>
      <template v-else>
        <MerchantPlaceCard
          v-for="merchant in merchants"
          :key="merchant.merchantId"
          :merchant="merchant"
          @detail="openMerchantDetail"
        />
        <text class="load-more-text">{{ loadMoreText }}</text>
      </template>
    </view>
    <view v-if="categoryFilterVisible" class="filter-mask" @click="categoryFilterVisible = false">
      <view class="filter-sheet" @click.stop>
        <view class="filter-heading"><text>选择主营</text><button @click="categoryFilterVisible = false">关闭</button></view>
        <text class="filter-hint">可多选，展示符合任一主营的档口</text>
        <checkbox-group class="category-options" @change="pendingCategories = $event.detail.value">
          <label v-for="category in MERCHANT_CATEGORY_OPTIONS" :key="category" :class="['category-option', { selected: pendingCategories.includes(category) }]">
            <checkbox :value="category" :checked="pendingCategories.includes(category)" color="#c23a00" />{{ category }}
          </label>
        </checkbox-group>
        <view class="filter-footer"><button class="filter-reset" @click="pendingCategories = []">重置</button><button class="filter-confirm" @click="confirmCategoryFilter">查看档口</button></view>
      </view>
    </view>
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onLoad, onPullDownRefresh, onReachBottom, onShow } from '@dcloudio/uni-app'
import MerchantPlaceCard from '../../components/MerchantPlaceCard.vue'
import { listMerchantDirectory, submitMerchantLocationReport } from '../../api/merchantDirectory'
import { requireLogin } from '../../common/auth'
import { MERCHANT_CATEGORY_OPTIONS } from '../../common/merchantCategories'
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
const categoryFilterVisible = ref(false)
const pendingCategories = ref([])
let hasShown = false
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
  // 从店铺管理返回时重新读取公开目录，暂停或地址变化不会留在旧列表中。
  if (hasShown) loadMerchants({ reset: true })
  hasShown = true
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
      console.error('加载拿货档口目录失败', { page: targetPage, reset: Boolean(reset) })
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

function openOnboarding() {
  if (!requireLogin({ redirect: '/pages/merchant/profile' })) return
  uni.navigateTo({ url: '/pages/merchant/profile' })
}

function openCategoryFilter() {
  pendingCategories.value = categoriesText.value.split(',').filter(Boolean)
  categoryFilterVisible.value = true
}

function confirmCategoryFilter() {
  categoriesText.value = pendingCategories.value.join(',')
  categoryFilterVisible.value = false
  return applyCategoryFilter()
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
    console.error('提交目录导航位置反馈失败', { merchantId: target.merchantId })
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
  background: #f6f7f8;
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
  background: #c23a00;
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
  margin-top: 64rpx;
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
.directory-intro { display: flex; justify-content: space-between; align-items: center; gap: 20rpx; padding: 32rpx 28rpx 28rpx; background: #fff; }
.directory-intro > view { display: grid; gap: 10rpx; }
.intro-label { color: #8a4b2a; font-size: 22rpx; letter-spacing: 2rpx; }
.intro-title { color: #19232e; font-size: 42rpx; font-weight: 750; }
.intro-desc { color: #78818c; font-size: 24rpx; }
.onboard-button { flex-shrink: 0; border: 1rpx solid #ead4c9; color: #a7380b; padding: 16rpx 20rpx; border-radius: 12rpx; background: #fff8f4; font-size: 24rpx; gap: 8rpx; }
.category-picker { max-width: 75%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; justify-content: flex-start; gap: 16rpx; padding: 15rpx 20rpx; font-size: 25rpx; color: #303a45; background: #f3f4f5; border-radius: 10rpx; }
.directory-scope { margin-left: auto; white-space: nowrap; color: #858d97; font-size: 22rpx; }
.filter-mask { position: fixed; inset: 0; z-index: 90; background: rgba(20,28,36,.4); display: flex; align-items: flex-end; }
.filter-sheet { width: 100%; max-height: 85vh; overflow: auto; background: #fff; border-radius: 28rpx 28rpx 0 0; padding: 32rpx 28rpx calc(28rpx + env(safe-area-inset-bottom)); }
.filter-heading { display: flex; align-items: center; justify-content: space-between; font-size: 34rpx; font-weight: 700; }
.filter-heading button { background: transparent; color: #6b7280; padding: 16rpx; font-size: 25rpx; font-weight: 400; }
.filter-hint { display: block; color: #777f89; margin: 12rpx 0 28rpx; font-size: 24rpx; }
.category-options { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 16rpx; }
.category-option { display: flex; align-items: center; justify-content: center; min-height: 76rpx; border: 1rpx solid #e2e5e9; border-radius: 10rpx; font-size: 24rpx; }
.category-option.selected { color: #b6400d; background: #fff4ed; border-color: #d9a588; }
.category-option checkbox { transform: scale(.7); }
.filter-footer { display: flex; gap: 20rpx; margin-top: 32rpx; }
.filter-reset { flex: 1; background: #f3f4f5; color: #48525d; height: 88rpx; }
.filter-confirm { flex: 2; background: #c23a00; color: white; height: 88rpx; }
</style>
