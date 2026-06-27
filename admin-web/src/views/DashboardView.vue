<template>
  <section>
    <div class="page-title">
      <h2>数据概览</h2>
      <el-button type="primary" plain>刷新</el-button>
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
      <el-table :data="tasks" stripe>
        <el-table-column prop="type" label="类型" width="140" />
        <el-table-column prop="title" label="内容" />
        <el-table-column prop="city" label="城市站" width="120" />
        <el-table-column prop="createdAt" label="提交时间" width="180" />
      </el-table>
    </section>
  </section>
</template>

<script setup>
const metrics = [
  { label: '待审核资源', value: '18', trend: '今日新增 6' },
  { label: '待认证商家', value: '7', trend: '本周新增 12' },
  { label: '采购需求', value: '25', trend: '今日新增 9' },
  { label: '联系次数', value: '138', trend: '较昨日 +16' },
]

const tasks = [
  { type: '资源审核', title: '女童春款卫衣库存整包清', city: '织里', createdAt: '2026-06-27 10:20' },
  { type: '认证审核', title: '织里样板童装厂', city: '织里', createdAt: '2026-06-27 09:45' },
  { type: '采购需求', title: '找 90-140 童装夏款现货', city: '织里', createdAt: '2026-06-27 09:10' },
]
</script>
