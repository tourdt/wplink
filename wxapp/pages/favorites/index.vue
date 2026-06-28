<template>
  <view class="favorites-page">
    <view class="tab-row">
      <button
        v-for="item in tabs"
        :key="item.value"
        :class="['tab-button', activeTab === item.value ? 'active' : '']"
        @click="selectTab(item.value)"
      >
        {{ item.label }}
      </button>
    </view>

    <view v-if="activeTab === 'resources'" class="content-list">
      <view v-if="favoriteResources.length === 0" class="empty-card">暂无收藏资源</view>
      <ResourceCard v-for="item in favoriteResources" :key="item.id" :resource="item" @open="openResource" />
    </view>

    <view v-if="activeTab === 'merchants'" class="content-list">
      <view v-if="followedMerchants.length === 0" class="empty-card">暂无关注商家</view>
      <view v-for="item in followedMerchants" :key="item.id" class="merchant-item" @click="openMerchant(item)">
        <text class="merchant-name">{{ item.name }}</text>
        <text class="merchant-meta">{{ merchantTypeText[item.merchantType] || item.merchantType }} · {{ item.verificationStatus === 'verified' ? '已认证' : '未认证' }}</text>
      </view>
    </view>

    <view v-if="activeTab === 'searches'" class="content-list">
      <view v-if="savedSearches.length === 0" class="empty-card">暂无保存搜索</view>
      <view v-for="item in savedSearches" :key="item.id" class="search-item">
        <view @click="applySavedSearch(item)">
          <text class="merchant-name">{{ item.name }}</text>
          <text class="merchant-meta">{{ item.keyword || item.typeCode || '认证资源' }}</text>
        </view>
        <button class="delete-button" @click="removeSavedSearch(item)">删除</button>
      </view>
    </view>
  </view>
</template>

<script setup>
import { ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import ResourceCard from '../../components/ResourceCard.vue'
import { deleteSavedSearch, listFavoriteResources, listFollowedMerchants, listSavedSearches } from '../../api/favorite'

const tabs = [
  { label: '资源', value: 'resources' },
  { label: '商家', value: 'merchants' },
  { label: '搜索', value: 'searches' },
]
const activeTab = ref('resources')
const favoriteResources = ref([])
const followedMerchants = ref([])
const savedSearches = ref([])
const SEARCH_KEY = 'wplink_pending_search_keyword'
const merchantTypeText = {
  factory: '工厂',
  stall: '档口',
  stockist: '库存商',
  service_provider: '服务商',
  buyer: '采购商',
}

onLoad(loadAll)

function selectTab(value) {
  activeTab.value = value
}

async function loadAll() {
  await Promise.all([loadFavoriteResources(), loadFollowedMerchants(), loadSavedSearches()])
}

async function loadFavoriteResources() {
  try {
    const resp = await listFavoriteResources({ page: 1, pageSize: 20 })
    favoriteResources.value = resp.items || []
  } catch (err) {
    favoriteResources.value = []
  }
}

async function loadFollowedMerchants() {
  try {
    const resp = await listFollowedMerchants({ page: 1, pageSize: 20 })
    followedMerchants.value = resp.items || []
  } catch (err) {
    followedMerchants.value = []
  }
}

async function loadSavedSearches() {
  try {
    const resp = await listSavedSearches({ page: 1, pageSize: 20 })
    savedSearches.value = resp.items || []
  } catch (err) {
    savedSearches.value = []
  }
}

function openResource(item) {
  uni.navigateTo({ url: `/pages/resource/detail?id=${item.id}` })
}

function openMerchant(item) {
  uni.navigateTo({ url: `/pages/merchant/detail?id=${item.id}` })
}

function applySavedSearch(item) {
  uni.setStorageSync(SEARCH_KEY, {
    keyword: item.keyword || item.category || '',
    typeCode: item.typeCode || '',
    cityCode: item.cityCode || '',
  })
  uni.switchTab({ url: '/pages/search/index' })
}

async function removeSavedSearch(item) {
  try {
    await deleteSavedSearch(item.id)
    uni.showToast({ title: '已删除保存的搜索', icon: 'none' })
    await loadSavedSearches()
  } catch (err) {
    uni.showToast({ title: err.message || '删除失败，请稍后重试', icon: 'none' })
  }
}
</script>

<style scoped>
.favorites-page {
  min-height: 100vh;
  padding: 24rpx;
  background: #f4f6f8;
}

.tab-row {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12rpx;
  margin-bottom: 20rpx;
}

.tab-button {
  height: 72rpx;
  border-radius: 10rpx;
  background: #ffffff;
  color: #364152;
  font-size: 26rpx;
}

.tab-button.active {
  background: #0f766e;
  color: #ffffff;
}

.content-list {
  display: grid;
  gap: 16rpx;
}

.empty-card,
.merchant-item,
.search-item {
  padding: 24rpx;
  border-radius: 12rpx;
  background: #ffffff;
}

.merchant-name {
  display: block;
  margin-bottom: 8rpx;
  color: #1f2933;
  font-size: 30rpx;
  font-weight: 700;
}

.merchant-meta {
  color: #697586;
  font-size: 26rpx;
}

.search-item {
  display: grid;
  grid-template-columns: 1fr 120rpx;
  gap: 12rpx;
  align-items: center;
}

.delete-button {
  height: 64rpx;
  border-radius: 10rpx;
  background: #fff1f2;
  color: #be123c;
  font-size: 24rpx;
}
</style>
