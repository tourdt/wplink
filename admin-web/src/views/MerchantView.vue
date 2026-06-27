<template>
  <section>
    <div class="page-title">
      <h2>商家管理</h2>
      <el-button type="primary">新增商家</el-button>
    </div>
    <section class="panel">
      <el-table :data="rows" stripe>
        <el-table-column prop="name" label="商家名称" min-width="180" />
        <el-table-column prop="type" label="类型" width="120" />
        <el-table-column prop="status" label="认证状态" width="120" />
        <el-table-column prop="activeAt" label="最近活跃" width="180" />
        <el-table-column label="操作" width="160">
          <template #default>
            <el-button type="primary" link>查看</el-button>
            <el-button link>发券</el-button>
          </template>
        </el-table-column>
      </el-table>
    </section>
  </section>
</template>

<script setup>
const rows = [
  { name: '织里样板童装厂', type: '工厂', status: '已认证', activeAt: '2026-06-27 10:00' },
  { name: '中区库存档口', type: '库存商', status: '待认证', activeAt: '2026-06-26 18:30' },
]
</script>
