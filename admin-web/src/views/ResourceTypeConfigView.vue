<template>
  <section>
    <div class="page-title">
      <h2>资源类型配置</h2>
      <el-button :loading="loading" plain @click="loadConfigs">刷新</el-button>
    </div>

    <section class="panel">
      <el-form :inline="true" class="filter-bar">
        <el-form-item label="城市站">
          <el-select v-model="filters.cityCode" style="width: 140px">
            <el-option label="织里" value="zhili" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="filters.status" style="width: 140px">
            <el-option label="全部" value="" />
            <el-option label="启用" value="active" />
            <el-option label="停用" value="disabled" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="loadConfigs">查询</el-button>
        </el-form-item>
      </el-form>

      <div v-if="errorText" class="table-state table-state-error">
        <span>{{ errorText }}</span>
        <el-button type="danger" plain @click="loadConfigs">重试</el-button>
      </div>
      <el-table v-loading="loading" :data="configs" stripe empty-text="暂无资源类型配置">
        <el-table-column prop="typeName" label="类型名称" width="140" />
        <el-table-column prop="typeCode" label="编码" width="140" />
        <el-table-column prop="defaultValidDays" label="有效期" width="100">
          <template #default="{ row }">{{ row.defaultValidDays }} 天</template>
        </el-table-column>
        <el-table-column label="必填字段" min-width="220">
          <template #default="{ row }">
            <el-tag v-for="field in row.requiredFields" :key="field" class="field-tag" size="small">
              {{ field }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'success' : 'info'">
              {{ row.status === 'active' ? '启用' : '停用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="openEditor(row)">配置</el-button>
          </template>
        </el-table-column>
      </el-table>
    </section>

    <el-drawer v-model="drawerVisible" title="编辑资源类型配置" size="560px">
      <el-form v-if="editing" label-position="top">
        <el-form-item label="类型">
          <el-input :model-value="`${editing.typeName}（${editing.typeCode}）`" disabled />
        </el-form-item>
        <el-form-item label="默认有效期">
          <el-input-number v-model="editing.defaultValidDays" :min="1" :max="365" />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="editing.status">
            <el-option label="启用" value="active" />
            <el-option label="停用" value="disabled" />
          </el-select>
        </el-form-item>
        <el-form-item label="配置 JSON">
          <el-input v-model="configJson" type="textarea" :rows="16" spellcheck="false" />
        </el-form-item>
        <el-alert
          v-if="jsonError"
          :title="jsonError"
          type="error"
          show-icon
          :closable="false"
          class="json-alert"
        />
        <div class="drawer-actions">
          <el-button @click="drawerVisible = false">取消</el-button>
          <el-button type="primary" :loading="saving" @click="saveConfig">保存</el-button>
        </div>
      </el-form>
    </el-drawer>
  </section>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listResourceTypeConfigs, updateResourceTypeConfig } from '../api/city'

const filters = reactive({
  cityCode: 'zhili',
  status: '',
})
const configs = ref([])
const loading = ref(false)
const errorText = ref('')
const saving = ref(false)
const drawerVisible = ref(false)
const editing = ref(null)
const configJson = ref('')
const jsonError = ref('')

onMounted(loadConfigs)

async function loadConfigs() {
  loading.value = true
  errorText.value = ''
  try {
    const resp = await listResourceTypeConfigs(filters)
    configs.value = resp.items || []
  } catch {
    errorText.value = '资源类型配置加载失败，请重试'
  } finally {
    loading.value = false
  }
}

function openEditor(row) {
  editing.value = { ...row }
  configJson.value = JSON.stringify(
    {
      fieldSchema: row.fieldSchema || {},
      requiredFields: row.requiredFields || [],
      filterFields: row.filterFields || [],
      displayTemplate: row.displayTemplate || {},
      reviewRules: row.reviewRules || {},
      sortWeights: row.sortWeights || {},
      messageRules: row.messageRules || {},
    },
    null,
    2,
  )
  jsonError.value = ''
  drawerVisible.value = true
}

async function saveConfig() {
  jsonError.value = ''
  let parsed
  try {
    parsed = JSON.parse(configJson.value)
  } catch {
    jsonError.value = '配置 JSON 格式不正确，请检查后再保存'
    return
  }
  try {
    await ElMessageBox.confirm(`确认保存「${editing.value.typeName}」的资源类型配置吗？`, '确认保存配置', {
      type: 'warning',
      confirmButtonText: '确认保存',
      cancelButtonText: '取消',
    })
  } catch {
    return
  }

  saving.value = true
  try {
    await updateResourceTypeConfig(editing.value.id, {
      ...parsed,
      defaultValidDays: editing.value.defaultValidDays,
      status: editing.value.status,
    })
    ElMessage.success('资源类型配置已保存')
    drawerVisible.value = false
    await loadConfigs()
  } finally {
    saving.value = false
  }
}
</script>
