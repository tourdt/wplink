<template>
  <view class="publish-page">
    <view class="form-card">
      <text class="page-title">发布资源</text>
      <picker :range="resourceTypeNames" :value="selectedTypeIndex" @change="selectType">
        <view class="field picker-field">{{ selectedTypeLabel }}</view>
      </picker>
      <input v-model="form.title" class="field" placeholder="标题" />
      <input v-model="form.category" class="field" placeholder="品类" />
      <input v-model="form.quantityText" class="field" placeholder="数量/产能" />
      <input v-model="form.priceText" class="field" placeholder="价格描述" />
      <textarea v-model="form.description" class="textarea" placeholder="资源描述" />
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
const selectedTypeLabel = computed(() => resourceTypes.value[selectedTypeIndex.value]?.typeName || '请选择资源类型')

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
  form.typeCode = resourceTypes.value[selectedTypeIndex.value]?.typeCode || ''
}

async function submit() {
  if (!validatePublishForm()) {
    return
  }
  saveMerchantId(form.merchantId)
  const result = await createResource({ ...form })
  if (result.id) {
    await submitResource(result.id)
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

<style scoped>
.publish-page {
  min-height: 100vh;
  padding: 24rpx;
  background: #f4f6f8;
}

.form-card {
  display: grid;
  gap: 18rpx;
  padding: 24rpx;
  border-radius: 12rpx;
  background: #ffffff;
}

.page-title {
  font-size: 36rpx;
  font-weight: 700;
}

.field,
.textarea {
  min-height: 80rpx;
  padding: 0 20rpx;
  border: 1rpx solid #d8dde6;
  border-radius: 10rpx;
}

.picker-field {
  display: flex;
  align-items: center;
  color: #1f2933;
}

.textarea {
  min-height: 160rpx;
  padding-top: 18rpx;
}

.action-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16rpx;
}

.image-list {
  display: grid;
  gap: 8rpx;
}

.image-url {
  color: #697586;
  font-size: 24rpx;
  word-break: break-all;
}

.secondary-button,
.primary-button {
  height: 88rpx;
  border-radius: 12rpx;
}

.secondary-button {
  background: #edf2f7;
  color: #364152;
}

.primary-button {
  background: #0f766e;
  color: #ffffff;
}
</style>
