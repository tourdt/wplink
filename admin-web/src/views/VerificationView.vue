<template>
  <section>
    <div class="page-title">
      <h2>认证审核</h2>
    </div>
    <section class="panel billing-panel">
      <div class="billing-header">
        <div>
          <strong>认证收费设置</strong>
          <p>{{ billingSummary }}</p>
        </div>
        <el-button type="primary" plain @click="billingDrawerVisible = true">调整设置</el-button>
      </div>
    </section>
    <section class="panel">
      <div v-if="errorText" class="table-state table-state-error">
        <span>{{ errorText }}</span>
        <el-button type="danger" plain @click="loadRows">重试</el-button>
      </div>
      <el-table v-loading="loading" :data="rows" stripe empty-text="暂无待审核认证">
        <el-table-column prop="merchantName" label="商家" min-width="180" />
        <el-table-column label="营业主体" min-width="180">
          <template #default="{ row }">{{ displayValue(row.businessName) }}</template>
        </el-table-column>
        <el-table-column label="认证身份" width="140">
          <template #default="{ row }">{{ typeText[row.verificationType] || row.verificationType }}</template>
        </el-table-column>
        <el-table-column prop="submittedAt" label="提交时间" width="180" />
        <el-table-column label="操作" width="180">
          <template #default="{ row }">
            <el-button type="primary" link @click="approve(row)">通过</el-button>
            <el-button type="danger" link @click="openReject(row)">驳回</el-button>
            <el-button link @click="openMaterial(row)">查看资料</el-button>
          </template>
        </el-table-column>
      </el-table>
    </section>

    <el-dialog v-model="rejectVisible" title="驳回认证" width="420px">
      <el-input v-model="rejectReason" type="textarea" :rows="4" placeholder="请填写驳回原因" />
      <template #footer>
        <el-button @click="rejectVisible = false">取消</el-button>
        <el-button type="danger" :loading="submitting" @click="submitReject">确认驳回</el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="billingDrawerVisible" title="认证收费设置" size="460px">
      <el-form label-position="top">
        <el-form-item label="城市站">
          <el-input v-model.trim="billingForm.cityCode" />
        </el-form-item>
        <el-form-item label="认证收费">
          <el-switch v-model="billingForm.chargeEnabled" active-text="开启" inactive-text="关闭" />
        </el-form-item>
        <el-form-item label="认证费用">
          <el-input-number v-model="billingFeeYuan" :min="0" :step="10" :precision="2" />
        </el-form-item>
        <el-form-item label="限时免费">
          <el-switch v-model="billingForm.freeEnabled" active-text="开启" inactive-text="关闭" />
        </el-form-item>
        <el-form-item label="限免开始">
          <el-date-picker v-model="billingForm.freeStartAt" type="datetime" value-format="YYYY-MM-DDTHH:mm:ssZ" />
        </el-form-item>
        <el-form-item label="限免结束">
          <el-date-picker v-model="billingForm.freeEndAt" type="datetime" value-format="YYYY-MM-DDTHH:mm:ssZ" />
        </el-form-item>
        <el-form-item label="前端说明">
          <el-input v-model.trim="billingForm.notice" type="textarea" :rows="3" />
        </el-form-item>
        <div class="drawer-actions">
          <el-button @click="billingDrawerVisible = false">取消</el-button>
          <el-button type="primary" :loading="savingBilling" @click="saveBillingConfig">保存设置</el-button>
        </div>
      </el-form>
    </el-drawer>

    <el-drawer v-model="materialVisible" title="认证材料" size="620px">
      <div v-if="materialRow" class="material-drawer">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="商家">{{ materialRow.merchantName }}</el-descriptions-item>
          <el-descriptions-item label="营业主体">{{ displayValue(materialRow.businessName) }}</el-descriptions-item>
          <el-descriptions-item label="认证身份">{{ typeText[materialRow.verificationType] || materialRow.verificationType }}</el-descriptions-item>
          <el-descriptions-item label="状态">{{ materialRow.status }}</el-descriptions-item>
          <el-descriptions-item label="提交时间">{{ materialRow.submittedAt }}</el-descriptions-item>
        </el-descriptions>

        <section class="material-section">
          <h3>联系人和地址</h3>
          <el-descriptions :column="1" border>
            <el-descriptions-item v-for="item in materialInfoItems" :key="item.label" :label="item.label">
              {{ item.value }}
            </el-descriptions-item>
          </el-descriptions>
        </section>

        <section class="material-section">
          <h3>图片材料</h3>
          <div class="material-image-grid">
            <div v-for="item in materialImageItems" :key="item.label" class="material-image-item">
              <span class="material-image-label">{{ item.label }}</span>
              <el-image
                v-if="item.url"
                class="material-image"
                :src="item.url"
                fit="cover"
                :preview-src-list="materialImagePreviewList"
                preview-teleported
              />
              <div v-else class="material-empty">未提交</div>
            </div>
          </div>
        </section>
      </div>
    </el-drawer>
  </section>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getVerificationBillingConfig, listPendingVerifications, reviewVerification, updateVerificationBillingConfig } from '../api/verification'
