<template>
  <view class="home-page">
    <view class="home-fixed-header" :style="fixedHeaderStyle">
      <view class="custom-title-bar" :style="customTitleBarStyle">
        <view class="home-brand">
          <image class="brand-logo" src="/static/brand/yihuotong-header-logo.png" mode="aspectFit" />
        </view>
      </view>

      <view class="search-entry" @click="openSearch()">
        <view class="search-icon" aria-hidden="true"></view>
        <text class="search-placeholder">搜索现货、厂家或求购需求...</text>
      </view>
    </view>

    <view class="home-content" :style="homeContentStyle">
      <view class="banner-list">
        <swiper
          class="banner-swiper"
          autoplay
          circular
          :interval="4200"
          :duration="450"
          :current="activeBannerIndex"
          @change="handleBannerChange"
        >
          <swiper-item v-for="(item, index) in displayBanners" :key="item.id">
            <view
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
          </swiper-item>
        </swiper>
        <view v-if="displayBanners.length > 1" class="banner-dots">
          <view
            v-for="(item, index) in displayBanners"
            :key="`${item.id}-dot`"
            :class="['banner-dot', { active: activeBannerIndex === index }]"
          ></view>
        </view>
      </view>

      <view class="quick-action-grid">
        <button
          v-for="item in sceneEntries"
          :key="item.title"
          :class="['quick-action', item.tone]"
          @click="openScene(item)"
        >
          <view :class="['quick-icon', item.icon]">
            <view v-if="item.icon === 'market'" class="icon-market-stall">
              <text></text>
              <text></text>
              <text></text>
            </view>
            <view v-if="item.icon === 'clearance'" class="icon-clearance-tag">
              <text></text>
            </view>
            <view v-if="item.icon === 'factory'" class="icon-factory-grid">
              <text></text>
              <text></text>
              <text></text>
              <text></text>
              <text></text>
              <text></text>
            </view>
            <view v-if="item.icon === 'orders'" class="icon-orders-board">
              <text></text>
              <text></text>
              <text></text>
            </view>
          </view>
          <text class="quick-title">{{ item.title }}</text>
        </button>
      </view>

      <view class="section-head">
        <text class="section-title">精选资源</text>
        <text class="section-link" @click="openSearch()">更多</text>
      </view>

      <view class="recommend-card" v-if="displayRecommendCard" @click="openRecommendCard(displayRecommendCard)">
        <view>
          <text class="recommend-tag">{{ displayRecommendCard.tag || '平台推荐' }}</text>
          <text class="recommend-title">{{ displayRecommendCard.title }}</text>
          <text v-if="displayRecommendCard.subtitle" class="recommend-desc">{{ displayRecommendCard.subtitle }}</text>
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
import { listHomeBanners, listHomeRecommendCards } from '../../api/discovery'
import { listResources } from '../../api/resource'

