<template>
  <view class="resource-list-shell">
    <view v-if="resources.length === 0 && !loading" class="empty-text">{{ emptyText }}</view>
    <view v-else class="resource-list">
      <ResourceCard
        v-for="item in resources"
        :key="item.id"
        :resource="item"
        :variant="variant"
        @open="emit('open', $event)"
      />
    </view>

    <button v-if="hasMore || loading" class="load-more" :disabled="loading" @click="emit('load-more')">
      {{ loading ? loadingText : loadMoreText }}
    </button>
  </view>
</template>

<script setup>
import ResourceCard from './ResourceCard.vue'

defineProps({
  resources: {
    type: Array,
    default: () => [],
  },
  emptyText: {
    type: String,
    default: '暂无资源',
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
  padding: 28rpx 0;
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