import { verificationTypeText as typeText } from '../common/merchantIdentity'
import { useAuthStore } from '../stores/auth'

const rows = ref([])
const loading = ref(false)
const errorText = ref('')
const submitting = ref(false)
const rejectVisible = ref(false)
const rejectTarget = ref(null)
const rejectReason = ref('')
const materialVisible = ref(false)
const materialRow = ref(null)
const billingDrawerVisible = ref(false)
const savingBilling = ref(false)
const billingForm = reactive({
  cityCode: 'zhili',
  chargeEnabled: false,
  feeAmount: 0,
  currency: 'CNY',
  freeEnabled: false,
  freeStartAt: '',
  freeEndAt: '',
  notice: '',
})
const auth = useAuthStore()

const billingFeeYuan = computed({
  get: () => Number((billingForm.feeAmount / 100).toFixed(2)),
  set: (value) => {
    billingForm.feeAmount = Math.round(Number(value || 0) * 100)
  },
})

const billingSummary = computed(() => {
  if (!billingForm.chargeEnabled) return '当前关闭收费，认证审核通过后直接生效。'
  if (isFreeWindowActive()) return `当前限时免费，费用设置为 ${billingFeeText.value}，审核通过后直接生效。`
  return `当前收费 ${billingFeeText.value}，审核通过后用户需在线支付，支付成功后认证生效。`
})

const billingFeeText = computed(() => `¥${(billingForm.feeAmount / 100).toFixed(2)}`)
const materialInfoItems = computed(() => {
  const row = materialRow.value
  if (!row) return []
  return [
    { label: '统一社会信用代码', value: materialValue(row, 'socialCreditCode') },
    { label: '联系人姓名', value: materialValue(row, 'applicantName') },
    { label: '联系电话', value: materialValue(row, 'contactPhone') },
    { label: '联系微信', value: materialValue(row, 'contactWechat') },
    { label: '经营地址', value: materialValue(row, 'addressText') },
    { label: '资料真实承诺', value: materialValue(row, 'commitmentAccepted') === true ? '已承诺' : '未确认' },
  ].map((item) => ({ ...item, value: displayValue(item.value) }))
})
const materialImageItems = computed(() => {
  const row = materialRow.value
  if (!row) return []
  return [
    { label: '营业执照', url: row.licenseUrl },
    { label: '门头/场地', url: row.storefrontUrl },
    { label: '经营实拍', url: materialValue(row, 'sceneUrl') },
    { label: '授权证明', url: materialValue(row, 'authorizationUrl') },
    { label: '其他证明', url: materialValue(row, 'qualificationUrl') },
  ].map((item) => ({ ...item, url: materialUrl(item.url) }))
})
const materialImagePreviewList = computed(() => materialImageItems.value.map((item) => item.url).filter(Boolean))

onMounted(async () => {
  await Promise.all([loadRows(), loadBillingConfig()])
})

async function loadRows() {
  loading.value = true
  errorText.value = ''
  try {
    const resp = await listPendingVerifications({ page: 1, pageSize: 20 })
    rows.value = resp.items || []
  } catch {
    errorText.value = '认证审核列表加载失败，请重试'
  } finally {
    loading.value = false
  }
}

async function loadBillingConfig() {
  try {
    const config = await getVerificationBillingConfig({ cityCode: billingForm.cityCode })
    Object.assign(billingForm, normalizeBillingConfig(config))
  } catch {
    ElMessage.warning('认证收费设置加载失败')
  }
}