const banners = ref([])
const recommendCards = ref([])
const homeResources = ref([])
const activeBannerIndex = ref(0)
const headerMetrics = ref({
  statusBarHeight: 44,
  navBarHeight: 44,
  headerHeight: 154,
})
const SEARCH_KEY = 'wplink_pending_search_keyword'
const PUBLISH_TYPE_KEY = 'wplink_pending_publish_type_code'
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
    id: 'default-order',
    kindText: '订单需求 · 工厂接单',
    title: '有空档产能？查看订单',
    jumpType: 'search',
    jumpTarget: '订单',
    typeCode: 'order',
    tone: 'order',
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
  { title: '货源市场', tone: 'navy', icon: 'market', typeCode: 'goods', keyword: '现货' },
  { title: '库存清仓', tone: 'red', icon: 'clearance', typeCode: 'inventory', keyword: '库存' },
  { title: '工厂产能', tone: 'teal', icon: 'factory', typeCode: 'factory', keyword: '小单快返' },
  { title: '订单大厅', tone: 'amber', icon: 'orders', typeCode: 'order', keyword: '订单' },
]
const displayBanners = computed(() => {
  const bannerSource = banners.value.length ? banners.value : defaultBanners
  return bannerSource.filter((item) => item.title).map(normalizeBanner)
})
const displayRecommendCard = computed(() => {
  const item = recommendCards.value.find((card) => card.title)
  return item ? normalizeRecommendCard(item) : null
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
  await Promise.all([loadBanners(), loadRecommendCards(), loadHomeResources()])
}

async function loadBanners() {
  // 首页首屏由运营配置驱动，失败时保留空列表，不阻断搜索和发布入口。
  try {
    const resp = await listHomeBanners({ cityCode: DEFAULT_CITY_CODE })
    banners.value = resp.items || []
    activeBannerIndex.value = 0
  } catch {
    banners.value = []
    activeBannerIndex.value = 0
  }
}

async function loadHomeResources() {
  const resp = await listResources({ cityCode: DEFAULT_CITY_CODE, page: 1, pageSize: 2 })
  homeResources.value = resp.items || []
}

async function loadRecommendCards() {
  // 首页推荐卡由后台运营位配置驱动；加载失败只隐藏该卡片，不影响下方资源列表。
  try {
    const resp = await listHomeRecommendCards({ cityCode: DEFAULT_CITY_CODE })
    recommendCards.value = resp.items || []
  } catch {
    recommendCards.value = []
  }
}

function handleBannerChange(event) {
  const current = Number(event?.detail?.current)
  activeBannerIndex.value = Number.isFinite(current) ? current : 0
}

function openBanner(item) {
  if (item.keyword) {
    openSearch({ keyword: item.keyword, typeCode: item.typeCode })
    return
  }
  if (item.jumpType === 'search') {
    openSearch({ keyword: item.jumpTarget, typeCode: item.typeCode })
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
  if (item.jumpType === 'demand') {
    openInternal(item.jumpTarget || '/pages/demand/index')
    return
  }
  if (item.jumpType === 'webview') {
    uni.navigateTo({ url: `/pages/webview/index?url=${encodeURIComponent(item.jumpTarget)}` })
    return
  }
  if (item.jumpType === 'publish') {
    openPublish(item.typeCode)
    return
  }
  if (item.jumpType === 'internal' && item.jumpTarget) {
    openInternal(item.jumpTarget)
    return
  }
  openSearch()
}

function openScene(item) {
  openSearch({ keyword: item.keyword, typeCode: item.typeCode })
}

function openRecommendCard(item) {
  openBanner(item)
}

function openSearch(options = {}) {
  const searchOptions = typeof options === 'string' ? { keyword: options } : { ...options }
  if (searchOptions.keyword || searchOptions.typeCode || searchOptions.cityCode) {
    uni.setStorageSync(SEARCH_KEY, searchOptions)
  } else {
    uni.removeStorageSync(SEARCH_KEY)
  }
  uni.navigateTo({ url: '/pages/search/result' })
}

function openPublish(typeCode = '') {
  if (typeCode) {
    uni.setStorageSync(PUBLISH_TYPE_KEY, typeCode)
  } else {
    uni.removeStorageSync(PUBLISH_TYPE_KEY)
  }
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
    const typePair = query.split('&').find((item) => item.startsWith('typeCode='))
    const keyword = keywordPair ? decodeURIComponent(keywordPair.split('=')[1] || '') : ''
    const typeCode = typePair ? decodeURIComponent(typePair.split('=')[1] || '') : ''
    openSearch({ keyword, typeCode })
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

function normalizeRecommendCard(item) {
  return {
    ...item,
    id: item.id || `recommend-card-${item.title}`,
    tag: item.tag || (item.tags && item.tags[0]) || '平台推荐',
  }
}

function bannerKindText(item) {
  if (item.tags && item.tags.length) return item.tags.slice(0, 2).join(' · ')
  const kindMap = {
    topic: '专题推荐',
    resource: '资源推荐',
    merchant: '认证商家',
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
    publish: 'publish',
    search: 'topic',
  }
  return toneMap[jumpType] || 'topic'
}
</script>

<style lang="scss" scoped>
.home-page {
  min-height: 100vh;
  background: $wplink-bg;
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
  background: $wplink-bg;
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
  min-width: 0;
}

.brand-logo {
  display: block;
  width: 246rpx;
  height: 82rpx;
}

.search-entry {
  display: flex;
  align-items: center;
  gap: 18rpx;
  min-height: 78rpx;
  margin-top: 18rpx;
  padding: 0 34rpx;
  border-radius: 22rpx;
  background: #ffffff;
  box-shadow: inset 0 0 0 1rpx rgba(176, 186, 200, 0.56), 0 8rpx 18rpx rgba(15, 23, 42, 0.04);
}

.search-icon {
  position: relative;
  flex: 0 0 32rpx;
  width: 32rpx;
  height: 32rpx;
}

.search-icon::before {
  position: absolute;
  top: 3rpx;
  left: 2rpx;
  width: 20rpx;
  height: 20rpx;
  border: 3rpx solid #7b8492;
  border-radius: 999rpx;
  content: '';
}

.search-icon::after {
  position: absolute;
  right: 4rpx;
  bottom: 5rpx;
  width: 14rpx;
  height: 3rpx;
  border-radius: 999rpx;
  background: #7b8492;
  content: '';
  transform-origin: right center;
  transform: rotate(45deg);
}

.search-placeholder {
  flex: 1;
  min-width: 0;
  color: #707987;
  font-size: 28rpx;
  font-weight: 600;
  line-height: 1.3;
  word-break: break-word;
}

.banner-list {
  position: relative;
  width: 100%;
  height: 326rpx;
  margin-bottom: 48rpx;
}

.banner-swiper {
  width: 100%;
  height: 326rpx;
}

.banner-card {
  position: relative;
  display: block;
  width: 100%;
  height: 326rpx;
  overflow: hidden;
  border-radius: 26rpx;
  background: $wplink-primary;
  box-shadow: 0 18rpx 34rpx rgba($wplink-primary, 0.12);
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
    linear-gradient(135deg, rgba($wplink-blue, 0.86), rgba($wplink-blue, 0.48)),
    radial-gradient(circle at 20% 20%, rgba(255, 255, 255, 0.24), transparent 34%);
}

.banner-card.factory .banner-pattern {
  background:
    linear-gradient(135deg, rgba($wplink-warning, 0.9), rgba($wplink-primary, 0.74)),
    repeating-linear-gradient(90deg, rgba(255, 255, 255, 0.14) 0 20rpx, transparent 20rpx 40rpx);
}

.banner-card.order .banner-pattern {
  background:
    linear-gradient(135deg, rgba($wplink-coral, 0.9), rgba($wplink-warning, 0.78)),
    radial-gradient(circle at 80% 22%, rgba(255, 255, 255, 0.22), transparent 30%);
}

.banner-card.publish .banner-pattern {
  background:
    linear-gradient(135deg, rgba($wplink-text, 0.92), rgba($wplink-primary, 0.78)),
    repeating-linear-gradient(135deg, rgba(255, 255, 255, 0.12) 0 14rpx, transparent 14rpx 30rpx);
}

.banner-shade {
  position: absolute;
  inset: 0;
  background:
    linear-gradient(180deg, rgba($wplink-primary, 0.04), rgba($wplink-primary, 0.8)),
    linear-gradient(90deg, rgba($wplink-primary, 0.74), rgba($wplink-primary, 0.12) 68%);
}

.banner-copy {
  position: absolute;
  right: 40rpx;
  bottom: 38rpx;
  left: 40rpx;
  display: grid;
  gap: 12rpx;
  color: $wplink-card;
}

.banner-kicker {
  justify-self: start;
  min-height: 42rpx;
  padding: 0 20rpx;
  border-radius: 999rpx;
  background: $wplink-warning;
  color: #222222;
  font-size: 22rpx;
  font-weight: 700;
  line-height: 42rpx;
}

.banner-title {
  max-width: 590rpx;
  color: $wplink-card;
  font-size: 34rpx;
  font-weight: 800;
  line-height: 1.24;
  word-break: break-word;
}

.banner-subcopy {
  color: $wplink-card;
  font-size: 24rpx;
  font-weight: 700;
  line-height: 1.3;
  opacity: 0.95;
}

.banner-dots {
  position: absolute;
  right: 28rpx;
  bottom: 24rpx;
  z-index: 3;
  display: flex;
  align-items: center;
  gap: 10rpx;
}

.banner-dot {
  width: 10rpx;
  height: 10rpx;
  border-radius: 999rpx;
  background: rgba(255, 255, 255, 0.58);
}

.banner-dot.active {
  width: 22rpx;
  background: #ffffff;
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

.icon-factory-grid {
  display: grid;
  grid-template-columns: repeat(3, 9rpx);
  grid-auto-rows: 9rpx;
  gap: 5rpx;
  padding: 9rpx;
  border: 5rpx solid $wplink-success;
}

.icon-factory-grid text {
  background: $wplink-success;
}

.icon-market-stall {
  position: relative;
  display: grid;
  grid-template-columns: repeat(3, 12rpx);
  gap: 2rpx;
  width: 42rpx;
  height: 38rpx;
  padding-top: 10rpx;
}

.icon-market-stall::before,
.icon-market-stall::after {
  position: absolute;
  right: 0;
  left: 0;
  content: '';
}

.icon-market-stall::before {
  top: 0;
  height: 11rpx;
  border-radius: 6rpx 6rpx 2rpx 2rpx;
  background: repeating-linear-gradient(90deg, $wplink-primary 0 10rpx, rgba($wplink-primary, 0.45) 10rpx 20rpx);
}

.icon-market-stall::after {
  bottom: 0;
  height: 5rpx;
  border-radius: 999rpx;
  background: $wplink-primary;
}

.icon-market-stall text {
  align-self: end;
  height: 20rpx;
  border-radius: 2rpx 2rpx 0 0;
  background: rgba($wplink-primary, 0.72);
}

.icon-clearance-tag {
  position: relative;
  width: 42rpx;
  height: 30rpx;
  border-radius: 6rpx 8rpx 8rpx 6rpx;
  background: $wplink-warning;
  transform: rotate(-10deg);
}

.icon-clearance-tag::before {
  position: absolute;
  top: 10rpx;
  left: 7rpx;
  width: 8rpx;
  height: 8rpx;
  border-radius: 999rpx;
  background: $wplink-card;
  content: '';
}

.icon-clearance-tag::after {
  position: absolute;
  top: 12rpx;
  right: -10rpx;
  width: 20rpx;
  height: 20rpx;
  border-radius: 3rpx;
  background: $wplink-warning;
  content: '';
  transform: rotate(45deg);
}

.icon-clearance-tag text {
  position: absolute;
  right: 8rpx;
  bottom: 7rpx;
  width: 18rpx;
  height: 5rpx;
  border-radius: 999rpx;
  background: $wplink-card;
  z-index: 1;
}

.icon-orders-board {
  display: grid;
  gap: 7rpx;
  width: 40rpx;
  padding: 8rpx 7rpx;
  border: 5rpx solid #f59e0b;
  border-radius: 4rpx;
}

.icon-orders-board text {
  display: block;
  height: 5rpx;
  border-radius: 999rpx;
  background: #f59e0b;
}

.icon-orders-board text:nth-child(2) {
  width: 26rpx;
}

.icon-orders-board text:nth-child(3) {
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
  background: $wplink-card;
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
  background: $wplink-warning-soft;
  color: #9a5b00;
  font-size: 24rpx;
  font-weight: 700;
}

.recommend-title {
  display: block;
  margin-bottom: 8rpx;
  color: $wplink-primary;
  font-size: 36rpx;
  font-weight: 700;
  line-height: 1.25;
  word-break: break-word;
}

.recommend-desc {
  color: $wplink-muted;
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
  background: $wplink-primary-soft;
  color: $wplink-primary;
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
  color: $wplink-primary;
  font-size: 36rpx;
  font-weight: 700;
}

.section-link {
  color: $wplink-primary;
  font-size: 26rpx;
  font-weight: 700;
}

.home-resource-list {
  display: grid;
  gap: 20rpx;
  margin-top: 20rpx;
}
</style>
