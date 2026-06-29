<template>
  <view class="my-page">
    <view class="mine-head">
      <view class="avatar">衣</view>
      <view>
        <text class="mine-name">{{ mineName }}</text>
        <text class="mine-role">衣货通商家 · 商家管理员</text>
      </view>
    </view>

    <view class="profile-card">
      <text class="page-title">我的</text>
      <button class="secondary-button" @click="loginWithWechat">微信登录</button>
      <view class="sms-row">
        <input v-model="phone" class="field" type="number" placeholder="手机号" />
        <button class="sms-button" :disabled="smsSending || smsCountdown > 0" @click="sendSmsCodeForPhone">
          {{ smsCountdown > 0 ? `${smsCountdown}s` : '验证码' }}
        </button>
      </view>
      <input v-model="smsCode" class="field" type="number" placeholder="短信验证码" />
      <button class="secondary-button" :disabled="bindingPhone" @click="bindCurrentPhone">绑定手机号</button>
      <input v-model="userId" class="field" placeholder="用户 ID" />
      <input v-model="merchantId" class="field" placeholder="商家 ID" />
      <button class="primary-button" @click="saveIdentity">保存身份</button>
    </view>

    <view class="benefit-card">
      <text class="benefit-tag">权益提醒</text>
      <text class="benefit-title">{{ benefitTitle }}</text>
      <text class="benefit-desc">建议用于急清库存或空档产能资源，展示为“置顶”。</text>
      <button :disabled="!merchantId.trim()" @click="openPublish">去发布资源</button>
    </view>

    <view class="entitlement-card">
      <view class="section-head">
        <text class="section-title">我的权益</text>
        <button class="mini-button" :disabled="entitlementLoading || !merchantId.trim()" @click="loadEntitlements">刷新</button>
      </view>
      <view v-if="!merchantId.trim()" class="empty-state">保存商家 ID 后查看权益余量</view>
      <view v-else-if="entitlementLoading" class="empty-state">权益加载中</view>
      <view v-else class="entitlement-grid">
        <view v-for="row in entitlementRows" :key="row.type" class="entitlement-item">
          <text class="entitlement-label">{{ row.label }}</text>
          <text class="entitlement-value">{{ row.remaining }}</text>
          <text class="entitlement-meta">已用 {{ row.used }} / 总 {{ row.total }}</text>
        </view>
        <view class="entitlement-item">
          <text class="entitlement-label">置顶券</text>
          <text class="entitlement-value">{{ availableTopVoucherCount }}</text>
          <text class="entitlement-meta">可用于我的发布</text>
        </view>
      </view>
      <button class="secondary-button" :disabled="!merchantId.trim()" @click="openMyResources">去使用权益</button>
    </view>

    <view class="action-list">
      <view class="action-item" @click="openMerchantProfile">
        <text>商家资料</text>
        <text class="action-meta">入驻信息和主页介绍</text>
      </view>
      <view class="action-item" @click="openMyResources">
        <text>我的发布</text>
        <text class="action-meta">资源状态和效果数据</text>
      </view>
      <view class="action-item" @click="openMyDemands">
        <text>我的需求</text>
        <text class="action-meta">采购需求和处理进展</text>
      </view>
      <view class="action-item" @click="openFavorites">
        <text>收藏关注</text>
        <text class="action-meta">收藏资源、关注商家和保存搜索</text>
      </view>
      <view class="action-item" @click="openVerification">
        <text>商家认证</text>
        <text class="action-meta">认证状态和提交资料</text>
      </view>
      <view class="action-item" @click="openPublish">
        <text>发布资源</text>
        <text class="action-meta">新增库存、货源、工厂或服务</text>
      </view>
    </view>
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onLoad, onUnload } from '@dcloudio/uni-app'
import { DEFAULT_CITY_CODE } from '../../common/constants'
import { bindPhone, sendSmsCode, wechatLogin } from '../../api/auth'
import { getMerchantEntitlements, listTopVouchers } from '../../api/entitlement'
import { getMerchantId, getUserId, saveMerchantId, saveToken, saveUserId } from '../../store/session'

const userId = ref('')
const merchantId = ref('')
const phone = ref('')
const smsCode = ref('')
const entitlements = ref([])
const topVouchers = ref([])
const entitlementLoading = ref(false)
const smsSending = ref(false)
const bindingPhone = ref(false)
const smsCountdown = ref(0)
let smsTimer = 0

onLoad(() => {
  userId.value = getUserId()
  merchantId.value = getMerchantId()
  loadEntitlements()
})

onUnload(() => {
  clearSMSCountdown()
})

