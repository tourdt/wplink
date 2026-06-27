<template>
  <view class="home-page">
    <view class="search-entry" @click="openSearch">
      <text>搜索库存、货源、工厂、服务</text>
    </view>

    <view class="banner-list">
      <view v-for="item in banners" :key="item.id" class="banner-card" @click="openBanner(item)">
        <image v-if="item.coverUrl" class="banner-image" :src="item.coverUrl" mode="aspectFill" />
        <view class="banner-copy">
          <text class="banner-title">{{ item.title }}</text>
          <text class="banner-subtitle">{{ item.subtitle }}</text>
        </view>
      </view>
    </view>

    <view class="quick-grid">
      <button @click="openSearch">找资源</button>
      <button @click="openDemand">提需求</button>
      <button @click="openPublish">发资源</button>
    </view>
  </view>
</template>

<script setup>
import { ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { DEFAULT_CITY_CODE } from '../../common/constants'
import { listHomeBanners } from '../../api/discovery'

const banners = ref([])

onLoad(loadBanners)

async function loadBanners() {
  // 首页首屏由运营配置驱动，失败时保留空列表，不阻断搜索和发布入口。
  const resp = await listHomeBanners({ cityCode: DEFAULT_CITY_CODE })
  banners.value = resp.items || []
}

function openBanner(item) {
  if (item.jumpType === 'topic') {
    uni.navigateTo({ url: `/pages/topic/index?id=${item.jumpTarget}` })
    return
  }
  if (item.jumpType === 'resource') {
    uni.navigateTo({ url: `/pages/resource/detail?id=${item.jumpTarget}` })
    return
  }
  if (item.jumpType === 'merchant') {
    uni.navigateTo({ url: `/pages/merchant/detail?id=${item.jumpTarget}` })
    return
  }
  if (item.jumpType === 'webview') {
    uni.navigateTo({ url: `/pages/webview/index?url=${encodeURIComponent(item.jumpTarget)}` })
    return
  }
  if (item.jumpType === 'internal' && item.jumpTarget) {
    uni.navigateTo({ url: item.jumpTarget })
    return
  }
  openDemand()
}

function openSearch() {
  uni.navigateTo({ url: '/pages/search/index' })
}

function openDemand() {
  uni.navigateTo({ url: '/pages/demand/index' })
}

function openPublish() {
  uni.navigateTo({ url: '/pages/publish/index' })
}
</script>

<style scoped>
.home-page {
  min-height: 100vh;
  padding: 24rpx;
  background: #f4f6f8;
}

.search-entry {
  display: flex;
  align-items: center;
  height: 82rpx;
  margin-bottom: 20rpx;
  padding: 0 24rpx;
  border-radius: 12rpx;
  background: #ffffff;
  color: #697586;
}

.banner-list {
  display: grid;
  gap: 18rpx;
}

.banner-card {
  position: relative;
  min-height: 220rpx;
  overflow: hidden;
  border-radius: 12rpx;
  background: #0f766e;
}

.banner-image {
  width: 100%;
  height: 220rpx;
}

.banner-copy {
  position: absolute;
  right: 24rpx;
  bottom: 24rpx;
  left: 24rpx;
  display: grid;
  gap: 8rpx;
  color: #ffffff;
}

.banner-title {
  font-size: 38rpx;
  font-weight: 700;
}

.banner-subtitle {
  font-size: 26rpx;
}

.quick-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16rpx;
  margin-top: 20rpx;
}

.quick-grid button {
  height: 84rpx;
  border-radius: 12rpx;
  background: #ffffff;
  color: #1f2933;
}
</style>
