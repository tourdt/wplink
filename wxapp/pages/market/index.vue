<template>
  <view class="resource-page" :style="resourcePageStyle">
    <view class="resource-nav" :style="resourceNavStyle">
      <view class="resource-title-bar" :style="resourceTitleBarStyle">
        <text class="resource-title">供需市场</text>
      </view>
    </view>

    <view class="resource-toolbar" :style="resourceToolbarStyle">
      <view class="search-entry" @click="openSearchPage()">
        <text class="search-placeholder">{{ searchPlaceholder }}</text>
        <text class="search-action">搜索</text>
      </view>

      <view class="filter-shell">
        <scroll-view class="filter-row group-row" scroll-x scroll-with-animation>
          <button
            v-for="item in groupFilterOptions"
            :key="item.code"
            :class="['filter-button', item.code === filters.groupCode ? 'active' : '']"
            @click="selectGroup(item.code)"
          >
            {{ item.name }}
          </button>
        </scroll-view>
        <scroll-view
          class="filter-row"
          scroll-x
          scroll-with-animation
          :scroll-into-view="scrollIntoTypeId"
          :scroll-left="typeScrollLeft"
          @scroll="handleTypeScroll"
        >
          <button
            v-for="item in visibleResourceTypes"
            :key="item.value"
            :id="getTypeButtonId(item.value)"
            :class="['filter-button', item.value === filters.typeCode ? 'active' : '']"
            @click="selectType(item.value)"
          >
            {{ item.label }}
          </button>
        </scroll-view>
        <button
          v-if="resourceTypes.length > 1"
          class="all-type-button"
          @click="openTypeDrawer"
        >
          全部分类
        </button>
      </view>
    </view>

    <view v-if="rows.length" class="result-list">
      <template v-for="item in rows" :key="item.id">
        <DemandCard v-if="item.direction === RESOURCE_DIRECTION_DEMAND" :resource="item" @open="openResource" />
        <ResourceCard v-else :resource="item" @open="openResource" />
      </template>
      <text class="load-more-text">{{ loading ? '加载中...' : hasMore ? '上拉加载更多' : '没有更多了' }}</text>
    </view>

    <view v-else class="empty-card">
      <view class="empty-visual"></view>
      <text class="empty-title">{{ recommendationEmptyTitle }}</text>
      <text class="empty-desc">换个类型或搜索关键词。</text>
      <button class="primary-button" @click="openSearchPage()">去搜索</button>
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
import { computed, nextTick, reactive, ref } from 'vue'
import { onLoad, onPageScroll, onPullDownRefresh, onReachBottom } from '@dcloudio/uni-app'
import DemandCard from '../../components/DemandCard.vue'
import ResourceCard from '../../components/ResourceCard.vue'
import { DEFAULT_CITY_CODE } from '../../common/constants'
import { groupResourceTypes } from '../../common/resourceCategories'
import { listCityResourceTypes } from '../../api/city'
import { listResources } from '../../api/resource'

const resourceTypes = ref([{ label: '全部', value: '' }])
const categoryGroups = ref([])
const SEARCH_KEY = 'wplink_pending_search_keyword'
const PAGE_TITLE = '供需市场'
const RESOURCE_DIRECTION_DEMAND = 'demand'
const NAV_BOTTOM_RPX = 12
const headerMetrics = ref({
  statusBarHeight: 44,
  navBarHeight: 44,
  headerHeight: 94,
})
const rows = ref([])
const page = ref(1)
const pageSize = 20
const total = ref(0)
const hasMore = ref(true)
const loading = ref(false)
const filters = reactive({
  cityCode: DEFAULT_CITY_CODE,
  groupCode: '',
  typeCode: '',
})
const showTypeDrawer = ref(false)
const scrollIntoTypeId = ref('')
const typeScrollLeft = ref(0)
const pageScrollTop = ref(0)
const groupFilterOptions = computed(() => [{ code: '', name: '全部类目' }, ...categoryGroups.value])
const visibleResourceTypes = computed(() => resourceTypes.value)
const resourceNavStyle = computed(() => `padding-top: ${headerMetrics.value.statusBarHeight}px;`)
const resourceTitleBarStyle = computed(() => `height: ${headerMetrics.value.navBarHeight}px;`)
const resourcePageStyle = computed(() => `padding-top: calc(${headerMetrics.value.headerHeight}px + 24rpx);`)
const resourceToolbarStyle = computed(() => `top: ${headerMetrics.value.headerHeight}px;`)
const recommendationEmptyTitle = '暂无推荐内容'
const searchPlaceholder = '搜供应、需求、场地或服务'

