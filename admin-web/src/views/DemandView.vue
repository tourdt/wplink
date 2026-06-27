<template>
  <section>
    <div class="page-title">
      <h2>采购需求</h2>
    </div>

    <section class="panel">
      <el-form :inline="true" class="filter-bar">
        <el-form-item label="城市站">
          <el-select v-model="filters.cityCode" style="width: 140px">
            <el-option label="织里" value="zhili" />
          </el-select>
        </el-form-item>
        <el-form-item label="需求类型">
          <el-select v-model="filters.demandType" placeholder="全部" style="width: 150px">
            <el-option label="全部" value="" />
            <el-option label="找库存" value="inventory" />
            <el-option label="找货源" value="goods" />
            <el-option label="找工厂" value="factory" />
            <el-option label="找服务" value="service" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="filters.status" placeholder="全部" style="width: 130px">
            <el-option label="全部" value="" />
            <el-option label="待撮合" value="pending" />
            <el-option label="跟进中" value="matching" />
            <el-option label="已关闭" value="closed" />
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
      <el-table v-loading="loading" :data="rows" stripe empty-text="暂无采购需求">
        <el-table-column prop="title" label="需求" min-width="220" />
        <el-table-column label="类型" width="120">
          <template #default="{ row }">{{ demandTypeText[row.demandType] || row.demandType }}</template>
        </el-table-column>
        <el-table-column prop="category" label="品类" width="120" />
        <el-table-column prop="contactName" label="提交人" width="140" />
        <el-table-column label="状态" width="120">
          <template #default="{ row }">
            <el-tag :type="statusTagType[row.status] || 'info'">
              {{ statusText[row.status] || row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" label="提交时间" width="180" />
        <el-table-column label="操作" width="160">
          <template #default="{ row }">
            <el-button type="primary" link @click="openDetail(row)">查看</el-button>
            <el-button link @click="markMatching(row)">跟进</el-button>
            <el-button type="warning" link @click="closeDemand(row)">关闭</el-button>
          </template>
        </el-table-column>
      </el-table>
    </section>

    <el-drawer v-model="detailVisible" title="需求详情" size="520px">
      <el-descriptions v-if="detail" :column="1" border>
        <el-descriptions-item label="标题">{{ detail.title }}</el-descriptions-item>
        <el-descriptions-item label="类型">{{ demandTypeText[detail.demandType] || detail.demandType }}</el-descriptions-item>
        <el-descriptions-item label="品类">{{ detail.category }}</el-descriptions-item>
        <el-descriptions-item label="状态">{{ statusText[detail.status] || detail.status }}</el-descriptions-item>
        <el-descriptions-item label="联系人">{{ detail.contact?.name }}</el-descriptions-item>
        <el-descriptions-item label="电话">{{ detail.contact?.phone }}</el-descriptions-item>
        <el-descriptions-item label="微信">{{ detail.contact?.wechat }}</el-descriptions-item>
        <el-descriptions-item label="数量要求">{{ formatObject(detail.quantityRequirement) }}</el-descriptions-item>
        <el-descriptions-item label="价格范围">{{ formatObject(detail.priceRange) }}</el-descriptions-item>
        <el-descriptions-item label="补充属性">{{ formatObject(detail.attributes) }}</el-descriptions-item>
      </el-descriptions>
    </el-drawer>
  </section>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getDemand, listDemands, updateDemandStatus } from '../api/demand'

const demandTypeText = {
  inventory: '找库存',
  goods: '找货源',
  factory: '找工厂',
  service: '找服务',
}
const statusText = {
  pending: '待撮合',
  matching: '跟进中',
  closed: '已关闭',
}
const statusTagType = {
  pending: 'warning',
  matching: 'primary',
  closed: 'info',
}

const filters = reactive({
  cityCode: 'zhili',
  demandType: '',
  status: '',
})
const rows = ref([])
const detail = ref(null)
const loading = ref(false)
const errorText = ref('')
const detailVisible = ref(false)

onMounted(loadRows)

async function loadRows() {
  loading.value = true
  errorText.value = ''
  try {
    const resp = await listDemands({ ...filters, page: 1, pageSize: 20 })
    rows.value = resp.items || []
  } catch {
    errorText.value = '采购需求列表加载失败，请重试'
  } finally {
    loading.value = false
  }
}

async function openDetail(row) {
  detail.value = await getDemand(row.id)
  detailVisible.value = true
}

async function markMatching(row) {
  await updateStatus(row, 'matching', '需求已标记为跟进中')
}

async function closeDemand(row) {
  await updateStatus(row, 'closed', '需求已关闭')
}

async function updateStatus(row, status, message) {
  try {
    await ElMessageBox.confirm(`确认将「${row.title}」更新为「${statusText[status] || status}」吗？`, '确认更新需求状态', {
      type: status === 'closed' ? 'warning' : 'info',
      confirmButtonText: '确认更新',
      cancelButtonText: '取消',
    })
  } catch {
    return
  }
  await updateDemandStatus(row.id, { status })
  ElMessage.success(message)
  await loadRows()
}

function formatObject(value) {
  if (!value || !Object.keys(value).length) {
    return '-'
  }
  return Object.entries(value)
    .map(([key, val]) => `${key}: ${val}`)
    .join('，')
}
</script>
