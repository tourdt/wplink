<template>
  <view v-if="!selectedDirection" class="publish-entry-page">
    <view class="entry-head">
      <text class="entry-title">发布</text>
      <text class="entry-desc">选择本次要发布的内容类型，后续字段会按资源或需求自动切换。</text>
    </view>
    <view class="publish-direction-grid">
      <button
        v-for="item in publishDirectionOptions"
        :key="item.value"
        :class="['publish-direction-card', item.value]"
        @click="startPublish(item.value)"
      >
        <text class="direction-title">{{ item.title }}</text>
        <text class="direction-desc">{{ item.desc }}</text>
      </button>
    </view>
  </view>

  <ResourcePublishForm
    v-if="selectedDirection"
    :initial-options="initialOptions"
    mode="create"
    :reserve-bottom-safe-area="false"
  />
</template>

<script setup>
import { ref } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import ResourcePublishForm from '../../components/ResourcePublishForm.vue'

const PUBLISH_TYPE_KEY = 'wplink_pending_publish_type_code'
const RESOURCE_DIRECTION_SUPPLY = 'supply'
const RESOURCE_DIRECTION_DEMAND = 'demand'
const publishDirectionOptions = [
  {
    title: '发布资源',
    desc: '发布现货、库存、工厂、招聘、服务等供给信息',
    value: RESOURCE_DIRECTION_SUPPLY,
  },
  {
    title: '发布需求',
    desc: '发布找现货、找库存、找工厂、找服务等采购需求',
    value: RESOURCE_DIRECTION_DEMAND,
  },
]
const selectedDirection = ref('')
const initialOptions = ref({})

onLoad(applyPendingPublishType)
onShow(applyPendingPublishType)

function applyPendingPublishType() {
  const pendingPublish = uni.getStorageSync(PUBLISH_TYPE_KEY)
  if (!pendingPublish) return
  uni.removeStorageSync(PUBLISH_TYPE_KEY)
  const pendingTypeCode = typeof pendingPublish === 'object'
    ? pendingPublish.typeCode || ''
    : pendingPublish
  const pendingDirection = normalizePublishDirection(
    typeof pendingPublish === 'object' ? pendingPublish.direction : '',
  ) || RESOURCE_DIRECTION_SUPPLY
  selectedDirection.value = pendingDirection
  initialOptions.value = { typeCode: pendingTypeCode, direction: pendingDirection }
}

function startPublish(direction) {
  selectedDirection.value = normalizePublishDirection(direction) || RESOURCE_DIRECTION_SUPPLY
  initialOptions.value = { typeCode: '', direction: selectedDirection.value }
}

function normalizePublishDirection(value) {
  return [RESOURCE_DIRECTION_SUPPLY, RESOURCE_DIRECTION_DEMAND].includes(value) ? value : ''
}
</script>

<style lang="scss" scoped>
.publish-entry-page {
  min-height: 100vh;
  padding: 32rpx 24rpx;
  background: $wplink-bg;
}

.entry-head {
  display: grid;
  gap: 10rpx;
  margin-bottom: 24rpx;
}

.entry-title {
  color: $wplink-primary;
  font-size: 40rpx;
  font-weight: 700;
  line-height: 1.25;
}

.entry-desc {
  color: $wplink-muted;
  font-size: 26rpx;
  line-height: 1.5;
}

.publish-direction-grid {
  display: grid;
  gap: 18rpx;
}

.publish-direction-card {
  display: grid;
  gap: 12rpx;
  justify-items: start;
  min-height: 176rpx;
  padding: 28rpx;
  border: 1rpx solid $wplink-line;
  border-radius: 12rpx;
  background: $wplink-card;
  text-align: left;
  box-shadow: 0 8rpx 24rpx rgba(15, 23, 42, 0.04);
}

.publish-direction-card.supply {
  border-color: rgba($wplink-primary, 0.18);
}

.publish-direction-card.demand {
  border-color: rgba($wplink-warning, 0.22);
}

.direction-title {
  color: $wplink-primary;
  font-size: 34rpx;
  font-weight: 700;
  line-height: 1.3;
}

.direction-desc {
  color: $wplink-muted;
  font-size: 26rpx;
  line-height: 1.5;
}
</style>
