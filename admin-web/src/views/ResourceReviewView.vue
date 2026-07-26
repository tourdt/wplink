<template>
  <section>
    <div class="page-title">
      <h2>供需信息审核</h2>
      <el-button type="primary" @click="openProxyCreate">代发供需信息</el-button>
    </div>

    <section class="panel">
      <el-form :inline="true" class="filter-bar">
        <el-form-item label="城市站">
          <el-select v-model="filters.cityCode" style="width: 140px" @change="handleCityChange">
            <el-option v-for="station in cityStationOptions" :key="station.value" :label="station.label" :value="station.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="供需类型">
          <el-select v-model="filters.typeCode" placeholder="全部" style="width: 160px">
            <el-option label="全部" value="" />
            <el-option v-for="item in resourceTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
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
      <el-table v-loading="loading" :data="rows" stripe empty-text="暂无待审核供需信息">
        <el-table-column prop="title" label="信息标题" min-width="220" />
        <el-table-column label="类型" width="120">
          <template #default="{ row }">{{ typeText[row.typeCode] || row.typeCode }}</template>
        </el-table-column>
        <el-table-column prop="merchantName" label="商家" width="180" />
        <el-table-column prop="createdAt" label="提交时间" width="180" />
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="approve(row)">通过</el-button>
            <el-button type="danger" link @click="openReject(row)">驳回</el-button>
            <el-button link @click="openRowDetail(row)">详情</el-button>
          </template>
        </el-table-column>
      </el-table>
    </section>

    <el-dialog v-model="reasonVisible" title="驳回供需信息" width="420px">
      <el-input v-model="reasonText" type="textarea" :rows="4" placeholder="请填写处理原因" />
      <template #footer>
        <el-button @click="reasonVisible = false">取消</el-button>
        <el-button type="danger" :loading="submitting" @click="submitReasonAction">
          确认
        </el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="proxyVisible" title="代发供需信息" size="520px">
      <el-form label-position="top">
        <el-form-item label="商家 ID">
          <el-input v-model.trim="proxyForm.merchantId" />
        </el-form-item>
        <el-form-item label="供需类型">
          <el-select v-model="proxyForm.typeCode">
            <el-option v-for="(label, value) in typeText" :key="value" :label="label" :value="value" />
          </el-select>
        </el-form-item>
        <el-form-item label="标题">
          <el-input v-model.trim="proxyForm.title" />
        </el-form-item>
        <el-form-item label="品类">
          <el-input v-model.trim="proxyForm.category" />
        </el-form-item>
        <el-form-item label="数量/产能">
          <el-input v-model.trim="proxyForm.quantityText" />
        </el-form-item>
        <el-form-item label="价格">
          <el-input v-model.trim="proxyForm.priceText" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="proxyForm.description" type="textarea" :rows="3" />
        </el-form-item>
        <el-form-item label="联系人">
          <el-input v-model.trim="proxyForm.contact.name" />
        </el-form-item>
        <el-form-item label="联系电话">
          <el-input v-model.trim="proxyForm.contact.phone" />
        </el-form-item>
        <div class="drawer-actions">
          <el-button @click="proxyVisible = false">取消</el-button>
          <el-button type="primary" :loading="savingProxy" @click="submitProxyCreate">提交审核</el-button>
        </div>
      </el-form>
    </el-drawer>

    <el-drawer v-model="detailVisible" title="供需信息详情" size="420px">
      <el-descriptions v-if="detailRow" :column="1" border>
        <el-descriptions-item label="信息标题">{{ detailRow.title }}</el-descriptions-item>
        <el-descriptions-item label="供需类型">{{ typeText[detailRow.typeCode] || detailRow.typeCode }}</el-descriptions-item>
        <el-descriptions-item label="商家">{{ detailRow.merchantName }}</el-descriptions-item>
        <el-descriptions-item label="提交时间">{{ detailRow.createdAt }}</el-descriptions-item>
      </el-descriptions>
    </el-drawer>
  </section>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from '../plugins/elementPlus'
import { createResource, listPendingResources, reviewResource } from '../api/resource'
import { listCityResourceTypes } from '../api/city'
import { cityStationOptions, defaultCityCode } from '../common/cityStations'
import { useAuthStore } from '../stores/auth'

const typeText = reactive({})
const resourceTypeOptions = ref([])

const filters = reactive({
  cityCode: defaultCityCode,
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
const proxyVisible = ref(false)
const savingProxy = ref(false)
const detailVisible = ref(false)
const detailRow = ref(null)
const proxyForm = reactive(defaultProxyForm())
const auth = useAuthStore()

onMounted(async () => {
  await loadResourceTypes()
  await loadRows()
})

async function loadResourceTypes() {
  try {
    const resp = await listCityResourceTypes(filters.cityCode)
    resourceTypeOptions.value = (resp.items || []).map((item) => ({
      value: item.typeCode,
      label: item.typeName,
    }))
    Object.keys(typeText).forEach((key) => delete typeText[key])
    resourceTypeOptions.value.forEach((item) => {
      typeText[item.value] = item.label
    })
  } catch {
    resourceTypeOptions.value = []
  }
}

async function handleCityChange() {
  filters.typeCode = ''
  await loadResourceTypes()
}

async function loadRows() {
  loading.value = true
  errorText.value = ''
  try {
    const resp = await listPendingResources({ ...filters, page: 1, pageSize: 20 })
    rows.value = resp.items || []
  } catch {
    errorText.value = '供需信息审核列表加载失败，请重试'
  } finally {
    loading.value = false
  }
}

async function approve(row) {
  try {
    await ElMessageBox.confirm(`确认通过「${row.title}」的供需信息审核吗？`, '确认审核通过', {
      type: 'warning',
      confirmButtonText: '确认通过',
      cancelButtonText: '取消',
    })
  } catch {
    return
  }
  submitting.value = true
  try {
    await reviewResource(row.id, { action: 'approve', reviewerId: currentOperatorId() })
    ElMessage.success('供需信息已审核通过')
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

async function submitReasonAction() {
  if (!reasonText.value.trim()) {
    ElMessage.warning('请填写处理原因')
    return
  }
  submitting.value = true
  try {
    await reviewResource(reasonTarget.value.id, { action: reasonAction.value, reason: reasonText.value.trim(), reviewerId: currentOperatorId() })
    ElMessage.success('供需信息已驳回')
    reasonVisible.value = false
    await loadRows()
  } finally {
    submitting.value = false
  }
}

function openProxyCreate() {
  Object.assign(proxyForm, defaultProxyForm())
  proxyVisible.value = true
}

async function submitProxyCreate() {
  if (!proxyForm.merchantId || !proxyForm.title || !proxyForm.category || !proxyForm.contact.name || !proxyForm.contact.phone) {
    ElMessage.warning('请补充商家、标题、品类和联系方式')
    return
  }
  savingProxy.value = true
  try {
    await createResource({ ...proxyForm, contact: { ...proxyForm.contact } })
    ElMessage.success('供需信息已提交审核')
    proxyVisible.value = false
    await loadRows()
  } finally {
    savingProxy.value = false
  }
}

function openRowDetail(row) {
  detailRow.value = row
  detailVisible.value = true
}

function defaultProxyForm() {
  return {
    merchantId: '',
    cityCode: defaultCityCode,
    typeCode: 'inventory',
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
  }
}

function currentOperatorId() {
  return auth.user?.operatorId || ''
}
</script>
