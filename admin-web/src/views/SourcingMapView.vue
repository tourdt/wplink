<template>
  <section>
    <div class="page-title">
      <h2>拿货地图</h2>
      <div class="title-actions">
        <el-button type="primary">添加档口</el-button>
        <el-button>添加配套</el-button>
        <el-button>批量生成</el-button>
      </div>
    </div>

    <section class="panel sourcing-map-shell">
      <aside class="scene-sidebar">
        <h3>地图场景</h3>
        <div class="table-state">请选择或创建地图场景</div>
      </aside>

      <main class="map-workbench">
        <div class="map-toolbar">
          <span>标注画布</span>
          <el-tag type="info">第一期支持矩形和点位</el-tag>
        </div>
        <div class="map-canvas">
          <span>底图上传后可在此标注档口和配套点位</span>
        </div>
      </main>

      <aside class="object-panel">
        <h3>点位属性</h3>
        <div class="table-state">点击画布对象后编辑</div>
      </aside>
    </section>
  </section>
</template>

<script setup>
</script>

<style scoped>
.title-actions {
  display: flex;
  gap: 8px;
}

.sourcing-map-shell {
  display: grid;
  grid-template-columns: 260px minmax(0, 1fr) 320px;
  gap: 16px;
  min-height: calc(100vh - 150px);
}

.scene-sidebar,
.object-panel,
.map-workbench {
  min-width: 0;
}

.scene-sidebar h3,
.object-panel h3 {
  margin: 0 0 12px;
  font-size: 16px;
}

.map-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.map-canvas {
  position: relative;
  display: grid;
  min-height: 560px;
  place-items: center;
  overflow: hidden;
  border: 1px dashed #cdd5df;
  border-radius: 8px;
  background: #f8fafc;
  color: #697586;
}
</style>
