<template>
  <view class="resource-list-shell">
    <view v-if="resources.length === 0 && !loading" class="empty-text">{{ emptyText }}</view>
    <view v-else class="resource-list">
      <ResourceExposure
        v-for="item in resources"
        :key="item.id"
        :resource-id="item.id"
        :source="exposureSource"
      >
        <ResourceFeedCard
          v-if="variant === 'feed'"
          :resource="item"
          @open="emit('open', $event)"
        />
        <ResourceCard
          v-else
          :resource="item"
          :variant="variant"
          @open="emit('open', $event)"
        />
      </ResourceExposure>
    </view>

    <button v-if="hasMore || loading" class="load-more" :disabled="loading" @click="emit('load-more')">
      {{ loading ? loadingText : loadMoreText }}
    </button>
  </view>
</template>

<script setup>
import ResourceCard from './ResourceCard.vue'
import ResourceExposure from './ResourceExposure.vue'
import ResourceFeedCard from './ResourceFeedCard.vue'

defineProps({
  resources: {
    type: Array,
    default: () => [],
  },
  emptyText: {
    type: String,
    default: '暂无供应',
  },
  loading: {
    type: Boolean,
    default: false,
  },
  hasMore: {
    type: Boolean,
    default: false,
  },
  loadMoreText: {
    type: String,
    default: '加载更多',
  },
  loadingText: {
    type: String,
    default: '加载中...',
  },
  variant: {
    type: String,
    default: '',
  },
  exposureSource: {
    type: String,
    default: 'list',
  },
})

const emit = defineEmits(['open', 'load-more'])
</script>

<style lang="scss" scoped>
.resource-list-shell {
  display: grid;
  gap: 18rpx;
}

.resource-list {
  display: grid;
  gap: 18rpx;
}

.empty-text {
  display: grid;
  place-items: center;
  min-height: 360rpx;
  padding: 32rpx 0;
  padding-bottom: 80rpx;
  color: $wplink-muted;
  font-size: 26rpx;
  text-align: center;
}

.load-more {
  height: 72rpx;
  border: 1rpx solid $wplink-line;
  border-radius: 10rpx;
  background: $wplink-card;
  color: $wplink-primary;
  font-size: 26rpx;
  font-weight: 700;
  line-height: 72rpx;
}

.load-more[disabled] {
  background: #f8fafc;
  color: $wplink-muted;
}
</style>
