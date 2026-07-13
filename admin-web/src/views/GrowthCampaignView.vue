<template>
  <section>
    <div class="page-title">
      <h2>增长活动</h2>
      <div class="title-actions">
        <el-button :loading="loading" plain @click="loadCampaigns">刷新</el-button>
        <el-button type="primary" @click="openCampaignEditor()">新增活动</el-button>
      </div>
    </div>

    <section class="panel">
      <el-table
        v-loading="loading"
        :data="campaigns"
        stripe
        highlight-current-row
        empty-text="暂无增长活动"
        @row-click="selectCampaign"
      >
        <el-table-column prop="name" label="活动名称" min-width="160" />
        <el-table-column prop="code" label="编码" min-width="180" />
        <el-table-column label="状态" width="90">
          <template #default="{ row }">
            <el-tag :type="campaignStatusType(row.status)">{{ campaignStatusText(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="startsAt" label="开始时间" min-width="170" />
        <el-table-column prop="endsAt" label="结束时间" min-width="170" />
        <el-table-column label="操作" width="230" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click.stop="openCampaignEditor(row)">编辑</el-button>
            <el-button link @click.stop="pauseCampaign(row)">暂停</el-button>
            <el-button type="danger" link @click.stop="disableCampaign(row)">停用</el-button>
          </template>
        </el-table-column>
      </el-table>
    </section>

    <section class="panel">
      <div class="table-toolbar">
        <h3>规则配置</h3>
        <el-button type="primary" :disabled="!selectedCampaignCode" @click="openRuleEditor()">新增规则</el-button>
      </div>
      <el-table v-loading="ruleLoading" :data="rules" stripe empty-text="请选择活动">
        <el-table-column prop="ruleName" label="规则名称" min-width="190" />
        <el-table-column prop="triggerEvent" label="触发事件" min-width="210" />
        <el-table-column label="奖励类型" width="120">
          <template #default="{ row }">{{ rewardTypeText(row.rewardType) }}</template>
        </el-table-column>
        <el-table-column prop="rewardAmount" label="奖励数量" width="100" />
        <el-table-column prop="validDays" label="有效期" width="90" />
        <el-table-column prop="perUserDailyLimit" label="每日上限" width="100" />
        <el-table-column label="状态" width="90">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'success' : 'info'">{{ ruleStatusText(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="90" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="openRuleEditor(row)">编辑</el-button>
          </template>
        </el-table-column>
      </el-table>
    </section>

    <section class="panel">
      <div class="table-toolbar">
        <h3>发放记录</h3>
        <el-button :disabled="!selectedCampaignCode" :loading="grantLoading" plain @click="loadGrants">刷新记录</el-button>
      </div>
      <el-table v-loading="grantLoading" :data="grants" stripe empty-text="暂无发放记录">
        <el-table-column prop="merchantId" label="商家" min-width="150" />
        <el-table-column label="规则" min-width="190">
          <template #default="{ row }">{{ displayGrantRuleName(row) }}</template>
        </el-table-column>
        <el-table-column label="权益" width="120">
          <template #default="{ row }">{{ rewardTypeText(row.rewardType) }}</template>
        </el-table-column>
        <el-table-column prop="rewardAmount" label="数量" width="80" />
        <el-table-column prop="status" label="状态" width="90" />
        <el-table-column prop="reason" label="原因" width="120" />
        <el-table-column prop="createdAt" label="时间" min-width="170" />
      </el-table>
    </section>

    <el-drawer v-model="campaignDrawerVisible" title="增长活动" size="540px">
      <el-form label-position="top">
        <el-form-item label="活动编码">
          <el-input v-model.trim="campaignForm.code" :disabled="Boolean(editingCampaignCode)" />
        </el-form-item>
        <el-form-item label="活动名称">
          <el-input v-model.trim="campaignForm.name" />
        </el-form-item>
        <div class="form-grid">
          <el-form-item label="状态">
            <el-select v-model="campaignForm.status">
              <el-option label="草稿" value="draft" />
              <el-option label="启用" value="active" />
              <el-option label="暂停" value="paused" />
              <el-option label="结束" value="ended" />
              <el-option label="停用" value="disabled" />
            </el-select>
          </el-form-item>
          <el-form-item label="停用原因">
            <el-input v-model.trim="campaignForm.disableReason" />
          </el-form-item>
        </div>
        <div class="form-grid">
          <el-form-item label="开始时间">
            <el-input v-model.trim="campaignForm.startsAt" placeholder="2026-07-10T00:00:00+08:00" />
          </el-form-item>
          <el-form-item label="结束时间">
            <el-input v-model.trim="campaignForm.endsAt" placeholder="2026-10-08T00:00:00+08:00" />
          </el-form-item>
        </div>
        <div class="drawer-actions">
          <el-button @click="campaignDrawerVisible = false">取消</el-button>
          <el-button type="primary" :loading="saving" @click="saveCampaign">保存</el-button>
        </div>
      </el-form>
    </el-drawer>

    <el-drawer v-model="ruleDrawerVisible" title="规则配置" size="560px">
      <el-form label-position="top">
        <el-form-item label="规则名称">
          <el-input v-model.trim="ruleForm.ruleName" />
        </el-form-item>
        <div class="form-grid">
          <el-form-item label="触发事件">
            <el-select v-model="ruleForm.triggerEvent">
              <el-option v-for="item in triggerEvents" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
          </el-form-item>
          <el-form-item label="状态">
            <el-select v-model="ruleForm.status">
              <el-option label="启用" value="active" />
              <el-option label="停用" value="inactive" />
            </el-select>
          </el-form-item>
        </div>
        <div class="form-grid">
          <el-form-item label="奖励类型">
            <el-select v-model="ruleForm.rewardType">
              <el-option label="发布次数" value="publish_quota" />
              <el-option label="刷新次数" value="refresh_quota" />
            </el-select>
          </el-form-item>
          <el-form-item label="奖励数量">
            <el-input-number v-model="ruleForm.rewardAmount" :min="1" />
          </el-form-item>
          <el-form-item label="有效期">
            <el-input-number v-model="ruleForm.validDays" :min="1" />
          </el-form-item>
          <el-form-item label="优先级">
            <el-input-number v-model="ruleForm.priority" :min="1" />
          </el-form-item>
        </div>
        <div class="form-grid">
          <el-form-item label="总上限">
            <el-input-number v-model="ruleForm.perUserLimit" :min="0" />
          </el-form-item>
          <el-form-item label="每日上限">
            <el-input-number v-model="ruleForm.perUserDailyLimit" :min="0" />
          </el-form-item>
          <el-form-item label="供需信息每日上限">
            <el-input-number v-model="ruleForm.perResourceDailyLimit" :min="0" />
          </el-form-item>
        </div>
        <el-form-item label="说明">
          <el-input v-model.trim="ruleForm.description" />
        </el-form-item>
        <div class="drawer-actions">
          <el-button @click="ruleDrawerVisible = false">取消</el-button>
          <el-button type="primary" :loading="saving" @click="saveRule">保存</el-button>
        </div>
      </el-form>
    </el-drawer>
  </section>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from '../plugins/elementPlus'
import {
  createGrowthCampaign,
  createGrowthRule,
  listGrowthCampaigns,
  listGrowthRewardGrants,
  listGrowthRules,
  updateGrowthCampaign,
  updateGrowthRule,
} from '../api/growthCampaign'

const loading = ref(false)
const ruleLoading = ref(false)
const grantLoading = ref(false)
const saving = ref(false)
const campaigns = ref([])
const rules = ref([])
const grants = ref([])
const selectedCampaignCode = ref('')
const campaignDrawerVisible = ref(false)
const ruleDrawerVisible = ref(false)
const editingCampaignCode = ref('')
const editingRuleCode = ref('')
const campaignForm = reactive(createCampaignForm())
const ruleForm = reactive(createRuleForm())

const triggerEvents = [
  { label: '首次登录', value: 'user_first_login' },
  { label: '首条供需信息审核通过', value: 'resource_first_approved' },
  { label: '审核通过数量达标', value: 'resource_approved_count_reached' },
  { label: '分享有效浏览', value: 'resource_share_effective_view' },
  { label: '分享有效联系', value: 'resource_share_effective_contact' },
  { label: '邀请成功', value: 'invitee_first_resource_approved' },
]

onMounted(loadCampaigns)

async function loadCampaigns() {
  loading.value = true
  try {
    const resp = await listGrowthCampaigns()
    campaigns.value = resp.items || []
    if (!selectedCampaignCode.value && campaigns.value[0]) {
      await selectCampaign(campaigns.value[0])
    }
  } finally {
    loading.value = false
  }
}

async function selectCampaign(row) {
  selectedCampaignCode.value = row.code
  await Promise.all([loadRules(), loadGrants()])
}

async function loadRules() {
  if (!selectedCampaignCode.value) return
  ruleLoading.value = true
  try {
    const resp = await listGrowthRules(selectedCampaignCode.value)
    rules.value = resp.items || []
  } finally {
    ruleLoading.value = false
  }
}

async function loadGrants() {
  if (!selectedCampaignCode.value) return
  grantLoading.value = true
  try {
    const resp = await listGrowthRewardGrants(selectedCampaignCode.value, { pageSize: 50 })
    grants.value = resp.items || []
  } finally {
    grantLoading.value = false
  }
}

function openCampaignEditor(row) {
  editingCampaignCode.value = row?.code || ''
  Object.assign(campaignForm, row ? {
    code: row.code,
    name: row.name,
    status: row.status || 'active',
    startsAt: row.startsAt || '',
    endsAt: row.endsAt || '',
    disableReason: '',
  } : createCampaignForm())
  campaignDrawerVisible.value = true
}

function openRuleEditor(row) {
  editingRuleCode.value = row?.ruleCode || ''
  Object.assign(ruleForm, row ? {
    ruleCode: row.ruleCode,
    ruleName: row.ruleName || '',
    triggerEvent: row.triggerEvent,
    status: row.status || 'active',
    priority: row.priority || 100,
    rewardType: row.rewardType || 'publish_quota',
    rewardAmount: row.rewardAmount || 1,
    validDays: row.validDays || 30,
    perUserLimit: row.perUserLimit || 0,
    perUserDailyLimit: row.perUserDailyLimit || 0,
    perResourceDailyLimit: row.perResourceDailyLimit || 0,
    description: row.description || '',
  } : createRuleForm())
  ruleDrawerVisible.value = true
}

async function saveCampaign() {
  saving.value = true
  try {
    const payload = {
      code: campaignForm.code.trim(),
      name: campaignForm.name.trim(),
      status: campaignForm.status,
      startsAt: campaignForm.startsAt.trim(),
      endsAt: campaignForm.endsAt.trim(),
      disableReason: campaignForm.disableReason.trim(),
    }
    if (editingCampaignCode.value) {
      await updateGrowthCampaign(editingCampaignCode.value, payload)
    } else {
      await createGrowthCampaign(payload)
    }
    ElMessage.success('增长活动已保存')
    campaignDrawerVisible.value = false
    await loadCampaigns()
  } finally {
    saving.value = false
  }
}

async function pauseCampaign(row) {
  await updateGrowthCampaign(row.code, { ...row, status: 'paused' })
  ElMessage.success('活动已暂停')
  await loadCampaigns()
}

async function disableCampaign(row) {
  const { value } = await ElMessageBox.prompt('请输入停用原因', '停用增长活动', {
    confirmButtonText: '停用',
    cancelButtonText: '取消',
    inputPattern: /\S+/,
    inputErrorMessage: '请填写停用原因',
  })
  await updateGrowthCampaign(row.code, { ...row, status: 'disabled', disableReason: value })
  ElMessage.success('活动已停用')
  await loadCampaigns()
}

async function saveRule() {
  if (!selectedCampaignCode.value) return
  const ruleName = ruleForm.ruleName.trim()
  if (!ruleName) {
    ElMessage.warning('请填写规则名称')
    return
  }
  saving.value = true
  try {
    const payload = {
      ruleCode: editingRuleCode.value || ruleForm.ruleCode.trim() || createInternalRuleCode(),
      ruleName,
      triggerEvent: ruleForm.triggerEvent,
      status: ruleForm.status,
      priority: Number(ruleForm.priority || 100),
      rewardType: ruleForm.rewardType,
      rewardAmount: Number(ruleForm.rewardAmount || 0),
      validDays: Number(ruleForm.validDays || 0),
      perUserLimit: Number(ruleForm.perUserLimit || 0),
      perUserDailyLimit: Number(ruleForm.perUserDailyLimit || 0),
      perResourceDailyLimit: Number(ruleForm.perResourceDailyLimit || 0),
      description: ruleForm.description.trim(),
    }
    if (editingRuleCode.value) {
      await updateGrowthRule(selectedCampaignCode.value, editingRuleCode.value, payload)
    } else {
      await createGrowthRule(selectedCampaignCode.value, payload)
    }
    ElMessage.success('规则配置已保存')
    ruleDrawerVisible.value = false
    await loadRules()
  } finally {
    saving.value = false
  }
}

function createCampaignForm() {
  return {
    code: '',
    name: '',
    status: 'active',
    startsAt: '',
    endsAt: '',
    disableReason: '',
  }
}

function createRuleForm() {
  return {
    ruleCode: '',
    ruleName: '',
    triggerEvent: 'user_first_login',
    status: 'active',
    priority: 100,
    rewardType: 'publish_quota',
    rewardAmount: 1,
    validDays: 30,
    perUserLimit: 0,
    perUserDailyLimit: 0,
    perResourceDailyLimit: 0,
    description: '',
  }
}

function createInternalRuleCode() {
  return `custom_rule_${Date.now().toString(36)}`
}

function displayGrantRuleName(row = {}) {
  return row.ruleName || row.ruleCode || '-'
}

function rewardTypeText(rewardType) {
  const texts = {
    publish_quota: '发布次数',
    refresh_quota: '刷新次数',
  }
  return texts[rewardType] || rewardType || '-'
}

function campaignStatusText(status) {
  const texts = { draft: '草稿', active: '启用', paused: '暂停', ended: '结束', disabled: '停用' }
  return texts[status] || status
}

function campaignStatusType(status) {
  if (status === 'active') return 'success'
  if (status === 'disabled') return 'danger'
  if (status === 'paused') return 'warning'
  return 'info'
}

function ruleStatusText(status) {
  return status === 'active' ? '启用' : '停用'
}
</script>

<style scoped>
.title-actions,
.table-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.panel + .panel {
  margin-top: 16px;
}

.table-toolbar {
  margin-bottom: 12px;
}

.table-toolbar h3 {
  margin: 0;
  font-size: 16px;
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 16px;
}

@media (max-width: 720px) {
  .title-actions,
  .table-toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  .form-grid {
    grid-template-columns: 1fr;
  }
}
</style>
