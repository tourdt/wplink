<template>
  <view class="publish-page">
    <view class="quota-card">
      <view>
        <text class="quota-label">认证商家权益</text>
        <text class="quota-title">本月可发布资源</text>
        <text class="quota-desc">审核通过后进入搜索、首页分类和商家主页，置顶券可提升曝光。</text>
      </view>
      <button @click="openMyResources">查看权益</button>
    </view>

    <scroll-view class="publish-types" scroll-x>
      <button
        v-for="item in publishTypeOptions"
        :key="item.value"
        :class="['type-button', form.typeCode === item.value ? 'active' : '']"
        @click="selectTypeByCode(item.value)"
      >
        {{ item.label }}
      </button>
    </scroll-view>

    <view class="form-card">
      <text class="page-title">发布资源</text>
      <view class="form-status">
        <text>{{ publishReadyText }}</text>
        <strong>{{ canSubmit ? '可提交审核' : '继续补充必填项' }}</strong>
      </view>
      <picker :range="resourceTypeNames" :value="selectedTypeIndex" @change="selectType">
        <view class="field picker-field">{{ selectedTypeLabel }}</view>
      </picker>
      <input v-model="form.title" class="field" placeholder="标题" />
      <input v-model="form.category" class="field" placeholder="品类" />
      <input v-model="form.quantityText" class="field" placeholder="数量/产能" />
      <input v-model="form.priceText" class="field" placeholder="价格描述" />
      <textarea v-model="form.description" class="textarea" placeholder="资源描述" />
      <view class="effect-preview">
        <text class="effect-label">发布后可获得</text>
        <text class="effect-value">搜索曝光 · 商家主页展示 · 联系统计</text>
      </view>
      <button class="secondary-button" @click="uploadResourceImage">上传资源图片</button>
      <view v-if="form.images.length" class="image-list">
        <text v-for="image in form.images" :key="image" class="image-url">{{ image }}</text>
      </view>
      <input v-model="form.contact.name" class="field" placeholder="联系人" />
      <input v-model="form.contact.phone" class="field" placeholder="联系电话" />
      <view class="action-row">
        <button class="secondary-button" @click="saveDraft">保存草稿</button>
        <button class="primary-button" @click="submit">提交审核</button>
      </view>
    </view>
  </view>
</template>

<script setup>
import { computed, reactive, ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { DEFAULT_CITY_CODE } from '../../common/constants'
import { getMerchantId, saveMerchantId } from '../../store/session'
import { listCityResourceTypes } from '../../api/city'
import { createResource, createResourceDraft, submitResource } from '../../api/resource'
import { chooseAndUploadImage } from '../../common/upload'

const resourceTypes = ref([])
const selectedTypeIndex = ref(0)
const publishTypeOptions = [
  { label: '发布库存', value: 'inventory' },
  { label: '发布货源', value: 'goods' },
  { label: '发布工厂产能', value: 'factory' },
  { label: '发布服务', value: 'service' },
]
const form = reactive({
  merchantId: '',
  cityCode: DEFAULT_CITY_CODE,
  typeCode: '',
  title: '',
  category: '',
  quantityText: '',
  priceText: '',
  description: '',
  attributes: {},
  tags: [],
  images: [],
  contact: {
    name: '',
    phone: '',
    wechat: '',
  },
})

const resourceTypeNames = computed(() => resourceTypes.value.map((item) => item.typeName))
const selectedTypeLabel = computed(() => {
  const current = resourceTypes.value[selectedTypeIndex.value] || {}
  return current.typeName || '请选择资源类型'
})
const canSubmit = computed(() => Boolean(form.typeCode && form.title.trim() && form.category.trim() && form.contact.name.trim() && form.contact.phone.trim()))
const publishReadyText = computed(() => (canSubmit.value ? '必填项已完整' : '标题、品类、联系人和联系电话为必填项'))

onLoad(async (options) => {
  // 发布页优先使用路由带入的商家 ID，其次使用我的页保存的商家，减少商家重复输入。
  form.merchantId = options.merchantId || getMerchantId()
  form.typeCode = options.typeCode || ''
  await loadResourceTypes()
})

async function loadResourceTypes() {
  const resp = await listCityResourceTypes(form.cityCode)
  resourceTypes.value = resp.items || []
  if (!resourceTypes.value.length) {
    form.typeCode = ''
    return
  }
  const matchIndex = resourceTypes.value.findIndex((item) => item.typeCode === form.typeCode)
  selectedTypeIndex.value = matchIndex >= 0 ? matchIndex : 0
  form.typeCode = resourceTypes.value[selectedTypeIndex.value].typeCode
}

function selectType(event) {
  selectedTypeIndex.value = Number(event.detail.value)
  const current = resourceTypes.value[selectedTypeIndex.value] || {}
  form.typeCode = current.typeCode || ''
}

function selectTypeByCode(typeCode) {
  const index = resourceTypes.value.findIndex((item) => item.typeCode === typeCode)
  if (index >= 0) {
    selectedTypeIndex.value = index
  }
  form.typeCode = typeCode
}

function openMyResources() {
  uni.navigateTo({ url: `/pages/my-resources/index?merchantId=${form.merchantId}` })
}

async function submit() {
  if (!validatePublishForm()) {
    return
  }
  saveMerchantId(form.merchantId)
  const result = await createResource({ ...form })
  if (result.id) {
    await submitResource(result.id, form.merchantId)
  }
  uni.showToast({ title: '已提交审核', icon: 'none' })
  uni.navigateTo({ url: '/pages/publish-success/index' })
}

async function saveDraft() {
  if (!validatePublishForm()) {
    return
  }
  saveMerchantId(form.merchantId)
  await createResourceDraft({ ...form })
  uni.showToast({ title: '草稿已保存', icon: 'none' })
  uni.navigateTo({ url: `/pages/my-resources/index?merchantId=${form.merchantId}` })
}

async function uploadResourceImage() {
  try {
    // 资源图片先上传到对象存储，业务接口只保存最终 CDN URL，避免后端转发大文件。
    const url = await chooseAndUploadImage('resource')
    form.images.push(url)
    uni.showToast({ title: '图片已上传', icon: 'none' })
  } catch (err) {
    uni.showToast({ title: err.message || '图片上传失败，请重试', icon: 'none' })
  }
}

function validatePublishForm() {
  if (!form.merchantId) {
    uni.showToast({ title: '请先选择商家', icon: 'none' })
    return false
  }
  if (!form.typeCode) {
    uni.showToast({ title: '请选择资源类型', icon: 'none' })
    return false
  }
  if (!form.title.trim()) {
    uni.showToast({ title: '请填写标题', icon: 'none' })
    return false
  }
  if (!form.category.trim()) {
    uni.showToast({ title: '请填写品类', icon: 'none' })
    return false
  }
  if (!form.contact.name.trim()) {
    uni.showToast({ title: '请填写联系人', icon: 'none' })
    return false
  }
  if (!form.contact.phone.trim()) {
    uni.showToast({ title: '请填写联系电话', icon: 'none' })
    return false
  }
  return true
}
</script>

<style lang="scss" scoped>
.publish-page {
  min-height: 100vh;
  padding: 24rpx;
  background: $wplink-bg;
}

.form-card {
  display: grid;
  gap: 18rpx;
  margin-bottom: 20rpx;
  padding: 24rpx;
  border-radius: 12rpx;
  background: $wplink-card;
}

.quota-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18rpx;
  margin-bottom: 20rpx;
  padding: 24rpx;
  border-radius: 12rpx;
  background: linear-gradient(135deg, $wplink-warning-soft, $wplink-primary-soft);
}

