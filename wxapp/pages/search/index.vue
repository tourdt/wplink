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

    <view v-if="rows.length" class="result-list">
      <ResourceCard v-for="item in rows" :key="item.id" :resource="item" @open="openResource" />
    </view>

    <view v-else-if="searched" class="empty-card">
      <text class="empty-title">暂未找到合适资源</text>
      <text class="empty-desc">提交采购需求后，运营会协助撮合。</text>
      <button class="primary-button" @click="openDemand">提交需求</button>
    </view>
  </view>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import ResourceCard from '../../components/ResourceCard.vue'
import { DEFAULT_CITY_CODE } from '../../common/constants'
import { listCityResourceTypes } from '../../api/city'
import { searchResources } from '../../api/resource'

const resourceTypes = ref([{ label: '全部', value: '' }])
const keyword = ref('')
const rows = ref([])
const searched = ref(false)
const filters = reactive({
  cityCode: DEFAULT_CITY_CODE,
  typeCode: '',
})

onLoad(loadResourceTypes)

async function loadResourceTypes() {
  const resp = await listCityResourceTypes(filters.cityCode)
  const items = (resp.items || []).map((item) => ({
    label: item.typeName,
    value: item.typeCode,
  }))
  resourceTypes.value = [{ label: '全部', value: '' }, ...items]
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
  margin-bottom: 20rpx;
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
