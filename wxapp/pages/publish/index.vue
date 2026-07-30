<template>
  <view class="publish-entry-page">
    <view class="publish-rule-card">
      <view class="rule-head">
        <text class="rule-title">发布前须知</text>
        <text class="rule-badge">每月 3 条免费</text>
      </view>
      <text class="rule-summary">提交后由系统自动安全检测，请确保信息可核验、可履约。</text>
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
    <view class="publish-category-section">
      <view class="section-headline">
        <text class="direction-section-title">选择发布大类</text>
        <button class="reload-button" :disabled="loadingCategories" @click="loadPublishCategories">刷新</button>
      </view>
      <text class="category-helper">按童装批发、厂房仓库、本地服务等场景选择，系统会自动匹配供应或需求表单。</text>
      <view v-if="categoryGroups.length" class="publish-category-list">
        <button
          v-for="group in categoryGroups"
          :key="group.code"
          class="category-group-button"
          @click="startPublishGroup(group)"
        >
          <text class="category-group-name">{{ group.name }}</text>
        </button>
      </view>
      <view v-else class="category-empty">
        <text>{{ loadingCategories ? '类目加载中...' : '暂无可发布类目，请稍后重试' }}</text>
      </view>
    </view>

    <view v-if="showTypeSheet" class="type-sheet-mask" @click="closeTypeSheet">
      <view class="type-sheet-panel" @click.stop>
        <view class="type-sheet-head">
          <view class="type-sheet-copy">
            <text class="type-sheet-title">选择具体发布类型</text>
            <text class="type-sheet-subtitle">{{ selectedCategoryName }}</text>
          </view>
          <button class="type-sheet-close" @click="closeTypeSheet">关闭</button>
        </view>
        <view class="type-option-list">
          <button
            v-for="item in selectedCategoryItems"
            :key="item.typeCode"
            class="type-option-button"
            @click="startPublishCategory(item)"
          >
            <text class="type-option-name">{{ item.typeName }}</text>
          </button>
        </view>
      </view>
    </view>
  </view>
</template>

<script setup>
import { onLoad, onShow } from '@dcloudio/uni-app'
import { computed, ref } from 'vue'
import { requireLogin } from '../../common/auth'
import { DEFAULT_CITY_CODE } from '../../common/constants'
import { groupResourceTypes } from '../../common/resourceCategories'
import { listCityResourceTypes } from '../../api/city'

const PUBLISH_TYPE_KEY = 'wplink_pending_publish_type_code'
const categoryGroups = ref([])
const loadingCategories = ref(false)
const selectedCategoryGroup = ref(null)
const showTypeSheet = ref(false)
const selectedCategoryItems = computed(() => selectedCategoryGroup.value?.items || [])
const selectedCategoryName = computed(() => selectedCategoryGroup.value?.name || '发布类型')
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
    title: '安全与权益',
    content: '违规内容可能下架；严重时限制功能、封停账号、收回相关权益，费用不予退还。',
  },
]

onLoad(async () => {
  await loadPublishCategories()
  applyPendingPublishType()
})
onShow(applyPendingPublishType)

async function loadPublishCategories() {
  if (loadingCategories.value) return
  loadingCategories.value = true
  try {
    // 发布入口只读取一份二级类目配置；供需方向由被点击的二级类目决定。
    const resp = await listCityResourceTypes(DEFAULT_CITY_CODE)
    categoryGroups.value = groupResourceTypes(resp.items || [])
    if (selectedCategoryGroup.value) {
      selectedCategoryGroup.value = categoryGroups.value.find((group) => group.code === selectedCategoryGroup.value?.code) || null
      showTypeSheet.value = Boolean(selectedCategoryGroup.value)
    }
  } finally {
    loadingCategories.value = false
  }
}

async function applyPendingPublishType() {
  // 首页快捷入口会先写入待发布类型，tab onShow 消费后立即进入独立表单页。
  const pendingPublish = uni.getStorageSync(PUBLISH_TYPE_KEY)
  if (!pendingPublish) return
  if (!requireLogin()) return
  uni.removeStorageSync(PUBLISH_TYPE_KEY)
  const pendingTypeCode = typeof pendingPublish === 'object'
    ? pendingPublish.typeCode || ''
    : pendingPublish
  navigateToPublishForm({ typeCode: pendingTypeCode })
}

