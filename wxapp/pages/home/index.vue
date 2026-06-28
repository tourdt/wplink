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
    <view class="banner-dots">
      <text class="active"></text>
      <text></text>
      <text></text>
    </view>

    <view class="search-entry" @click="openSearch()">
      <text class="search-placeholder">搜索童装库存、工厂、货源</text>
      <text class="search-action">搜索</text>
    </view>

    <view class="trust-strip">
      <text>平台核实</text>
      <text>认证商家</text>
      <text>过期下架</text>
    </view>

    <view class="activity-card" @click="openActivity">
      <view class="activity-cover">
        <text>活动</text>
      </view>
      <view class="activity-copy">
        <text class="activity-tag">活动推广</text>
        <text class="activity-title">织里童装夏款供需对接会</text>
        <text class="activity-desc">活动链接通过业务域名白名单校验，点击打开活动网页。</text>
      </view>
      <text class="activity-action">打开</text>
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
    actionText: '看主页',
    keyword: '小单快返',
    tone: 'factory',
  },
]
const sceneEntries = [
  { label: '找现货', title: '爆款童装', tone: 'green', keyword: '现货' },
  { label: '清库存', title: '整包急清', tone: 'coral', keyword: '急清库存' },
  { label: '找工厂', title: '小单快返', tone: 'blue', keyword: '小单快返' },
  { label: '商家发布', title: '资源上架', tone: 'amber', action: 'publish' },
]
const displayBanners = computed(() => defaultBanners)

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

function openActivity() {
  uni.navigateTo({ url: `/pages/webview/index?url=${encodeURIComponent('https://m.fulink.example/events/zhili-summer')}` })
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
  padding: 20rpx 20rpx 28rpx;
  background: #f4f6f8;
}

.search-entry {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 82rpx;
  margin: 18rpx 0 16rpx;
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
  position: relative;
  width: 100%;
  margin-bottom: 0;
  white-space: nowrap;
}

.banner-card {
  position: relative;
  display: inline-block;
  width: 100%;
  height: 352rpx;
  margin-right: 20rpx;
  overflow: hidden;
  border-radius: 12rpx;
  background: #0f766e;
  vertical-align: top;
}

.banner-image {
  width: 100%;
  height: 352rpx;
}

.banner-pattern {
  width: 100%;
  height: 352rpx;
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

.banner-copy {
  position: absolute;
  top: 42rpx;
  right: 136rpx;
  left: 28rpx;
  display: grid;
  gap: 12rpx;
  color: #ffffff;
}

.banner-kicker {
  font-size: 24rpx;
  opacity: 0.88;
}

.banner-title {
  font-size: 52rpx;
  font-weight: 700;
  line-height: 1.12;
}

.banner-subtitle {
  font-size: 28rpx;
  line-height: 1.45;
  opacity: 0.88;
}

.banner-pill {
  position: absolute;
  right: 28rpx;
  top: 50%;
  transform: translateY(-50%);
  padding: 14rpx 18rpx;
  border-radius: 999rpx;
  background: rgba(255, 255, 255, 0.18);
  color: #ffffff;
  font-size: 24rpx;
  white-space: nowrap;
}

.banner-dots {
  display: flex;
  justify-content: flex-end;
  gap: 8rpx;
  margin: -36rpx 20rpx 28rpx 0;
}

.banner-dots text {
  width: 10rpx;
  height: 10rpx;
  border-radius: 999rpx;
  background: rgba(255, 255, 255, 0.56);
}

.banner-dots .active {
  width: 30rpx;
  background: #ffffff;
}

.trust-strip {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12rpx;
  margin: -4rpx 0 20rpx;
}

.trust-strip text {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 58rpx;
  border-radius: 12rpx;
  background: #ffffff;
  color: #0f766e;
  font-size: 24rpx;
  font-weight: 700;
  box-shadow: 0 8rpx 24rpx rgba(15, 23, 42, 0.04);
}

.activity-card {
  display: grid;
  grid-template-columns: 144rpx minmax(0, 1fr) 64rpx;
  align-items: center;
  gap: 20rpx;
  margin-bottom: 20rpx;
  padding: 20rpx;
  border-radius: 12rpx;
  background: #ffffff;
  box-shadow: 0 8rpx 24rpx rgba(15, 23, 42, 0.05);
}

.activity-cover {
  display: flex;
  align-items: flex-end;
  width: 144rpx;
  height: 144rpx;
  padding: 12rpx;
  border-radius: 10rpx;
  background:
    radial-gradient(circle at 32% 24%, rgba(255, 255, 255, 0.26), transparent 28%),
    #7b8fc7;
  color: #ffffff;
  font-size: 22rpx;
  font-weight: 700;
}

.activity-copy {
  display: grid;
  gap: 8rpx;
  min-width: 0;
}

.activity-tag {
  color: #b7791f;
  font-size: 22rpx;
  font-weight: 700;
}

.activity-title {
  color: #1f2933;
  font-size: 30rpx;
  font-weight: 700;
  line-height: 1.35;
}

.activity-desc {
  color: #697586;
  font-size: 24rpx;
  line-height: 1.45;
}

.activity-action {
  color: #0f766e;
  font-size: 26rpx;
  font-weight: 700;
}

.focus-card,
.recommend-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20rpx;
  margin-bottom: 20rpx;
  padding: 28rpx;
  border-radius: 12rpx;
  background: #ffffff;
  box-shadow: 0 8rpx 24rpx rgba(15, 23, 42, 0.05);
}

.focus-card {
  border: 1rpx solid rgba(15, 118, 110, 0.18);
  background:
    linear-gradient(135deg, rgba(15, 118, 110, 0.1), rgba(37, 99, 235, 0.08)),
    #ffffff;
}

.focus-label,
.recommend-tag {
  display: inline-flex;
  align-items: center;
  min-height: 34rpx;
  margin-bottom: 12rpx;
  padding: 0 12rpx;
  border-radius: 8rpx;
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
  padding: 12rpx 16rpx;
  border-radius: 10rpx;
  background: #ffffff;
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
  min-height: 164rpx;
  padding: 28rpx;
  border-radius: 12rpx;
  text-align: left;
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
  font-size: 24rpx;
}

.scene-title {
  font-size: 36rpx;
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
  font-size: 36rpx;
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
