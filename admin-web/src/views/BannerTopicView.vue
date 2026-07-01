<template>
  <section>
    <div class="page-title">
      <h2>首页运营位</h2>
      <el-button type="primary" @click="openCreate">新增配置</el-button>
    </div>

    <section class="panel">
      <el-form :inline="true" class="filter-bar">
        <el-form-item label="城市站">
          <el-select v-model="filters.cityCode" style="width: 140px">
            <el-option v-for="station in cityStationOptions" :key="station.value" :label="station.label" :value="station.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="运营位">
          <el-select v-model="filters.kind" style="width: 160px">
            <el-option v-for="item in kindOptions" :key="item.value" :label="item.label" :value="item.value" />
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
      <el-table v-loading="loading" :data="rows" stripe empty-text="暂无首页运营位配置">
        <el-table-column label="运营位" width="130">
          <template #default="{ row }">{{ kindText[row.kind] || row.kind }}</template>
        </el-table-column>
        <el-table-column prop="title" label="标题" min-width="180" />
        <el-table-column label="资源范围" width="180">
          <template #default="{ row }">{{ row.typeScope?.join('、') || '-' }}</template>
        </el-table-column>
        <el-table-column label="跳转" min-width="180">
          <template #default="{ row }">{{ jumpTypeText[row.jumpType] || row.jumpType }}：{{ targetDisplay(row) }}</template>
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
          <el-select v-model="form.cityCode" @change="loadTargetOptions">
            <el-option v-for="station in cityStationOptions" :key="station.value" :label="station.label" :value="station.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="运营位">
          <el-select v-model="form.kind" @change="handleFormKindChange">
            <el-option v-for="item in kindOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="标题">
          <el-input v-model="form.title" />
        </el-form-item>
        <el-form-item label="副标题">
          <el-input v-model="form.subtitle" />
        </el-form-item>
        <el-form-item v-if="isBannerKind" label="封面">
          <div class="cover-upload-field">
            <el-upload
              class="cover-uploader"
              accept="image/png,image/jpeg,image/webp"
              :show-file-list="false"
              :http-request="uploadCover"
              :disabled="uploadingCover"
            >
              <img v-if="form.coverUrl" class="cover-preview" :src="form.coverUrl" alt="Banner 封面" />
              <div v-else class="cover-placeholder">
                <span>{{ uploadingCover ? '上传中...' : '上传封面' }}</span>
                <small>建议比例 2.2:1</small>
              </div>
            </el-upload>
            <div class="cover-url-field">
              <el-input v-model="form.coverUrl" clearable placeholder="上传后自动生成，也可粘贴图片 URL" />
              <p>Banner 图片建议比例 2.2:1，例如 1320 x 600 px；支持 JPG、PNG、WebP。</p>
            </div>
          </div>
        </el-form-item>
        <el-form-item v-if="isBannerKind" label="资源类型范围">
          <el-select v-model="form.typeScope" multiple>
            <el-option label="库存" value="inventory" />
            <el-option label="货源" value="goods" />
            <el-option label="工厂" value="factory" />
            <el-option label="服务" value="service" />
          </el-select>
        </el-form-item>
        <el-form-item label="跳转类型">
          <el-select v-model="form.jumpType" @change="handleJumpTypeChange">
            <el-option v-for="item in visibleJumpTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="跳转目标">
          <el-input
            v-if="form.jumpType === 'webview'"
            v-model="form.jumpTarget"
            placeholder="请输入允许访问的活动网页 URL"
          />
          <el-select
            v-else-if="form.jumpType === 'internal'"
            v-model="form.jumpTarget"
            filterable
            placeholder="选择内部页面"
          >
            <el-option v-for="item in internalPageOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
          <el-select
            v-else-if="form.jumpType === 'resource'"
            v-model="form.jumpTarget"
            filterable
            remote
            reserve-keyword
            :remote-method="searchResourceTargets"
            :loading="resourceTargetLoading"
            placeholder="选择资源"
          >
            <el-option v-for="item in resourceOptions" :key="item.id" :label="resourceOptionLabel(item)" :value="item.id" />
          </el-select>
          <el-select
            v-else-if="form.jumpType === 'merchant'"
            v-model="form.jumpTarget"
            filterable
            remote
            reserve-keyword
            :remote-method="searchMerchantTargets"
            :loading="merchantTargetLoading"
            placeholder="选择商家"
          >
            <el-option v-for="item in merchantOptions" :key="item.id" :label="merchantOptionLabel(item)" :value="item.id" />
          </el-select>
          <el-select
            v-else-if="form.jumpType === 'demand'"
            v-model="form.jumpTarget"
            disabled
          >
            <el-option label="采购需求页" value="/pages/demand/index" />
          </el-select>
          <el-alert
            v-else
            title="专题会使用当前 Banner 作为落地页，无需选择跳转目标。"
            type="info"
            :closable="false"
            show-icon
          />
        </el-form-item>
        <el-form-item :label="form.kind === 'home_recommend_card' ? '角标' : '标签'">
          <el-select v-model="form.tags" multiple filterable allow-create>
            <el-option label="平台推荐" value="平台推荐" />
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
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { createBannerTopic, listBannerTopics, updateBannerTopic } from '../api/bannerTopic'
import { listMerchants } from '../api/merchant'
import { listResources } from '../api/resource'
import { uploadBannerImage } from '../api/upload'
import { cityStationOptions, defaultCityCode } from '../common/cityStations'
import { merchantTypeLabel } from '../common/merchantIdentity'

