<template>
  <view class="demand-page">
    <view class="form-card">
      <text class="page-title">提交采购需求</text>
      <input v-model="form.title" class="field" placeholder="需求标题" />
      <input v-model="form.category" class="field" placeholder="品类" />
      <input v-model="form.quantityRequirement.quantity" class="field" type="number" placeholder="数量" />
      <input v-model="form.contact.name" class="field" placeholder="联系人" />
      <input v-model="form.contact.phone" class="field" placeholder="联系电话" />
      <input v-model="form.contact.wechat" class="field" placeholder="微信号" />
      <button class="primary-button" @click="submit">提交需求</button>
    </view>
  </view>
</template>

<script setup>
import { reactive } from 'vue'
import { DEFAULT_CITY_CODE } from '../../common/constants'
import { createDemand } from '../../api/demand'
import { getUserId } from '../../store/session'

const form = reactive({
  cityCode: DEFAULT_CITY_CODE,
  demandType: 'inventory',
  title: '',
  category: '',
  priceRange: {},
  quantityRequirement: {
    quantity: '',
    unit: '件',
  },
  attributes: {},
  contact: {
    name: '',
    phone: '',
    wechat: '',
  },
})

async function submit() {
  const userId = getUserId()
  if (!userId) {
    uni.showToast({ title: '请先在我的页保存用户 ID', icon: 'none' })
    return
  }
  if (!form.title || !form.category || !form.contact.name || !form.contact.phone) {
    uni.showToast({ title: '请补充需求和联系方式', icon: 'none' })
    return
  }
  // 采购需求必须绑定发布人，后续“我的需求”和消息触达都依赖该用户 ID。
  await createDemand({
    ...form,
    userId,
    quantityRequirement: {
      quantity: Number(form.quantityRequirement.quantity) || 0,
      unit: form.quantityRequirement.unit,
    },
  })
  uni.showToast({ title: '需求已提交', icon: 'none' })
  uni.navigateTo({ url: '/pages/demand-success/index' })
}
</script>

<style scoped>
.demand-page {
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
  color: #1f2933;
  font-size: 36rpx;
  font-weight: 700;
}

.field {
  min-height: 80rpx;
  padding: 0 20rpx;
  border: 1rpx solid #d8dde6;
  border-radius: 10rpx;
}

.primary-button {
  height: 88rpx;
  border-radius: 12rpx;
  background: #0f766e;
  color: #ffffff;
}
</style>
