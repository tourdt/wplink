<template>
  <section>
    <div class="page-title">
      <h2>Banner 专题</h2>
      <el-button type="primary" @click="openCreate">新增配置</el-button>
    </div>

    <section class="panel">
      <el-form :inline="true" class="filter-bar">
        <el-form-item label="城市站">
          <el-select v-model="filters.cityCode" style="width: 140px">
            <el-option label="织里" value="zhili" />
          </el-select>
        </el-form-item>
        <el-form-item label="类型">
          <el-select v-model="filters.kind" placeholder="全部" style="width: 140px">
            <el-option label="全部" value="" />
            <el-option label="Banner" value="banner" />
            <el-option label="专题" value="topic" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="filters.status" placeholder="全部" style="width: 140px">
            <el-option label="全部" value="" />
            <el-option label="草稿" value="draft" />
            <el-option label="启用" value="active" />
            <el-option label="停用" value="disabled" />
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
      <el-table v-loading="loading" :data="rows" stripe empty-text="暂无 Banner 或专题配置">
        <el-table-column prop="title" label="标题" min-width="180" />
        <el-table-column label="类型" width="100">
          <template #default="{ row }">{{ kindText[row.kind] || row.kind }}</template>
        </el-table-column>
        <el-table-column label="资源范围" width="180">
          <template #default="{ row }">{{ row.typeScope?.join('、') || '-' }}</template>
        </el-table-column>
        <el-table-column label="跳转" min-width="180">
          <template #default="{ row }">{{ jumpTypeText[row.jumpType] || row.jumpType }}：{{ row.jumpTarget }}</template>
        </el-table-column>
        <el-table-column prop="sortOrder" label="排序" width="90" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="statusTagType[row.status] || 'info'">{{ statusText[row.status] || row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="updatedAt" label="更新时间" width="180" />
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="openEdit(row)">编辑</el-button>
          </template>
        </el-table-column>
      </el-table>
    </section>

    <el-drawer v-model="drawerVisible" :title="editingId ? '编辑配置' : '新增配置'" size="560px">
      <el-form label-position="top">
        <el-form-item label="城市站">
          <el-select v-model="form.cityCode">
            <el-option label="织里" value="zhili" />
          </el-select>
        </el-form-item>
        <el-form-item label="类型">
          <el-segmented v-model="form.kind" :options="kindOptions" />
        </el-form-item>
        <el-form-item label="标题">
          <el-input v-model="form.title" />
        </el-form-item>
        <el-form-item label="副标题">
          <el-input v-model="form.subtitle" />
        </el-form-item>
        <el-form-item label="封面">
          <el-input v-model="form.coverUrl" placeholder="https://..." />
        </el-form-item>
        <el-form-item label="资源类型范围">
          <el-select v-model="form.typeScope" multiple>
            <el-option label="库存" value="inventory" />
            <el-option label="货源" value="goods" />
            <el-option label="工厂" value="factory" />
            <el-option label="服务" value="service" />
          </el-select>
        </el-form-item>
        <el-form-item label="跳转类型">
          <el-select v-model="form.jumpType">
            <el-option label="专题" value="topic" />
            <el-option label="资源详情" value="resource" />
            <el-option label="商家主页" value="merchant" />
            <el-option label="需求入口" value="demand" />
            <el-option label="内部页面" value="internal" />
            <el-option label="活动网页" value="webview" />
          </el-select>
        </el-form-item>
        <el-form-item label="跳转目标">
          <el-input v-model="form.jumpTarget" />
        </el-form-item>
        <el-form-item label="标签">
          <el-select v-model="form.tags" multiple filterable allow-create>
            <el-option label="首页推荐" value="首页推荐" />
            <el-option label="童装" value="童装" />
            <el-option label="库存" value="库存" />
          </el-select>
        </el-form-item>
        <el-form-item label="上线时间">
          <el-date-picker v-model="form.startAt" type="datetime" value-format="YYYY-MM-DDTHH:mm:ssZ" />
        </el-form-item>
        <el-form-item label="下线时间">
          <el-date-picker v-model="form.endAt" type="datetime" value-format="YYYY-MM-DDTHH:mm:ssZ" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="form.sortOrder" :min="0" />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="form.status">
            <el-option label="草稿" value="draft" />
            <el-option label="启用" value="active" />
            <el-option label="停用" value="disabled" />
          </el-select>
        </el-form-item>
        <div class="drawer-actions">
          <el-button @click="drawerVisible = false">取消</el-button>
          <el-button type="primary" :loading="saving" @click="submit">保存</el-button>
        </div>
      </el-form>
    </el-drawer>
  </section>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { createBannerTopic, listBannerTopics, updateBannerTopic } from '../api/bannerTopic'

const kindText = { banner: 'Banner', topic: '专题' }
const jumpTypeText = { topic: '专题', resource: '资源', merchant: '商家', demand: '需求', internal: '内部页', webview: '网页' }
const statusText = { draft: '草稿', active: '启用', disabled: '停用' }
const statusTagType = { draft: 'info', active: 'success', disabled: 'warning' }
const kindOptions = [
  { label: 'Banner', value: 'banner' },
  { label: '专题', value: 'topic' },
]

const filters = reactive({ cityCode: 'zhili', kind: '', status: '' })
const rows = ref([])
const loading = ref(false)
const errorText = ref('')
const saving = ref(false)
const drawerVisible = ref(false)
const editingId = ref('')
const form = reactive(defaultForm())

onMounted(loadRows)

function defaultForm() {
  return {
    cityCode: 'zhili',
    kind: 'banner',
    title: '',
    subtitle: '',
    coverUrl: '',
    typeScope: [],
    jumpType: 'topic',
    jumpTarget: '',
    tags: [],
    startAt: '',
    endAt: '',
    sortOrder: 0,
    status: 'draft',
  }
}

async function loadRows() {
  loading.value = true
  errorText.value = ''
  try {
    const resp = await listBannerTopics({ ...filters })
    rows.value = resp.items || []
  } catch {
    errorText.value = 'Banner 专题配置加载失败，请重试'
  } finally {
    loading.value = false
  }
}

function resetForm(data = defaultForm()) {
  Object.assign(form, defaultForm(), data)
}

function openCreate() {
  editingId.value = ''
  resetForm()
  drawerVisible.value = true
}

function openEdit(row) {
  editingId.value = row.id
  resetForm(row)
  drawerVisible.value = true
}

async function submit() {
  try {
    await ElMessageBox.confirm('确认保存当前 Banner/专题配置吗？', '确认保存配置', {
      type: 'warning',
      confirmButtonText: '确认保存',
      cancelButtonText: '取消',
    })
  } catch {
    return
  }
  saving.value = true
  try {
    if (editingId.value) {
      await updateBannerTopic(editingId.value, { ...form })
    } else {
      await createBannerTopic({ ...form })
    }
    ElMessage.success('配置已保存')
    drawerVisible.value = false
    await loadRows()
  } finally {
    saving.value = false
  }
}
</script>
