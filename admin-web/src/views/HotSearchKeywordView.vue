<template>
  <section>
    <div class="page-title">
      <h2>热门搜索词</h2>
      <el-button type="primary" @click="openCreate">新增搜索词</el-button>
    </div>

    <section class="panel">
      <el-form :inline="true" class="filter-bar">
        <el-form-item label="城市站">
          <el-select v-model="filters.cityCode" style="width: 140px">
            <el-option label="织里" value="zhili" />
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
      <el-table v-loading="loading" :data="rows" stripe empty-text="暂无热门搜索词">
        <el-table-column prop="keyword" label="搜索词" min-width="180" />
        <el-table-column prop="sortOrder" label="排序" width="90" />
        <el-table-column label="展示时间" min-width="220">
          <template #default="{ row }">
            {{ timeRangeText(row) }}
          </template>
        </el-table-column>
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

    <el-drawer v-model="drawerVisible" :title="editingId ? '编辑搜索词' : '新增搜索词'" size="460px">
      <el-form label-position="top">
        <el-form-item label="城市站">
          <el-select v-model="form.cityCode">
            <el-option label="织里" value="zhili" />
          </el-select>
        </el-form-item>
        <el-form-item label="搜索词">
          <el-input v-model="form.keyword" maxlength="64" show-word-limit />
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
import { createHotSearchKeyword, listHotSearchKeywords, updateHotSearchKeyword } from '../api/hotSearchKeyword'

const statusText = { draft: '草稿', active: '启用', disabled: '停用' }
const statusTagType = { draft: 'info', active: 'success', disabled: 'warning' }
const filters = reactive({ cityCode: 'zhili', status: '' })
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
    keyword: '',
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
    const resp = await listHotSearchKeywords({ ...filters })
    rows.value = resp.items || []
  } catch {
    errorText.value = '热门搜索词加载失败，请重试'
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

function timeRangeText(row) {
  if (!row.startAt && !row.endAt) return '长期展示'
  return `${row.startAt || '立即'} 至 ${row.endAt || '不限'}`
}

async function submit() {
  try {
    await ElMessageBox.confirm('确认保存当前热门搜索词吗？', '确认保存配置', {
      type: 'warning',
      confirmButtonText: '确认保存',
      cancelButtonText: '取消',
    })
  } catch {
    return
  }
  saving.value = true
  try {
    const payload = { ...form }
    if (editingId.value) {
      await updateHotSearchKeyword(editingId.value, payload)
    } else {
      await createHotSearchKeyword(payload)
    }
    ElMessage.success('配置已保存')
    drawerVisible.value = false
    await loadRows()
  } catch (err) {
    ElMessage.error(err.message || '热门搜索词保存失败，请重试')
  } finally {
    saving.value = false
  }
}
</script>
