<template>
  <section>
    <div class="page-title">
      <h2>操作日志</h2>
    </div>
    <section class="panel">
      <el-table :data="rows" stripe>
        <el-table-column prop="operator" label="操作人" width="140" />
        <el-table-column prop="action" label="动作" width="160" />
        <el-table-column prop="object" label="对象" min-width="220" />
        <el-table-column prop="createdAt" label="时间" width="180" />
      </el-table>
    </section>
  </section>
</template>

<script setup>
const rows = [
  { operator: '运营A', action: '审核通过', object: '女童春款卫衣库存整包清', createdAt: '2026-06-27 10:30' },
]
</script>
