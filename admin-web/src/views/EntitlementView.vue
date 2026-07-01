<template>
  <section>
    <div class="page-title">
      <h2>权益发放</h2>
      <el-button type="primary" @click="drawerVisible = true">发放权益</el-button>
    </div>
    <section class="panel">
      <div v-if="errorText" class="table-state table-state-error">
        <span>{{ errorText }}</span>
        <el-button type="danger" plain @click="loadRows">重试</el-button>
      </div>
      <el-table v-loading="loading" :data="rows" stripe empty-text="暂无可发放商家">
        <el-table-column prop="name" label="商家" min-width="180" />
        <el-table-column label="主要身份" width="140">
          <template #default="{ row }">{{ merchantTypeText[row.merchantType] || row.merchantType }}</template>
        </el-table-column>
        <el-table-column label="认证状态" width="120">
          <template #default="{ row }">{{ row.verificationStatus === 'verified' ? '已认证' : '未认证' }}</template>
        </el-table-column>
        <el-table-column label="状态" width="120">
          <template #default="{ row }">{{ row.status === 'active' ? '正常' : row.status }}</template>
        </el-table-column>
      </el-table>
    </section>

    <el-drawer v-model="drawerVisible" title="发放权益" size="480px">
      <el-form label-position="top">
        <el-form-item label="商家">
          <el-select v-model="form.merchantId" filterable>
            <el-option v-for="item in rows" :key="item.id" :label="item.name" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="权益类型">
          <el-select v-model="form.entitlementType">
            <el-option label="发布额度" value="publish_quota" />
            <el-option label="刷新额度" value="refresh_quota" />
          </el-select>
        </el-form-item>
        <el-form-item label="发放数量">
          <el-input-number v-model="form.totalAmount" :min="1" />
        </el-form-item>
        <el-form-item label="来源">
          <el-input v-model="form.sourceType" />
        </el-form-item>
        <el-form-item label="原因">
          <el-input v-model="form.reason" type="textarea" :rows="3" />
        </el-form-item>
        <div class="drawer-actions">
          <el-button @click="drawerVisible = false">取消</el-button>
          <el-button type="primary" :loading="saving" @click="submitGrant">保存</el-button>
        </div>
      </el-form>
    </el-drawer>
  </section>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { grantMerchantEntitlement } from '../api/entitlement'
import { listMerchants } from '../api/merchant'
import { defaultCityCode } from '../common/cityStations'
import { merchantTypeText } from '../common/merchantIdentity'
import { useAuthStore } from '../stores/auth'

const rows = ref([])
const loading = ref(false)
const errorText = ref('')
const saving = ref(false)
const drawerVisible = ref(false)
const form = reactive({
  merchantId: '',
  entitlementType: 'publish_quota',
  sourceType: 'manual',
  totalAmount: 10,
  reason: '',
})
const auth = useAuthStore()

onMounted(loadRows)

async function loadRows() {
  loading.value = true
  errorText.value = ''
  try {
    const resp = await listMerchants({ cityCode: defaultCityCode, page: 1, pageSize: 50 })
    rows.value = resp.items || []
  } catch {
    errorText.value = '商家列表加载失败，请重试'
  } finally {
    loading.value = false
  }
}

async function submitGrant() {
  if (!form.merchantId || !form.reason.trim()) {
    ElMessage.warning('请选择商家并填写原因')
    return
  }
  const merchantName = rows.value.find((item) => item.id === form.merchantId)?.name || '当前商家'
  try {
    await ElMessageBox.confirm(`确认向「${merchantName}」发放 ${form.totalAmount} 个权益额度吗？`, '确认发放权益', {
      type: 'warning',
      confirmButtonText: '确认发放',
      cancelButtonText: '取消',
    })
  } catch {
    return
  }
  saving.value = true
  try {
    await grantMerchantEntitlement(form.merchantId, {
      operatorId: auth.user?.userId || '',
      entitlementType: form.entitlementType,
      sourceType: form.sourceType,
      totalAmount: form.totalAmount,
      reason: form.reason.trim(),
    })
    ElMessage.success('权益已发放')
    drawerVisible.value = false
  } finally {
    saving.value = false
  }
}
</script>
