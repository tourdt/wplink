<template>
  <section>
    <div class="page-title">
      <h2>举报审核</h2>
      <el-button type="primary" @click="loadRows">刷新</el-button>
    </div>

    <section class="panel">
      <el-form :inline="true" class="filter-bar">
        <el-form-item label="处理状态">
          <el-select v-model="filters.status" style="width: 140px">
            <el-option label="待处理" value="pending" />
            <el-option label="举报成立" value="valid" />
            <el-option label="举报不成立" value="invalid" />
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
      <el-table v-loading="loading" :data="rows" stripe empty-text="暂无举报信息">
        <el-table-column prop="resourceTitle" label="供需信息" min-width="240" />
        <el-table-column prop="merchantName" label="商家" width="180" />
        <el-table-column label="资源状态" width="120">
          <template #default="{ row }">{{ resourceStatusText[row.resourceStatus] || row.resourceStatus }}</template>
        </el-table-column>
        <el-table-column label="举报数" width="90">
          <template #default="{ row }">
            <el-tag :type="row.reportCount > 1 ? 'danger' : 'warning'">{{ row.reportCount }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="最新原因" min-width="180">
          <template #default="{ row }">
            <div class="reason-cell">
              <span>{{ reportReasonText[row.reasonCode] || row.reasonCode }}</span>
              <small v-if="row.reasonText">{{ row.reasonText }}</small>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="latestReportedAt" label="最近举报时间" width="180" />
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <el-button v-if="row.status === 'pending'" type="primary" link @click="openReview(row)">处理</el-button>
            <span v-else>{{ reportStatusText[row.status] || row.status }}</span>
          </template>
        </el-table-column>
      </el-table>
    </section>

    <el-dialog v-model="reviewVisible" title="处理举报" width="760px">
      <el-form v-if="reviewTarget" label-position="top">
        <section class="resource-preview" v-loading="previewLoading">
          <div class="preview-heading">
            <div class="preview-title-group">
              <span class="preview-kicker">供需详情预览</span>
              <strong>{{ previewTitle }}</strong>
            </div>
            <el-tag :type="resourceStatusTagType(previewStatus)">
              {{ resourceStatusText[previewStatus] || previewStatus }}
            </el-tag>
          </div>

          <div v-if="previewError" class="preview-error">
            <span>{{ previewError }}</span>
            <el-button type="primary" link @click="loadResourcePreview(reviewTarget)">重试</el-button>
          </div>
          <div v-else class="preview-body">
            <el-image
              v-if="mainPreviewImage"
              class="preview-cover"
              :src="mainPreviewImage"
              fit="cover"
              :preview-src-list="previewImages"
              preview-teleported
            />
            <div v-else class="preview-cover preview-cover-empty">暂无图片</div>
            <div class="preview-content">
              <div class="preview-meta-grid">
                <div v-for="item in previewMetaItems" :key="item.label" class="preview-meta-item">
                  <span>{{ item.label }}</span>
                  <strong>{{ item.value }}</strong>
                </div>
              </div>
              <p class="preview-description">{{ previewDescription }}</p>
              <div v-if="previewAttributeItems.length" class="preview-attributes">
                <div v-for="item in previewAttributeItems" :key="item.key || item.label" class="preview-attribute">
                  <span>{{ item.label }}</span>
                  <strong>{{ item.value }}</strong>
                </div>
              </div>
              <div v-if="previewTags.length" class="preview-tags">
                <el-tag v-for="tag in previewTags" :key="tag" type="info" effect="plain">{{ tag }}</el-tag>
              </div>
              <div v-if="previewContactItems.length" class="preview-contact">
                <div v-for="item in previewContactItems" :key="item.label" class="preview-contact-item">
                  <span>{{ item.label }}</span>
                  <strong>{{ item.value }}</strong>
                </div>
              </div>
            </div>
          </div>
        </section>

        <el-descriptions class="report-summary" title="举报摘要" :column="2" border>
          <el-descriptions-item label="举报数">{{ reviewTarget.reportCount }}</el-descriptions-item>
          <el-descriptions-item label="最新原因">{{ reportReasonText[reviewTarget.reasonCode] || reviewTarget.reasonCode }}</el-descriptions-item>
          <el-descriptions-item label="补充说明" :span="2">{{ reviewTarget.reasonText || '无' }}</el-descriptions-item>
        </el-descriptions>

        <el-form-item label="处理结论">
          <el-button-group class="review-action-group">
            <el-button
              :type="reviewForm.action === 'valid' ? 'danger' : 'default'"
              :plain="reviewForm.action !== 'valid'"
              @click="setReviewAction('valid')"
            >
              举报成立并下架
            </el-button>
            <el-button
              :type="reviewForm.action === 'invalid' ? 'primary' : 'default'"
              :plain="reviewForm.action !== 'invalid'"
              @click="setReviewAction('invalid')"
            >
              举报不成立
            </el-button>
          </el-button-group>
        </el-form-item>
        <el-form-item v-if="reviewForm.action === 'valid'" label="资源处理">
          <el-tag type="warning">下架供需信息</el-tag>
        </el-form-item>
        <el-form-item v-if="reviewForm.action === 'valid'" label="退还发布次数">
          <el-switch
            v-model="reviewForm.refundPublishQuota"
            active-text="退还 1 次"
            inactive-text="不退还"
          />
        </el-form-item>
        <el-form-item label="处理原因">
          <el-input
            v-model="reviewForm.reason"
            type="textarea"
            :rows="4"
            maxlength="300"
            show-word-limit
            placeholder="可选，不填将使用默认处理原因"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="reviewVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" :disabled="previewLoading" @click="submitReview">确认处理</el-button>
      </template>
    </el-dialog>
  </section>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage } from '../plugins/elementPlus'
