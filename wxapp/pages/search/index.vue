<template>
  <view class="search-page">
    <view class="search-guide">
      <text class="guide-title">找资源</text>
      <text class="guide-desc">输入关键词或先选热门条件，找不到时可直接提交采购需求。</text>
    </view>

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

    <view v-if="rows.length" class="promotion-note">
      <text class="promotion-title">置顶资源</text>
      <text class="promotion-desc">推广资源均需审核通过，置顶只提升曝光，不替代真实性判断。</text>
    </view>

    <view v-if="rows.length" class="result-list">
      <ResourceCard v-for="item in rows" :key="item.id" :resource="item" @open="openResource" />
    </view>

    <view v-else-if="searched" class="empty-card">
      <text class="empty-title">暂未找到合适资源</text>
      <text class="empty-desc">提交采购需求后，运营会协助撮合。</text>
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

onLoad(loadResourceTypes)
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
  const pendingKeyword = uni.getStorageSync(SEARCH_KEY)
  if (!pendingKeyword) return
  uni.removeStorageSync(SEARCH_KEY)
  keyword.value = pendingKeyword
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

function openResource(item) {
  uni.navigateTo({ url: `/pages/resource/detail?id=${item.id}` })
}

function openDemand() {
  uni.navigateTo({ url: '/pages/demand/index' })
}
</script>

<style scoped>
.search-page {
  min-height: 100vh;
  padding: 24rpx;
  background: #f4f6f8;
}

.search-guide {
  display: grid;
  gap: 8rpx;
  margin-bottom: 20rpx;
  padding: 24rpx;
  border-radius: 12rpx;
  background: #ffffff;
}

.guide-title {
  color: #1f2933;
  font-size: 36rpx;
  font-weight: 700;
}

.guide-desc {
  color: #697586;
  font-size: 26rpx;
  line-height: 1.5;
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
  border: 1rpx solid #d8dde6;
  background: #ffffff;
}

.search-button,
.primary-button {
  background: #0f766e;
  color: #ffffff;
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
  background: #ffffff;
  color: #364152;
}

.filter-button.active {
  background: #d9f3ef;
  color: #0f766e;
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
  color: #697586;
  font-size: 26rpx;
}

.hot-button {
  flex: 0 0 auto;
  min-width: 136rpx;
  height: 62rpx;
  padding: 0 18rpx;
  border-radius: 10rpx;
  background: #ffffff;
  color: #364152;
  font-size: 24rpx;
}

.promotion-note {
  display: grid;
  gap: 6rpx;
  margin-bottom: 18rpx;
  padding: 18rpx 20rpx;
  border-radius: 12rpx;
  background: #fff7e6;
}

.promotion-title {
  color: #b7791f;
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
  gap: 12rpx;
  padding: 24rpx;
  border-radius: 12rpx;
  background: #ffffff;
}

.empty-title {
  color: #1f2933;
  font-size: 32rpx;
  font-weight: 700;
}

.empty-desc {
  color: #697586;
  font-size: 28rpx;
}

.primary-button {
  height: 84rpx;
  border-radius: 12rpx;
}
</style>
