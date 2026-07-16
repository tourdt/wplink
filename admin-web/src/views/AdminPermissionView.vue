<template>
  <section>
    <div class="page-title">
      <h2>管理员权限</h2>
      <el-button type="primary" @click="openCreate">新增管理员</el-button>
    </div>

    <section class="panel">
      <el-form :inline="true" class="filter-bar">
        <el-form-item label="关键词">
          <el-input v-model="filters.keyword" placeholder="账号 / 姓名" style="width: 220px" clearable />
        </el-form-item>
        <el-form-item label="角色">
          <el-select v-model="filters.role" style="width: 160px">
            <el-option label="全部" value="" />
            <el-option v-for="item in roleOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="filters.status" style="width: 140px">
            <el-option label="全部" value="" />
            <el-option v-for="item in statusOptions" :key="item.value" :label="item.label" :value="item.value" />
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

      <el-table v-loading="loading" :data="rows" stripe empty-text="暂无管理员账号">
        <el-table-column prop="loginName" label="登录账号" min-width="180" show-overflow-tooltip />
        <el-table-column prop="realName" label="姓名" width="140" />
        <el-table-column label="角色" min-width="190">
          <template #default="{ row }">
            <el-tag v-for="role in row.roles" :key="role" class="field-tag" :type="roleTagType(role)">
              {{ roleText[role] || role }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="110">
          <template #default="{ row }">
            <el-tag :type="row.status === 'enabled' ? 'success' : 'info'">
              {{ statusText[row.status] || row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" label="创建时间" width="180" />
        <el-table-column prop="lastLoginAt" label="最近登录" width="180" />
        <el-table-column label="操作" width="170" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="openEdit(row)">编辑</el-button>
            <el-button :type="row.status === 'enabled' ? 'danger' : 'success'" link @click="toggleStatus(row)">
              {{ row.status === 'enabled' ? '停用' : '启用' }}
            </el-button>
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

    <section class="panel module-permission-panel">
      <div class="panel-header module-permission-header">
        <div>
          <h3>平台运营可见模块</h3>
          <p>配置后，平台运营重新登录即可按新权限展示后台菜单。</p>
        </div>
        <el-button type="primary" :loading="moduleSaving" @click="savePlatformModules">保存配置</el-button>
      </div>

      <div v-if="moduleErrorText" class="table-state table-state-error">
        <span>{{ moduleErrorText }}</span>
        <el-button type="danger" plain @click="loadModulePermissions">重试</el-button>
      </div>

      <div v-loading="moduleLoading" class="module-grid">
        <section v-for="group in moduleGroups" :key="group.name" class="module-group">
          <h4>{{ group.name }}</h4>
          <div class="module-options">
            <el-checkbox
              v-for="module in group.items"
              :key="module.code"
              :model-value="selectedPlatformModules.includes(module.code)"
              @change="setPlatformModule(module.code, $event)"
            >
              {{ module.label }}
            </el-checkbox>
          </div>
        </section>
      </div>
    </section>

    <el-drawer v-model="drawerVisible" :title="editingOperatorId ? '编辑管理员' : '新增管理员'" size="520px">
      <el-form label-position="top">
        <el-form-item label="登录账号">
          <el-input v-model="form.loginName" autocomplete="off" />
        </el-form-item>
        <el-form-item label="姓名">
          <el-input v-model="form.realName" autocomplete="off" />
        </el-form-item>
        <el-form-item :label="editingOperatorId ? '登录密码' : '初始密码'">
          <el-input v-model="form.password" type="password" show-password autocomplete="new-password" :placeholder="editingOperatorId ? '留空不修改' : ''" />
        </el-form-item>
        <el-form-item label="角色">
          <el-select v-model="form.roles" multiple>
            <el-option v-for="item in roleOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="form.status">
            <el-option v-for="item in statusOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <div class="drawer-actions">
          <el-button @click="drawerVisible = false">取消</el-button>
          <el-button type="primary" :loading="saving" @click="submitOperator">保存</el-button>
        </div>
      </el-form>
    </el-drawer>
  </section>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from '../plugins/elementPlus'
import {
  createAdminOperator,
  listAdminModulePermissions,
  listAdminOperators,
  updateAdminOperator,
  updateAdminOperatorStatus,
  updateAdminRoleModulePermissions,
} from '../api/adminPermission'

const roleOptions = [
  { label: '平台运营', value: 'platform_operator' },
  { label: '超级管理员', value: 'super_admin' },
]
const statusOptions = [
  { label: '启用', value: 'enabled' },
  { label: '停用', value: 'disabled' },
]
const roleText = Object.fromEntries(roleOptions.map((item) => [item.value, item.label]))
const statusText = Object.fromEntries(statusOptions.map((item) => [item.value, item.label]))

const filters = reactive({
  keyword: '',
  role: '',
  status: '',
})
const pagination = reactive({ page: 1, pageSize: 20, total: 0 })
const form = reactive(defaultForm())
const rows = ref([])
const availableModules = ref([])
const selectedPlatformModules = ref([])
const loading = ref(false)
const moduleLoading = ref(false)
const saving = ref(false)
const moduleSaving = ref(false)
const errorText = ref('')
const moduleErrorText = ref('')
const drawerVisible = ref(false)
const editingOperatorId = ref('')
const moduleGroups = computed(() => {
  const grouped = new Map()
  for (const module of availableModules.value) {
    const group = module.group || '其他'
    if (!grouped.has(group)) grouped.set(group, [])
    grouped.get(group).push(module)
  }
  return Array.from(grouped.entries()).map(([name, items]) => ({ name, items }))
})

onMounted(() => {
  loadRows()
  loadModulePermissions()
})

async function loadRows(page = pagination.page) {
  pagination.page = page
  loading.value = true
  errorText.value = ''
  try {
    const resp = await listAdminOperators({
      keyword: filters.keyword.trim(),
      role: filters.role,
      status: filters.status,
      page: pagination.page,
      pageSize: pagination.pageSize,
    })
    rows.value = resp.items || []
    pagination.total = resp.total || 0
  } catch {
    errorText.value = '管理员账号加载失败，请重试'
  } finally {
    loading.value = false
  }
}

async function loadModulePermissions() {
  moduleLoading.value = true
  moduleErrorText.value = ''
  try {
    const resp = await listAdminModulePermissions()
    availableModules.value = resp.modules || []
    const platformRole = (resp.roles || []).find((item) => item.roleCode === 'platform_operator')
    selectedPlatformModules.value = [...(platformRole?.modules || [])]
  } catch {
    moduleErrorText.value = '模块权限配置加载失败，请重试'
  } finally {
    moduleLoading.value = false
  }
}

function setPlatformModule(moduleCode, checked) {
  if (checked && !selectedPlatformModules.value.includes(moduleCode)) {
    selectedPlatformModules.value = [...selectedPlatformModules.value, moduleCode]
    return
  }
  if (!checked) {
    selectedPlatformModules.value = selectedPlatformModules.value.filter((code) => code !== moduleCode)
  }
}

async function savePlatformModules() {
  moduleSaving.value = true
  try {
    await updateAdminRoleModulePermissions('platform_operator', {
      modules: [...selectedPlatformModules.value],
    })
    ElMessage.success('模块权限配置已保存，平台运营重新登录后生效')
    await loadModulePermissions()
  } finally {
    moduleSaving.value = false
  }
}

function openCreate() {
  editingOperatorId.value = ''
  Object.assign(form, defaultForm())
  drawerVisible.value = true
}

function openEdit(row) {
  editingOperatorId.value = row.operatorId
  Object.assign(form, {
    loginName: row.loginName || '',
    realName: row.realName || '',
    password: '',
    roles: [...(row.roles || [])],
    status: row.status || 'enabled',
  })
  drawerVisible.value = true
}

async function submitOperator() {
  saving.value = true
  try {
    const payload = {
      loginName: form.loginName.trim(),
      realName: form.realName.trim(),
      password: form.password.trim(),
      roles: [...form.roles],
      status: form.status,
    }
    if (editingOperatorId.value) {
      await updateAdminOperator(editingOperatorId.value, payload)
      ElMessage.success('管理员账号已更新')
    } else {
      await createAdminOperator(payload)
      ElMessage.success('管理员账号已创建')
    }
    drawerVisible.value = false
    await loadRows(editingOperatorId.value ? pagination.page : 1)
  } finally {
    saving.value = false
  }
}

async function toggleStatus(row) {
  const nextStatus = row.status === 'enabled' ? 'disabled' : 'enabled'
  const actionText = nextStatus === 'enabled' ? '启用' : '停用'
  try {
    await ElMessageBox.confirm(`确认${actionText}${row.realName || row.loginName}？`, '更新账号状态', {
      type: nextStatus === 'enabled' ? 'success' : 'warning',
    })
  } catch {
    return
  }
  await updateAdminOperatorStatus(row.operatorId, { status: nextStatus })
  ElMessage.success(`管理员账号已${actionText}`)
  await loadRows(pagination.page)
}

function defaultForm() {
  return {
    loginName: '',
    realName: '',
    password: '',
    roles: ['platform_operator'],
    status: 'enabled',
  }
}

function roleTagType(role) {
  if (role === 'super_admin') return 'danger'
  return 'primary'
}
</script>

<style scoped>
.module-permission-panel {
  margin-top: 16px;
}

.module-permission-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.module-permission-header p {
  margin: 6px 0 0;
  color: #697586;
}

.module-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 18px;
  min-height: 120px;
}

.module-group h4 {
  margin: 0 0 10px;
  font-size: 14px;
}

.module-options {
  display: grid;
  gap: 8px;
}
</style>
