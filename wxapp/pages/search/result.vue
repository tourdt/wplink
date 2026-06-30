<template>
  <view class="search-page">
    <view class="search-bar">
      <input v-model="keyword" class="search-input" placeholder="搜索库存、货源、工厂、服务" @confirm="search" />
      <button class="search-button" @click="search">搜索</button>
    </view>

    <view class="filter-shell">
      <scroll-view
        class="filter-row"
        scroll-x
        scroll-with-animation
        :scroll-into-view="scrollIntoTypeId"
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

    <view class="hot-row">
      <text class="hot-label">热门：</text>
      <button v-for="item in hotKeywords" :key="item" class="hot-button" @click="searchHotKeyword(item)">
        {{ item }}
      </button>
    </view>

    <view v-if="rows.length" class="result-list">
      <ResourceCard v-for="item in rows" :key="item.id" :resource="item" @open="openResource" />
    </view>

    <view v-else-if="searched" class="empty-card">
      <view class="empty-visual">
        <view class="empty-sheet">
          <text></text>
          <text></text>
          <text></text>
        </view>
        <view class="empty-magnifier"></view>
      </view>
      <text class="empty-title">{{ emptyTitle }}</text>
      <text class="empty-desc">{{ emptyDesc }}</text>
      <view v-if="emptySuggestions.length" class="empty-suggestions">
        <button
          v-for="item in emptySuggestions"
          :key="item"
          class="empty-suggestion"
          @click="searchHotKeyword(item)"
        >
          {{ item }}
        </button>
      </view>
      <view class="empty-actions">
        <button class="secondary-button" @click="resetSearchConditions">换个条件</button>
      </view>
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
import { onLoad, onShow } from '@dcloudio/uni-app'
import ResourceCard from '../../components/ResourceCard.vue'
import { DEFAULT_CITY_CODE } from '../../common/constants'
import { listCityResourceTypes } from '../../api/city'
import { searchResources } from '../../api/resource'

const resourceTypes = ref([{ label: '全部', value: '' }])
const hotKeywords = ['夏款现货', '急清库存', '小单快返', '直播供货']
const SEARCH_KEY = 'wplink_pending_search_keyword'
const keyword = ref('')
const rows = ref([])
const searched = ref(false)
const filters = reactive({
  cityCode: DEFAULT_CITY_CODE,
  typeCode: '',
})
const showTypeDrawer = ref(false)
const scrollIntoTypeId = ref('')
const visibleResourceTypes = computed(() => resourceTypes.value)
const trimmedKeyword = computed(() => keyword.value.trim())
const emptyTitle = '暂无匹配资源'
const emptyDesc = '换个关键词或分类试试。'
const emptySuggestions = computed(() => hotKeywords
  .filter((item) => item !== trimmedKeyword.value)
  .slice(0, 3))

onLoad(async (options = {}) => {
  await loadResourceTypes()
  await applyRouteSearch(options)
})
onShow(applyPendingKeyword)

async function loadResourceTypes() {
  const resp = await listCityResourceTypes(filters.cityCode)
  const items = (resp.items || []).map((item) => ({
    label: item.typeName,
    value: item.typeCode,
  }))
  resourceTypes.value = [{ label: '全部', value: '' }, ...items]
  if (filters.typeCode) {
    await scrollToSelectedType(filters.typeCode)
  }
}

async function applyRouteSearch(options = {}) {
  const routeKeyword = decodeSearchValue(options.keyword || options.q || '')
  const routeTypeCode = decodeSearchValue(options.typeCode || '')
  if (!routeKeyword && !routeTypeCode) return
  keyword.value = routeKeyword
  filters.typeCode = routeTypeCode
  filters.cityCode = decodeSearchValue(options.cityCode || '') || DEFAULT_CITY_CODE
  await scrollToSelectedType(routeTypeCode)
  await search()
}

async function applyPendingKeyword() {
  const pendingSearch = uni.getStorageSync(SEARCH_KEY)
  if (!pendingSearch) return
  uni.removeStorageSync(SEARCH_KEY)
  if (typeof pendingSearch === 'string') {
    keyword.value = pendingSearch
  } else {
    keyword.value = pendingSearch.keyword || ''
    filters.typeCode = pendingSearch.typeCode || ''
    filters.cityCode = pendingSearch.cityCode || DEFAULT_CITY_CODE
  }
  await scrollToSelectedType(filters.typeCode)
  await search()
}

async function search() {
  const resp = await searchResources({
    ...filters,
    keyword: keyword.value.trim(),
    page: 1,
    pageSize: 20,
  })
  rows.value = resp.items || []
  searched.value = true
}

function searchHotKeyword(value) {
  keyword.value = value
  search()
}

async function resetSearchConditions() {
  keyword.value = ''
  filters.typeCode = ''
  rows.value = []
  searched.value = false
  await scrollToSelectedType('')
}

