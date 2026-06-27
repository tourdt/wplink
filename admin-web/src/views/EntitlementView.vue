<template>
  <section>
    <div class="page-title">
      <h2>权益发放</h2>
      <el-button type="primary">发放权益</el-button>
    </div>
    <section class="panel">
      <el-table :data="rows" stripe>
        <el-table-column prop="merchant" label="商家" min-width="180" />
        <el-table-column prop="type" label="权益" width="160" />
        <el-table-column prop="source" label="来源" width="140" />
        <el-table-column prop="createdAt" label="发放时间" width="180" />
      </el-table>
    </section>
  </section>
</template>

<script setup>
const rows = [
  { merchant: '织里样板童装厂', type: '置顶券 3 张', source: '认证赠送', createdAt: '2026-06-27 10:05' },
]
</script>
