<template>
  <view class="home-page">
    <view class="home-fixed-header" :style="fixedHeaderStyle">
      <view class="custom-title-bar" :style="customTitleBarStyle">
        <view class="home-brand">
          <view class="brand-icon" aria-hidden="true">
            <view class="brand-roof"></view>
            <view class="brand-window-row">
              <text></text>
              <text></text>
              <text></text>
            </view>
          </view>
          <text class="brand-name">衣货通</text>
        </view>
      </view>

      <view class="search-entry" @click="openSearch()">
        <view class="search-icon" aria-hidden="true">
          <text></text>
        </view>
        <text class="search-placeholder">搜索现货、厂家或求购需求...</text>
      </view>
    </view>

    <view class="home-content" :style="homeContentStyle">
      <scroll-view class="banner-list" scroll-x>
        <view
          v-for="item in displayBanners"
          :key="item.id"
          :class="['banner-card', 'factory-hero', item.tone]"
          @click="openBanner(item)"
        >
          <image v-if="item.coverUrl" class="banner-image" :src="item.coverUrl" mode="aspectFill" />
          <view v-else class="banner-pattern"></view>
          <view class="banner-shade"></view>
          <view class="banner-copy">
            <text class="banner-kicker">{{ item.kindText || '平台推荐' }}</text>
            <text class="banner-title">{{ item.title }}</text>
            <text class="banner-subcopy">{{ item.subTitle || '本周新增 128 家金牌工厂' }}</text>
          </view>
        </view>
      </scroll-view>

      <view class="quick-action-grid">
        <button
          v-for="item in sceneEntries"
          :key="item.title"
          :class="['quick-action', item.tone]"
          @click="openScene(item)"
        >
          <view :class="['quick-icon', item.icon]">
            <text v-if="item.icon === 'stock'" class="icon-stock-line"></text>
            <text v-if="item.icon === 'stock'" class="icon-stock-dot"></text>
            <text v-if="item.icon === 'clear'" class="icon-clear-flame"></text>
            <view v-if="item.icon === 'factory'" class="icon-factory-grid">
              <text></text>
              <text></text>
              <text></text>
              <text></text>
              <text></text>
              <text></text>
            </view>
            <view v-if="item.icon === 'order'" class="icon-order-lines">
              <text></text>
              <text></text>
              <text></text>
            </view>
          </view>
          <text class="quick-title">{{ item.title }}</text>
        </button>
      </view>

      <view class="section-head">
        <text class="section-title">平台推荐资源</text>
        <text class="section-link" @click="openSearch()">更多</text>
      </view>

      <view class="recommend-card" @click="openSearch('小单快返')">
        <view>
          <text class="recommend-tag">平台推荐</text>
          <text class="recommend-title">本周空档工厂：4 条针织生产线</text>
          <text class="recommend-desc">认证工厂 · 适合小单快返 · 运营已核实</text>
        </view>
        <text class="recommend-action">查看</text>
      </view>

      <view v-if="homeResources.length" class="home-resource-list">
        <ResourceCard
          v-for="item in homeResources"
          :key="item.id"
          :resource="item"
          variant="home"
          @open="openResource"
        />
      </view>
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
const headerMetrics = ref({
  statusBarHeight: 44,
  navBarHeight: 44,
  headerHeight: 154,
})
const SEARCH_KEY = 'wplink_pending_search_keyword'
const SEARCH_BLOCK_RPX = 116
const defaultBanners = [
  {
    id: 'default-topic',
    kindText: '织里站 · 精选工厂',
    title: '童装产业带数字化撮合中心',
    subTitle: '本周新增 128 家金牌工厂',
    coverUrl: '/static/home/factory-hero.jpg',
    jumpType: 'topic',
    jumpTarget: 'default-topic',
    tone: 'topic',
  },
  {
    id: 'default-activity',
    kindText: '活动推广 · 白名单网页',
    title: '夏款供需对接会',
    jumpType: 'webview',
    jumpTarget: 'https://m.fulink.example/events/zhili-summer',
    tone: 'activity',
  },
  {
    id: 'default-merchant',
    kindText: '平台推荐 · 认证工厂',
    title: '本周空档工厂',
    jumpType: 'search',
    jumpTarget: '小单快返',
    tone: 'factory',
  },
  {
    id: 'default-demand',
    kindText: '找货需求 · 运营跟进',
    title: '没找到合适货源？',
    jumpType: 'demand',
    jumpTarget: '/pages/demand/index',
    tone: 'demand',
  },
  {
    id: 'default-publish',
    kindText: '商家发布 · 增加曝光',
    title: '库存和产能可直接上架',
    jumpType: 'publish',
    jumpTarget: '/pages/publish/index',
    tone: 'publish',
  },
]
const sceneEntries = [
  { title: '我要找货', tone: 'navy', icon: 'stock', keyword: '现货' },
  { title: '我要清货', tone: 'red', icon: 'clear', action: 'publish' },
  { title: '我要找厂', tone: 'teal', icon: 'factory', keyword: '小单快返' },
  { title: '我要接单', tone: 'amber', icon: 'order', action: 'demand' },
]
const displayBanners = computed(() => {
  const bannerSource = banners.value.length ? banners.value : defaultBanners
  return bannerSource.filter((item) => item.title).map(normalizeBanner)
})
const fixedHeaderStyle = computed(() => `padding-top: ${headerMetrics.value.statusBarHeight}px;`)
const customTitleBarStyle = computed(() => `height: ${headerMetrics.value.navBarHeight}px;`)
const homeContentStyle = computed(() => `padding-top: ${headerMetrics.value.headerHeight}px;`)