async function approve(row) {
  try {
    await ElMessageBox.confirm(approveConfirmText(row), '确认认证通过', {
      type: 'warning',
      confirmButtonText: '确认通过',
      cancelButtonText: '取消',
    })
  } catch {
    return
  }
  submitting.value = true
  try {
    await reviewVerification(row.id, { action: 'approve', reviewerId: currentOperatorId() })
    ElMessage.success(billingForm.chargeEnabled && !isFreeWindowActive() ? '资料已通过，等待用户支付' : '认证已通过')
    await loadRows()
  } finally {
    submitting.value = false
  }
}

async function saveBillingConfig() {
  if (billingForm.chargeEnabled && billingForm.feeAmount <= 0) {
    ElMessage.warning('开启收费时请填写认证费用')
    return
  }
  savingBilling.value = true
  try {
    const config = await updateVerificationBillingConfig({ ...billingForm })
    Object.assign(billingForm, normalizeBillingConfig(config))
    billingDrawerVisible.value = false
    ElMessage.success('认证收费设置已保存')
  } finally {
    savingBilling.value = false
  }
}

function openReject(row) {
  rejectTarget.value = row
  rejectReason.value = ''
  rejectVisible.value = true
}

async function submitReject() {
  if (!rejectReason.value.trim()) {
    ElMessage.warning('请填写驳回原因')
    return
  }
  submitting.value = true
  try {
    await reviewVerification(rejectTarget.value.id, { action: 'reject', reviewNote: rejectReason.value.trim(), reviewerId: currentOperatorId() })
    ElMessage.success('认证已驳回')
    rejectVisible.value = false
    await loadRows()
  } finally {
    submitting.value = false
  }
}

function openMaterial(row) {
  materialRow.value = row
  materialVisible.value = true
}

function materialValue(row, key) {
  return row?.materials?.[key]
}

function materialUrl(value) {
  if (typeof value !== 'string') return ''
  return value.trim()
}

function displayValue(value) {
  if (value === true) return '是'
  if (value === false) return '否'
  const text = String(value ?? '').trim()
  return text || '未提交'
}

function currentOperatorId() {
  return auth.user?.userId || ''
}

function normalizeBillingConfig(config = {}) {
  return {
    cityCode: config.cityCode || 'zhili',
    chargeEnabled: Boolean(config.chargeEnabled),
    feeAmount: Number(config.feeAmount || 0),
    currency: config.currency || 'CNY',
    freeEnabled: Boolean(config.freeEnabled),
    freeStartAt: config.freeStartAt || '',
    freeEndAt: config.freeEndAt || '',
    notice: config.notice || '',
  }
}

function approveConfirmText(row) {
  if (billingForm.chargeEnabled && !isFreeWindowActive()) {
    return `确认「${row.merchantName}」资料审核通过吗？通过后用户需在线支付 ${billingFeeText.value}，支付成功后认证才生效。`
  }
  return `确认通过「${row.merchantName}」的认证审核吗？`
}

function isFreeWindowActive() {
  if (!billingForm.freeEnabled) return false
  const now = Date.now()
  const start = billingForm.freeStartAt ? Date.parse(billingForm.freeStartAt) : null
  const end = billingForm.freeEndAt ? Date.parse(billingForm.freeEndAt) : null
  if (start && now < start) return false
  if (end && now > end) return false
  return true
}
</script>

<style scoped>
.billing-panel {
  margin-bottom: 16px;
  padding: 18px 20px;
}

.billing-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.billing-header p {
  margin: 6px 0 0;
  color: #697586;
}

.material-drawer {
  display: grid;
  gap: 18px;
}

.material-section {
  display: grid;
  gap: 12px;
}

.material-section h3 {
  margin: 0;
  color: #202939;
  font-size: 15px;
}

.material-image-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.material-image-item {
  display: grid;
  gap: 8px;
}

.material-image-label {
  color: #364152;
  font-size: 13px;
  font-weight: 600;
}

.material-image,
.material-empty {
  width: 100%;
  height: 150px;
  border-radius: 8px;
}

.material-image {
  border: 1px solid #e3e8ef;
}

.material-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px dashed #cdd5df;
  color: #697586;
  background: #f8fafc;
  font-size: 13px;
}
</style>