import { getResource, listResourceReports, reviewResourceReport } from '../api/resource'
import { useAuthStore } from '../stores/auth'

const typeText = {
  inventory: '库存清仓',
  goods: '现货货源',
  factory: '工厂接单',
  order: '订单找厂',
  job: '招工招聘',
  rental: '出租转让',
  service: '配套服务',
}

const reportReasonText = {
  fake_info: '虚假信息',
  unreachable: '联系不上',
  inaccurate_price_quantity: '价格或数量不实',
  image_infringement: '图片侵权或盗图',
  illegal_content: '违法违规内容',
  malicious_redirect: '恶意导流',
  other: '其他',
}

const reportStatusText = {
  pending: '待处理',
  valid: '举报成立',
  invalid: '举报不成立',
}

const resourceStatusText = {
  draft: '草稿',
  pending: '内容审核中',
  manual_review: '内容审核中',
  audit_retry: '内容审核中',
  published: '已发布',
  rejected: '已驳回',
  taken_down: '已下架',
  expired: '已过期',
}

const filters = reactive({ status: 'pending' })
const rows = ref([])
const loading = ref(false)
const errorText = ref('')
const reviewVisible = ref(false)
const reviewTarget = ref(null)
const previewResource = ref(null)
const previewLoading = ref(false)
const previewError = ref('')
const submitting = ref(false)
const reviewForm = reactive(defaultReviewForm())
const auth = useAuthStore()
let previewRequestId = 0

watch(() => reviewForm.action, (action) => {
  if (action === 'invalid') {
    reviewForm.refundPublishQuota = false
  }
})

const previewStatus = computed(() => previewResource.value?.status || reviewTarget.value?.resourceStatus || '')
const previewTitle = computed(() => previewResource.value?.title || reviewTarget.value?.resourceTitle || '')
const previewImages = computed(() => Array.isArray(previewResource.value?.images) ? previewResource.value.images.filter(Boolean) : [])
const mainPreviewImage = computed(() => previewImages.value[0] || '')
const previewTags = computed(() => Array.isArray(previewResource.value?.tags) ? previewResource.value.tags.filter(Boolean) : [])
const previewDescription = computed(() => previewResource.value?.description || '商家暂未填写详细描述')
const previewAttributeItems = computed(() => {
  const items = previewResource.value?.attributeItems
  return Array.isArray(items) ? items.filter((item) => item?.label && item?.value) : []
})
const previewMetaItems = computed(() => {
  const resource = previewResource.value || {}
  return [
    { label: '商家', value: resource.merchant?.name || reviewTarget.value?.merchantName },
    { label: '类型', value: resource.typeName || typeText[resource.typeCode] || resource.typeCode },
    { label: '品类', value: resource.category },
    { label: '价格', value: resource.priceText },
    { label: '数量/产能', value: resource.quantityText },
    { label: '发布时间', value: resource.publishedAt },
    { label: '过期时间', value: resource.expiresAt },
  ].filter((item) => item.value)
})
const previewContactItems = computed(() => {
  const contact = previewResource.value?.contact || {}
  return [
    { label: '联系人', value: contact.name },
    { label: '电话', value: contact.phoneMasked },
    { label: '微信', value: contact.wechatMasked },
  ].filter((item) => item.value)
})

onMounted(loadRows)

async function loadRows() {
  loading.value = true
  errorText.value = ''
  try {
    const resp = await listResourceReports({ ...filters, page: 1, pageSize: 20 })
    rows.value = resp.items || []
  } catch {
    errorText.value = '举报列表加载失败，请重试'
  } finally {
    loading.value = false
  }
}

