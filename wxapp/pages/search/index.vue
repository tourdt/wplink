<template>
  <view class="resource-page">
    <view class="search-entry" @click="openSearchPage()">
      <text class="search-placeholder">搜索库存、货源、工厂、服务</text>
      <text class="search-action">搜索</text>
    </view>

    <view class="filter-shell">
      <scroll-view class="filter-row" scroll-x>
        <button
          v-for="item in visibleResourceTypes"
          :key="item.value"
          :class="['filter-button', item.value === filters.typeCode ? 'active' : '']"
          @click="selectType(item.value)"
        >
          {{ item.label }}
        </button>
      </scroll-view>
      <button
        v-if="resourceTypes.length > visibleResourceTypes.length"
        class="all-type-button"
        @click="openTypeDrawer"
      >
        全部分类
      </button>
    </view>
    </view>

    <view v-if="rows.length" class="result-list">
      <ResourceCard v-for="item in rows" :key="item.id" :resource="item" @open="openResource" />
      <text class="load-more-text">{{ loading ? '加载中...' : hasMore ? '上拉加载更多' : '没有更多了' }}</text>
    </view>

    <view v-else class="empty-card">
      <view class="empty-visual">
        <text>资源</text>
      </view>
      <text class="empty-title">当前类型暂无推荐资源</text>
      <text class="empty-desc">可以换个类型继续浏览，或直接搜索关键词、提交采购需求。</text>
      <button class="primary-button" @click="openSearchPage()">去搜索</button>
      <button class="secondary-button" @click="openDemand">提交采购需求</button>
    </view>

    <view v-if="showTypeDrawer" class="type-drawer-mask" @click="closeTypeDrawer">
      <view class="type-drawer-panel" @click.stop>
        <view class="type-drawer-head">
          <text class="type-drawer-title">全部分类</text>
          <button class="type-drawer-close" @click="closeTypeDrawer">关闭</button>
        </view>
        <text class="type-drawer-subtitle">常用分类</text>
        <view class="drawer-type-grid">
          <button
            v-for="item in resourceTypes"
            :key="item.value"
            :class="['drawer-type-button', item.value === filters.typeCode ? 'active' : '']"
            @click="selectType(item.value)"
          >
            {{ item.label }}
          </button>
        </view>
      </view>
    </view>
  </view>
</template>

<script setup>
import { computed, reactive, ref } from 'vue'
import { onLoad, onPullDownRefresh, onReachBottom } from '@dcloudio/uni-app'
import ResourceCard from '../../components/ResourceCard.vue'
import { DEFAULT_CITY_CODE } from '../../common/constants'
import { listCityResourceTypes } from '../../api/city'
import { listResources } from '../../api/resource'

const resourceTypes = ref([{ label: '全部', value: '' }])
const MAX_VISIBLE_RESOURCE_TYPES = 6
const SEARCH_KEY = 'wplink_pending_search_keyword'
const PAGE_TITLE = '资源推荐'
const rows = ref([])
const page = ref(1)
const pageSize = 20
const total = ref(0)
const hasMore = ref(true)
const loading = ref(false)
const filters = reactive({
  cityCode: DEFAULT_CITY_CODE,
  typeCode: '',
})
const showTypeDrawer = ref(false)
const visibleResourceTypes = computed(() => resourceTypes.value.slice(0, MAX_VISIBLE_RESOURCE_TYPES))

onLoad(initResourcePage)

async function initResourcePage() {
  uni.setNavigationBarTitle({ title: PAGE_TITLE })
  await loadResourceTypes()
  await loadRecommendedResources({ reset: true })
}

onPullDownRefresh(async () => {
  try {
    await loadRecommendedResources({ reset: true })
  } finally {
    uni.stopPullDownRefresh()
  }
})

onReachBottom(() => {
  loadRecommendedResources({ reset: false })
})

async function loadResourceTypes() {
  const resp = await listCityResourceTypes(filters.cityCode)
  const items = (resp.items || []).map((item) => ({
    label: item.typeName,
    value: item.typeCode,
  }))
  resourceTypes.value = [{ label: '全部', value: '' }, ...items]
}

async function loadRecommendedResources({ reset = true } = {}) {
  if (loading.value) return
  if (!reset && !hasMore.value) return
  loading.value = true
  try {
    const nextPage = reset ? 1 : page.value + 1
    const resp = await listResources({
      cityCode: filters.cityCode,
      typeCode: filters.typeCode,
      page: nextPage,
      pageSize,
    })
    const items = resp.items || []
    rows.value = reset ? items : [...rows.value, ...items]
    page.value = nextPage
    total.value = resp.total || rows.value.length
    hasMore.value = rows.value.length < total.value
  } finally {
    loading.value = false
  }
}

async function selectType(typeCode) {
  filters.typeCode = typeCode
  showTypeDrawer.value = false
  await loadRecommendedResources({ reset: true })
}

function openTypeDrawer() {
  showTypeDrawer.value = true
}

