<template>
  <view class="binding-page">
    <view class="status-card">
      <view>
        <text class="status-title">拿货地图档口</text>
        <text class="status-desc">{{ statusSummary }}</text>
      </view>
      <text class="status-badge">{{ statusText }}</text>
    </view>

    <view v-if="!boundObject" class="search-panel">
      <view class="field-row">
        <text class="field-label">地图场景</text>
        <picker :range="sceneOptions" range-key="name" @change="changeScene">
          <view class="picker-field">{{ selectedSceneName }}</view>
        </picker>
      </view>
      <view class="field-row">
        <text class="field-label">档口关键词</text>
        <view class="search-row">
          <input v-model="keyword" class="field" placeholder="档口编号、名称或地址" @confirm="searchCandidates" />
          <button class="search-button" :disabled="candidateLoading" @click="searchCandidates">搜索</button>
        </view>
      </view>
    </view>

    <view v-if="!boundObject" class="candidate-list">
      <view
        v-for="item in candidates"
        :key="item.objectId"
        :class="['candidate-item', { selected: selectedObjectId === item.objectId, disabled: item.isBound }]"
        @click="selectCandidate(item)"
      >
        <view>
          <text class="candidate-title">{{ item.code }} {{ item.name }}</text>
          <text class="candidate-desc">{{ item.sceneName }} · {{ item.address || '地址待补充' }}</text>
          <text v-if="item.isBound" class="candidate-warning">已绑定 {{ item.merchantName || '其他商家' }}</text>
        </view>
        <text class="candidate-check">{{ selectedObjectId === item.objectId ? '已选' : item.isBound ? '占用' : '选择' }}</text>
      </view>
      <view v-if="!candidateLoading && candidates.length === 0" class="empty-state">
        <text>暂无匹配档口</text>
      </view>
    </view>

    <view v-if="!boundObject" class="submit-panel">
      <view class="field-row">
        <text class="field-label">档口说明（选填）</text>
        <textarea v-model="note" class="textarea" placeholder="例如：A001 档口，门头名为 XX 童装。" />
        <text class="candidate-desc">每个商家暂时只能绑定一个主档口，绑定后可通过位置纠错处理变更。</text>
      </view>
    </view>

    <view v-if="!boundObject" class="fixed-submit-spacer" />
    <view v-if="!boundObject" class="fixed-submit-bar">
      <button class="primary-button" :disabled="submitting || !selectedObjectId" @click="submitBindingRequest">确认绑定</button>
    </view>
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { ensureMerchantProfileReady } from '../../common/merchantProfileGuard'
import { getMerchantMapBinding, listMapBindCandidates, listMapScenes, submitMapBindRequest } from '../../api/sourcingMap'
import { DEFAULT_CITY_CODE } from '../../common/constants'
import { getMerchantId } from '../../store/session'

const merchantId = ref('')
const scenes = ref([])
const selectedSceneCode = ref('')
const keyword = ref('')
const candidates = ref([])
const selectedObjectId = ref('')
const note = ref('')
const bindingStatus = ref({})
const sceneLoading = ref(false)
const candidateLoading = ref(false)
const submitting = ref(false)

const sceneOptions = computed(() => scenes.value)
const selectedSceneName = computed(() => {
  const scene = scenes.value.find((item) => item.code === selectedSceneCode.value)
  return scene?.name || '请选择地图场景'
})
const latestRequest = computed(() => bindingStatus.value.latestRequest || {})
const boundObject = computed(() => bindingStatus.value.boundObject || null)
const statusText = computed(() => {
  if (boundObject.value) return '已绑定'
  if (latestRequest.value.status === 'pending') return '处理中'
  if (latestRequest.value.status === 'rejected') return '已驳回'
  return '未绑定'
})
const statusSummary = computed(() => {
  if (boundObject.value) return `${boundObject.value.sceneName || '拿货地图'} · ${boundObject.value.code || ''} ${boundObject.value.name || ''}`
  if (latestRequest.value.status === 'pending') return '历史绑定记录正在处理中，请稍后查看结果。'
  if (latestRequest.value.status === 'rejected') return latestRequest.value.reviewNote || '申请未通过，可重新选择档口后提交。'
  return '请选择你的主档口，确认后会立即关联到拿货地图。'
})

onLoad(async (options) => {
  merchantId.value = options.merchantId || getMerchantId()
  if (!(await ensureMerchantProfileReady(merchantId.value))) return
  loadInitialData()
})

onShow(() => {
  if (merchantId.value) loadBindingStatus()
})

async function loadInitialData() {
  await Promise.all([loadBindingStatus(), loadScenes()])
  await searchCandidates()
}

async function loadBindingStatus() {
  try {
    bindingStatus.value = await getMerchantMapBinding(merchantId.value, { suppressErrorToast: true })
  } catch (err) {
    bindingStatus.value = {}
  }
}

async function loadScenes() {
  sceneLoading.value = true
  try {
    const resp = await listMapScenes({ cityCode: DEFAULT_CITY_CODE })
    scenes.value = resp.items || []
    if (!selectedSceneCode.value && scenes.value.length) {
      selectedSceneCode.value = scenes.value[0].code
    }
  } catch (err) {
    uni.showToast({ title: '地图场景加载失败', icon: 'none' })
  } finally {
    sceneLoading.value = false
  }
}

