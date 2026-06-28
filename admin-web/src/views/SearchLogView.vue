<template>
  <section>
    <div class="page-title">
      <h2>搜索日志</h2>
    </div>
    <section class="panel">
      <el-form :inline="true" class="filter-bar">
        <el-form-item label="城市">
          <el-input v-model="filters.cityCode" placeholder="zhili" style="width: 160px" />
        </el-form-item>
        <el-form-item label="关键词">
          <el-input v-model="filters.keyword" placeholder="搜索关键词" style="width: 220px" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="loadRows(1)">查询</el-button>
        </el-form-item>
      </el-form>

      <div v-if="errorText" class="table-state table-state-error">
        <span>{{ errorText }}</span>
        <el-button type="danger" plain @click="loadRows(1)">重试</el-button>
      </div>
      <el-table v-loading="loading" :data="rows" stripe empty-text="暂无搜索日志">
        <el-table-column prop="keyword" label="关键词" min-width="180" show-overflow-tooltip />
        <el-table-column prop="cityName" label="城市" width="120" />
        <el-table-column prop="resultCount" label="结果数" width="100" />
        <el-table-column label="筛选条件" min-width="260" show-overflow-tooltip>
          <template #default="{ row }">
            {{ formatFilters(row.filters) }}
          </template>
        </el-table-column>
        <el-table-column prop="userId" label="用户 ID" min-width="220" show-overflow-tooltip />
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
import { listSearchLogs } from '../api/searchLog'

const filters = reactive({
  cityCode: 'zhili',
  keyword: '',
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
    const resp = await listSearchLogs({
      cityCode: filters.cityCode.trim(),
      keyword: filters.keyword.trim(),
      page: pagination.page,
      pageSize: pagination.pageSize,
    })
    rows.value = resp.items || []
    pagination.total = resp.total || 0
  } catch {
    errorText.value = '搜索日志加载失败，请重试'
  } finally {
    loading.value = false
  }
}

function formatFilters(value) {
  if (!value || Object.keys(value).length === 0) {
    return '-'
  }
  return JSON.stringify(value)
}
</script>
