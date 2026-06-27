<template>
  <section>
    <div class="page-title">
      <h2>认证审核</h2>
    </div>
    <section class="panel">
      <el-table :data="rows" stripe>
        <el-table-column prop="merchant" label="商家" min-width="180" />
        <el-table-column prop="type" label="认证类型" width="140" />
        <el-table-column prop="createdAt" label="提交时间" width="180" />
        <el-table-column label="操作" width="180">
          <template #default>
            <el-button type="primary" link>通过</el-button>
            <el-button type="danger" link>驳回</el-button>
            <el-button link>材料</el-button>
          </template>
        </el-table-column>
      </el-table>
    </section>
  </section>
</template>

<script setup>
const rows = [
  { merchant: '织里样板童装厂', type: '工厂认证', createdAt: '2026-06-27 09:45' },
]
</script>
