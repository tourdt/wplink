<template>
  <section>
    <div class="page-title">
      <h2>商家管理</h2>
      <el-button type="primary" @click="drawerVisible = true">新增商家</el-button>
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
        <el-form-item>
          <el-button type="primary" @click="loadMerchants">查询</el-button>
        </el-form-item>
      </el-form>

      <div v-if="errorText" class="table-state table-state-error">
        <span>{{ errorText }}</span>
        <el-button type="danger" plain @click="loadMerchants">重试</el-button>
      </div>
      <el-table v-loading="loading" :data="rows" stripe empty-text="暂无商家数据">
        <el-table-column prop="name" label="商家名称" min-width="180" />
        <el-table-column label="主要身份" width="120">
          <template #default="{ row }">{{ merchantTypeText[row.merchantType] || row.merchantType }}</template>
        </el-table-column>
        <el-table-column label="认证状态" width="120">
          <template #default="{ row }">{{ verificationText[row.verificationStatus] || row.verificationStatus }}</template>
        </el-table-column>
        <el-table-column label="商家状态" width="120">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'success' : 'warning'">
              {{ row.status === 'active' ? '正常' : row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="lastActiveAt" label="最近活跃" width="180" />
        <el-table-column label="操作" width="160">
          <template #default="{ row }">
            <el-button type="primary" link @click="openDetail(row)">查看</el-button>
            <el-button link @click="openEntitlement">发券</el-button>
          </template>
        </el-table-column>
      </el-table>
    </section>

    <el-drawer v-model="detailVisible" title="商家详情" size="520px">
      <el-descriptions v-if="detail" :column="1" border>
        <el-descriptions-item label="商家名称">{{ detail.name }}</el-descriptions-item>
        <el-descriptions-item label="主要身份">{{ merchantTypeText[detail.merchantType] || detail.merchantType }}</el-descriptions-item>
        <el-descriptions-item label="主营品类">{{ detail.mainCategories?.join('、') }}</el-descriptions-item>
        <el-descriptions-item label="联系人">{{ detail.contact?.name }}</el-descriptions-item>
        <el-descriptions-item label="电话">{{ detail.contact?.phoneMasked }}</el-descriptions-item>
        <el-descriptions-item label="微信">{{ detail.contact?.wechatMasked }}</el-descriptions-item>
        <el-descriptions-item label="发布概况">
          已发布 {{ detail.resourcesSummary?.publishedCount || 0 }}，已成交 {{ detail.resourcesSummary?.dealtCount || 0 }}
        </el-descriptions-item>
      </el-descriptions>
    </el-drawer>

    <el-drawer v-model="drawerVisible" title="新增商家" size="520px">
      <el-form label-position="top">
        <el-form-item label="城市站">
          <el-select v-model="form.cityCode">
            <el-option v-for="station in cityStationOptions" :key="station.value" :label="station.label" :value="station.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="商家名称">
          <el-input v-model="form.name" />
        </el-form-item>
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
        <el-form-item label="联系人">
          <el-input v-model="form.contactName" />
        </el-form-item>
        <el-form-item label="联系电话">
          <el-input v-model="form.contactPhone" />
        </el-form-item>
        <el-form-item label="微信">
          <el-input v-model="form.contactWechat" />
        </el-form-item>
        <el-form-item label="地址">
          <el-input v-model="form.addressText" />
        </el-form-item>
        <el-form-item label="简介">
          <el-input v-model="form.description" type="textarea" :rows="3" />
        </el-form-item>
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
import { ElMessage } from 'element-plus'
import { createMerchant, getMerchant, listMerchants } from '../api/merchant'
import { cityStationOptions, defaultCityCode } from '../common/cityStations'
import { merchantIdentityOptions, merchantTypeText } from '../common/merchantIdentity'
const verificationText = {
  unverified: '未认证',
  pending: '待审核',
  verified: '已认证',
  rejected: '已驳回',
  expired: '已过期',
}

const filters = reactive({
  cityCode: defaultCityCode,
  merchantType: '',
})
const form = reactive({
  cityCode: defaultCityCode,
  name: '',
  merchantType: 'factory',
  mainCategories: ['童装'],
  contactName: '',
  contactPhone: '',
  contactWechat: '',
  addressText: '',
  description: '',
})

const rows = ref([])
const detail = ref(null)
const loading = ref(false)
const errorText = ref('')
const saving = ref(false)
const drawerVisible = ref(false)
const detailVisible = ref(false)
const router = useRouter()

onMounted(loadMerchants)

async function loadMerchants() {
  loading.value = true
  errorText.value = ''
  try {
    const resp = await listMerchants({ ...filters, page: 1, pageSize: 20 })
    rows.value = resp.items || []
  } catch {
    errorText.value = '商家列表加载失败，请重试'
  } finally {
    loading.value = false
  }
}

async function openDetail(row) {
  detail.value = await getMerchant(row.id)
  detailVisible.value = true
}

async function submitMerchant() {
  saving.value = true
  try {
    await createMerchant({ ...form })
    ElMessage.success('商家已创建')
    drawerVisible.value = false
    await loadMerchants()
  } finally {
    saving.value = false
  }
}

function openEntitlement() {
  router.push({ name: 'entitlements' })
}
</script>
