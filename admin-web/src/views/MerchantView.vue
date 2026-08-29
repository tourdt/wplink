<template>
  <section>
    <div class="page-title">
      <h2>商家管理</h2>
      <div class="page-actions">
        <el-button @click="governance.openReports(1)">位置错误反馈</el-button>
        <el-button type="primary" @click="drawerVisible = true">新增商家</el-button>
      </div>
    </div>

    <section class="panel">
      <el-form :inline="true" class="filter-bar">
        <el-form-item label="城市站">
          <el-select v-model="filters.cityCode" style="width: 140px">
            <el-option v-for="station in cityStationOptions" :key="station.value" :label="station.label" :value="station.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="主要身份">
          <el-select v-model="filters.merchantType" placeholder="全部" style="width: 150px">
            <el-option label="全部" value="" />
            <el-option v-for="item in merchantIdentityOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="商家状态">
          <el-select v-model="filters.status" placeholder="全部" style="width: 130px">
            <el-option label="全部" value="" />
            <el-option label="正常" value="active" />
            <el-option label="停用" value="disabled" />
          </el-select>
        </el-form-item>
        <el-form-item label="位置状态">
          <el-select v-model="filters.locationStatus" placeholder="全部" style="width: 140px">
            <el-option v-for="item in locationStatusOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="governance.loadMerchants(1)">查询</el-button>
        </el-form-item>
      </el-form>

      <div v-if="state.merchantError" class="table-state table-state-error">
        <span>{{ state.merchantError }}</span>
        <el-button type="danger" plain @click="governance.loadMerchants()">重试</el-button>
      </div>
      <el-table v-loading="state.merchantsLoading" :data="state.merchants" stripe empty-text="暂无商家数据">
        <el-table-column prop="name" label="商家名称" min-width="170" show-overflow-tooltip />
        <el-table-column label="主要身份" width="120">
          <template #default="{ row }">{{ merchantTypeText[row.merchantType] || row.merchantType }}</template>
        </el-table-column>
        <el-table-column label="商家状态" width="105">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'success' : 'warning'">
              {{ row.status === 'active' ? '正常' : row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="位置治理" min-width="390">
          <template #default="{ row }">
            <div class="location-signal">
              <el-tag size="small" :type="merchantLocationSignal(row).statusType">
                {{ merchantLocationSignal(row).statusText }}
              </el-tag>
              <span class="booth-summary">{{ merchantLocationSignal(row).boothSummary }}</span>
              <span
                class="pending-count"
                :class="{ 'has-pending': row.pendingLocationReportCount > 0 }"
              >
                {{ merchantLocationSignal(row).pendingText }}
              </span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="lastActiveAt" label="最近活跃" width="170" />
        <el-table-column label="操作" width="310" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="governance.openDetail(row)">查看</el-button>
            <el-button link @click="governance.openReports(1)">查看反馈</el-button>
            <el-button
              v-if="row.locationStatus === 'visible'"
              type="danger"
              link
              :loading="governance.isActionLoading('merchant', row.id, 'hide')"
              :disabled="governance.isActionLoading('merchant', row.id)"
              @click="confirmLocationAction(row, 'hide')"
            >
              隐藏位置
            </el-button>
            <el-button
              v-else
              type="success"
              link
              :loading="governance.isActionLoading('merchant', row.id, 'restore')"
              :disabled="governance.isActionLoading('merchant', row.id)"
              @click="confirmLocationAction(row, 'restore')"
            >
              恢复位置
            </el-button>
            <el-button link @click="openEntitlement">发券</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination-row">
        <el-pagination
          v-model:current-page="state.merchantPage.page"
          :page-size="state.merchantPage.pageSize"
          layout="total, prev, pager, next"
          :total="state.merchantPage.total"
          @current-change="governance.loadMerchants"
        />
      </div>
    </section>

    <el-drawer
      v-model="state.detailDrawerOpen"
      title="商家详情"
      size="560px"
      @closed="governance.closeDetail"
    >
      <div v-loading="state.detailLoading" class="detail-content">
        <el-descriptions v-if="state.detail" :column="1" border>
          <el-descriptions-item label="商家名称">{{ state.detail.name }}</el-descriptions-item>
          <el-descriptions-item label="主要身份">
            {{ merchantTypeText[state.detail.merchantType] || state.detail.merchantType }}
          </el-descriptions-item>
          <el-descriptions-item label="主营品类">{{ state.detail.mainCategories?.join('、') || '-' }}</el-descriptions-item>
          <el-descriptions-item label="联系人">{{ state.detail.contact?.name || '-' }}</el-descriptions-item>
          <el-descriptions-item label="电话">{{ state.detail.contact?.phoneMasked || '-' }}</el-descriptions-item>
          <el-descriptions-item label="微信">{{ state.detail.contact?.wechatMasked || '-' }}</el-descriptions-item>
          <el-descriptions-item label="位置摘要">
            <div class="detail-location">
              <div class="location-signal">
                <el-tag size="small" :type="merchantLocationSignal(state.detail).statusType">
                  {{ merchantLocationSignal(state.detail).statusText }}
                </el-tag>
                <span>{{ merchantLocationSignal(state.detail).boothSummary }}</span>
                <span class="pending-count">{{ merchantLocationSignal(state.detail).pendingText }}</span>
              </div>
              <small>{{ state.detail.locationName || state.detail.addressText || '未设置地图位置' }}</small>
              <small v-if="state.detail.locationUpdatedAt">最近更新：{{ state.detail.locationUpdatedAt }}</small>
            </div>
          </el-descriptions-item>
          <el-descriptions-item label="发布概况">
            已发布 {{ state.detail.resourcesSummary?.publishedCount || 0 }}，已成交 {{ state.detail.resourcesSummary?.dealtCount || 0 }}
          </el-descriptions-item>
        </el-descriptions>
      </div>
    </el-drawer>

    <el-drawer
      v-model="state.reportDrawerOpen"
      title="位置错误反馈"
      size="860px"
      @closed="governance.closeReports"
    >
      <el-form :inline="true" class="report-filter">
        <el-form-item label="反馈状态">
          <el-select v-model="state.reportFilters.status" style="width: 150px" @change="governance.loadReports(1)">
            <el-option v-for="item in reportStatusOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
      </el-form>
      <div v-if="state.reportError" class="table-state table-state-error">
        <span>{{ state.reportError }}</span>
        <el-button type="danger" plain @click="governance.loadReports()">重试</el-button>
      </div>
      <el-table v-loading="state.reportsLoading" :data="state.reports" stripe empty-text="暂无位置错误反馈">
        <el-table-column prop="merchantName" label="商家" min-width="150" show-overflow-tooltip />
        <el-table-column label="反馈原因" min-width="180">
          <template #default="{ row }">
            <strong>{{ reportReasonText[row.reasonCode] || row.reasonCode }}</strong>
            <small class="report-description">{{ row.description || '无补充说明' }}</small>
          </template>
        </el-table-column>
        <el-table-column label="报告时位置" min-width="210" show-overflow-tooltip>
          <template #default="{ row }">{{ snapshotSummary(row.locationSnapshot) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="95">
          <template #default="{ row }">
            <el-tag size="small" :type="reportStatusTagType[row.status] || 'info'">
              {{ reportStatusText[row.status] || row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" label="提交时间" width="165" />
        <el-table-column label="操作" width="245" fixed="right">
          <template #default="{ row }">
            <template v-if="row.status === 'pending'">
              <el-button
                type="danger"
                link
                :loading="governance.isActionLoading('report', row.reportId, 'valid')"
                :disabled="governance.isActionLoading('report', row.reportId)"
                @click="confirmReview(row, 'valid')"
              >
                标记有效
              </el-button>
              <el-button
                link
                :loading="governance.isActionLoading('report', row.reportId, 'invalid')"
                :disabled="governance.isActionLoading('report', row.reportId)"
                @click="confirmReview(row, 'invalid')"
              >
                标记无效
              </el-button>
              <el-button
                type="success"
                link
                :loading="governance.isActionLoading('report', row.reportId, 'resolved')"
                :disabled="governance.isActionLoading('report', row.reportId)"
                @click="confirmReview(row, 'resolved')"
              >
                标记已解决
              </el-button>
            </template>
            <span v-else class="muted-text">{{ row.reviewNote || '已处理' }}</span>
          </template>
        </el-table-column>
      </el-table>
      <div class="pagination-row">
        <el-pagination
          v-model:current-page="state.reportPage.page"
          :page-size="state.reportPage.pageSize"
          layout="total, prev, pager, next"
          :total="state.reportPage.total"
          @current-change="governance.loadReports"
        />
      </div>
    </el-drawer>

    <el-drawer v-model="drawerVisible" title="新增商家" size="520px">
      <el-form label-position="top">
        <el-form-item label="城市站">
          <el-select v-model="form.cityCode">
            <el-option v-for="station in cityStationOptions" :key="station.value" :label="station.label" :value="station.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="商家名称"><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="主要身份">
          <el-select v-model="form.merchantType">
            <el-option v-for="item in merchantIdentityOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="主营品类">
          <el-select v-model="form.mainCategories" multiple filterable allow-create>
            <el-option label="童装" value="童装" />
            <el-option label="卫衣" value="卫衣" />
            <el-option label="套装" value="套装" />
          </el-select>
        </el-form-item>
        <el-form-item label="联系人"><el-input v-model="form.contactName" /></el-form-item>
        <el-form-item label="联系电话"><el-input v-model="form.contactPhone" /></el-form-item>
        <el-form-item label="微信"><el-input v-model="form.contactWechat" /></el-form-item>
        <el-form-item label="地址"><el-input v-model="form.addressText" /></el-form-item>
        <el-form-item label="简介"><el-input v-model="form.description" type="textarea" :rows="3" /></el-form-item>
        <div class="drawer-actions">
          <el-button @click="drawerVisible = false">取消</el-button>
          <el-button type="primary" :loading="saving" @click="submitMerchant">保存</el-button>
        </div>
      </el-form>
    </el-drawer>
  </section>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from '../plugins/elementPlus'
import {
  createMerchant,
  getMerchant,
  listMerchantLocationReports,
  listMerchants,
  reviewMerchantLocationReport,
  updateMerchantLocationStatus,
} from '../api/merchant'
import { cityStationOptions, defaultCityCode } from '../common/cityStations'
import { merchantIdentityOptions, merchantTypeText } from '../common/merchantIdentity'
import { createMerchantLocationGovernance, merchantLocationSignal } from './merchantLocationGovernance'

const locationStatusOptions = [
  { label: '全部', value: '' },
  { label: '已公开', value: 'visible' },
  { label: '已隐藏', value: 'hidden' },
  { label: '风险隐藏', value: 'risk_hidden' },
]
const reportStatusOptions = [
  { label: '待处理', value: 'pending' },
  { label: '有效', value: 'valid' },
  { label: '无效', value: 'invalid' },
  { label: '已解决', value: 'resolved' },
]
const reportStatusText = Object.fromEntries(reportStatusOptions.map((item) => [item.value, item.label]))
const reportStatusTagType = { pending: 'warning', valid: 'danger', invalid: 'info', resolved: 'success' }
const reportReasonText = {
  navigation_inaccurate: '导航位置不准确',
  wrong_market: '市场信息错误',
  wrong_booth: '档口信息错误',
  other: '其他问题',
}
const reviewActionCopy = {
  valid: { actionText: '标记有效', prompt: '确认将该位置反馈标记为有效？' },
  invalid: { actionText: '标记无效', prompt: '确认将该位置反馈标记为无效？' },
  resolved: { actionText: '标记已解决', prompt: '确认将该位置反馈标记为已解决？' },
}
const locationActionCopy = {
  hide: { actionText: '隐藏位置', prompt: '隐藏后该商家不会出现在拿货档口目录，确认隐藏位置？' },
  restore: { actionText: '恢复位置', prompt: '仅坐标有效的商家可以恢复公开，确认恢复位置？' },
}

const governance = createMerchantLocationGovernance({
  getMerchant,
  listMerchants,
  listMerchantLocationReports,
  reviewMerchantLocationReport,
  updateMerchantLocationStatus,
}, (type, message) => ElMessage[type](message), reactive)
const state = governance.state
const filters = state.merchantFilters
const form = reactive({
  cityCode: defaultCityCode,
  name: '',
  merchantType: 'individual',
  mainCategories: ['童装'],
  contactName: '',
  contactPhone: '',
  contactWechat: '',
  addressText: '',
  description: '',
})
const saving = ref(false)
const drawerVisible = ref(false)
const router = useRouter()

onMounted(() => governance.loadMerchants(1))

async function confirmReview(row, status) {
  const copy = reviewActionCopy[status]
  try {
    await ElMessageBox.confirm(copy.prompt, copy.actionText, {
      type: status === 'valid' ? 'warning' : 'info',
      confirmButtonText: copy.actionText,
      cancelButtonText: '取消',
    })
  } catch {
    return
  }
  await governance.reviewReport(row, status)
}

async function confirmLocationAction(row, action) {
  const copy = locationActionCopy[action]
  try {
    await ElMessageBox.confirm(copy.prompt, copy.actionText, {
      type: action === 'hide' ? 'warning' : 'info',
      confirmButtonText: copy.actionText,
      cancelButtonText: '取消',
    })
  } catch {
    return
  }
  await governance.updateLocation(row, action)
}

async function submitMerchant() {
  saving.value = true
  try {
    await createMerchant({ ...form })
    ElMessage.success('商家已创建')
    drawerVisible.value = false
    await governance.loadMerchants(1)
  } finally {
    saving.value = false
  }
}

function snapshotSummary(snapshot = {}) {
  return [
    snapshot.marketName,
    snapshot.buildingName,
    snapshot.floorNo,
    snapshot.boothNo,
    snapshot.locationName,
  ].filter(Boolean).join(' · ') || '报告时未记录位置摘要'
}

function openEntitlement() {
  router.push({ name: 'entitlements' })
}
</script>

<style scoped>
.page-actions,
.location-signal {
  display: flex;
  align-items: center;
  gap: 8px;
}

.location-signal {
  min-width: 0;
  white-space: nowrap;
}

.booth-summary {
  overflow: hidden;
  min-width: 0;
  color: #364152;
  text-overflow: ellipsis;
}

.pending-count {
  flex: none;
  padding-left: 8px;
  border-left: 1px solid #e4e7ed;
  color: #697586;
  font-size: 12px;
}

.pending-count.has-pending {
  color: #b42318;
  font-weight: 600;
}

.detail-content {
  min-height: 180px;
}

.detail-location {
  display: grid;
  gap: 8px;
}

.detail-location small,
.report-description {
  display: block;
  color: #697586;
}

.report-filter {
  margin-bottom: 8px;
}

.muted-text {
  color: #697586;
  font-size: 13px;
}
</style>
