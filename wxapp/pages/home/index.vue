<template>
  <view class="home-page">
    <scroll-view class="banner-list" scroll-x>
      <view v-for="item in displayBanners" :key="item.id" :class="['banner-card', item.tone]" @click="openBanner(item)">
        <image v-if="item.coverUrl" class="banner-image" :src="item.coverUrl" mode="aspectFill" />
        <view v-else class="banner-pattern"></view>
        <view class="banner-copy">
          <text class="banner-kicker">{{ item.kindText || '平台推荐' }}</text>
          <text class="banner-title">{{ item.title }}</text>
          <text class="banner-subtitle">{{ item.subtitle || '运营精选产业资源，点击查看详情' }}</text>
        </view>
        <text class="banner-pill">{{ item.actionText }}</text>
      </view>
    </scroll-view>
    <view class="banner-dots" :style="{ width: bannerDotsWidth }">
      <text v-for="(item, index) in displayBanners" :key="`dot-${item.id}`" :class="{ active: index === 0 }"></text>
    </view>

    <view class="search-entry" @click="openSearch()">
      <text class="search-placeholder">搜索童装库存、工厂、货源</text>
      <text class="search-action">搜索</text>
    </view>

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
    kindText: '本周重点 · 平台核实资源',
    title: '急清库存专题',
    subtitle: '32 条可看样库存，过期自动下架',
    actionText: '查看专题',
    jumpType: 'topic',
    jumpTarget: 'default-topic',
    tone: 'topic',
  },
  {
    id: 'default-activity',
    kindText: '活动推广 · 白名单网页',
    title: '夏款供需对接会',
    subtitle: '配置封面、文案和网页链接',
    actionText: '打开活动',
    jumpType: 'webview',
    jumpTarget: 'https://m.fulink.example/events/zhili-summer',
    tone: 'activity',
  },
  {
    id: 'default-merchant',
    kindText: '平台推荐 · 认证工厂',
    title: '本周空档工厂',
    subtitle: '4 条针织生产线，适合小单快返',
    actionText: '去搜索',
    jumpType: 'search',
    jumpTarget: '小单快返',
    tone: 'factory',
  },
  {
    id: 'default-demand',
    kindText: '找货需求 · 运营跟进',
    title: '没找到合适货源？',
    subtitle: '提交采购需求，平台继续帮你留意库存、货源和工厂',
    actionText: '提交需求',
    jumpType: 'demand',
    jumpTarget: '/pages/demand/index',
    tone: 'demand',
  },
  {
    id: 'default-publish',
    kindText: '商家发布 · 增加曝光',
    title: '库存和产能可直接上架',
    subtitle: '发布后进入审核，审核通过即可被搜索和推荐',
    actionText: '去发布',
    jumpType: 'publish',
    jumpTarget: '/pages/publish/index',
    tone: 'publish',
  },
]
const sceneEntries = [
  { label: '找现货', title: '爆款童装', tone: 'green', keyword: '现货' },
  { label: '清库存', title: '整包急清', tone: 'coral', keyword: '急清库存' },
  { label: '找工厂', title: '小单快返', tone: 'blue', keyword: '小单快返' },
  { label: '商家发布', title: '资源上架', tone: 'amber', action: 'publish' },
]
const displayBanners = computed(() => {
  const bannerSource = banners.value.length ? banners.value : defaultBanners
  return bannerSource.filter((item) => item.title).map(normalizeBanner)
})
const bannerDotsWidth = computed(() => {
  const count = Math.max(displayBanners.value.length, 3)
  return `${64 + (count - 1) * 22}rpx`
})

onLoad(loadHomeData)

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
    subtitle: item.subtitle || '运营精选产业资源，点击查看详情',
    actionText: item.actionText || bannerActionText(item.jumpType),
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

function bannerActionText(jumpType) {
  const actionMap = {
    topic: '查看专题',
    resource: '看详情',
    merchant: '看主页',
    demand: '提交需求',
    publish: '去发布',
    search: '去搜索',
    webview: '打开活动',
    internal: '立即进入',
  }
  return actionMap[jumpType] || '查看'
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
  padding: 28rpx 28rpx 36rpx;
  background: #f3f5f7;
}

.search-entry {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 92rpx;
  margin: 0 0 24rpx;
  padding: 0 28rpx;
  border-radius: 16rpx;
  background: #ffffff;
  box-shadow: 0 12rpx 40rpx rgba(15, 23, 42, 0.06);
}

.search-placeholder {
  flex: 1;
  min-width: 0;
  color: #697586;
  font-size: 28rpx;
  line-height: 1.35;
  word-break: break-word;
}

.search-action {
  flex: 0 0 auto;
  color: #0f766e;
  font-size: 28rpx;
  font-weight: 700;
  white-space: nowrap;
}

.banner-list {
  position: relative;
  width: 100%;
  margin-bottom: 0;
  white-space: nowrap;
}

.banner-card {
  position: relative;
  display: inline-block;
  width: 100%;
  height: 320rpx;
  margin-right: 20rpx;
  overflow: hidden;
  border-radius: 16rpx;
  background: #0f766e;
  vertical-align: top;
}