onLoad(initHomePage)

function initHomePage() {
  updateHeaderMetrics()
  loadHomeData()
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
    const searchBlockHeight = typeof uni.upx2px === 'function' ? uni.upx2px(SEARCH_BLOCK_RPX) : 66
    headerMetrics.value = {
      statusBarHeight,
      navBarHeight,
      headerHeight: statusBarHeight + navBarHeight + searchBlockHeight,
    }
  } catch {
    headerMetrics.value = {
      statusBarHeight: 44,
      navBarHeight: 44,
      headerHeight: 154,
    }
  }
}

async function loadHomeData() {
  await Promise.all([loadBanners(), loadHomeResources()])
}

async function loadBanners() {
  // 首页首屏由运营配置驱动，失败时保留空列表，不阻断搜索和发布入口。
  try {
    const resp = await listHomeBanners({ cityCode: DEFAULT_CITY_CODE })
    banners.value = resp.items || []
  } catch {
    banners.value = []
  }
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
  if (item.jumpType === 'search') {
    openSearch(item.jumpTarget)
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
  if (item.jumpType === 'demand') {
    openDemand()
    return
  }
  if (item.jumpType === 'publish') {
    openPublish()
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
  if (item.action === 'demand') {
    openDemand()
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
  if (path === '/pages/search/index' && url.includes('?')) {
    const query = url.split('?')[1] || ''
    const keywordPair = query.split('&').find((item) => item.startsWith('keyword=') || item.startsWith('q='))
    const keyword = keywordPair ? decodeURIComponent(keywordPair.split('=')[1] || '') : ''
    openSearch(keyword)
    return
  }
  if (tabPages.includes(path)) {
    uni.switchTab({ url: path })
    return
  }
  uni.navigateTo({ url })
}

function normalizeBanner(item) {
  return {
    ...item,
    id: item.id || `${item.jumpType || 'banner'}-${item.title}`,
    kindText: item.kindText || bannerKindText(item),
    tone: item.tone || bannerTone(item.jumpType),
  }
}

function bannerKindText(item) {
  if (item.tags && item.tags.length) return item.tags.slice(0, 2).join(' · ')
  const kindMap = {
    topic: '专题推荐',
    resource: '资源推荐',
    merchant: '认证商家',
    demand: '找货需求',
    publish: '商家发布',
    search: '热门搜索',
    webview: '活动推广',
    internal: '平台入口',
  }
  return kindMap[item.jumpType] || '平台推荐'
}

function bannerTone(jumpType) {
  const toneMap = {
    webview: 'activity',
    merchant: 'factory',
    demand: 'demand',
    publish: 'publish',
    search: 'topic',
  }
  return toneMap[jumpType] || 'topic'
}
</script>

<style scoped>
.home-page {
  min-height: 100vh;
  background: #f6f8fc;
}

.home-fixed-header {
  position: fixed;
  top: 0;
  right: 0;
  left: 0;
  z-index: 20;
  padding-right: 28rpx;
  padding-bottom: 14rpx;
  padding-left: 28rpx;
  background: #f6f8fc;
  box-shadow: 0 8rpx 24rpx rgba(15, 23, 42, 0.03);
}

.custom-title-bar {
  display: flex;
  align-items: center;
  box-sizing: border-box;
  padding-right: 190rpx;
}

.home-content {
  box-sizing: border-box;
  min-height: 100vh;
  padding-right: 28rpx;
  padding-bottom: 44rpx;
  padding-left: 28rpx;
}

.home-brand {
  display: flex;
  align-items: center;
  gap: 20rpx;
  min-width: 0;
}

.brand-icon {
  display: grid;
  place-items: center;
  width: 56rpx;
  height: 56rpx;
  border-radius: 8rpx;
  background: #052a46;
}

.brand-roof {
  width: 34rpx;
  height: 12rpx;
  background: #ffffff;
  clip-path: polygon(0 45%, 18% 45%, 18% 24%, 38% 24%, 38% 45%, 58% 45%, 58% 0, 76% 0, 76% 45%, 100% 45%, 100% 100%, 0 100%);
}

.brand-window-row {
  display: flex;
  gap: 4rpx;
  margin-top: -2rpx;
}

.brand-window-row text {
  width: 6rpx;
  height: 8rpx;
  background: #ffffff;
}

.brand-name {
  color: #12243a;
  font-size: 32rpx;
  font-weight: 800;
  line-height: 1.2;
}

.search-entry {
  display: flex;
  align-items: center;
  gap: 22rpx;
  min-height: 78rpx;
  margin-top: 18rpx;
  padding: 0 34rpx;
  border-radius: 22rpx;
  background: #fbfcff;
  box-shadow: inset 0 0 0 1rpx rgba(226, 232, 240, 0.34);
}

.search-icon {
  position: relative;
  flex: 0 0 30rpx;
  width: 30rpx;
  height: 30rpx;
}

.search-icon::before {
  position: absolute;
  top: 0;
  left: 0;
  width: 20rpx;
  height: 20rpx;
  border: 4rpx solid #a9afb8;
  border-radius: 999rpx;
  content: '';
}

.search-icon text {
  position: absolute;
  right: 1rpx;
  bottom: 2rpx;
  width: 14rpx;
  height: 4rpx;
  border-radius: 999rpx;
  background: #a9afb8;
  transform: rotate(45deg);
}

.search-placeholder {
  flex: 1;
  min-width: 0;
  color: #aab0ba;
  font-size: 28rpx;
  font-weight: 700;
  line-height: 1.3;
  word-break: break-word;
}

.banner-list {
  position: relative;
  width: 100%;
  margin-bottom: 48rpx;
  white-space: nowrap;
}

.banner-card {
  position: relative;
  display: inline-block;
  width: 100%;
  height: 326rpx;
  margin-right: 20rpx;
  overflow: hidden;
  border-radius: 26rpx;
  background: #052a46;
  box-shadow: 0 18rpx 34rpx rgba(5, 42, 70, 0.12);
  vertical-align: top;
}

.banner-image {
  width: 100%;
  height: 326rpx;
}

.banner-pattern {
  width: 100%;
  height: 326rpx;
  background:
    linear-gradient(135deg, rgba(7, 49, 74, 0.94), rgba(12, 86, 108, 0.72)),
    repeating-linear-gradient(90deg, rgba(255, 255, 255, 0.14) 0 18rpx, transparent 18rpx 42rpx);
}

.banner-card.activity .banner-pattern {
  background:
    linear-gradient(135deg, rgba(37, 99, 235, 0.86), rgba(123, 143, 199, 0.78)),
    radial-gradient(circle at 20% 20%, rgba(255, 255, 255, 0.24), transparent 34%);
}

.banner-card.factory .banner-pattern {
  background:
    linear-gradient(135deg, rgba(183, 121, 31, 0.9), rgba(15, 118, 110, 0.74)),
    repeating-linear-gradient(90deg, rgba(255, 255, 255, 0.14) 0 20rpx, transparent 20rpx 40rpx);
}

.banner-card.demand .banner-pattern {
  background:
    linear-gradient(135deg, rgba(220, 107, 74, 0.9), rgba(183, 121, 31, 0.78)),
    radial-gradient(circle at 80% 22%, rgba(255, 255, 255, 0.22), transparent 30%);
}

.banner-card.publish .banner-pattern {
  background:
    linear-gradient(135deg, rgba(31, 41, 51, 0.92), rgba(15, 118, 110, 0.78)),
    repeating-linear-gradient(135deg, rgba(255, 255, 255, 0.12) 0 14rpx, transparent 14rpx 30rpx);
}

.banner-shade {
  position: absolute;
  inset: 0;
  background:
    linear-gradient(180deg, rgba(5, 42, 70, 0.04), rgba(5, 42, 70, 0.8)),
    linear-gradient(90deg, rgba(5, 42, 70, 0.74), rgba(5, 42, 70, 0.12) 68%);
}

.banner-copy {
  position: absolute;
  right: 40rpx;
  bottom: 38rpx;
  left: 40rpx;
  display: grid;
  gap: 12rpx;
  color: #ffffff;
}

.banner-kicker {
  justify-self: start;
  min-height: 42rpx;
  padding: 0 20rpx;
  border-radius: 999rpx;
  background: #ff6d2d;
  color: #222222;
  font-size: 22rpx;
  font-weight: 700;
  line-height: 42rpx;
}

.banner-title {
  max-width: 590rpx;
  color: #ffffff;
  font-size: 34rpx;
  font-weight: 800;
  line-height: 1.24;
  word-break: break-word;
}

.banner-subcopy {
  color: #ffffff;
  font-size: 24rpx;
  font-weight: 700;
  line-height: 1.3;
  opacity: 0.95;
}

.quick-action-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 28rpx;
  margin: 0 0 42rpx;
}

.quick-action {
  display: grid;
  justify-items: center;
  gap: 16rpx;
  min-width: 0;
  padding: 0;
  border: 0;
  background: transparent;
  text-align: center;
  line-height: 1.2;
  box-shadow: none;
}

.quick-action::after {
  border: 0;
}

.quick-icon {
  position: relative;
  display: grid;
  place-items: center;
  width: 98rpx;
  height: 98rpx;
  border-radius: 20rpx;
  background: #eef1f7;
}

.quick-action.red .quick-icon {
  background: #f6eeee;
}

.quick-action.teal .quick-icon {
  background: #eef7fa;
}

.quick-action.amber .quick-icon {
  background: #f7f1ee;
}

.quick-title {
  color: #12243a;
  font-size: 24rpx;
  font-weight: 800;
  line-height: 1.2;
  word-break: keep-all;
}

.icon-stock-line {
  width: 36rpx;
  height: 28rpx;
  border: 5rpx solid #052a46;
  border-radius: 4rpx;
}

.icon-stock-dot {
  position: absolute;
  top: 36rpx;
  width: 20rpx;
  height: 5rpx;
  border-radius: 999rpx;
  background: #052a46;
}

.icon-clear-flame {
  width: 30rpx;
  height: 38rpx;
  border-radius: 50% 50% 48% 48%;
  background: #c73a12;
  clip-path: polygon(50% 0, 78% 28%, 94% 58%, 82% 88%, 50% 100%, 18% 88%, 6% 58%, 28% 30%);
}

.icon-clear-flame::after {
  position: absolute;
  top: 42rpx;
  left: 43rpx;
  width: 14rpx;
  height: 20rpx;
  border-radius: 50%;
  background: #ffffff;
  content: '';
}

.icon-factory-grid {
  display: grid;
  grid-template-columns: repeat(3, 9rpx);
  grid-auto-rows: 9rpx;
  gap: 5rpx;
  padding: 9rpx;
  border: 5rpx solid #16b98f;
}

.icon-factory-grid text {
  background: #16b98f;
}

.icon-order-lines {
  display: grid;
  gap: 8rpx;
  width: 36rpx;
}

.icon-order-lines text {
  display: block;
  height: 5rpx;
  border-radius: 999rpx;
  background: #f59e0b;
}

.icon-order-lines text:nth-child(2) {
  width: 26rpx;
}

.icon-order-lines text:nth-child(3) {
  width: 18rpx;
}

.recommend-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20rpx;
  margin-bottom: 0;
  padding: 28rpx;
  border-radius: 16rpx;
  background: #ffffff;
  box-shadow: 0 16rpx 48rpx rgba(15, 23, 42, 0.06);
}

.recommend-card > view {
  min-width: 0;
}

.recommend-tag {
  display: inline-flex;
  align-items: center;
  min-height: 34rpx;
  margin-bottom: 12rpx;
  padding: 0 12rpx;
  border-radius: 10rpx;
  background: #fff7e6;
  color: #9a5b00;
  font-size: 24rpx;
  font-weight: 700;
}

.recommend-title {
  display: block;
  margin-bottom: 8rpx;
  color: #1f2933;
  font-size: 36rpx;
  font-weight: 700;
  line-height: 1.25;
  word-break: break-word;
}

.recommend-desc {
  color: #697586;
  font-size: 26rpx;
  line-height: 1.5;
  word-break: break-word;
}

.recommend-action {
  flex: 0 0 auto;
  min-width: 96rpx;
  min-height: 64rpx;
  padding: 14rpx 18rpx;
  border-radius: 16rpx;
  background: #e6f4f1;
  color: #0f766e;
  font-size: 26rpx;
  font-weight: 700;
  line-height: 1.25;
  text-align: center;
}

.section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin: 32rpx 0 20rpx;
}

.section-title {
  color: #1f2933;
  font-size: 36rpx;
  font-weight: 700;
}

.section-link {
  color: #0f766e;
  font-size: 26rpx;
  font-weight: 700;
}

.home-resource-list {
  display: grid;
  gap: 20rpx;
  margin-top: 20rpx;
}
</style>
