<template>
  <view class="home-page">
    <view class="search-entry" @click="openSearch()">
      <text class="search-placeholder">搜索童装库存、工厂、货源</text>
      <text class="search-action">搜索</text>
    </view>

    <scroll-view class="banner-list" scroll-x>
      <view v-for="item in displayBanners" :key="item.id" class="banner-card" @click="openBanner(item)">
        <image v-if="item.coverUrl" class="banner-image" :src="item.coverUrl" mode="aspectFill" />
        <view class="banner-copy">
          <text class="banner-kicker">{{ item.kindText || '平台推荐' }}</text>
          <text class="banner-title">{{ item.title }}</text>
          <text class="banner-subtitle">{{ item.subtitle || '运营精选产业资源，点击查看详情' }}</text>
        </view>
      </view>
    </scroll-view>

    <view class="focus-card" @click="openSearch('急清库存')">
      <view>
        <text class="focus-label">本周重点</text>
        <text class="focus-title">急清库存榜</text>
        <text class="focus-desc">优先看整包可看样资源，适合快速补货和直播拿货。</text>
      </view>
      <text class="focus-action">去看看</text>
    </view>

    <view class="quick-grid">
      <button
        v-for="item in sceneEntries"
        :key="item.title"
        :class="['scene-card', item.tone]"
        @click="openScene(item)"
      >
        <text class="scene-label">{{ item.label }}</text>
        <text class="scene-title">{{ item.title }}</text>
      </button>
    </view>

    <view class="section-head">
      <text class="section-title">平台推荐资源</text>
      <text class="section-link" @click="openSearch()">更多</text>
    </view>

    <view class="recommend-card" @click="openSearch('小单快返')">
      <view>
        <text class="recommend-tag">运营推荐</text>
        <text class="recommend-title">本周空档工厂和急清资源</text>
        <text class="recommend-desc">推广资源需审核通过，置顶只提升曝光，不替代真实性判断。</text>
      </view>
      <text class="recommend-action">查看</text>
    </view>

    <view v-if="homeResources.length" class="home-resource-list">
      <ResourceCard v-for="item in homeResources" :key="item.id" :resource="item" @open="openResource" />
    </view>
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import ResourceCard from '../../components/ResourceCard.vue'
import { DEFAULT_CITY_CODE } from '../../common/constants'
import { listHomeBanners } from '../../api/discovery'
import { listResources } from '../../api/resource'

const banners = ref([])
const homeResources = ref([])
const SEARCH_KEY = 'wplink_pending_search_keyword'
const defaultBanners = [
  {
    id: 'default-topic',
    kindText: '本周重点 · 织里童装产业带',
    title: '急清库存专题',
    subtitle: '整包可看样资源，适合快速补货',
    jumpType: 'internal',
    jumpTarget: '/pages/search/index',
    keyword: '急清库存',
  },
  {
    id: 'default-factory',
    kindText: '平台推荐 · 认证工厂',
    title: '本周空档工厂',
    subtitle: '小单快返、针织童装产能优先看',
    jumpType: 'internal',
    jumpTarget: '/pages/search/index',
    keyword: '小单快返',
  },
]
const sceneEntries = [
  { label: '找现货', title: '爆款童装', tone: 'green', keyword: '现货' },
  { label: '清库存', title: '整包急清', tone: 'coral', keyword: '急清库存' },
  { label: '找工厂', title: '小单快返', tone: 'blue', keyword: '小单快返' },
  { label: '商家发布', title: '资源上架', tone: 'amber', action: 'publish' },
]
const displayBanners = computed(() => (banners.value.length ? banners.value : defaultBanners))

onLoad(loadHomeData)

async function loadHomeData() {
  await Promise.all([loadBanners(), loadHomeResources()])
}

async function loadBanners() {
  // 首页首屏由运营配置驱动，失败时保留空列表，不阻断搜索和发布入口。
  const resp = await listHomeBanners({ cityCode: DEFAULT_CITY_CODE })
  banners.value = resp.items || []
}

async function loadHomeResources() {
  const resp = await listResources({ cityCode: DEFAULT_CITY_CODE, page: 1, pageSize: 2 })
  homeResources.value = resp.items || []
}