function startPublishGroup(group) {
  if (!requireLogin()) return
  const items = group?.items || []
  if (items.length === 0) {
    uni.showToast({ title: '该类目暂无可发布类型', icon: 'none' })
    return
  }
  selectedCategoryGroup.value = group
  if (items.length === 1) {
    navigateToPublishForm({ typeCode: items[0].typeCode })
    return
  }
  showTypeSheet.value = true
}

function startPublishCategory(item) {
  if (!requireLogin()) return
  showTypeSheet.value = false
  navigateToPublishForm({ typeCode: item.typeCode })
}

function closeTypeSheet() {
  showTypeSheet.value = false
}

function navigateToPublishForm(options = {}) {
  const initialPublishOptions = {
    typeCode: options.typeCode || '',
  }
  const query = []
  initialPublishOptions.typeCode && query.push(`typeCode=${encodeURIComponent(initialPublishOptions.typeCode)}`)
  uni.navigateTo({ url: `/pages/publish/edit?${query.join('&')}` })
}
</script>

<style lang="scss" scoped>
.publish-entry-page {
  min-height: 100vh;
  padding: 28rpx 24rpx 40rpx;
  background: linear-gradient(180deg, #f8fbff 0%, $wplink-bg 280rpx);
}

.publish-category-section {
  display: grid;
  gap: 14rpx;
}

.direction-section-title {
  padding: 0 4rpx;
  color: $wplink-muted;
  font-size: 22rpx;
  font-weight: 700;
  line-height: 1.35;
}

.section-headline {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16rpx;
}

.reload-button {
  flex: 0 0 auto;
  height: 54rpx;
  padding: 0 18rpx;
  border: 1rpx solid $wplink-line;
  border-radius: 8rpx;
  background: #fff;
  color: $wplink-muted;
  font-size: 22rpx;
  line-height: 54rpx;
}

.reload-button::after {
  border: 0;
}

.category-helper {
  padding: 0 4rpx;
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 1.5;
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

.publish-category-list {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12rpx;
}

.category-group-button {
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 0;
  min-height: 92rpx;
  margin: 0;
  padding: 0 16rpx;
  border: 1rpx solid $wplink-line;
  border-radius: 10rpx;
  background: $wplink-card;
  line-height: normal;
}

.category-group-button::after {
  border: 0;
}

.category-group-button:active {
  background: rgba($wplink-primary, 0.035);
}

.category-group-name {
  max-width: 100%;
  overflow: hidden;
  color: $wplink-primary;
  font-size: 27rpx;
  font-weight: 700;
  line-height: 1.35;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.type-sheet-mask {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 50;
  display: flex;
  align-items: flex-end;
  background: rgba(15, 23, 42, 0.38);
}

.type-sheet-panel {
  width: 100%;
  max-height: 70vh;
  padding: 28rpx 24rpx calc(32rpx + env(safe-area-inset-bottom));
  border-radius: 18rpx 18rpx 0 0;
  background: $wplink-bg;
}

.type-sheet-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16rpx;
  margin-bottom: 20rpx;
}

.type-sheet-copy {
  display: grid;
  gap: 4rpx;
  min-width: 0;
}

.type-sheet-title {
  color: $wplink-primary;
  font-size: 31rpx;
  font-weight: 700;
  line-height: 1.35;
}

.type-sheet-subtitle {
  color: $wplink-muted;
  font-size: 23rpx;
  line-height: 1.35;
}

.type-sheet-close {
  flex: 0 0 auto;
  width: 112rpx;
  height: 58rpx;
  margin: 0;
  border-radius: 10rpx;
  background: $wplink-card;
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 58rpx;
}

.type-sheet-close::after {
  border: 0;
}

.type-option-list {
  display: grid;
  gap: 12rpx;
}

.type-option-button {
  display: flex;
  align-items: center;
  min-height: 78rpx;
  margin: 0;
  padding: 0 20rpx;
  border: 1rpx solid rgba($wplink-primary, 0.12);
  border-radius: 10rpx;
  background: $wplink-card;
  text-align: left;
  line-height: normal;
}

.type-option-button::after {
  border: 0;
}

.type-option-button:active {
  background: rgba($wplink-primary, 0.035);
}

.type-option-name {
  min-width: 0;
  color: $wplink-primary;
  font-size: 26rpx;
  font-weight: 700;
  line-height: 1.35;
}

.category-empty {
  padding: 28rpx;
  border: 1rpx dashed $wplink-line;
  border-radius: 12rpx;
  color: $wplink-muted;
  font-size: 24rpx;
  text-align: center;
}
</style>