onLoad(initResourcePage)
onPageScroll(handlePageScroll)

async function initResourcePage() {
  updateHeaderMetrics()
  uni.setNavigationBarTitle({ title: PAGE_TITLE })
  await loadResourceTypes()
  await loadRecommendedResources({ reset: true })
}

function updateHeaderMetrics() {
  try {
    const systemInfo = uni.getSystemInfoSync()
    const statusBarHeight = Number(systemInfo.statusBarHeight) || 44
    let menuTop = statusBarHeight + 6
    let menuHeight = 32

    if (typeof uni.getMenuButtonBoundingClientRect === 'function') {
      const menuButton = uni.getMenuButtonBoundingClientRect()
      menuTop = Number(menuButton.top) || menuTop
      menuHeight = Number(menuButton.height) || menuHeight
    }

    const navBarHeight = Math.max(44, (menuTop - statusBarHeight) * 2 + menuHeight)
    const bottomPadding = typeof uni.upx2px === 'function' ? uni.upx2px(NAV_BOTTOM_RPX) : 6
    headerMetrics.value = {
      statusBarHeight,
      navBarHeight,
      headerHeight: statusBarHeight + navBarHeight + bottomPadding,
    }
  } catch {
    headerMetrics.value = {
      statusBarHeight: 44,
      navBarHeight: 44,
      headerHeight: 94,
    }
  }
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
  categoryGroups.value = groupResourceTypes(resp.items || [])
  applyCurrentGroupTypes()
  await scrollToSelectedType(filters.typeCode)
}

