<template>
  <section>
    <div class="page-title">
      <h2>供需类型配置</h2>
      <el-button :loading="loading" plain @click="loadConfigs">刷新</el-button>
    </div>

    <section class="panel">
      <el-alert
        title="必填字段控制商家发布或保存供需信息时必须补全的信息；不同供需类型可以要求不同字段。"
        type="info"
        :closable="false"
        show-icon
        class="config-note"
      />
      <el-form :inline="true" class="filter-bar">
        <el-form-item label="城市站">
          <el-select v-model="filters.cityCode" style="width: 140px">
            <el-option v-for="station in cityStationOptions" :key="station.value" :label="station.label" :value="station.value" />
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
      <el-table v-loading="loading" :data="configs" stripe empty-text="暂无供需类型配置">
        <el-table-column prop="typeName" label="类型名称" width="140" />
        <el-table-column prop="typeCode" label="编码" width="140" />
        <el-table-column label="类型归属" width="110">
          <template #default="{ row }">
            <el-tag :type="directionTagType(row.direction)" effect="plain">
              {{ directionLabel(row.direction) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="defaultValidDays" label="有效期" width="100">
          <template #default="{ row }">{{ row.defaultValidDays }} 天</template>
        </el-table-column>
        <el-table-column label="必填字段" min-width="220">
          <template #default="{ row }">
            <template v-if="row.requiredFields?.length">
              <el-tooltip
                v-for="field in row.requiredFields"
                :key="field"
                :content="fieldDescription(field)"
                placement="top"
              >
                <el-tag class="field-tag" size="small">
                  {{ fieldLabel(field) }}
                </el-tag>
              </el-tooltip>
            </template>
            <span v-else>-</span>
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

    <el-drawer v-model="drawerVisible" title="编辑供需类型配置" size="820px">
      <el-form v-if="editing" label-position="top">
        <el-form-item label="类型">
          <div class="type-summary">
            <el-input :model-value="`${editing.typeName}（${editing.typeCode}）`" disabled />
            <el-tag :type="directionTagType(editing.direction)" effect="plain">
              {{ directionLabel(editing.direction) }}
            </el-tag>
          </div>
        </el-form-item>
        <div class="basic-config-grid">
          <el-form-item label="默认有效期">
            <el-input-number v-model="editing.defaultValidDays" :min="1" :max="365" />
          </el-form-item>
          <el-form-item label="状态">
            <el-select v-model="editing.status">
              <el-option label="启用" value="active" />
              <el-option label="停用" value="disabled" />
            </el-select>
          </el-form-item>
        </div>
        <section class="required-field-note">
          <h3>必填字段说明</h3>
          <p>这些字段决定商家发布或保存该类型供需信息时，哪些信息必须填写完整。</p>
          <dl>
            <template v-for="field in editing.requiredFields || []" :key="field">
              <dt>{{ fieldLabel(field) }}（{{ field }}）</dt>
              <dd>{{ fieldDescription(field) }}</dd>
            </template>
          </dl>
        </section>

        <el-form-item label="配置方式">
          <el-radio-group v-model="editorMode" @change="handleEditorModeChange">
            <el-radio-button value="visual">可视化配置</el-radio-button>
            <el-radio-button value="json">高级 JSON</el-radio-button>
          </el-radio-group>
        </el-form-item>

        <section v-if="editorMode === 'visual'" class="field-config-editor">
          <div class="field-config-head">
            <h3>字段配置</h3>
            <el-button type="primary" plain @click="addField">新增字段</el-button>
          </div>
          <el-form-item label="基础必填字段">
            <el-select v-model="baseRequiredFields" multiple filterable collapse-tags collapse-tags-tooltip>
              <el-option
                v-for="field in baseRequiredFieldOptions"
                :key="field.value"
                :label="field.label"
                :value="field.value"
              />
            </el-select>
          </el-form-item>
          <el-empty v-if="!fieldRows.length" description="暂无动态字段" :image-size="80" />
          <div v-else class="field-row-list">
            <section v-for="(field, index) in fieldRows" :key="field.id" class="field-config-row">
              <div class="field-row-head">
                <strong>字段 {{ index + 1 }}</strong>
                <div class="field-row-actions">
                  <el-button link :disabled="index === 0" @click="moveField(index, -1)">上移</el-button>
                  <el-button link :disabled="index === fieldRows.length - 1" @click="moveField(index, 1)">下移</el-button>
                  <el-button type="danger" link @click="removeField(index)">删除</el-button>
                </div>
              </div>
              <div class="field-grid">
                <el-form-item label="字段编码">
                  <el-input v-model.trim="field.key" placeholder="例如：season" />
                </el-form-item>
                <el-form-item label="字段名称">
                  <el-input v-model.trim="field.label" placeholder="例如：季节" />
                </el-form-item>
                <el-form-item label="字段类型">
                  <el-select v-model="field.type" @change="handleFieldTypeChange(field)">
                    <el-option v-for="item in fieldTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
                  </el-select>
                </el-form-item>
                <el-form-item label="占位提示">
                  <el-input v-model.trim="field.placeholder" placeholder="例如：请选择季节" />
                </el-form-item>
              </div>
              <div class="field-switches">
                <el-checkbox v-model="field.required">设为必填</el-checkbox>
                <el-checkbox v-model="field.filterable">可筛选</el-checkbox>
                <el-checkbox v-model="field.displayInList">列表展示</el-checkbox>
                <el-checkbox v-model="field.displayInDetail">详情展示</el-checkbox>
                <el-checkbox v-if="field.type === 'select'" v-model="field.allowCustom">允许手动输入</el-checkbox>
              </div>
              <el-form-item v-if="field.type === 'select'" label="下拉选项">
                <el-input
                  v-model="field.optionsText"
                  type="textarea"
                  :rows="3"
                  placeholder="每行一个选项，例如：&#10;春季&#10;夏季"
                />
              </el-form-item>
            </section>
          </div>
        </section>

        <section v-else class="advanced-config-json">
          <el-form-item label="高级 JSON">
            <el-input v-model="configJson" type="textarea" :rows="18" spellcheck="false" />
          </el-form-item>
        </section>

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
import { ElMessage, ElMessageBox } from '../plugins/elementPlus'
import { listResourceTypeConfigs, updateResourceTypeConfig } from '../api/city'
import { cityStationOptions, defaultCityCode } from '../common/cityStations'

const filters = reactive({
  cityCode: defaultCityCode,
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
const editorMode = ref('visual')
const fieldRows = ref([])
const baseRequiredFields = ref([])
const advancedConfig = ref(createEmptyAdvancedConfig())
let nextFieldRowId = 1

const fieldTypeOptions = [
  { value: 'text', label: '单行文本' },
  { value: 'select', label: '下拉选择' },
  { value: 'boolean', label: '是/否开关' },
  { value: 'number', label: '数字' },
  { value: 'textarea', label: '多行文本' },
]
const directionTextMap = {
  supply: '供应类型',
  demand: '需求类型',
}
const baseRequiredFieldOptions = [
  { value: 'title', label: '标题' },
  { value: 'description', label: '供需信息描述' },
  { value: 'contactName', label: '联系人' },
  { value: 'contactPhone', label: '联系电话' },
  { value: 'contactWechat', label: '微信号' },
  { value: 'images', label: '供需信息图片' },
  { value: 'tags', label: '标签' },
]
const summaryFieldValueSet = new Set(['category', 'quantityText', 'priceText'])
const baseRequiredFieldValueSet = new Set(baseRequiredFieldOptions.map((field) => field.value))
const fieldDescriptionMap = {
  merchantId: {
    label: '商家',
    description: '指定供需信息归属的商家，用于商家主页展示、权益校验和后台追溯。',
  },
  cityCode: {
    label: '城市站',
    description: '决定供需信息发布到哪个城市站，影响搜索、推荐和专题筛选范围。',
  },
  typeCode: {
    label: '供需类型',
    description: '决定供需信息属于库存清仓、现货货源、工厂接单、配套服务等哪一类，影响发布表单、搜索筛选和专题展示。',
  },
  title: {
    label: '标题',
    description: '标题用于搜索、列表卡片和详情页主标题，应该直接说明供需信息卖点。',
  },
  category: {
    label: '品类',
    description: '说明供需信息所属品类，例如童装、卫衣、套装，用于买家筛选和运营审核。',
  },
  quantityText: {
    label: '数量/产能',
    description: '说明库存数量、可供货数量或工厂可接单规模，帮助买家判断是否匹配需求。',
  },
  priceText: {
    label: '价格描述',
    description: '说明价格、报价方式或费用范围，减少无效咨询。',
  },
  contactName: {
    label: '联系人',
    description: '买家和平台审核联系供需信息发布人的姓名或称呼。',
  },
  contactPhone: {
    label: '联系电话',
    description: '联系电话用于买家联系和平台审核核验。',
  },
  contactWechat: {
    label: '微信号',
    description: '微信号用于买家补充联系，适合电话不便接听的场景。',
  },
  description: {
    label: '供需信息描述',
    description: '补充供需信息细节、交易条件和注意事项，帮助审核和买家理解。',
  },
  images: {
    label: '供需信息图片',
    description: '图片用于展示货品、厂房、服务案例或环境，提升买家判断效率。',
  },
  tags: {
    label: '标签',
    description: '标签用于补充供需信息特征，方便运营归类和买家快速识别。',
  },
}

onMounted(loadConfigs)

async function loadConfigs() {
  loading.value = true
  errorText.value = ''
  try {
    const resp = await listResourceTypeConfigs(filters)
    configs.value = resp.items || []
  } catch {
    errorText.value = '供需类型配置加载失败，请重试'
  } finally {
    loading.value = false
  }
}

function openEditor(row) {
  editing.value = { ...row }
  editorMode.value = 'visual'
  fieldRows.value = normalizeFieldRowsFromSchema(row)
  const dynamicKeys = new Set(fieldRows.value.map((field) => field.key))
  baseRequiredFields.value = normalizeStringList(row.requiredFields).filter((field) => {
    return baseRequiredFieldValueSet.has(field) && !summaryFieldValueSet.has(field) && !dynamicKeys.has(field)
  })
  advancedConfig.value = {
    displayTemplate: cloneConfig(row.displayTemplate || {}),
    reviewRules: cloneConfig(row.reviewRules || {}),
    sortWeights: cloneConfig(row.sortWeights || {}),
    messageRules: cloneConfig(row.messageRules || {}),
  }
  syncConfigJsonFromEditor()
  jsonError.value = ''
  drawerVisible.value = true
}

function fieldLabel(field) {
  return fieldDescriptionMap[field]?.label || field
}

function fieldDescription(field) {
  return fieldDescriptionMap[field]?.description || '该字段是发布此类供需信息时必须填写的信息。'
}

function directionLabel(direction) {
  return directionTextMap[direction] || '供应类型'
}

function directionTagType(direction) {
  return direction === 'demand' ? 'warning' : 'success'
}

function normalizeFieldRowsFromSchema(row) {
  const schemaFields = Array.isArray(row.fieldSchema?.fields) ? row.fieldSchema.fields : []
  const requiredSet = new Set(normalizeStringList(row.requiredFields))
  const filterSet = new Set(normalizeStringList(row.filterFields))
  const listSet = new Set(normalizeStringList(row.displayTemplate?.list))
  const detailSet = new Set(normalizeStringList(row.displayTemplate?.detail))

  return schemaFields
    .map((field) => {
      const key = String(field?.key || '').trim()
      const displayIn = Array.isArray(field?.displayIn) ? field.displayIn : []
      return {
        id: nextFieldRowId++,
        key,
        label: String(field?.label || key).trim(),
        type: normalizeFieldType(field?.type),
        placeholder: String(field?.placeholder || '').trim(),
        required: field?.required === true || requiredSet.has(key),
        filterable: field?.filterable === true || filterSet.has(key),
        displayInList: displayIn.includes('list') || listSet.has(key),
        displayInDetail: displayIn.includes('detail') || detailSet.has(key),
        allowCustom: field?.allowCustom === true,
        optionsText: normalizeStringList(field?.options).join('\n'),
      }
    })
    .filter((field) => field.key || field.label)
}

function normalizeStringList(value) {
  if (!Array.isArray(value)) return []
  return Array.from(new Set(value.map((item) => String(item || '').trim()).filter(Boolean)))
}

function normalizeFieldType(type) {
  const normalized = String(type || 'text').trim()
  return fieldTypeOptions.some((item) => item.value === normalized) ? normalized : 'text'
}

function createEmptyAdvancedConfig() {
  return {
    displayTemplate: {},
    reviewRules: {},
    sortWeights: {},
    messageRules: {},
  }
}

function cloneConfig(value) {
  return JSON.parse(JSON.stringify(value || {}))
}

function addField() {
  fieldRows.value.push({
    id: nextFieldRowId++,
    key: '',
    label: '',
    type: 'text',
    placeholder: '',
    required: false,
    filterable: false,
    displayInList: false,
    displayInDetail: true,
    allowCustom: false,
    optionsText: '',
  })
}

function removeField(index) {
  fieldRows.value.splice(index, 1)
}

function moveField(index, direction) {
  const targetIndex = index + direction
  if (targetIndex < 0 || targetIndex >= fieldRows.value.length) return
  const [field] = fieldRows.value.splice(index, 1)
  fieldRows.value.splice(targetIndex, 0, field)
}

function handleFieldTypeChange(field) {
  if (field.type !== 'select') {
    field.allowCustom = false
    field.optionsText = ''
  }
}

function handleEditorModeChange(mode) {
  jsonError.value = ''
  if (mode === 'json') {
    const payload = buildConfigPayloadFromVisualEditor()
    if (!payload) {
      editorMode.value = 'visual'
      return
    }
    configJson.value = JSON.stringify(payload, null, 2)
    return
  }
  if (!syncEditorFromConfigJson()) {
    editorMode.value = 'json'
  }
}

function syncEditorFromConfigJson() {
  let parsed
  try {
    parsed = JSON.parse(configJson.value)
  } catch {
    jsonError.value = '配置 JSON 格式不正确，请检查后再切换'
    return false
  }
  const row = {
    fieldSchema: parsed.fieldSchema || {},
    requiredFields: parsed.requiredFields || [],
    filterFields: parsed.filterFields || [],
    displayTemplate: parsed.displayTemplate || {},
    reviewRules: parsed.reviewRules || {},
    sortWeights: parsed.sortWeights || {},
    messageRules: parsed.messageRules || {},
  }
  fieldRows.value = normalizeFieldRowsFromSchema(row)
  const dynamicKeys = new Set(fieldRows.value.map((field) => field.key))
  baseRequiredFields.value = normalizeStringList(row.requiredFields).filter((field) => {
    return baseRequiredFieldValueSet.has(field) && !summaryFieldValueSet.has(field) && !dynamicKeys.has(field)
  })
  advancedConfig.value = {
    displayTemplate: cloneConfig(row.displayTemplate),
    reviewRules: cloneConfig(row.reviewRules),
    sortWeights: cloneConfig(row.sortWeights),
    messageRules: cloneConfig(row.messageRules),
  }
  jsonError.value = ''
  return true
}

function syncConfigJsonFromEditor() {
  const payload = buildConfigPayloadFromVisualEditor({ silent: true })
  configJson.value = JSON.stringify(payload, null, 2)
}

function buildConfigPayloadFromVisualEditor(options = {}) {
  const fields = []
  const seenKeys = new Set()

  for (const field of fieldRows.value) {
    const key = String(field.key || '').trim()
    const label = String(field.label || '').trim()
    if (!key || !label) {
      return failBuildConfig('请先补全字段编码和字段名称', options)
    }
    if (!/^[A-Za-z][A-Za-z0-9_]*$/.test(key)) {
      return failBuildConfig(`字段「${label}」的编码只能使用英文字母、数字和下划线，并且必须以字母开头`, options)
    }
    if (seenKeys.has(key)) {
      return failBuildConfig(`字段编码重复：${key}`, options)
    }
    seenKeys.add(key)

    const type = normalizeFieldType(field.type)
    const configField = {
      key,
      label,
      type,
      required: field.required === true,
      filterable: field.filterable === true,
      displayIn: [
        ...(field.displayInList ? ['list'] : []),
        ...(field.displayInDetail ? ['detail'] : []),
      ],
    }
    const placeholder = String(field.placeholder || '').trim()
    if (placeholder) {
      configField.placeholder = placeholder
    }
    if (type === 'select') {
      const optionsList = normalizeOptionsText(field.optionsText)
      if (!field.allowCustom && !optionsList.length) {
        return failBuildConfig(`请为下拉字段「${label}」配置选项，或开启允许手动输入`, options)
      }
      configField.options = optionsList
      configField.allowCustom = field.allowCustom === true
    }
    fields.push(configField)
  }

  const requiredFields = normalizeStringList([
    ...baseRequiredFields.value,
    ...fields.filter((field) => field.required).map((field) => field.key),
  ])
  const displayTemplate = cloneConfig(advancedConfig.value.displayTemplate)
  const dynamicFieldKeys = new Set(fields.map((field) => field.key))
  const baseListFields = normalizeStringList(displayTemplate.list).filter((field) => !dynamicFieldKeys.has(field))
  const dynamicListFields = fields.filter((field) => field.displayIn.includes('list')).map((field) => field.key)
  displayTemplate.list = normalizeStringList([...baseListFields, ...dynamicListFields])
  displayTemplate.detail = fields.filter((field) => field.displayIn.includes('detail')).map((field) => field.key)

  return {
    fieldSchema: { fields },
    requiredFields,
    filterFields: fields.filter((field) => field.filterable).map((field) => field.key),
    displayTemplate,
    reviewRules: cloneConfig(advancedConfig.value.reviewRules),
    sortWeights: cloneConfig(advancedConfig.value.sortWeights),
    messageRules: cloneConfig(advancedConfig.value.messageRules),
  }
}

function failBuildConfig(message, options = {}) {
  if (!options.silent) {
    jsonError.value = message
    ElMessage.warning(message)
  }
  return null
}

function normalizeOptionsText(value) {
  return normalizeStringList(String(value || '').split('\n'))
}

function parseConfigJsonPayload() {
  try {
    const parsed = JSON.parse(configJson.value)
    return {
      fieldSchema: parsed.fieldSchema || {},
      requiredFields: normalizeStringList(parsed.requiredFields),
      filterFields: normalizeStringList(parsed.filterFields),
      displayTemplate: parsed.displayTemplate || {},
      reviewRules: parsed.reviewRules || {},
      sortWeights: parsed.sortWeights || {},
      messageRules: parsed.messageRules || {},
    }
  } catch {
    jsonError.value = '配置 JSON 格式不正确，请检查后再保存'
    return null
  }
}

async function saveConfig() {
  jsonError.value = ''
  const parsed = editorMode.value === 'visual' ? buildConfigPayloadFromVisualEditor() : parseConfigJsonPayload()
  if (!parsed) {
    return
  }
  try {
    await ElMessageBox.confirm(`确认保存「${editing.value.typeName}」的供需类型配置吗？`, '确认保存配置', {
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
    ElMessage.success('供需类型配置已保存')
    drawerVisible.value = false
    await loadConfigs()
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.config-note {
  margin-bottom: 12px;
}

.basic-config-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.type-summary {
  display: flex;
  align-items: center;
  gap: 10px;
}

.type-summary :deep(.el-input) {
  flex: 1;
}

.required-field-note {
  margin-bottom: 18px;
  padding: 14px 16px;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  background: #f8fafc;
}

.required-field-note h3 {
  margin: 0 0 6px;
  font-size: 15px;
}

.required-field-note p {
  margin: 0 0 12px;
  color: #697586;
  line-height: 1.5;
}

.required-field-note dl {
  display: grid;
  gap: 8px;
  margin: 0;
}

.required-field-note dt {
  color: #1f2933;
  font-weight: 700;
}

.required-field-note dd {
  margin: -4px 0 0;
  color: #697586;
  line-height: 1.5;
}

.field-config-editor,
.advanced-config-json {
  margin-bottom: 18px;
}

.field-config-head,
.field-row-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.field-config-head {
  margin-bottom: 12px;
}

.field-config-head h3 {
  margin: 0;
  font-size: 15px;
}

.field-row-list {
  display: grid;
  gap: 12px;
}

.field-config-row {
  padding: 14px;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  background: #ffffff;
}

.field-row-head {
  margin-bottom: 12px;
}

.field-row-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.field-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.field-switches {
  display: flex;
  flex-wrap: wrap;
  gap: 10px 18px;
  margin-bottom: 12px;
}

@media (max-width: 720px) {
  .basic-config-grid,
  .field-grid {
    grid-template-columns: 1fr;
  }
}
</style>