async function selectType(typeCode) {
  filters.typeCode = typeCode
  showTypeDrawer.value = false
  await scrollToSelectedType(typeCode)
  await search()
}

function getTypeButtonId(typeCode) {
  const key = typeCode || 'all'
  return `search-type-${String(key).replace(/[^a-zA-Z0-9_-]/g, '-')}`
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

function decodeSearchValue(value) {
  if (!value) return ''
  try {
    return decodeURIComponent(value)
  } catch (err) {
    return value
  }
}

function openResource(item) {
  uni.navigateTo({ url: `/pages/resource/detail?id=${item.id}` })
}
</script>

<style lang="scss" scoped>
.search-page {
  min-height: 100vh;
  padding: 24rpx;
  background: $wplink-bg;
}

.search-bar {
  display: grid;
  grid-template-columns: 1fr 144rpx;
  gap: 16rpx;
  margin-bottom: 20rpx;
}

.search-input,
.search-button,
.filter-button,
.all-type-button {
  height: 80rpx;
  border-radius: 10rpx;
}

.search-input {
  padding: 0 20rpx;
  border: 1rpx solid $wplink-line;
  background: $wplink-card;
}

.search-button {
  background: $wplink-primary;
  color: $wplink-card;
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
  margin-right: 12rpx;
  padding: 0 20rpx;
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
  background: $wplink-primary-soft;
  color: $wplink-primary;
  font-size: 24rpx;
  font-weight: 700;
}

.hot-row {
  display: flex;
  align-items: center;
  gap: 12rpx;
  margin-bottom: 20rpx;
  overflow-x: auto;
}

.hot-label {
  flex: 0 0 auto;
  color: $wplink-muted;
  font-size: 26rpx;
}

.hot-button {
  flex: 0 0 auto;
  min-width: 136rpx;
  height: 62rpx;
  padding: 0 18rpx;
  border-radius: 10rpx;
  background: $wplink-card;
  color: #364152;
  font-size: 24rpx;
}

.result-list {
  display: grid;
  gap: 18rpx;
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
  position: relative;
  display: grid;
  place-items: center;
  width: 168rpx;
  height: 112rpx;
  margin-bottom: 2rpx;
  border-radius: 12rpx;
  background:
    linear-gradient(140deg, rgba(255, 255, 255, 0.28), transparent 42%),
    linear-gradient(135deg, rgba($wplink-primary, 0.32), rgba($wplink-blue, 0.18));
  overflow: hidden;
}

.empty-visual::before {
  position: absolute;
  top: 16rpx;
  right: 20rpx;
  width: 42rpx;
  height: 8rpx;
  border-radius: 999rpx;
  background: rgba(255, 255, 255, 0.34);
  content: '';
}

.empty-sheet {
  position: absolute;
  left: 26rpx;
  bottom: 18rpx;
  display: grid;
  gap: 7rpx;
  width: 64rpx;
  padding: 12rpx 10rpx;
  border-radius: 8rpx;
  background: rgba(255, 255, 255, 0.92);
  box-shadow: 0 8rpx 16rpx rgba(15, 23, 42, 0.08);
}

.empty-sheet text {
  display: block;
  height: 5rpx;
  border-radius: 999rpx;
  background: rgba($wplink-primary, 0.18);
}

.empty-sheet text:first-child {
  width: 44rpx;
  background: rgba($wplink-warning, 0.46);
}

.empty-magnifier {
  position: absolute;
  right: 32rpx;
  bottom: 28rpx;
  width: 42rpx;
  height: 42rpx;
  border: 6rpx solid rgba(255, 255, 255, 0.9);
  border-radius: 999rpx;
}

.empty-magnifier::after {
  position: absolute;
  right: -20rpx;
  bottom: -9rpx;
  width: 28rpx;
  height: 6rpx;
  border-radius: 999rpx;
  background: rgba(255, 255, 255, 0.9);
  content: '';
  transform: rotate(45deg);
}

.empty-title {
  color: $wplink-text;
  font-size: 28rpx;
  font-weight: 700;
}

.empty-desc {
  max-width: 560rpx;
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 1.5;
}

.empty-suggestions {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 12rpx;
  width: 100%;
  margin-top: 2rpx;
}

.empty-suggestion {
  height: 58rpx;
  padding: 0 18rpx;
  border-radius: 999rpx;
  background: $wplink-primary-soft;
  color: $wplink-primary;
  font-size: 24rpx;
}

.empty-actions {
  display: flex;
  justify-content: center;
  width: 100%;
  margin-top: 4rpx;
}

.secondary-button {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  max-width: 300rpx;
  height: 72rpx;
  border-radius: 12rpx;
  background: $wplink-card;
  color: $wplink-primary;
  font-size: 26rpx;
  font-weight: 700;
  box-shadow: inset 0 0 0 1rpx $wplink-line;
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
