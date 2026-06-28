<template>
  <section>
    <div class="page-title">
      <h2>人工撮合</h2>
      <el-button type="primary" @click="openCreate">创建撮合单</el-button>
    </div>

    <section class="panel">
      <el-form :inline="true" class="filter-bar">
        <el-form-item label="状态">
          <el-select v-model="filters.status" placeholder="全部" style="width: 150px">
            <el-option label="全部" value="" />
            <el-option label="待处理" value="open" />
            <el-option label="已联系" value="contacted" />
            <el-option label="已成功" value="succeeded" />
            <el-option label="未成功" value="failed" />
            <el-option label="已关闭" value="closed" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="loadRows(1)">查询</el-button>
        </el-form-item>
      </el-form>

      <div v-if="errorText" class="table-state table-state-error">
        <span>{{ errorText }}</span>
        <el-button type="danger" plain @click="loadRows(1)">重试</el-button>
      </div>
      <el-table v-loading="loading" :data="rows" stripe empty-text="暂无撮合记录">
        <el-table-column prop="demandTitle" label="采购需求" min-width="220" />
        <el-table-column label="状态" width="120">
          <template #default="{ row }">
            <el-tag :type="statusTagType[row.status] || 'info'">
              {{ statusText[row.status] || row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="resourceCount" label="候选资源" width="100" />
        <el-table-column prop="participantCount" label="参与商家" width="100" />
        <el-table-column prop="resultNote" label="结果说明" min-width="180" show-overflow-tooltip />
        <el-table-column prop="createdAt" label="创建时间" width="180" />
        <el-table-column label="操作" width="260">
          <template #default="{ row }">
            <el-button type="primary" link @click="openStatus(row)">更新状态</el-button>
            <el-button link @click="openAppend(row, 'resource')">加资源</el-button>
            <el-button link @click="openAppend(row, 'participant')">加商家</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination-row">
        <el-pagination
          v-model:current-page="pagination.page"
          :page-size="pagination.pageSize"
          layout="total, prev, pager, next"
          :total="pagination.total"
          @current-change="loadRows"
        />
      </div>
    </section>

    <el-drawer v-model="createVisible" title="创建撮合单" size="480px">
      <el-form label-position="top">
        <el-form-item label="采购需求 ID">
          <el-input v-model="createForm.purchaseDemandId" />
        </el-form-item>
        <el-form-item label="候选资源 ID">
          <el-input v-model="createForm.resourceIdsText" type="textarea" :rows="3" placeholder="多个 ID 用逗号或换行分隔" />
        </el-form-item>
        <el-form-item label="参与商家 ID">
          <el-input v-model="createForm.participantMerchantIdsText" type="textarea" :rows="3" placeholder="多个 ID 用逗号或换行分隔" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="createForm.resultNote" type="textarea" :rows="3" />
        </el-form-item>
        <div class="drawer-actions">
          <el-button @click="createVisible = false">取消</el-button>
          <el-button type="primary" :loading="saving" @click="submitCreate">保存</el-button>
        </div>
      </el-form>
    </el-drawer>

    <el-dialog v-model="statusVisible" title="更新撮合状态" width="420px">
      <el-form label-position="top">
        <el-form-item label="状态">
          <el-select v-model="statusForm.status" style="width: 100%">
            <el-option label="待处理" value="open" />
            <el-option label="已联系" value="contacted" />
            <el-option label="已成功" value="succeeded" />
            <el-option label="未成功" value="failed" />
            <el-option label="已关闭" value="closed" />
          </el-select>
        </el-form-item>
        <el-form-item label="结果说明">
          <el-input v-model="statusForm.resultNote" type="textarea" :rows="3" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="statusVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitStatus">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="appendVisible" :title="appendTitle" width="440px">
      <el-form label-position="top">
        <el-form-item :label="appendMode === 'resource' ? '资源 ID' : '商家 ID'">
          <el-input v-model="appendForm.idsText" type="textarea" :rows="4" placeholder="多个 ID 用逗号或换行分隔" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="appendVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitAppend">保存</el-button>
      </template>
    </el-dialog>
  </section>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  addMatchCaseParticipants,
  addMatchCaseResources,
  createMatchCase,
  listMatchCases,
  updateMatchCaseStatus,
} from '../api/match'
import { useAuthStore } from '../stores/auth'

const statusText = {
  open: '待处理',
  contacted: '已联系',
  succeeded: '已成功',
  failed: '未成功',
  closed: '已关闭',
}
const statusTagType = {
  open: 'warning',
  contacted: 'primary',
  succeeded: 'success',
  failed: 'danger',
  closed: 'info',
}

const filters = reactive({ status: '' })
const pagination = reactive({ page: 1, pageSize: 20, total: 0 })
const rows = ref([])
const loading = ref(false)
const errorText = ref('')
const saving = ref(false)
const createVisible = ref(false)
const statusVisible = ref(false)
const appendVisible = ref(false)
const currentCase = ref(null)
const appendMode = ref('resource')
const auth = useAuthStore()
const createForm = reactive({
  purchaseDemandId: '',
  resourceIdsText: '',
  participantMerchantIdsText: '',
  resultNote: '',
})
const statusForm = reactive({ status: 'contacted', resultNote: '' })
const appendForm = reactive({ idsText: '' })
const appendTitle = computed(() => (appendMode.value === 'resource' ? '追加候选资源' : '追加参与商家'))

onMounted(loadRows)

async function loadRows(page = pagination.page) {
  pagination.page = page
  loading.value = true
  errorText.value = ''
  try {
    const resp = await listMatchCases({
      status: filters.status,
      page: pagination.page,
      pageSize: pagination.pageSize,
    })
    rows.value = resp.items || []
    pagination.total = resp.total || 0
  } catch {
    errorText.value = '撮合记录加载失败，请重试'
  } finally {
    loading.value = false
  }
}

function openCreate() {
  createForm.purchaseDemandId = ''
  createForm.resourceIdsText = ''
  createForm.participantMerchantIdsText = ''
  createForm.resultNote = ''
  createVisible.value = true
}

async function submitCreate() {
  if (!createForm.purchaseDemandId.trim()) {
    ElMessage.warning('请填写采购需求 ID')
    return
  }
  try {
    await ElMessageBox.confirm('确认基于当前采购需求创建撮合单吗？', '确认创建撮合单', {
      type: 'warning',
      confirmButtonText: '确认创建',
      cancelButtonText: '取消',
    })
  } catch {
    return
  }
  saving.value = true
  try {
    await createMatchCase({
      operatorId: currentOperatorId(),
      purchaseDemandId: createForm.purchaseDemandId.trim(),
      resourceIds: parseIds(createForm.resourceIdsText),
      participantMerchantIds: parseIds(createForm.participantMerchantIdsText),
      resultNote: createForm.resultNote.trim(),
    })
    ElMessage.success('撮合单已创建')
    createVisible.value = false
    await loadRows(1)
  } finally {
    saving.value = false
  }
}

function openStatus(row) {
  currentCase.value = row
  statusForm.status = row.status || 'contacted'
  statusForm.resultNote = row.resultNote || ''
  statusVisible.value = true
}

async function submitStatus() {
  if ((statusForm.status === 'succeeded' || statusForm.status === 'failed') && !statusForm.resultNote.trim()) {
    ElMessage.warning('成功或失败时请填写结果说明')
    return
  }
  try {
    await ElMessageBox.confirm(`确认将撮合单更新为「${statusText[statusForm.status] || statusForm.status}」吗？`, '确认更新撮合状态', {
      type: statusForm.status === 'failed' || statusForm.status === 'closed' ? 'warning' : 'info',
      confirmButtonText: '确认更新',
      cancelButtonText: '取消',
    })
  } catch {
    return
  }
  saving.value = true
  try {
    await updateMatchCaseStatus(currentCase.value.id, {
      operatorId: currentOperatorId(),
      status: statusForm.status,
      resultNote: statusForm.resultNote.trim(),
    })
    ElMessage.success('撮合状态已更新')
    statusVisible.value = false
    await loadRows()
  } finally {
    saving.value = false
  }
}

function openAppend(row, mode) {
  currentCase.value = row
  appendMode.value = mode
  appendForm.idsText = ''
  appendVisible.value = true
}

async function submitAppend() {
  const ids = parseIds(appendForm.idsText)
  if (!ids.length) {
    ElMessage.warning('请填写要追加的 ID')
    return
  }
  try {
    await ElMessageBox.confirm(`确认追加 ${ids.length} 条${appendMode.value === 'resource' ? '候选资源' : '参与商家'}吗？`, '确认追加', {
      type: 'warning',
      confirmButtonText: '确认追加',
      cancelButtonText: '取消',
    })
  } catch {
    return
  }
  saving.value = true
  try {
    if (appendMode.value === 'resource') {
      await addMatchCaseResources(currentCase.value.id, { operatorId: currentOperatorId(), resourceIds: ids })
    } else {
      await addMatchCaseParticipants(currentCase.value.id, { operatorId: currentOperatorId(), participantMerchantIds: ids })
    }
    ElMessage.success('已追加')
    appendVisible.value = false
    await loadRows()
  } finally {
    saving.value = false
  }
}

function parseIds(value) {
  return value
    .split(/[\s,，]+/)
    .map((item) => item.trim())
    .filter(Boolean)
}

function currentOperatorId() {
  return auth.user?.userId || ''
}
</script>
