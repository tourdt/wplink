<template>
  <section>
    <div class="page-title">
      <h2>操作日志</h2>
    </div>
    <section class="panel">
      <el-form :inline="true" class="filter-bar">
        <el-form-item label="对象类型">
          <el-input v-model="filters.objectType" placeholder="resource / match_case" style="width: 180px" />
        </el-form-item>
        <el-form-item label="对象 ID">
          <el-input v-model="filters.objectId" style="width: 220px" />
        </el-form-item>
        <el-form-item label="操作人 ID">
          <el-input v-model="filters.operatorId" style="width: 220px" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="loadRows(1)">查询</el-button>
        </el-form-item>
      </el-form>

      <div v-if="errorText" class="table-state table-state-error">
        <span>{{ errorText }}</span>
        <el-button type="danger" plain @click="loadRows(1)">重试</el-button>
      </div>
      <el-table v-loading="loading" :data="rows" stripe empty-text="暂无操作日志">
        <el-table-column prop="operatorId" label="操作人" min-width="220" show-overflow-tooltip />
        <el-table-column prop="operatorRole" label="角色" width="150" />
        <el-table-column prop="action" label="动作" width="170" />
        <el-table-column prop="objectType" label="对象类型" width="140" />
        <el-table-column prop="objectId" label="对象 ID" min-width="220" show-overflow-tooltip />
        <el-table-column prop="createdAt" label="时间" width="180" />
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
  </section>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { listOperationLogs } from '../api/operationLog'

const filters = reactive({
  objectType: '',
  objectId: '',
  operatorId: '',
})
const pagination = reactive({ page: 1, pageSize: 20, total: 0 })
const rows = ref([])
const loading = ref(false)
const errorText = ref('')

onMounted(loadRows)

async function loadRows(page = pagination.page) {
  pagination.page = page
  loading.value = true
  errorText.value = ''
  try {
    const resp = await listOperationLogs({
      objectType: filters.objectType.trim(),
      objectId: filters.objectId.trim(),
      operatorId: filters.operatorId.trim(),
      page: pagination.page,
      pageSize: pagination.pageSize,
    })
    rows.value = resp.items || []
    pagination.total = resp.total || 0
  } catch {
    errorText.value = '操作日志加载失败，请重试'
  } finally {
    loading.value = false
  }
}
</script>
