<template>
  <section>
    <div class="page-title">
      <h2>采购需求</h2>
    </div>
    <section class="panel">
      <el-table :data="rows" stripe>
        <el-table-column prop="title" label="需求" min-width="220" />
        <el-table-column prop="buyer" label="提交人" width="140" />
        <el-table-column prop="status" label="状态" width="120" />
        <el-table-column prop="createdAt" label="提交时间" width="180" />
        <el-table-column label="操作" width="160">
          <template #default>
            <el-button type="primary" link>撮合</el-button>
            <el-button link>记录</el-button>
          </template>
        </el-table-column>
      </el-table>
    </section>
  </section>
</template>

<script setup>
const rows = [
  { title: '找 90-140 童装夏款现货', buyer: '张先生', status: '待撮合', createdAt: '2026-06-27 09:10' },
]
</script>