function openBanner(item) {
  if (item.keyword) {
    openSearch(item.keyword)
    return
  }
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
    openInternal(item.jumpTarget)
    return
  }
  openDemand()
}

function openScene(item) {
  if (item.action === 'publish') {
    openPublish()
    return
  }
  openSearch(item.keyword)
}

function openSearch(keyword = '') {
  if (keyword) {
    uni.setStorageSync(SEARCH_KEY, keyword)
  } else {
    uni.removeStorageSync(SEARCH_KEY)
  }
  uni.switchTab({ url: '/pages/search/index' })
}

function openDemand() {
  uni.navigateTo({ url: '/pages/demand/index' })
}

function openPublish() {
  uni.switchTab({ url: '/pages/publish/index' })
}

function openResource(item) {
  uni.navigateTo({ url: `/pages/resource/detail?id=${item.id}` })
}

function openInternal(url) {
  const tabPages = ['/pages/home/index', '/pages/search/index', '/pages/publish/index', '/pages/messages/index', '/pages/my/index']
  const path = url.split('?')[0]
  if (tabPages.includes(path)) {
    uni.switchTab({ url: path })
    return
  }
  uni.navigateTo({ url })
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
  justify-content: space-between;
  height: 82rpx;
  margin-bottom: 20rpx;
  padding: 0 24rpx;
  border-radius: 12rpx;
  background: #ffffff;
}

.search-placeholder {
  color: #697586;
  font-size: 28rpx;
}

.search-action {
  color: #0f766e;
  font-size: 28rpx;
  font-weight: 700;
}

.banner-list {
  width: 100%;
  margin-bottom: 18rpx;
  white-space: nowrap;
}

.banner-card {
  position: relative;
  display: inline-block;
  width: 620rpx;
  min-height: 220rpx;
  margin-right: 18rpx;
  overflow: hidden;
  border-radius: 12rpx;
  background: #0f766e;
  vertical-align: top;
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

.banner-kicker {
  font-size: 22rpx;
  opacity: 0.88;
}

.banner-title {
  font-size: 38rpx;
  font-weight: 700;
}

.banner-subtitle {
  font-size: 26rpx;
}

.focus-card,
.recommend-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20rpx;
  margin-bottom: 20rpx;
  padding: 24rpx;
  border-radius: 12rpx;
  background: #ffffff;
}

.focus-label,
.recommend-tag {
  display: block;
  margin-bottom: 8rpx;
  color: #0f766e;
  font-size: 24rpx;
  font-weight: 700;
}

.focus-title,
.recommend-title {
  display: block;
  margin-bottom: 8rpx;
  color: #1f2933;
  font-size: 32rpx;
  font-weight: 700;
}

.focus-desc,
.recommend-desc {
  color: #697586;
  font-size: 26rpx;
  line-height: 1.5;
}

.focus-action,
.recommend-action {
  flex: 0 0 auto;
  color: #0f766e;
  font-size: 26rpx;
  font-weight: 700;
}

.quick-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16rpx;
  margin-bottom: 24rpx;
}

.scene-card {
  display: grid;
  gap: 8rpx;
  min-height: 122rpx;
  padding: 20rpx;
  border-radius: 12rpx;
  text-align: left;
}

.scene-card.green {
  background: #e6f4f1;
  color: #0f766e;
}

.scene-card.coral {
  background: #fff0eb;
  color: #c2410c;
}

.scene-card.blue {
  background: #eaf1ff;
  color: #2563eb;
}

.scene-card.amber {
  background: #fff7e6;
  color: #b7791f;
}

.scene-label {
  font-size: 24rpx;
}

.scene-title {
  font-size: 32rpx;
  font-weight: 700;
}

.section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 14rpx;
}

.section-title {
  color: #1f2933;
  font-size: 32rpx;
  font-weight: 700;
}

.section-link {
  color: #697586;
  font-size: 26rpx;
}

.recommend-card {
  margin-bottom: 0;
  background: #ffffff;
}

.home-resource-list {
  display: grid;
  gap: 18rpx;
  margin-top: 18rpx;
}
</style>