function saveIdentity() {
  if (userId.value.trim()) {
    saveUserId(userId.value.trim())
  }
  if (!merchantId.value.trim()) {
    uni.showToast({ title: '请填写商家 ID', icon: 'none' })
    return
  }
  saveMerchantId(merchantId.value.trim())
  loadEntitlements()
  uni.showToast({ title: '已保存身份', icon: 'none' })
}

async function loginWithWechat() {
  try {
    const code = await getWechatLoginCode()
    const resp = await wechatLogin({ code, defaultCityCode: DEFAULT_CITY_CODE })
    if (resp.token) {
      saveToken(resp.token)
    }
    const loginUser = resp.user || {}
    if (loginUser.id) {
      userId.value = loginUser.id
      saveUserId(loginUser.id)
    }
    uni.showToast({ title: '登录成功', icon: 'none' })
  } catch (err) {
    uni.showToast({ title: err.message || '登录失败，请稍后重试', icon: 'none' })
  }
}

async function sendSmsCodeForPhone() {
  const normalizedPhone = phone.value.trim()
  if (!normalizedPhone) {
    uni.showToast({ title: '请填写手机号', icon: 'none' })
    return
  }
  try {
    smsSending.value = true
    await sendSmsCode({ phone: normalizedPhone })
    uni.showToast({ title: '验证码已发送', icon: 'none' })
    startSMSCountdown()
  } catch (err) {
    uni.showToast({ title: err.message || '验证码发送失败', icon: 'none' })
  } finally {
    smsSending.value = false
  }
}

async function bindCurrentPhone() {
  const normalizedPhone = phone.value.trim()
  const normalizedCode = smsCode.value.trim()
  if (!normalizedPhone || !normalizedCode) {
    uni.showToast({ title: '请填写手机号和验证码', icon: 'none' })
    return
  }
  try {
    bindingPhone.value = true
    await bindPhone({ phone: normalizedPhone, smsCode: normalizedCode })
    uni.showToast({ title: '手机号已绑定', icon: 'none' })
  } catch (err) {
    uni.showToast({ title: err.message || '绑定失败，请稍后重试', icon: 'none' })
  } finally {
    bindingPhone.value = false
  }
}

function startSMSCountdown() {
  clearSMSCountdown()
  smsCountdown.value = 60
  smsTimer = setInterval(() => {
    smsCountdown.value -= 1
    if (smsCountdown.value <= 0) {
      clearSMSCountdown()
    }
  }, 1000)
}

function clearSMSCountdown() {
  if (smsTimer) {
    clearInterval(smsTimer)
    smsTimer = 0
  }
}

function getWechatLoginCode() {
  return new Promise((resolve) => {
    uni.login({
      provider: 'weixin',
      success: (res) => {
        resolve(res.code || localDevLoginCode())
      },
      // 本地 H5/模拟环境可能没有微信登录能力，使用开发 code 仍可完成后端链路验收。
      fail: () => {
        resolve(localDevLoginCode())
      },
    })
  })
}

function localDevLoginCode() {
  const key = 'wplink_dev_login_code'
  const existing = uni.getStorageSync(key)
  if (existing) return existing
  const code = `local-dev-${Date.now()}`
  uni.setStorageSync(key, code)
  return code
}

const entitlementRows = computed(() => [
  buildEntitlementRow(['publish_quota', 'posting_quota'], '发布额度'),
  buildEntitlementRow(['refresh_quota'], '刷新额度'),
])

const availableTopVoucherCount = computed(() => topVouchers.value.filter((item) => item.status === 'unused').length)
const mineName = computed(() => (merchantId.value ? '商家工作台' : '我的账号'))
const defaultBenefitTitle = '3 张置顶券本月可用'
const benefitTitle = computed(() => {
  const count = availableTopVoucherCount.value
  return count > 0 ? `${count} 张置顶券本月可用` : defaultBenefitTitle
})

function buildEntitlementRow(types, label) {
  const item = entitlements.value.find((entry) => types.includes(entry.type)) || {}
  return {
    type: types[0],
    label,
    total: Number(item.totalAmount || 0),
    used: Number(item.usedAmount || 0),
    remaining: Number(item.remainingAmount || 0),
  }
}

async function loadEntitlements() {
  const currentMerchantId = merchantId.value.trim()
  if (!currentMerchantId) {
    entitlements.value = []
    topVouchers.value = []
    return
  }
  try {
    entitlementLoading.value = true
    // 我的页是商家运营入口，进入时同步权益余量，避免商家只能在发布失败后才知道额度不足。
    const [entitlementResp, voucherResp] = await Promise.all([
      getMerchantEntitlements(currentMerchantId),
      listTopVouchers(currentMerchantId),
    ])
    entitlements.value = entitlementResp.items || []
    topVouchers.value = voucherResp.items || []
  } catch (err) {
    entitlements.value = []
    topVouchers.value = []
    uni.showToast({ title: err.message || '权益加载失败，请稍后重试', icon: 'none' })
  } finally {
    entitlementLoading.value = false
  }
}

