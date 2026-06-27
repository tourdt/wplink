<template>
  <section>
    <div class="page-title">
      <h2>资源审核</h2>
      <el-button type="primary">代发资源</el-button>
    </div>

    <section class="panel">
      <el-form :inline="true" class="filter-bar">
        <el-form-item label="城市站">
          <el-select model-value="zhili" style="width: 140px">
            <el-option label="织里" value="zhili" />
          </el-select>
        </el-form-item>
        <el-form-item label="资源类型">
          <el-select model-value="" placeholder="全部" style="width: 160px">
            <el-option label="全部" value="" />
            <el-option label="库存" value="inventory" />
            <el-option label="工厂产能" value="factory" />
            <el-option label="招聘" value="job" />
            <el-option label="出租" value="rental" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button>查询</el-button>
        </el-form-item>
      </el-form>

      <el-table :data="rows" stripe>
        <el-table-column prop="title" label="资源标题" min-width="220" />
        <el-table-column prop="type" label="类型" width="120" />
        <el-table-column prop="merchant" label="商家" width="180" />
        <el-table-column prop="createdAt" label="提交时间" width="180" />
        <el-table-column label="操作" width="180" fixed="right">
          <template #default>
            <el-button type="primary" link>通过</el-button>
            <el-button type="danger" link>驳回</el-button>
            <el-button link>详情</el-button>
          </template>
        </el-table-column>
      </el-table>
    </section>
  </section>
</template>

<script setup>
const rows = [
  { title: '女童春款卫衣库存整包清', type: '库存', merchant: '织里样板童装厂', createdAt: '2026-06-27 10:20' },
  { title: '承接童装小单快返', type: '工厂产能', merchant: '东区针织加工厂', createdAt: '2026-06-27 09:35' },
]
</script>