async function loadRecommendedResources({ reset = true } = {}) {
  if (loading.value) return
  if (!reset && !hasMore.value) return
  loading.value = true
  try {
    const nextPage = reset ? 1 : page.value + 1
    const resp = await listResources({
      cityCode: filters.cityCode,
      groupCode: filters.groupCode,
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

async function selectGroup(groupCode) {
  if (filters.groupCode === groupCode) return
  filters.groupCode = groupCode
  filters.typeCode = ''
  showTypeDrawer.value = false
  applyCurrentGroupTypes()
  await scrollToSelectedType('')
  await loadRecommendedResources({ reset: true })
  await restorePageScroll()
}

async function selectType(typeCode) {
  filters.typeCode = typeCode
  showTypeDrawer.value = false
  await scrollToSelectedType(typeCode)
  await loadRecommendedResources({ reset: true })
}

function getTypeButtonId(typeCode) {
  const key = typeCode || 'all'
  return `resource-type-${String(key).replace(/[^a-zA-Z0-9_-]/g, '-')}`
}

function applyCurrentGroupTypes() {
  const selectedGroup = categoryGroups.value.find((item) => item.code === filters.groupCode)
  const groupedItems = selectedGroup
    ? selectedGroup.items
    : categoryGroups.value.flatMap((item) => item.items || [])
  const items = groupedItems.map((item) => ({
    label: item.typeName,
    value: item.typeCode,
  }))
  resourceTypes.value = [{ label: '全部', value: '' }, ...items]
  if (filters.typeCode && !items.some((item) => item.value === filters.typeCode)) {
    filters.typeCode = ''
  }
}

async function scrollToSelectedType(typeCode = filters.typeCode) {
  const nextId = getTypeButtonId(typeCode)
  if (scrollIntoTypeId.value === nextId) {
    scrollIntoTypeId.value = ''
    await nextTick()
  }
  scrollIntoTypeId.value = nextId
}

function openTypeDrawer() {
  showTypeDrawer.value = true
}

function closeTypeDrawer() {
  showTypeDrawer.value = false
}

function handlePageScroll(event = {}) {
  const scrollTop = Number(event.scrollTop) || 0
  pageScrollTop.value = scrollTop
}

function handleTypeScroll(event = {}) {
  const scrollLeft = Number(event.detail?.scrollLeft) || 0
  typeScrollLeft.value = scrollLeft
}

async function restorePageScroll(scrollTop = pageScrollTop.value) {
  pageScrollTop.value = Number(scrollTop) || 0
  await nextTick()
  if (typeof uni.pageScrollTo === 'function') {
    uni.pageScrollTo({ scrollTop, duration: 0 })
  }
}

function openSearchPage(keyword = '') {
  const searchOptions = {
    keyword,
    groupCode: filters.groupCode,
    typeCode: filters.typeCode,
    cityCode: filters.cityCode,
  }
  if (keyword || filters.groupCode || filters.typeCode || filters.cityCode !== DEFAULT_CITY_CODE) {
    uni.setStorageSync(SEARCH_KEY, searchOptions)
  } else {
    uni.removeStorageSync(SEARCH_KEY)
  }
  uni.navigateTo({ url: '/pages/search/index' })
}

function openResource(item) {
  uni.navigateTo({ url: `/pages/resource/detail?id=${item.id}` })
}
</script>

<style lang="scss" scoped>
.resource-page {
  min-height: 100vh;
  padding: 24rpx;
  background: $wplink-bg;
}

.resource-nav {
  position: fixed;
  top: 0;
  right: 0;
  left: 0;
  z-index: 30;
  box-sizing: border-box;
  padding-right: 24rpx;
  padding-bottom: 12rpx;
  padding-left: 24rpx;
  background: $wplink-bg;
  box-shadow: 0 8rpx 20rpx rgba(15, 23, 42, 0.04);
}

.resource-title-bar {
  display: flex;
  align-items: center;
  justify-content: center;
  box-sizing: border-box;
}

.resource-title {
  color: $wplink-primary;
  font-size: 30rpx;
  font-weight: 800;
}

.resource-toolbar {
  position: sticky;
  position: -webkit-sticky;
  z-index: 20;
  margin: -24rpx -24rpx 16rpx;
  padding: 24rpx 24rpx 16rpx;
  background: $wplink-bg;
  box-shadow: 0 10rpx 18rpx rgba(32, 42, 68, 0.06);
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
  display: grid;
  grid-template-columns: minmax(0, 1fr) 156rpx;
  align-items: center;
  gap: 12rpx;
}

.filter-row {
  flex: 1;
  min-width: 0;
  white-space: nowrap;
  overflow-x: auto;
}

.group-row {
  grid-column: 1 / -1;
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
  gap: 12rpx;
  padding: 32rpx 24rpx;
  border-radius: 12rpx;
  background: $wplink-card;
  text-align: center;
}

.empty-visual {
  display: block;
  width: 148rpx;
  height: 104rpx;
  padding: 0;
  border-radius: 12rpx;
  background:
    linear-gradient(140deg, rgba(255, 255, 255, 0.42), transparent 42%),
    repeating-linear-gradient(45deg, rgba(255, 255, 255, 0.22) 0 10rpx, transparent 10rpx 20rpx),
    rgba($wplink-primary, 0.18);
}

.empty-title {
  color: $wplink-text;
  font-size: 28rpx;
  font-weight: 700;
}

.empty-desc {
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 1.5;
}

.primary-button {
  width: 100%;
  max-width: 360rpx;
  height: 84rpx;
  border-radius: 12rpx;
}

.primary-button {
  background: $wplink-primary;
  color: $wplink-card;
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