function openMyResources() {
  uni.navigateTo({ url: `/pages/my-resources/index?merchantId=${merchantId.value}` })
}

function openMerchantProfile() {
  const query = merchantId.value ? `?merchantId=${merchantId.value}` : ''
  uni.navigateTo({ url: `/pages/merchant/profile${query}` })
}

function openMyDemands() {
  uni.navigateTo({ url: '/pages/my-demands/index' })
}

function openFavorites() {
  uni.navigateTo({ url: '/pages/favorites/index' })
}

function openVerification() {
  uni.navigateTo({ url: '/pages/verification/index' })
}

function openPublish() {
  uni.switchTab({ url: '/pages/publish/index' })
}
</script>

<style scoped>
.my-page {
  min-height: 100vh;
  padding: 24rpx;
  background: #f4f6f8;
}

.mine-head,
.profile-card,
.benefit-card,
.entitlement-card,
.action-list {
  display: grid;
  gap: 18rpx;
  margin-bottom: 20rpx;
  padding: 24rpx;
  border-radius: 12rpx;
  background: #ffffff;
}

.mine-head {
  grid-template-columns: 96rpx 1fr;
  align-items: center;
}

.avatar {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 88rpx;
  height: 88rpx;
  border-radius: 12rpx;
  background: #0f766e;
  color: #ffffff;
  font-size: 36rpx;
  font-weight: 700;
}

.mine-name {
  display: block;
  margin-bottom: 8rpx;
  color: #1f2933;
  font-size: 36rpx;
  font-weight: 700;
}

.mine-role {
  color: #697586;
  font-size: 26rpx;
}

.benefit-card {
  background: #fff7e6;
}

.benefit-tag {
  width: 128rpx;
  padding: 6rpx 12rpx;
  border-radius: 8rpx;
  background: #fff0cc;
  color: #b7791f;
  font-size: 24rpx;
  text-align: center;
}

.benefit-title {
  color: #1f2933;
  font-size: 34rpx;
  font-weight: 700;
}

.benefit-desc {
  color: #7c5a22;
  font-size: 26rpx;
  line-height: 1.5;
}

.benefit-card button {
  height: 80rpx;
  border-radius: 10rpx;
  background: #0f766e;
  color: #ffffff;
  font-size: 28rpx;
  font-weight: 700;
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

.sms-row {
  display: grid;
  grid-template-columns: 1fr 180rpx;
  gap: 14rpx;
  align-items: center;
}

.primary-button {
  height: 84rpx;
  border-radius: 12rpx;
  background: #0f766e;
  color: #ffffff;
}

.secondary-button {
  height: 84rpx;
  border: 1rpx solid #0f766e;
  border-radius: 12rpx;
  background: #ffffff;
  color: #0f766e;
}

.sms-button {
  height: 80rpx;
  border: 1rpx solid #0f766e;
  border-radius: 10rpx;
  background: #ffffff;
  color: #0f766e;
  font-size: 26rpx;
}

.section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16rpx;
}

.section-title {
  color: #1f2933;
  font-size: 32rpx;
  font-weight: 700;
}

.mini-button {
  width: 128rpx;
  height: 60rpx;
  border: 1rpx solid #d8dde6;
  border-radius: 10rpx;
  background: #ffffff;
  color: #364152;
  font-size: 24rpx;
  line-height: 60rpx;
}

.empty-state {
  padding: 18rpx 0;
  color: #697586;
  font-size: 26rpx;
}

.entitlement-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 14rpx;
}

.entitlement-item {
  display: grid;
  gap: 8rpx;
  min-height: 148rpx;
  padding: 18rpx;
  border: 1rpx solid #e3e8ef;
  border-radius: 10rpx;
  background: #f8fafc;
}

.entitlement-label {
  color: #364152;
  font-size: 24rpx;
}

.entitlement-value {
  color: #0f766e;
  font-size: 38rpx;
  font-weight: 700;
}

.entitlement-meta {
  color: #697586;
  font-size: 22rpx;
}

.action-item {
  display: grid;
  gap: 8rpx;
  padding: 18rpx 0;
  border-bottom: 1rpx solid #edf2f7;
  color: #1f2933;
  font-size: 32rpx;
}

.action-item:last-child {
  border-bottom: 0;
}

.action-meta {
  color: #697586;
  font-size: 26rpx;
}
</style>
