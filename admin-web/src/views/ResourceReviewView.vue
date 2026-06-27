<template>
  <section>
    <div class="page-title">
      <h2>资源审核</h2>
      <el-button type="primary">代发资源</el-button>
    </div>

    <section class="panel">
      <el-form :inline="true" class="filter-bar">
        <el-form-item label="城市站">
          <el-select v-model="filters.cityCode" style="width: 140px">
            <el-option label="织里" value="zhili" />
          </el-select>
        </el-form-item>
        <el-form-item label="资源类型">
          <el-select v-model="filters.typeCode" placeholder="全部" style="width: 160px">
            <el-option label="全部" value="" />
            <el-option label="库存" value="inventory" />
            <el-option label="货源" value="goods" />
            <el-option label="工厂产能" value="factory" />
            <el-option label="订单需求" value="order" />
            <el-option label="招聘" value="job" />
            <el-option label="出租" value="rental" />
            <el-option label="服务" value="service" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="loadRows">查询</el-button>
        </el-form-item>
      </el-form>

      <div v-if="errorText" class="table-state table-state-error">
        <span>{{ errorText }}</span>
        <el-button type="danger" plain @click="loadRows">重试</el-button>
      </div>
      <el-table v-loading="loading" :data="rows" stripe empty-text="暂无待审核资源">
        <el-table-column prop="title" label="资源标题" min-width="220" />
        <el-table-column label="类型" width="120">
          <template #default="{ row }">{{ typeText[row.typeCode] || row.typeCode }}</template>
        </el-table-column>
        <el-table-column prop="merchantName" label="商家" width="180" />
        <el-table-column prop="createdAt" label="提交时间" width="180" />
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="approve(row)">通过</el-button>
            <el-button type="danger" link @click="openReject(row)">驳回</el-button>
            <el-button type="warning" link @click="openTakeDown(row)">下架</el-button>
            <el-button link>详情</el-button>
          </template>
        </el-table-column>
      </el-table>
    </section>

    <el-dialog v-model="reasonVisible" :title="reasonAction === 'reject' ? '驳回资源' : '下架资源'" width="420px">
      <el-input v-model="reasonText" type="textarea" :rows="4" placeholder="请填写处理原因" />
      <template #footer>
        <el-button @click="reasonVisible = false">取消</el-button>
        <el-button :type="reasonAction === 'reject' ? 'danger' : 'warning'" :loading="submitting" @click="submitReasonAction">
          确认
        </el-button>
      </template>
    </el-dialog>
  </section>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listPendingResources, reviewResource } from '../api/resource'

const typeText = {
  inventory: '库存',
  goods: '货源',
  factory: '工厂产能',
  order: '订单需求',
  job: '招聘',
  rental: '出租',
  service: '服务',
}

const filters = reactive({
  cityCode: 'zhili',
  typeCode: '',
})
const rows = ref([])
const loading = ref(false)
const errorText = ref('')
const submitting = ref(false)
const reasonVisible = ref(false)
const reasonTarget = ref(null)
const reasonAction = ref('reject')
const reasonText = ref('')

onMounted(loadRows)

async function loadRows() {
  loading.value = true
  errorText.value = ''
  try {
    const resp = await listPendingResources({ ...filters, page: 1, pageSize: 20 })
    rows.value = resp.items || []
  } catch {
    errorText.value = '资源审核列表加载失败，请重试'
  } finally {
    loading.value = false
  }
}

async function approve(row) {
  try {
    await ElMessageBox.confirm(`确认通过「${row.title}」的资源审核吗？`, '确认审核通过', {
      type: 'warning',
      confirmButtonText: '确认通过',
      cancelButtonText: '取消',
    })
  } catch {
    return
  }
  submitting.value = true
  try {
    await reviewResource(row.id, { action: 'approve' })
    ElMessage.success('资源已审核通过')
    await loadRows()
  } finally {
    submitting.value = false
  }
}

function openReject(row) {
  reasonTarget.value = row
  reasonAction.value = 'reject'
  reasonText.value = ''
  reasonVisible.value = true
}

function openTakeDown(row) {
  reasonTarget.value = row
  reasonAction.value = 'take_down'
  reasonText.value = ''
  reasonVisible.value = true
}

async function submitReasonAction() {
  if (!reasonText.value.trim()) {
    ElMessage.warning('请填写处理原因')
    return
  }
  submitting.value = true
  try {
    await reviewResource(reasonTarget.value.id, { action: reasonAction.value, reason: reasonText.value.trim() })
    ElMessage.success(reasonAction.value === 'reject' ? '资源已驳回' : '资源已下架')
    reasonVisible.value = false
    await loadRows()
  } finally {
    submitting.value = false
  }
}
</script>
