<template>
  <view class="topic-page">
    <view class="topic-hero">
      <image v-if="topic.coverUrl" class="topic-cover" :src="topic.coverUrl" mode="aspectFill" />
      <view class="topic-copy">
        <text class="topic-title">{{ topic.title }}</text>
        <text class="topic-subtitle">{{ topic.subtitle }}</text>
      </view>
    </view>

    <view v-if="rows.length" class="result-list">
      <ResourceCard v-for="item in rows" :key="item.id" :resource="item" @open="openResource" />
    </view>

    <view v-else-if="demandEntry" class="empty-card">
      <text class="empty-title">{{ demandEntry.title }}</text>
      <button class="primary-button" @click="openDemand">{{ demandEntry.buttonText }}</button>
    </view>
  </view>
</template>

<script setup>
import { ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import ResourceCard from '../../components/ResourceCard.vue'
import { DEFAULT_CITY_CODE } from '../../common/constants'
import { getTopicResources } from '../../api/discovery'

const topic = ref({})
const rows = ref([])
const demandEntry = ref(null)

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

<style scoped>
.topic-page {
  min-height: 100vh;
  padding: 24rpx;
  background: #f4f6f8;
}

.topic-hero {
  position: relative;
  min-height: 240rpx;
  margin-bottom: 20rpx;
  overflow: hidden;
  border-radius: 12rpx;
  background: #0f766e;
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
  color: #ffffff;
}

.topic-title,
.empty-title {
  font-size: 36rpx;
  font-weight: 700;
}

.topic-subtitle {
  font-size: 26rpx;
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

.primary-button {
  height: 84rpx;
  border-radius: 12rpx;
  background: #0f766e;
  color: #ffffff;
}
</style>
