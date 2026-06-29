<template>
  <view class="search-page">
    <view class="search-guide">
      <text class="guide-title">找资源</text>
      <text class="guide-desc">输入关键词或先选热门条件，快速判断平台核实、认证状态、数量、价格和刷新时间。</text>
    </view>

    <view class="search-bar">
      <input v-model="keyword" class="search-input" placeholder="搜索库存、货源、工厂、服务" @confirm="search" />
      <button class="search-button" @click="search">搜索</button>
    </view>
    <view class="save-row">
      <button class="save-button" @click="saveCurrentSearch">保存搜索</button>
      <button class="save-button" @click="loadSavedSearches">刷新保存</button>
    </view>

    <view v-if="savedSearches.length" class="saved-row">
      <button v-for="item in savedSearches" :key="item.id" class="saved-button" @click="applySavedSearch(item)">
        {{ item.name }}
      </button>
    </view>

    <view class="filter-row">
      <button
        v-for="item in resourceTypes"
        :key="item.value"
        :class="['filter-button', item.value === filters.typeCode ? 'active' : '']"
        @click="selectType(item.value)"
      >
        {{ item.label }}
      </button>
    </view>

    <view class="hot-row">
      <text class="hot-label">热门：</text>
      <button v-for="item in hotKeywords" :key="item" class="hot-button" @click="searchHotKeyword(item)">
        {{ item }}
      </button>
    </view>

    <view v-if="rows.length" class="promotion-note">
      <text class="promotion-title">置顶资源</text>
      <text class="promotion-desc">推广资源均需审核通过，置顶只提升曝光，不替代真实性判断。</text>
    </view>

    <view v-if="rows.length" class="result-list">
      <ResourceCard v-for="item in rows" :key="item.id" :resource="item" @open="openResource" />
    </view>

    <view v-else-if="searched" class="empty-card">
      <view class="empty-visual">
        <text>找货</text>
      </view>
      <text class="empty-title">暂未找到合适资源</text>
      <text class="empty-desc">可以提交采购需求，平台运营会继续留意库存、货源或工厂。</text>
      <button class="primary-button" @click="openDemand">提交采购需求</button>
    </view>
  </view>
</template>

<script setup>
import { computed, reactive, ref } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import ResourceCard from '../../components/ResourceCard.vue'
import { DEFAULT_CITY_CODE } from '../../common/constants'
import { listCityResourceTypes } from '../../api/city'
import { createSavedSearch, listSavedSearches } from '../../api/favorite'
import { searchResources } from '../../api/resource'
import { getSession } from '../../store/session'

const resourceTypes = ref([{ label: '全部', value: '' }])
const hotKeywords = ['夏款现货', '急清库存', '小单快返', '直播供货']
const SEARCH_KEY = 'wplink_pending_search_keyword'
const keyword = ref('')
const rows = ref([])
const savedSearches = ref([])
const searched = ref(false)
const filters = reactive({
  cityCode: DEFAULT_CITY_CODE,
  typeCode: '',
})

onLoad(() => {
  loadResourceTypes()
  loadSavedSearches()
})
onShow(applyPendingKeyword)

async function loadResourceTypes() {
  const resp = await listCityResourceTypes(filters.cityCode)
  const items = (resp.items || []).map((item) => ({
    label: item.typeName,
    value: item.typeCode,
  }))
  resourceTypes.value = [{ label: '全部', value: '' }, ...items]
}

function applyPendingKeyword() {
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
  search()
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

async function loadSavedSearches() {
  if (!getSession().token) {
    savedSearches.value = []
    return
  }
  try {
    const resp = await listSavedSearches({ page: 1, pageSize: 10 })
    savedSearches.value = resp.items || []
  } catch (err) {
    savedSearches.value = []
  }
}

async function saveCurrentSearch() {
  try {
    const name = keyword.value.trim() || selectedTypeLabel.value || '认证资源'
    await createSavedSearch({
      name,
      cityCode: filters.cityCode,
      typeCode: filters.typeCode,
      keyword: keyword.value.trim(),
    })
    uni.showToast({ title: '已保存搜索', icon: 'none' })
    await loadSavedSearches()
  } catch (err) {
    uni.showToast({ title: err.message || '保存搜索失败，请稍后重试', icon: 'none' })
  }
}

function applySavedSearch(item) {
  keyword.value = item.keyword || ''
  filters.typeCode = item.typeCode || ''
  filters.cityCode = item.cityCode || DEFAULT_CITY_CODE
  search()
}

function searchHotKeyword(value) {
  keyword.value = value
  search()
}

function selectType(typeCode) {
  filters.typeCode = typeCode
  search()
}

const selectedTypeLabel = computed(() => {
  const selected = resourceTypes.value.find((item) => item.value === filters.typeCode)
  return selected?.label || ''
})

function openResource(item) {
  uni.navigateTo({ url: `/pages/resource/detail?id=${item.id}` })
}

function openDemand() {
  uni.navigateTo({ url: '/pages/demand/index' })
}
</script>

<style lang="scss" scoped>
.search-page {
  min-height: 100vh;
  padding: 24rpx;
  background: $wplink-bg;
}

.search-guide {
  display: grid;
  gap: 8rpx;
  margin-bottom: 20rpx;
  padding: 24rpx;
  border-radius: 12rpx;
  background: $wplink-card;
}

.guide-title {
  color: $wplink-primary;
  font-size: 36rpx;
  font-weight: 700;
}

.guide-desc {
  color: $wplink-muted;
  font-size: 26rpx;
  line-height: 1.5;
}

.search-bar {
  display: grid;
  grid-template-columns: 1fr 144rpx;
  gap: 16rpx;
  margin-bottom: 20rpx;
}

.save-row,
.saved-row {
  display: flex;
  gap: 12rpx;
  margin-bottom: 16rpx;
  overflow-x: auto;
}

.save-button,
.saved-button {
  flex: 0 0 auto;
  height: 62rpx;
  padding: 0 18rpx;
  border-radius: 10rpx;
  background: $wplink-card;
  color: $wplink-primary;
  font-size: 24rpx;
}

.saved-button {
  background: $wplink-primary-soft;
}

.search-input,
.search-button,
.filter-button {
  height: 80rpx;
  border-radius: 10rpx;
}

.search-input {
  padding: 0 20rpx;
  border: 1rpx solid $wplink-line;
  background: $wplink-card;
}

.search-button,
.primary-button {
  background: $wplink-primary;
  color: $wplink-card;
}

.filter-row {
  display: flex;
  gap: 12rpx;
  margin-bottom: 16rpx;
  overflow-x: auto;
}

.filter-button {
  min-width: 112rpx;
  padding: 0 20rpx;
  background: $wplink-card;
  color: #364152;
}

.filter-button.active {
  background: $wplink-warning-soft;
  color: $wplink-primary;
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

.promotion-note {
  display: grid;
  gap: 6rpx;
  margin-bottom: 18rpx;
  padding: 18rpx 20rpx;
  border-radius: 12rpx;
  background: $wplink-warning-soft;
}

.promotion-title {
  color: $wplink-warning;
  font-size: 26rpx;
  font-weight: 700;
}

.promotion-desc {
  color: #7c5a22;
  font-size: 24rpx;
  line-height: 1.5;
}

.result-list {
  display: grid;
  gap: 18rpx;
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

.primary-button {
  width: 100%;
  height: 84rpx;
  border-radius: 12rpx;
}
</style>