function changeScene(event) {
  const scene = scenes.value[Number(event.detail.value)]
  selectedSceneCode.value = scene?.code || ''
  selectedObjectId.value = ''
  searchCandidates()
}

async function searchCandidates() {
  if (!merchantId.value) return
  candidateLoading.value = true
  try {
    const resp = await listMapBindCandidates({
      merchantId: merchantId.value,
      sceneCode: selectedSceneCode.value,
      keyword: keyword.value.trim(),
      limit: 30,
    })
    candidates.value = resp.items || []
  } catch (err) {
    candidates.value = []
    uni.showToast({ title: '档口加载失败，请重试', icon: 'none' })
  } finally {
    candidateLoading.value = false
  }
}

function selectCandidate(item) {
  if (item.isBound) {
    uni.showToast({ title: '该档口已绑定其他商家', icon: 'none' })
    return
  }
  selectedObjectId.value = item.objectId
}

async function submitBindingRequest() {
  if (!selectedObjectId.value) {
    uni.showToast({ title: '请选择要绑定的档口', icon: 'none' })
    return
  }
  submitting.value = true
  try {
    await submitMapBindRequest(merchantId.value, {
      objectId: selectedObjectId.value,
      note: note.value.trim(),
      evidenceImages: [],
    })
    uni.showToast({ title: '档口已绑定', icon: 'none' })
    await loadBindingStatus()
  } catch (err) {
    uni.showToast({ title: err.message || '档口绑定失败', icon: 'none' })
  } finally {
    submitting.value = false
  }
}
</script>

<style lang="scss" scoped>
.binding-page {
  min-height: 100vh;
  padding: 24rpx 24rpx 0;
  background: $wplink-bg;
}

.status-card,
.search-panel,
.submit-panel {
  display: grid;
  gap: 18rpx;
  margin-bottom: 20rpx;
  padding: 24rpx;
  border-radius: 12rpx;
  background: $wplink-card;
}

.status-card {
  grid-template-columns: minmax(0, 1fr) 120rpx;
  align-items: center;
}

.status-title,
.field-label,
.candidate-title {
  color: $wplink-primary;
  font-size: 28rpx;
  font-weight: 700;
}

.status-desc,
.candidate-desc,
.candidate-warning {
  display: block;
  margin-top: 8rpx;
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 1.45;
}

.status-badge {
  padding: 8rpx 12rpx;
  border-radius: 999rpx;
  background: $wplink-primary-soft;
  color: $wplink-primary;
  font-size: 24rpx;
  text-align: center;
}

.field-row {
  display: grid;
  gap: 12rpx;
}

.picker-field,
.field,
.textarea {
  width: 100%;
  min-height: 80rpx;
  padding: 0 20rpx;
  border: 1rpx solid $wplink-line;
  border-radius: 10rpx;
  box-sizing: border-box;
  background: #fff;
}

.picker-field {
  display: flex;
  align-items: center;
  color: $wplink-primary;
}

.search-row,
.evidence-row,
.image-title-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 148rpx;
  gap: 12rpx;
  align-items: center;
}

.search-button,
.remove-button,
.link-button {
  height: 72rpx;
  border-radius: 10rpx;
  font-size: 24rpx;
  line-height: 72rpx;
}

.search-button {
  background: $wplink-primary;
  color: #fff;
}

.remove-button,
.link-button {
  background: #f8fafc;
  color: $wplink-primary;
}

.candidate-list {
  display: grid;
  gap: 14rpx;
}

.candidate-item {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 96rpx;
  gap: 12rpx;
  align-items: center;
  padding: 22rpx;
  border: 1rpx solid $wplink-line;
  border-radius: 12rpx;
  background: $wplink-card;
}

.candidate-item.selected {
  border-color: $wplink-primary;
  background: $wplink-primary-soft;
}

.candidate-item.disabled {
  opacity: 0.72;
}

.candidate-warning {
  color: $wplink-warning;
}

.candidate-check {
  color: $wplink-primary;
  font-size: 24rpx;
  text-align: right;
}

.empty-state {
  padding: 56rpx 0;
  color: $wplink-muted;
  font-size: 26rpx;
  text-align: center;
}

.textarea {
  min-height: 150rpx;
  padding-top: 18rpx;
}

.fixed-submit-spacer {
  height: 132rpx;
}

.fixed-submit-bar {
  position: fixed;
  right: 0;
  bottom: 0;
  left: 0;
  padding: 18rpx 24rpx calc(18rpx + env(safe-area-inset-bottom));
  background: rgba(255, 255, 255, 0.96);
  box-shadow: 0 -8rpx 24rpx rgba(15, 23, 42, 0.08);
}

.primary-button {
  height: 88rpx;
  border-radius: 12rpx;
  background: $wplink-primary;
  color: #fff;
  font-size: 30rpx;
  font-weight: 700;
  line-height: 88rpx;
}

button::after {
  border: 0;
}
</style>
