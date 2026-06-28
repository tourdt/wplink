<template>
  <section>
    <div class="page-title">
      <h2>认证审核</h2>
    </div>
    <section class="panel">
      <div v-if="errorText" class="table-state table-state-error">
        <span>{{ errorText }}</span>
        <el-button type="danger" plain @click="loadRows">重试</el-button>
      </div>
      <el-table v-loading="loading" :data="rows" stripe empty-text="暂无待审核认证">
        <el-table-column prop="merchantName" label="商家" min-width="180" />
        <el-table-column label="认证类型" width="140">
          <template #default="{ row }">{{ typeText[row.verificationType] || row.verificationType }}</template>
        </el-table-column>
        <el-table-column prop="submittedAt" label="提交时间" width="180" />
        <el-table-column label="操作" width="180">
          <template #default="{ row }">
            <el-button type="primary" link @click="approve(row)">通过</el-button>
            <el-button type="danger" link @click="openReject(row)">驳回</el-button>
            <el-button link @click="openMaterial(row)">材料</el-button>
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

    <el-drawer v-model="materialVisible" title="认证材料" size="420px">
      <el-descriptions v-if="materialRow" :column="1" border>
        <el-descriptions-item label="商家">{{ materialRow.merchantName }}</el-descriptions-item>
        <el-descriptions-item label="认证类型">{{ typeText[materialRow.verificationType] || materialRow.verificationType }}</el-descriptions-item>
        <el-descriptions-item label="状态">{{ materialRow.status }}</el-descriptions-item>
        <el-descriptions-item label="提交时间">{{ materialRow.submittedAt }}</el-descriptions-item>
      </el-descriptions>
    </el-drawer>
  </section>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listPendingVerifications, reviewVerification } from '../api/verification'
import { useAuthStore } from '../stores/auth'

const typeText = {
  factory: '工厂认证',
  stall: '档口认证',
  stockist: '库存商认证',
  service_provider: '服务商认证',
}

const rows = ref([])
const loading = ref(false)
const errorText = ref('')
const submitting = ref(false)
const rejectVisible = ref(false)
const rejectTarget = ref(null)
const rejectReason = ref('')
const materialVisible = ref(false)
const materialRow = ref(null)
const auth = useAuthStore()

onMounted(loadRows)

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

async function approve(row) {
  try {
    await ElMessageBox.confirm(`确认通过「${row.merchantName}」的认证审核吗？`, '确认认证通过', {
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
    ElMessage.success('认证已通过')
    await loadRows()
  } finally {
    submitting.value = false
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

function currentOperatorId() {
  return auth.user?.userId || ''
}
</script>