function closeTypeDrawer() {
  showTypeDrawer.value = false
}

function openSearchPage(keyword = '') {
  if (keyword) {
    uni.setStorageSync(SEARCH_KEY, keyword)
  } else {
    uni.removeStorageSync(SEARCH_KEY)
  }
  uni.navigateTo({ url: '/pages/search/result' })
}

function openResource(item) {
  uni.navigateTo({ url: `/pages/resource/detail?id=${item.id}` })
}

function openDemand() {
  uni.navigateTo({ url: '/pages/demand/index' })
}
</script>

<style lang="scss" scoped>
.resource-page {
  min-height: 100vh;
  padding: 24rpx;
  background: $wplink-bg;
}

.search-entry {
  display: grid;
  grid-template-columns: 1fr 116rpx;
  align-items: center;
  gap: 16rpx;
  margin-bottom: 20rpx;
  padding: 0 18rpx 0 22rpx;
  min-height: 80rpx;
  border: 1rpx solid $wplink-line;
  border-radius: 10rpx;
  background: $wplink-card;
}

.search-placeholder {
  color: $wplink-muted;
  font-size: 28rpx;
}

.search-action {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 56rpx;
  border-radius: 10rpx;
  background: $wplink-primary;
  color: $wplink-card;
  font-size: 26rpx;
  font-weight: 700;
}

.filter-shell {
  display: flex;
  align-items: center;
  gap: 12rpx;
  margin-bottom: 16rpx;
}

.filter-row {
  flex: 1;
  min-width: 0;
  white-space: nowrap;
  overflow-x: auto;
}

.filter-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 112rpx;
  height: 80rpx;
  margin-right: 12rpx;
  padding: 0 20rpx;
  border-radius: 10rpx;
  background: $wplink-card;
  color: #364152;
  font-size: 26rpx;
}

.filter-button.active {
  background: $wplink-warning-soft;
  color: $wplink-primary;
}

.all-type-button {
  flex: 0 0 auto;
  width: 156rpx;
  height: 80rpx;
  border-radius: 10rpx;
  background: $wplink-primary-soft;
  color: $wplink-primary;
  font-size: 24rpx;
  font-weight: 700;
}

.result-list {
  display: grid;
  gap: 18rpx;
}

.load-more-text {
  padding: 8rpx 0 18rpx;
  color: $wplink-muted;
  font-size: 24rpx;
  text-align: center;
}

.empty-card {
  display: grid;
  justify-items: center;
  gap: 14rpx;
  padding: 40rpx 24rpx;
  border-radius: 12rpx;
  background: $wplink-card;
  text-align: center;
}

.empty-visual {
  display: flex;
  align-items: flex-end;
  justify-content: flex-start;
  width: 188rpx;
  height: 136rpx;
  padding: 18rpx;
  border-radius: 12rpx;
  background:
    linear-gradient(140deg, rgba(255, 255, 255, 0.22), transparent 38%),
    repeating-linear-gradient(45deg, rgba(255, 255, 255, 0.18) 0 12rpx, transparent 12rpx 24rpx),
    #7b8fc7;
  color: $wplink-card;
  font-size: 26rpx;
  font-weight: 700;
}

.empty-title {
  color: $wplink-primary;
  font-size: 32rpx;
  font-weight: 700;
}

.empty-desc {
  color: $wplink-muted;
  font-size: 28rpx;
  line-height: 1.5;
}

.primary-button,
.secondary-button {
  width: 100%;
  height: 84rpx;
  border-radius: 12rpx;
}

.primary-button {
  background: $wplink-primary;
  color: $wplink-card;
}

.secondary-button {
  background: $wplink-primary-soft;
  color: $wplink-primary;
}

.type-drawer-mask {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 30;
  display: flex;
  align-items: flex-end;
  background: rgba(15, 23, 42, 0.38);
}

.type-drawer-panel {
  width: 100%;
  max-height: 72vh;
  padding: 26rpx 24rpx calc(30rpx + env(safe-area-inset-bottom));
  border-radius: 16rpx 16rpx 0 0;
  background: $wplink-bg;
}

.type-drawer-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16rpx;
}

.type-drawer-title {
  color: $wplink-primary;
  font-size: 32rpx;
  font-weight: 700;
}

.type-drawer-close {
  width: 112rpx;
  height: 58rpx;
  border-radius: 10rpx;
  background: $wplink-card;
  color: $wplink-muted;
  font-size: 24rpx;
}

.type-drawer-subtitle {
  display: block;
  margin-bottom: 16rpx;
  color: $wplink-muted;
  font-size: 24rpx;
}

.drawer-type-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12rpx;
}

.drawer-type-button {
  height: 72rpx;
  padding: 0 10rpx;
  border-radius: 10rpx;
  background: $wplink-card;
  color: #364152;
  font-size: 24rpx;
}

.drawer-type-button.active {
  background: $wplink-warning-soft;
  color: $wplink-primary;
  font-weight: 700;
}
</style>