const jumpTypeText = { topic: '专题落地页', resource: '资源', merchant: '商家', demand: '需求', internal: '内部页', webview: '网页' }
const kindText = { banner: '首页 Banner', home_recommend_card: '首页推荐卡' }
const statusText = { draft: '草稿', active: '启用', disabled: '停用' }
const statusTagType = { draft: 'info', active: 'success', disabled: 'warning' }
const kindOptions = [
  { label: '首页 Banner', value: 'banner' },
  { label: '首页推荐卡', value: 'home_recommend_card' },
]
const jumpTypeOptions = [
  { label: '专题落地页', value: 'topic' },
  { label: '资源详情', value: 'resource' },
  { label: '商家主页', value: 'merchant' },
  { label: '需求入口', value: 'demand' },
  { label: '内部页面', value: 'internal' },
  { label: '活动网页', value: 'webview' },
]
const internalPageOptions = [
  { label: '首页', value: '/pages/home/index' },
  { label: '搜索页', value: '/pages/search/index' },
  { label: '发布页', value: '/pages/publish/index' },
  { label: '消息页', value: '/pages/messages/index' },
  { label: '我的', value: '/pages/my/index' },
  { label: '采购需求页', value: '/pages/demand/index' },
  { label: '我的发布', value: '/pages/my-resources/index' },
  { label: '收藏页', value: '/pages/favorites/index' },
  { label: '认证页', value: '/pages/verification/index' },
]

const filters = reactive({ cityCode: defaultCityCode, kind: 'banner', status: '' })
const rows = ref([])
const resourceOptions = ref([])
const merchantOptions = ref([])
const resourceTargetLoading = ref(false)
const merchantTargetLoading = ref(false)
const loading = ref(false)
const errorText = ref('')
const saving = ref(false)
const uploadingCover = ref(false)
const drawerVisible = ref(false)
const editingId = ref('')
const form = reactive(defaultForm())
const isBannerKind = computed(() => form.kind === 'banner')
const visibleJumpTypeOptions = computed(() => (
  form.kind === 'home_recommend_card'
    ? jumpTypeOptions.filter((item) => item.value !== 'topic')
    : jumpTypeOptions
))

onMounted(() => {
  loadRows()
  loadTargetOptions()
})

function defaultForm() {
  return {
    cityCode: defaultCityCode,
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
    errorText.value = '首页运营位配置加载失败，请重试'
  } finally {
    loading.value = false
  }
}

async function loadTargetOptions() {
  await Promise.all([searchResourceTargets(''), searchMerchantTargets('')])
}

async function searchResourceTargets(query = '') {
  resourceTargetLoading.value = true
  try {
    const resp = await listResources({ cityCode: form.cityCode, keyword: query, page: 1, pageSize: 30 })
    resourceOptions.value = resp.items || []
  } catch {
    resourceOptions.value = []
  } finally {
    resourceTargetLoading.value = false
  }
}

async function searchMerchantTargets(query = '') {
  merchantTargetLoading.value = true
  try {
    const resp = await listMerchants({ cityCode: form.cityCode, status: 'active', keyword: query, page: 1, pageSize: 30 })
    merchantOptions.value = resp.items || []
  } catch {
    merchantOptions.value = []
  } finally {
    merchantTargetLoading.value = false
  }
}

function resetForm(data = {}) {
  Object.assign(form, defaultForm(), data, { kind: data.kind || filters.kind })
  applyKindDefaults()
  applyJumpTargetDefault()
}