.banner-image {
  width: 100%;
  height: 320rpx;
}

.banner-pattern {
  width: 100%;
  height: 320rpx;
  background:
    linear-gradient(135deg, rgba(15, 118, 110, 0.92), rgba(37, 99, 235, 0.76)),
    repeating-linear-gradient(45deg, rgba(255, 255, 255, 0.16) 0 16rpx, transparent 16rpx 32rpx);
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

.banner-copy {
  position: absolute;
  top: 36rpx;
  right: 164rpx;
  left: 36rpx;
  display: grid;
  gap: 10rpx;
  color: #ffffff;
}

.banner-kicker {
  font-size: 24rpx;
  opacity: 0.88;
}

.banner-title {
  font-size: 52rpx;
  font-weight: 700;
  line-height: 1.15;
  word-break: break-word;
}

.banner-subtitle {
  font-size: 28rpx;
  line-height: 1.45;
  opacity: 0.88;
  word-break: break-word;
}

.banner-pill {
  position: absolute;
  right: 36rpx;
  top: 50%;
  transform: translateY(-50%);
  max-width: 148rpx;
  min-height: 56rpx;
  padding: 14rpx 22rpx;
  border-radius: 999rpx;
  background: rgba(255, 255, 255, 0.18);
  color: #ffffff;
  font-size: 24rpx;
  line-height: 1.2;
  text-align: center;
  white-space: nowrap;
}

.banner-dots {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 10rpx;
  min-width: 108rpx;
  margin: -66rpx 28rpx 24rpx auto;
  padding: 10rpx 14rpx;
  border-radius: 999rpx;
  background: rgba(15, 23, 42, 0.18);
}

.banner-dots text {
  width: 12rpx;
  height: 12rpx;
  border-radius: 999rpx;
  background: rgba(255, 255, 255, 0.56);
}

.banner-dots .active {
  width: 36rpx;
  background: #ffffff;
}

.focus-card,
.recommend-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20rpx;
  margin-bottom: 20rpx;
  padding: 28rpx;
  border-radius: 16rpx;
  background: #ffffff;
  box-shadow: 0 16rpx 48rpx rgba(15, 23, 42, 0.06);
}

.focus-card {
  border: 1rpx solid rgba(15, 118, 110, 0.18);
  background:
    linear-gradient(135deg, rgba(15, 118, 110, 0.1), rgba(37, 99, 235, 0.08)),
    #ffffff;
}

.focus-card > view,
.recommend-card > view {
  min-width: 0;
}

.focus-label,
.recommend-tag {
  display: inline-flex;
  align-items: center;
  min-height: 34rpx;
  margin-bottom: 12rpx;
  padding: 0 12rpx;
  border-radius: 10rpx;
  background: #0f766e;
  color: #ffffff;
  font-size: 24rpx;
  font-weight: 700;
}

.recommend-tag {
  background: #fff7e6;
  color: #9a5b00;
}

.focus-title,
.recommend-title {
  display: block;
  margin-bottom: 8rpx;
  color: #1f2933;
  font-size: 36rpx;
  font-weight: 700;
  line-height: 1.25;
  word-break: break-word;
}

.focus-desc,
.recommend-desc {
  color: #697586;
  font-size: 26rpx;
  line-height: 1.5;
  word-break: break-word;
}

.focus-action,
.recommend-action {
  flex: 0 0 auto;
  min-width: 96rpx;
  min-height: 64rpx;
  padding: 14rpx 18rpx;
  border-radius: 16rpx;
  background: #ffffff;
  color: #0f766e;
  font-size: 26rpx;
  font-weight: 700;
  line-height: 1.25;
  text-align: center;
}

.quick-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 20rpx;
  margin: 8rpx 0 32rpx;
}

.scene-card {
  display: grid;
  align-content: center;
  gap: 14rpx;
  min-height: 156rpx;
  padding: 28rpx 24rpx;
  border: 0;
  border-radius: 16rpx;
  text-align: left;
  line-height: 1.2;
  box-shadow: none;
}

.scene-card::after {
  border: 0;
}

.scene-card.green {
  background: #0f766e;
  color: #ffffff;
}

.scene-card.coral {
  background: #dc6b4a;
  color: #ffffff;
}

.scene-card.blue {
  background: #2563eb;
  color: #ffffff;
}

.scene-card.amber {
  background: #b7791f;
  color: #ffffff;
}

.scene-label {
  display: block;
  width: 100%;
  color: rgba(255, 255, 255, 0.9);
  font-size: 26rpx;
  font-weight: 500;
  text-align: left;
}

.scene-title {
  display: block;
  width: 100%;
  color: #ffffff;
  font-size: 40rpx;
  font-weight: 700;
  line-height: 1.2;
  text-align: left;
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

.recommend-card {
  margin-bottom: 0;
  background: #ffffff;
}

.recommend-action {
  background: #e6f4f1;
}

.home-resource-list {
  display: grid;
  gap: 20rpx;
  margin-top: 20rpx;
}
</style>
