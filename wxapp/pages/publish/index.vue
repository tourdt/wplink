<template>
  <view class="publish-entry-page">
    <view class="publish-rule-card">
      <view class="rule-head">
        <text class="rule-title">发布前须知</text>
        <text class="rule-badge">平台审核</text>
      </view>
      <text class="rule-summary">提交后进入平台审核，请确保信息可核验、可履约。</text>
      <view class="rule-list">
        <view
          v-for="(rule, index) in publishRules"
          :key="rule.title"
          class="rule-item"
        >
          <view class="rule-index">{{ index + 1 }}</view>
          <view class="rule-copy">
            <text class="rule-label">{{ rule.title }}</text>
            <text class="rule-text">{{ rule.content }}</text>
          </view>
        </view>
      </view>
    </view>
    <view class="publish-direction-section">
      <text class="direction-section-title">发布类型</text>
      <view class="publish-direction-list">
        <button
          v-for="item in publishDirectionOptions"
          :key="item.value"
          :class="['publish-direction-row', item.value]"
          @click="startPublish(item.value)"
        >
          <view :class="['direction-marker', item.value]" />
          <view class="direction-copy">
            <text class="direction-title">{{ item.title }}</text>
            <text class="direction-desc">{{ item.desc }}</text>
          </view>
          <text class="direction-arrow">›</text>
        </button>
      </view>
    </view>
  </view>
</template>

<script setup>
import { onLoad, onShow } from '@dcloudio/uni-app'
import { requireLogin } from '../../common/auth'

const PUBLISH_TYPE_KEY = 'wplink_pending_publish_type_code'
const RESOURCE_DIRECTION_SUPPLY = 'supply'
const RESOURCE_DIRECTION_DEMAND = 'demand'
const publishRules = [
  {
    title: '真实有效',
    content: '发布内容须真实、合法、有效，联系方式、价格、数量、交期、服务范围等应与实际一致。',
  },
  {
    title: '禁止违规',
    content: '不得发布违法违规、虚假夸大、重复刷屏、无关推广、侵权或误导交易内容。',
  },
  {
    title: '审核与权益',
    content: '违规内容可能下架；严重时限制功能、封停账号、收回相关权益，费用不予退还。',
  },
]
const publishDirectionOptions = [
  {
    title: '我能提供',
    desc: '发布货源、库存、产能、服务等供需信息',
    value: RESOURCE_DIRECTION_SUPPLY,
  },
  {
    title: '我想寻找',
    desc: '发布找货、找工厂、找服务等需求',
    value: RESOURCE_DIRECTION_DEMAND,
  },
]

onLoad(applyPendingPublishType)
onShow(applyPendingPublishType)

async function applyPendingPublishType() {
  // 首页快捷入口会先写入待发布类型，tab onShow 消费后立即进入独立表单页。
  const pendingPublish = uni.getStorageSync(PUBLISH_TYPE_KEY)
  if (!pendingPublish) return
  if (!requireLogin()) return
  uni.removeStorageSync(PUBLISH_TYPE_KEY)
  const pendingTypeCode = typeof pendingPublish === 'object'
    ? pendingPublish.typeCode || ''
    : pendingPublish
  const pendingDirection = normalizePublishDirection(
    typeof pendingPublish === 'object' ? pendingPublish.direction : '',
  ) || RESOURCE_DIRECTION_SUPPLY
  navigateToPublishForm({ typeCode: pendingTypeCode, direction: pendingDirection })
}

function startPublish(direction) {
  if (!requireLogin()) return
  const publishDirection = normalizePublishDirection(direction) || RESOURCE_DIRECTION_SUPPLY
  navigateToPublishForm({ typeCode: '', direction: publishDirection })
}

function navigateToPublishForm(options = {}) {
  const initialPublishOptions = {
    typeCode: options.typeCode || '',
    direction: normalizePublishDirection(options.direction || '') || RESOURCE_DIRECTION_SUPPLY,
  }
  const query = []
  initialPublishOptions.direction && query.push(`direction=${encodeURIComponent(initialPublishOptions.direction)}`)
  initialPublishOptions.typeCode && query.push(`typeCode=${encodeURIComponent(initialPublishOptions.typeCode)}`)
  uni.navigateTo({ url: `/pages/publish/edit?${query.join('&')}` })
}