function openCreate() {
  editingId.value = ''
  resetForm()
  loadTargetOptions()
  drawerVisible.value = true
}

function openEdit(row) {
  editingId.value = row.id
  resetForm(row)
  loadTargetOptions()
  drawerVisible.value = true
}

function applyJumpTargetDefault() {
  if (form.jumpType === 'topic') {
    form.jumpTarget = ''
  }
  if (form.jumpType === 'demand') {
    form.jumpTarget = '/pages/demand/index'
  }
}

function applyKindDefaults() {
  if (form.kind === 'home_recommend_card') {
    form.coverUrl = ''
    form.typeScope = []
    if (form.jumpType === 'topic') {
      form.jumpType = 'internal'
      form.jumpTarget = '/pages/search/index'
    }
    if (!form.tags.length) {
      form.tags = ['平台推荐']
    }
  }
}

function handleFormKindChange() {
  applyKindDefaults()
  applyJumpTargetDefault()
}

function handleJumpTypeChange(value) {
  form.jumpTarget = ''
  if (value === 'demand') {
    form.jumpTarget = '/pages/demand/index'
  }
}

function targetDisplay(row) {
  if (row.jumpType === 'topic' && !row.jumpTarget) return '当前 Banner 专题页'
  if (row.jumpType === 'internal') return optionLabel(internalPageOptions, row.jumpTarget)
  if (row.jumpType === 'resource') return optionLabel(resourceOptions.value, row.jumpTarget, 'title')
  if (row.jumpType === 'merchant') return optionLabel(merchantOptions.value, row.jumpTarget, 'name')
  return row.jumpTarget || '-'
}

function optionLabel(options, value, labelKey = 'label') {
  const item = options.find((option) => option.value === value || option.id === value)
  return item?.[labelKey] || value || '-'
}

function resourceOptionLabel(item) {
  return `${item.title}${item.merchantName ? ` · ${item.merchantName}` : ''}`
}

function merchantOptionLabel(item) {
  return `${item.name}${item.merchantType ? ` · ${merchantTypeLabel(item.merchantType)}` : ''}`
}

function buildSubmitPayload() {
  const payload = { ...form }
  if (payload.jumpType === 'topic') {
    payload.jumpTarget = ''
  }
  if (payload.jumpType === 'demand') {
    payload.jumpTarget = '/pages/demand/index'
  }
  if (payload.kind === 'home_recommend_card') {
    payload.coverUrl = ''
    payload.typeScope = []
    if (!payload.tags.length) {
      payload.tags = ['平台推荐']
    }
  }
  return payload
}

async function uploadCover(options) {
  uploadingCover.value = true
  try {
    form.coverUrl = await uploadBannerImage(options.file)
    options.onSuccess?.({ url: form.coverUrl })
    ElMessage.success('封面已上传')
  } catch (err) {
    options.onError?.(err)
    ElMessage.error(err.message || '封面上传失败，请重试')
  } finally {
    uploadingCover.value = false
  }
}

async function submit() {
  try {
    await ElMessageBox.confirm('确认保存当前首页运营位配置吗？', '确认保存配置', {
      type: 'warning',
      confirmButtonText: '确认保存',
      cancelButtonText: '取消',
    })
  } catch {
    return
  }
  saving.value = true
  try {
    const payload = buildSubmitPayload()
    if (editingId.value) {
      await updateBannerTopic(editingId.value, payload)
    } else {
      await createBannerTopic(payload)
    }
    ElMessage.success('配置已保存')
    drawerVisible.value = false
    await loadRows()
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.cover-upload-field {
  display: grid;
  grid-template-columns: 220px minmax(0, 1fr);
  gap: 14px;
  width: 100%;
}

.cover-uploader {
  width: 220px;
}

.cover-uploader :deep(.el-upload) {
  width: 220px;
  height: 100px;
  overflow: hidden;
  border: 1px dashed #c7d0dd;
  border-radius: 8px;
  background: #f8fafc;
}

.cover-preview {
  display: block;
  width: 220px;
  height: 100px;
  object-fit: cover;
}

.cover-placeholder {
  display: grid;
  width: 100%;
  height: 100%;
  place-items: center;
  align-content: center;
  gap: 6px;
  color: #364152;
}

.cover-placeholder span {
  font-weight: 600;
}

.cover-placeholder small,
.cover-url-field p {
  color: #697586;
}

.cover-url-field {
  display: grid;
  align-content: start;
  gap: 8px;
}

.cover-url-field p {
  margin: 0;
  font-size: 13px;
  line-height: 1.5;
}
</style>
