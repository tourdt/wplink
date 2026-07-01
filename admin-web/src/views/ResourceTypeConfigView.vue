<template>
  <section>
    <div class="page-title">
      <h2>资源类型配置</h2>
      <el-button :loading="loading" plain @click="loadConfigs">刷新</el-button>
    </div>

    <section class="panel">
      <el-alert
        title="必填字段控制商家发布或保存资源时必须补全的信息；不同资源类型可以要求不同字段。"
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
      <el-table v-loading="loading" :data="configs" stripe empty-text="暂无资源类型配置">
        <el-table-column prop="typeName" label="类型名称" width="140" />
        <el-table-column prop="typeCode" label="编码" width="140" />
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

    <el-drawer v-model="drawerVisible" title="编辑资源类型配置" size="560px">
      <el-form v-if="editing" label-position="top">
        <el-form-item label="类型">
          <el-input :model-value="`${editing.typeName}（${editing.typeCode}）`" disabled />
        </el-form-item>
        <el-form-item label="默认有效期">
          <el-input-number v-model="editing.defaultValidDays" :min="1" :max="365" />
        </el-form-item>
        <section class="required-field-note">
          <h3>必填字段说明</h3>
          <p>这些字段决定商家发布或保存该类型资源时，哪些信息必须填写完整。</p>
          <dl>
            <template v-for="field in editing.requiredFields || []" :key="field">
              <dt>{{ fieldLabel(field) }}（{{ field }}）</dt>
              <dd>{{ fieldDescription(field) }}</dd>
            </template>
          </dl>
        </section>
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
const fieldDescriptionMap = {
  merchantId: {
    label: '商家',
    description: '指定资源归属的商家，用于商家主页展示、权益校验和后台追溯。',
  },
  cityCode: {
    label: '城市站',
    description: '决定资源发布到哪个城市站，影响搜索、推荐和专题筛选范围。',
  },
  typeCode: {
    label: '资源类型',
    description: '决定资源属于库存、货源、工厂、服务等哪一类，影响发布表单、搜索筛选和专题展示。',
  },
  title: {
    label: '标题',
    description: '标题用于搜索、列表卡片和详情页主标题，应该直接说明资源卖点。',
  },
  category: {
    label: '品类',
    description: '说明资源所属品类，例如童装、卫衣、套装，用于买家筛选和运营审核。',
  },
  quantityText: {
    label: '数量/产能',
    description: '说明库存数量、可供货数量或工厂产能，帮助买家判断是否匹配需求。',
  },
  priceText: {
    label: '价格描述',
    description: '说明价格、报价方式或费用范围，减少无效咨询。',
  },
  contactName: {
    label: '联系人',
    description: '买家和平台审核联系资源发布人的姓名或称呼。',
  },
  contactPhone: {
    label: '联系电话',
    description: '联系电话用于买家联系和平台审核核验。',
  },
  description: {
    label: '资源描述',
    description: '补充资源细节、交易条件和注意事项，帮助审核和买家理解资源。',
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

function fieldLabel(field) {
  return fieldDescriptionMap[field]?.label || field
}

function fieldDescription(field) {
  return fieldDescriptionMap[field]?.description || '该字段是发布此类资源时必须填写的信息。'
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

<style scoped>
.config-note {
  margin-bottom: 12px;
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
</style>