.quota-card button {
  flex: 0 0 auto;
  width: 148rpx;
  height: 68rpx;
  border-radius: 10rpx;
  background: $wplink-card;
  color: $wplink-primary;
  font-size: 26rpx;
  font-weight: 700;
}

.quota-label,
.effect-label {
  color: $wplink-muted;
  font-size: 24rpx;
}

.quota-title {
  display: block;
  margin: 8rpx 0;
  color: $wplink-primary;
  font-size: 36rpx;
  font-weight: 700;
}

.quota-desc {
  color: $wplink-muted;
  font-size: 26rpx;
  line-height: 1.5;
}

.publish-types {
  width: 100%;
  margin-bottom: 20rpx;
  white-space: nowrap;
}

.type-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 152rpx;
  height: 72rpx;
  margin-right: 12rpx;
  padding: 0 20rpx;
  border-radius: 10rpx;
  background: $wplink-card;
  color: #364152;
  font-size: 26rpx;
}

.type-button.active {
  background: $wplink-primary;
  color: $wplink-card;
}

.page-title {
  font-size: 36rpx;
  font-weight: 700;
}

.form-status {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16rpx;
  padding: 18rpx;
  border-radius: 10rpx;
  background: #f8fafc;
  min-width: 0;
}

.form-status text {
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 1.45;
}

.form-status strong {
  flex: 0 0 auto;
  color: $wplink-primary;
  font-size: 26rpx;
  line-height: 1.25;
  text-align: right;
}

.field,
.textarea {
  min-height: 80rpx;
  padding: 0 20rpx;
  border: 1rpx solid $wplink-line;
  border-radius: 10rpx;
}

.picker-field {
  display: flex;
  align-items: center;
  color: $wplink-primary;
}

.textarea {
  min-height: 160rpx;
  padding-top: 18rpx;
}

.action-row {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16rpx;
}

.image-list {
  display: grid;
  gap: 8rpx;
}

.effect-preview {
  display: grid;
  gap: 8rpx;
  padding: 18rpx;
  border-radius: 10rpx;
  background: #f8fafc;
}

.effect-value {
  color: $wplink-primary;
  font-size: 26rpx;
  font-weight: 700;
}

.image-url {
  color: $wplink-muted;
  font-size: 24rpx;
  word-break: break-all;
}

.secondary-button,
.primary-button {
  height: 88rpx;
  border-radius: 12rpx;
  line-height: 1.25;
}

.secondary-button {
  background: #edf2f7;
  color: #364152;
}

.primary-button {
  background: $wplink-primary;
  color: $wplink-card;
}
</style>