function openReview(row) {
  reviewTarget.value = row
  resetResourcePreview()
  Object.assign(reviewForm, defaultReviewForm())
  reviewVisible.value = true
  loadResourcePreview(row)
}

async function loadResourcePreview(row = reviewTarget.value) {
  const resourceId = row?.resourceId
  const requestId = ++previewRequestId
  previewResource.value = null
  previewError.value = ''
  if (!resourceId) {
    previewError.value = '供需信息不存在或已下架'
    return
  }
  previewLoading.value = true
  try {
    const detail = await getResource(resourceId)
    if (requestId !== previewRequestId) return
    previewResource.value = detail || null
  } catch {
    if (requestId !== previewRequestId) return
    previewError.value = '供需详情加载失败，可根据列表信息继续处理'
  } finally {
    if (requestId === previewRequestId) {
      previewLoading.value = false
    }
  }
}

async function submitReview() {
  if (!reviewTarget.value || submitting.value || previewLoading.value) return
  submitting.value = true
  try {
    const action = reviewForm.action
    const resp = await reviewResourceReport(reviewTarget.value.id, {
      action,
      resourceAction: action === 'valid' ? 'take_down' : 'none',
      refundPublishQuota: action === 'valid' && reviewForm.refundPublishQuota,
      reason: reviewForm.reason.trim(),
      reviewerId: currentOperatorId(),
    })
    ElMessage.success(resp.message || '举报已处理')
    reviewVisible.value = false
    await loadRows()
  } finally {
    submitting.value = false
  }
}

function setReviewAction(action) {
  if (action !== 'valid' && action !== 'invalid') return
  reviewForm.action = action
}

function defaultReviewForm() {
  return {
    action: 'valid',
    refundPublishQuota: false,
    reason: '',
  }
}

function resetResourcePreview() {
  previewRequestId += 1
  previewResource.value = null
  previewLoading.value = false
  previewError.value = ''
}

function resourceStatusTagType(status) {
  switch (status) {
    case 'published':
      return 'success'
    case 'pending':
    case 'manual_review':
    case 'audit_retry':
      return 'warning'
    case 'rejected':
    case 'taken_down':
      return 'danger'
    default:
      return 'info'
  }
}

function currentOperatorId() {
  return auth.user?.operatorId || ''
}
</script>

<style scoped>
.resource-preview {
  min-height: 190px;
  margin-bottom: 16px;
  padding: 16px;
  border: 1px solid #d8dde6;
  border-radius: 8px;
  background: #f8fafc;
}

.preview-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 14px;
}

.preview-title-group {
  display: grid;
  gap: 4px;
  min-width: 0;
}

.preview-kicker,
.preview-meta-item span,
.preview-contact-item span {
  color: #697586;
  font-size: 12px;
  line-height: 1.35;
}

.preview-title-group strong {
  color: #1f2933;
  font-size: 18px;
  line-height: 1.35;
  word-break: break-word;
}

.preview-body {
  display: grid;
  grid-template-columns: 180px minmax(0, 1fr);
  gap: 16px;
}

.preview-cover {
  width: 180px;
  height: 136px;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  background: #ffffff;
}

.preview-cover-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  color: #8a94a6;
  font-size: 13px;
}

.preview-content {
  min-width: 0;
}

.preview-meta-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px 16px;
}

.preview-meta-item {
  display: grid;
  gap: 2px;
  min-width: 0;
}

.preview-meta-item strong,
.preview-contact-item strong {
  color: #364152;
  font-size: 14px;
  font-weight: 600;
  line-height: 1.4;
  word-break: break-word;
}

.preview-description {
  margin: 12px 0 0;
  padding: 10px 12px;
  border-radius: 6px;
  background: #ffffff;
  color: #364152;
  line-height: 1.6;
  white-space: pre-wrap;
}

.preview-attributes {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px 12px;
  margin-top: 12px;
}

.preview-attribute {
  display: flex;
  gap: 8px;
  min-width: 0;
  color: #364152;
  font-size: 13px;
  line-height: 1.45;
}

.preview-attribute span {
  flex: 0 0 auto;
  color: #697586;
}

.preview-attribute strong {
  min-width: 0;
  font-weight: 600;
  word-break: break-word;
}

.preview-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 12px;
}

.preview-contact {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid #e4e7ed;
}

.preview-contact-item {
  display: grid;
  gap: 2px;
  min-width: 0;
}

.preview-error {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 96px;
  padding: 12px;
  border: 1px solid #fed7aa;
  border-radius: 8px;
  background: #fff7ed;
  color: #b45309;
}

.report-summary {
  margin-bottom: 16px;
}

.review-action-group {
  display: inline-flex;
}

.reason-cell {
  display: grid;
  gap: 4px;
  line-height: 1.35;
}

.reason-cell small {
  color: #6b7280;
}
</style>
