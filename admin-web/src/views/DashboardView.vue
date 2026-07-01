<template>
  <section>
    <div class="page-title">
      <h2>数据概览</h2>
      <el-button type="primary" plain :loading="loading" @click="loadOverview">刷新</el-button>
    </div>

    <div class="metric-grid">
      <article v-for="item in metrics" :key="item.label" class="metric-card">
        <span>{{ item.label }}</span>
        <strong>{{ item.value }}</strong>
        <small>{{ item.trend }}</small>
      </article>
    </div>

    <section class="panel">
      <div class="panel-header">
        <h3>待处理事项</h3>
      </div>
      <div v-if="errorText" class="table-state table-state-error">
        <span>{{ errorText }}</span>
        <el-button type="danger" plain @click="loadOverview">重试</el-button>
      </div>
      <el-table v-loading="loading" :data="tasks" stripe empty-text="暂无待处理事项">
        <el-table-column prop="type" label="类型" width="140" />
        <el-table-column prop="title" label="内容" />
        <el-table-column prop="cityName" label="城市站" width="120" />
        <el-table-column prop="createdAt" label="提交时间" width="180" />
      </el-table>
    </section>
  </section>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { getDashboardOverview } from '../api/dashboard'
import { defaultCityCode } from '../common/cityStations'

const loading = ref(false)
const errorText = ref('')
const overview = ref({
  metrics: {
    pendingResourceCount: 0,
    pendingVerificationCount: 0,
    pendingDemandCount: 0,
    todayContactCount: 0,
  },
  tasks: [],
})

const metrics = computed(() => [
  { label: '待审核资源', value: overview.value.metrics.pendingResourceCount, trend: '待运营处理' },
  { label: '待认证商家', value: overview.value.metrics.pendingVerificationCount, trend: '待运营审核' },
  { label: '采购需求', value: overview.value.metrics.pendingDemandCount, trend: '待运营跟进' },
  { label: '联系次数', value: overview.value.metrics.todayContactCount, trend: '今日联系行为' },
])
const tasks = computed(() => overview.value.tasks || [])

onMounted(loadOverview)

async function loadOverview() {
  loading.value = true
  errorText.value = ''
  try {
    overview.value = await getDashboardOverview({ cityCode: defaultCityCode })
  } catch {
    errorText.value = '数据概览加载失败，请重试'
  } finally {
    loading.value = false
  }
}
</script>
