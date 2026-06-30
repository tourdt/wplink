<template>
  <view class="search-page">
    <view class="search-bar">
      <input v-model="keyword" class="search-input" placeholder="搜索库存、货源、工厂、服务" @confirm="search" />
      <button class="search-button" @click="search">搜索</button>
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

    <view v-if="rows.length" class="result-list">
      <ResourceCard v-for="item in rows" :key="item.id" :resource="item" @open="openResource" />
    </view>

    <view v-else-if="searched" class="empty-card">
      <view class="empty-visual">
        <text>找货</text>
      </view>
      <text class="empty-title">暂未找到合适资源</text>
      <button class="primary-button" @click="openDemand">提交采购需求</button>
    </view>
  </view>
</template>

<script setup>
import { reactive, ref } from 'vue'
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

onLoad((options = {}) => {
  loadResourceTypes()
  applyRouteSearch(options)
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

function applyRouteSearch(options = {}) {
  const routeKeyword = decodeSearchValue(options.keyword || options.q || '')
  const routeTypeCode = decodeSearchValue(options.typeCode || '')
  if (!routeKeyword && !routeTypeCode) return
  keyword.value = routeKeyword
  filters.typeCode = routeTypeCode
  filters.cityCode = decodeSearchValue(options.cityCode || '') || DEFAULT_CITY_CODE
  search()
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

function searchHotKeyword(value) {
  keyword.value = value
  search()
}

function selectType(typeCode) {
  filters.typeCode = typeCode
  search()
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

.search-bar {
  display: grid;
  grid-template-columns: 1fr 144rpx;
  gap: 16rpx;
  margin-bottom: 20rpx;
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

.primary-button {
  width: 100%;
  height: 84rpx;
  border-radius: 12rpx;
}
</style>