function normalizePublishDirection(value) {
  return [RESOURCE_DIRECTION_SUPPLY, RESOURCE_DIRECTION_DEMAND].includes(value) ? value : ''
}
</script>

<style lang="scss" scoped>
.publish-entry-page {
  min-height: 100vh;
  padding: 28rpx 24rpx 40rpx;
  background: linear-gradient(180deg, #f8fbff 0%, $wplink-bg 280rpx);
}

.publish-direction-section {
  display: grid;
  gap: 12rpx;
}

.direction-section-title {
  padding: 0 4rpx;
  color: $wplink-muted;
  font-size: 22rpx;
  font-weight: 700;
  line-height: 1.35;
}

.publish-rule-card {
  display: grid;
  gap: 18rpx;
  box-sizing: border-box;
  margin-bottom: 26rpx;
  padding: 26rpx 24rpx;
  border: 1rpx solid rgba($wplink-warning, 0.22);
  border-radius: 12rpx;
  background: linear-gradient(180deg, rgba($wplink-warning, 0.08), $wplink-card 52%);
  box-shadow: 0 12rpx 30rpx rgba(15, 23, 42, 0.06);
}

.rule-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16rpx;
  min-width: 0;
}

.rule-title {
  color: $wplink-primary;
  font-size: 32rpx;
  font-weight: 700;
  line-height: 1.35;
}

.rule-badge {
  flex: 0 0 auto;
  padding: 5rpx 14rpx;
  border-radius: 999rpx;
  background: rgba($wplink-warning, 0.1);
  color: $wplink-warning;
  font-size: 22rpx;
  font-weight: 700;
  line-height: 1.35;
}

.rule-summary {
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 1.5;
}

.rule-list {
  display: grid;
  gap: 14rpx;
}

.rule-item {
  display: flex;
  align-items: flex-start;
  gap: 14rpx;
}

.rule-index {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 auto;
  width: 34rpx;
  height: 34rpx;
  margin-top: 2rpx;
  border-radius: 999rpx;
  background: rgba($wplink-warning, 0.12);
  color: $wplink-warning;
  font-size: 20rpx;
  font-weight: 700;
  line-height: 1;
}

.rule-copy {
  display: grid;
  gap: 4rpx;
  min-width: 0;
}

.rule-label {
  color: $wplink-primary;
  font-size: 24rpx;
  font-weight: 700;
  line-height: 1.35;
}

.rule-text {
  min-width: 0;
  color: $wplink-text;
  font-size: 23rpx;
  line-height: 1.55;
}

.publish-direction-list {
  display: grid;
  box-sizing: border-box;
  overflow: hidden;
  width: 100%;
  border: 1rpx solid $wplink-line;
  border-radius: 12rpx;
  background: $wplink-card;
  box-shadow: 0 8rpx 24rpx rgba(15, 23, 42, 0.04);
}

.publish-direction-row {
  display: flex;
  align-items: center;
  gap: 18rpx;
  box-sizing: border-box;
  width: 100%;
  min-height: 132rpx;
  margin: 0;
  padding: 26rpx 24rpx;
  border: 0;
  border-radius: 0;
  background: transparent;
  text-align: left;
  line-height: normal;
  transition: background 160ms ease;
}

.publish-direction-row + .publish-direction-row {
  border-top: 1rpx solid rgba($wplink-line, 0.72);
}

.publish-direction-row:active {
  background: rgba($wplink-primary, 0.035);
}

.publish-direction-row::after {
  border: 0;
}

.direction-marker {
  flex: 0 0 auto;
  width: 6rpx;
  height: 48rpx;
  border-radius: 999rpx;
  background: rgba($wplink-primary, 0.72);
}

.direction-marker.demand {
  background: rgba($wplink-warning, 0.72);
}

.direction-copy {
  display: grid;
  flex: 1 1 auto;
  gap: 10rpx;
  min-width: 0;
  text-align: left;
}

.direction-title {
  color: $wplink-primary;
  font-size: 32rpx;
  font-weight: 700;
  line-height: 1.3;
}

.direction-desc {
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 1.5;
}

.direction-arrow {
  flex: 0 0 auto;
  color: $wplink-muted;
  font-size: 38rpx;
  font-weight: 300;
  line-height: 1;
}
</style>
