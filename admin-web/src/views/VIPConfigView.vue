<template>
  <section>
    <div class="page-title">
      <h2>VIP 配置</h2>
      <el-button :loading="loading" plain @click="loadAll">刷新</el-button>
    </div>

    <section class="panel">
      <el-tabs v-model="activeTab">
        <el-tab-pane label="VIP 套餐" name="plans">
          <div class="table-toolbar">
            <el-button type="primary" @click="openPlanEditor()">新增套餐</el-button>
          </div>
          <el-table v-loading="loading" :data="plans" stripe empty-text="暂无 VIP 套餐">
            <el-table-column prop="name" label="套餐" min-width="140" />
            <el-table-column prop="code" label="编码" width="130" />
            <el-table-column label="周期" width="90">
              <template #default="{ row }">{{ row.durationMonths }} 月</template>
            </el-table-column>
            <el-table-column label="标准价" width="120">
              <template #default="{ row }">{{ formatMoney(row.standardPriceCent) }}</template>
            </el-table-column>
            <el-table-column label="权益" min-width="240">
              <template #default="{ row }">{{ benefitSummary(row.benefits) }}</template>
            </el-table-column>
            <el-table-column label="状态" width="90">
              <template #default="{ row }">
                <el-tag :type="row.status === 'active' ? 'success' : 'info'">{{ statusText(row.status) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="displayOrder" label="排序" width="80" />
            <el-table-column label="操作" width="100" fixed="right">
              <template #default="{ row }">
                <el-button type="primary" link @click="openPlanEditor(row)">编辑</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="次数包" name="quotaPacks">
          <div class="table-toolbar">
            <el-button type="primary" @click="openQuotaPackEditor()">新增次数包</el-button>
          </div>
          <el-table v-loading="loading" :data="quotaPacks" stripe empty-text="暂无次数包">
            <el-table-column prop="name" label="次数包" min-width="140" />
            <el-table-column prop="code" label="编码" width="130" />
            <el-table-column prop="description" label="描述" min-width="160" />
            <el-table-column label="价格" width="140">
              <template #default="{ row }">{{ formatMoney(row.salePriceCent || row.standardPriceCent) }}</template>
            </el-table-column>
            <el-table-column label="权益" min-width="240">
              <template #default="{ row }">{{ benefitSummary(row.benefits) }}</template>
            </el-table-column>
            <el-table-column label="状态" width="90">
              <template #default="{ row }">
                <el-tag :type="row.status === 'active' ? 'success' : 'info'">{{ statusText(row.status) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="displayOrder" label="排序" width="80" />
            <el-table-column label="操作" width="100" fixed="right">
              <template #default="{ row }">
                <el-button type="primary" link @click="openQuotaPackEditor(row)">编辑</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="优惠活动" name="promotions">
          <div class="table-toolbar">
            <el-button type="primary" @click="openPromotionEditor()">新增优惠</el-button>
          </div>
          <el-table v-loading="loading" :data="promotions" stripe empty-text="暂无优惠活动">
            <el-table-column prop="code" label="编码" min-width="150" />
            <el-table-column label="套餐" width="130">
              <template #default="{ row }">{{ row.planName || row.planCode }}</template>
            </el-table-column>
            <el-table-column label="类型" width="110">
              <template #default="{ row }">{{ promotionTypeText(row.promotionType) }}</template>
            </el-table-column>
            <el-table-column label="优惠价" width="120">
              <template #default="{ row }">{{ formatMoney(row.salePriceCent) }}</template>
            </el-table-column>
            <el-table-column label="名额" width="110">
              <template #default="{ row }">{{ row.quotaLimit ? `${row.usedCount || 0}/${row.quotaLimit}` : '不限' }}</template>
            </el-table-column>
            <el-table-column label="状态" width="90">
              <template #default="{ row }">
                <el-tag :type="row.status === 'active' ? 'success' : 'info'">{{ statusText(row.status) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="100" fixed="right">
              <template #default="{ row }">
                <el-button type="primary" link @click="openPromotionEditor(row)">编辑</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
      </el-tabs>
    </section>

    <el-drawer v-model="planDrawerVisible" title="VIP 套餐" size="560px">
      <el-form label-position="top">
        <div class="form-grid">
          <el-form-item label="套餐编码">
            <el-input v-model.trim="planForm.code" :disabled="Boolean(editingPlanCode)" />
          </el-form-item>
          <el-form-item label="套餐名称">
            <el-input v-model.trim="planForm.name" />
          </el-form-item>
          <el-form-item label="周期（月）">
            <el-input-number v-model="planForm.durationMonths" :min="1" :max="120" />
          </el-form-item>
          <el-form-item label="标准价（元）">
            <el-input-number v-model="planForm.standardPriceYuan" :min="0" :precision="2" :step="1" />
          </el-form-item>
          <el-form-item label="状态">
            <el-select v-model="planForm.status">
              <el-option label="启用" value="active" />
              <el-option label="停用" value="inactive" />
            </el-select>
          </el-form-item>
          <el-form-item label="排序">
            <el-input-number v-model="planForm.displayOrder" :min="0" />
          </el-form-item>
        </div>
        <BenefitEditor v-model="planForm.benefits" />
        <div class="drawer-actions">
          <el-button @click="planDrawerVisible = false">取消</el-button>
          <el-button type="primary" :loading="saving" @click="savePlan">保存</el-button>
        </div>
      </el-form>
    </el-drawer>

    <el-drawer v-model="quotaPackDrawerVisible" title="次数包" size="560px">
      <el-form label-position="top">
        <div class="form-grid">
          <el-form-item label="次数包编码">
            <el-input v-model.trim="quotaPackForm.code" :disabled="Boolean(editingQuotaPackCode)" />
          </el-form-item>
          <el-form-item label="名称">
            <el-input v-model.trim="quotaPackForm.name" />
          </el-form-item>
          <el-form-item label="标准价（元）">
            <el-input-number v-model="quotaPackForm.standardPriceYuan" :min="0" :precision="2" :step="1" />
          </el-form-item>
          <el-form-item label="优惠价（元）">
            <el-input-number v-model="quotaPackForm.salePriceYuan" :min="0" :precision="2" :step="1" />
          </el-form-item>
          <el-form-item label="状态">
            <el-select v-model="quotaPackForm.status">
              <el-option label="启用" value="active" />
              <el-option label="停用" value="inactive" />
            </el-select>
          </el-form-item>
          <el-form-item label="排序">
            <el-input-number v-model="quotaPackForm.displayOrder" :min="0" />
          </el-form-item>
        </div>
        <el-form-item label="描述">
          <el-input v-model.trim="quotaPackForm.description" />
        </el-form-item>
        <el-form-item label="优惠标签">
          <el-input v-model.trim="quotaPackForm.saleLabel" />
        </el-form-item>
        <BenefitEditor v-model="quotaPackForm.benefits" />
        <div class="drawer-actions">
          <el-button @click="quotaPackDrawerVisible = false">取消</el-button>
          <el-button type="primary" :loading="saving" @click="saveQuotaPack">保存</el-button>
        </div>
      </el-form>
    </el-drawer>

    <el-drawer v-model="promotionDrawerVisible" title="优惠活动" size="560px">
      <el-form label-position="top">
        <div class="form-grid">
          <el-form-item label="优惠编码">
            <el-input v-model.trim="promotionForm.code" :disabled="Boolean(editingPromotionCode)" />
          </el-form-item>
          <el-form-item label="适用套餐">
            <el-select v-model="promotionForm.planCode">
              <el-option v-for="plan in plans" :key="plan.code" :label="plan.name" :value="plan.code" />
            </el-select>
          </el-form-item>
          <el-form-item label="优惠类型">
            <el-select v-model="promotionForm.promotionType">
              <el-option label="首购优惠" value="first_purchase" />
              <el-option label="限时优惠" value="launch" />
            </el-select>
          </el-form-item>
          <el-form-item label="优惠价（元）">
            <el-input-number v-model="promotionForm.salePriceYuan" :min="0" :precision="2" :step="1" />
          </el-form-item>
          <el-form-item label="名额">
            <el-input-number v-model="promotionForm.quotaLimit" :min="0" />
          </el-form-item>
          <el-form-item label="状态">
            <el-select v-model="promotionForm.status">
              <el-option label="启用" value="active" />
              <el-option label="停用" value="inactive" />
            </el-select>
          </el-form-item>
        </div>
        <div class="form-grid">
          <el-form-item label="开始时间">
            <el-date-picker v-model="promotionForm.startsAt" type="datetime" value-format="YYYY-MM-DDTHH:mm:ssZ" />
          </el-form-item>
          <el-form-item label="结束时间">
            <el-date-picker v-model="promotionForm.endsAt" type="datetime" value-format="YYYY-MM-DDTHH:mm:ssZ" />
          </el-form-item>
        </div>
        <div class="drawer-actions">
          <el-button @click="promotionDrawerVisible = false">取消</el-button>
          <el-button type="primary" :loading="saving" @click="savePromotion">保存</el-button>
        </div>
      </el-form>
    </el-drawer>
  </section>
</template>

<script setup>
import { defineComponent, h, onMounted, reactive, ref, resolveComponent } from 'vue'
import { ElMessage } from '../plugins/elementPlus'
import {
  createQuotaPackConfig,
  createVIPPlanConfig,
  createVIPPromotionConfig,
  listQuotaPackConfigs,
  listVIPPlanConfigs,
  listVIPPromotionConfigs,
  updateQuotaPackConfig,
  updateVIPPlanConfig,
  updateVIPPromotionConfig,
} from '../api/vipConfig'

const BenefitEditor = defineComponent({
  props: {
    modelValue: { type: Object, required: true },
  },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    const ElFormItem = resolveComponent('el-form-item')
    const ElInputNumber = resolveComponent('el-input-number')
    function update(key, value) {
      emit('update:modelValue', { ...props.modelValue, [key]: Number(value || 0) })
    }
    const field = (label, key, min = 0) => h(ElFormItem, { label }, () => h(ElInputNumber, {
      modelValue: props.modelValue[key] || 0,
      min,
      'onUpdate:modelValue': (value) => update(key, value),
    }))
    return () => h('section', { class: 'benefit-grid' }, [
      field('发布额度', 'publishQuota'),
      field('刷新次数', 'refreshQuota'),
      field('置顶券', 'topVoucherCount'),
      field('置顶时长（小时）', 'topDurationHours'),
      field('主页图片上限', 'homepageImageLimit'),
    ])
  },
})

const activeTab = ref('plans')
const loading = ref(false)
const saving = ref(false)
const plans = ref([])
const quotaPacks = ref([])
const promotions = ref([])
const planDrawerVisible = ref(false)
const quotaPackDrawerVisible = ref(false)
const promotionDrawerVisible = ref(false)
const editingPlanCode = ref('')
const editingQuotaPackCode = ref('')
const editingPromotionCode = ref('')

const planForm = reactive(createPlanForm())
const quotaPackForm = reactive(createQuotaPackForm())
const promotionForm = reactive(createPromotionForm())

onMounted(loadAll)

async function loadAll() {
  loading.value = true
  try {
    const [planResp, quotaPackResp, promotionResp] = await Promise.all([
      listVIPPlanConfigs(),
      listQuotaPackConfigs(),
      listVIPPromotionConfigs(),
    ])
    plans.value = planResp.items || []
    quotaPacks.value = quotaPackResp.items || []
    promotions.value = promotionResp.items || []
  } finally {
    loading.value = false
  }
}

function openPlanEditor(row) {
  editingPlanCode.value = row?.code || ''
  Object.assign(planForm, row ? {
    code: row.code,
    name: row.name,
    durationMonths: row.durationMonths,
    standardPriceYuan: centToYuan(row.standardPriceCent),
    status: row.status || 'active',
    displayOrder: row.displayOrder || 0,
    benefits: cloneBenefits(row.benefits),
  } : createPlanForm())
  planDrawerVisible.value = true
}

function openQuotaPackEditor(row) {
  editingQuotaPackCode.value = row?.code || ''
  Object.assign(quotaPackForm, row ? {
    code: row.code,
    name: row.name,
    description: row.description || '',
    standardPriceYuan: centToYuan(row.standardPriceCent),
    salePriceYuan: centToYuan(row.salePriceCent),
    saleLabel: row.saleLabel || '',
    status: row.status || 'active',
    displayOrder: row.displayOrder || 0,
    benefits: cloneBenefits(row.benefits),
  } : createQuotaPackForm())
  quotaPackDrawerVisible.value = true
}

function openPromotionEditor(row) {
  editingPromotionCode.value = row?.code || ''
  Object.assign(promotionForm, row ? {
    code: row.code,
    planCode: row.planCode,
    promotionType: row.promotionType,
    salePriceYuan: centToYuan(row.salePriceCent),
    startsAt: row.startsAt || '',
    endsAt: row.endsAt || '',
    quotaLimit: row.quotaLimit || 0,
    status: row.status || 'active',
  } : createPromotionForm())
  promotionDrawerVisible.value = true
}

async function savePlan() {
  saving.value = true
  try {
    const payload = buildPlanPayload()
    if (editingPlanCode.value) {
      await updateVIPPlanConfig(editingPlanCode.value, payload)
    } else {
      await createVIPPlanConfig(payload)
    }
    ElMessage.success('VIP 套餐已保存')
    planDrawerVisible.value = false
    await loadAll()
  } finally {
    saving.value = false
  }
}

async function saveQuotaPack() {
  saving.value = true
  try {
    const payload = buildQuotaPackPayload()
    if (editingQuotaPackCode.value) {
      await updateQuotaPackConfig(editingQuotaPackCode.value, payload)
    } else {
      await createQuotaPackConfig(payload)
    }
    ElMessage.success('次数包已保存')
    quotaPackDrawerVisible.value = false
    await loadAll()
  } finally {
    saving.value = false
  }
}

async function savePromotion() {
  saving.value = true
  try {
    const payload = buildPromotionPayload()
    if (editingPromotionCode.value) {
      await updateVIPPromotionConfig(editingPromotionCode.value, payload)
    } else {
      await createVIPPromotionConfig(payload)
    }
    ElMessage.success('优惠活动已保存')
    promotionDrawerVisible.value = false
    await loadAll()
  } finally {
    saving.value = false
  }
}

function buildPlanPayload() {
  return {
    code: planForm.code.trim(),
    name: planForm.name.trim(),
    durationMonths: Number(planForm.durationMonths || 0),
    standardPriceCent: yuanToCent(planForm.standardPriceYuan),
    status: planForm.status,
    displayOrder: Number(planForm.displayOrder || 0),
    benefits: buildBenefitsPayload(planForm.benefits),
  }
}

function buildQuotaPackPayload() {
  return {
    code: quotaPackForm.code.trim(),
    name: quotaPackForm.name.trim(),
    description: quotaPackForm.description.trim(),
    standardPriceCent: yuanToCent(quotaPackForm.standardPriceYuan),
    salePriceCent: yuanToCent(quotaPackForm.salePriceYuan),
    saleLabel: quotaPackForm.saleLabel.trim(),
    status: quotaPackForm.status,
    displayOrder: Number(quotaPackForm.displayOrder || 0),
    benefits: buildBenefitsPayload(quotaPackForm.benefits),
  }
}

function buildPromotionPayload() {
  return {
    code: promotionForm.code.trim(),
    planCode: promotionForm.planCode,
    promotionType: promotionForm.promotionType,
    salePriceCent: yuanToCent(promotionForm.salePriceYuan),
    startsAt: promotionForm.startsAt || '',
    endsAt: promotionForm.endsAt || '',
    quotaLimit: Number(promotionForm.quotaLimit || 0),
    status: promotionForm.status,
  }
}

function buildBenefitsPayload(benefits) {
  return {
    publishPolicy: 'quota',
    publishQuota: Number(benefits.publishQuota || 0),
    refreshQuota: Number(benefits.refreshQuota || 0),
    topVoucherCount: Number(benefits.topVoucherCount || 0),
    topDurationHours: Number(benefits.topDurationHours || 0),
    homepageImageLimit: Number(benefits.homepageImageLimit || 0),
  }
}

function createPlanForm() {
  return {
    code: '',
    name: '',
    durationMonths: 1,
    standardPriceYuan: 0,
    status: 'active',
    displayOrder: 0,
    benefits: createBenefits(),
  }
}

function createQuotaPackForm() {
  return {
    code: '',
    name: '',
    description: '',
    standardPriceYuan: 0,
    salePriceYuan: 0,
    saleLabel: '',
    status: 'active',
    displayOrder: 0,
    benefits: createBenefits(),
  }
}

function createPromotionForm() {
  return {
    code: '',
    planCode: '',
    promotionType: 'launch',
    salePriceYuan: 0,
    startsAt: '',
    endsAt: '',
    quotaLimit: 0,
    status: 'active',
  }
}

function createBenefits() {
  return {
    publishQuota: 0,
    refreshQuota: 0,
    topVoucherCount: 0,
    topDurationHours: 0,
    homepageImageLimit: 0,
  }
}

function cloneBenefits(benefits = {}) {
  return { ...createBenefits(), ...benefits }
}

function centToYuan(value) {
  return Number(value || 0) / 100
}

function yuanToCent(value) {
  return Math.round(Number(value || 0) * 100)
}

function formatMoney(value) {
  return `¥${centToYuan(value).toFixed(2)}`
}

function benefitSummary(benefits = {}) {
  const parts = []
  if (benefits.publishQuota) parts.push(`${benefits.publishQuota} 条发布`)
  if (benefits.refreshQuota) parts.push(`${benefits.refreshQuota} 次刷新`)
  if (benefits.topVoucherCount) parts.push(`${benefits.topVoucherCount} 张置顶`)
  return parts.length ? parts.join(' · ') : '-'
}

function statusText(status) {
  return status === 'active' ? '启用' : '停用'
}

function promotionTypeText(type) {
  return type === 'first_purchase' ? '首购优惠' : '限时优惠'
}
</script>

<style scoped>
.table-toolbar {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 12px;
}

.form-grid,
.benefit-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 16px;
}

.benefit-grid {
  margin-top: 8px;
}

@media (max-width: 720px) {
  .form-grid,
  .benefit-grid {
    grid-template-columns: 1fr;
  }
}
</style>
