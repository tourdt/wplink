<template>
  <view class="topic-page">
    <view class="topic-hero">
      <image v-if="topic.coverUrl" class="topic-cover" :src="topic.coverUrl" mode="aspectFill" />
      <view class="topic-copy">
        <text class="topic-label">Banner 专题</text>
        <text class="topic-title">{{ topic.title }}</text>
        <text class="topic-subtitle">{{ topic.subtitle || '运营配置专题列表，只展示已审核、未过期、平台内资源。' }}</text>
      </view>
    </view>

    <view class="topic-summary">
      <view v-for="item in topicStats" :key="item.label" class="topic-stat">
        <text class="stat-value">{{ item.value }}</text>
        <text class="stat-label">{{ item.label }}</text>
      </view>
    </view>

    <scroll-view class="filter-row" scroll-x>
      <button class="filter-button active">全部</button>
      <button class="filter-button">整包清</button>
      <button class="filter-button">可直播</button>
      <button class="filter-button">90-140</button>
      <button class="filter-button">平台核实</button>
    </scroll-view>

    <view v-if="rows.length" class="result-list">
      <ResourceCard v-for="item in rows" :key="item.id" :resource="item" @open="openResource" />
    </view>

    <view v-else class="empty-card">
      <text class="empty-title">{{ emptyTitle }}</text>
      <text class="empty-desc">提交需求后，运营会按专题条件继续帮你匹配。</text>
      <button class="primary-button" @click="openDemand">{{ emptyButtonText }}</button>
    </view>
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import ResourceCard from '../../components/ResourceCard.vue'
import { DEFAULT_CITY_CODE } from '../../common/constants'
import { getTopicResources } from '../../api/discovery'

const topic = ref({})
const rows = ref([])
const demandEntry = ref(null)
const topicStats = computed(() => [
  { label: '专题资源', value: rows.value.length },
  { label: '平台内资源', value: rows.value.filter((item) => item.status === 'published').length || rows.value.length },
  { label: '可提交需求', value: demandEntry.value ? 1 : 0 },
])
const emptyTitle = computed(() => (demandEntry.value || {}).title || '没有找到想要的款？')
const emptyButtonText = computed(() => (demandEntry.value || {}).buttonText || '提交找货需求')

onLoad(async (options) => {
  if (!options.id) return
  const resp = await getTopicResources(options.id, { cityCode: DEFAULT_CITY_CODE, page: 1, pageSize: 20 })
  topic.value = resp.topic || {}
  rows.value = resp.items || []
  demandEntry.value = resp.demandEntry || null
})

function openResource(item) {
  uni.navigateTo({ url: `/pages/resource/detail?id=${item.id}` })
}

function openDemand() {
  uni.navigateTo({ url: '/pages/demand/index' })
}
</script>

<style lang="scss" scoped>
.topic-page {
  min-height: 100vh;
  padding: 24rpx;
  background: $wplink-bg;
}

.topic-hero {
  position: relative;
  min-height: 240rpx;
  margin-bottom: 20rpx;
  overflow: hidden;
  border-radius: 12rpx;
  background: $wplink-primary;
}

.topic-cover {
  width: 100%;
  height: 240rpx;
}

.topic-copy {
  position: absolute;
  right: 24rpx;
  bottom: 24rpx;
  left: 24rpx;
  display: grid;
  gap: 8rpx;
  color: $wplink-card;
}

.topic-label {
  justify-self: start;
  padding: 6rpx 12rpx;
  border-radius: 8rpx;
  background: rgba(255, 255, 255, 0.18);
  color: $wplink-card;
  font-size: 22rpx;
}

.topic-title,
.empty-title {
  font-size: 36rpx;
  font-weight: 700;
}

.topic-subtitle,
.empty-desc {
  font-size: 26rpx;
  line-height: 1.5;
}

.topic-summary {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12rpx;
  margin-bottom: 20rpx;
}

.topic-stat {
  display: grid;
  gap: 6rpx;
  padding: 18rpx;
  border-radius: 12rpx;
  background: $wplink-card;
  text-align: center;
}

.stat-value {
  color: $wplink-primary;
  font-size: 34rpx;
  font-weight: 700;
}

.stat-label {
  color: $wplink-muted;
  font-size: 24rpx;
}

.filter-row {
  width: 100%;
  margin-bottom: 20rpx;
  white-space: nowrap;
}

.filter-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 128rpx;
  height: 68rpx;
  margin-right: 12rpx;
  padding: 0 20rpx;
  border-radius: 10rpx;
  background: $wplink-card;
  color: #364152;
  font-size: 24rpx;
}

.filter-button.active {
  background: $wplink-warning-soft;
  color: $wplink-primary;
  font-weight: 700;
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
  background: $wplink-card;
}

.empty-desc {
  color: $wplink-muted;
}

.primary-button {
  height: 84rpx;
  border-radius: 12rpx;
  background: $wplink-primary;
  color: $wplink-card;
}
</style>
