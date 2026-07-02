<template>
  <section>
    <div class="page-title">
      <h2>拿货地图</h2>
      <div class="title-actions">
        <el-button type="primary" :disabled="!selectedScene" @click="createBooth">添加档口</el-button>
        <el-button :disabled="!selectedScene" @click="createPoi">添加配套</el-button>
        <el-button :disabled="!selectedScene" @click="batchDrawerVisible = true">批量生成</el-button>
      </div>
    </div>

    <section class="panel sourcing-map-shell">
      <aside class="scene-sidebar">
        <div class="panel-heading">
          <h3>地图场景</h3>
          <el-button type="primary" link @click="newScene">新增</el-button>
        </div>

        <div v-if="sceneErrorText" class="table-state table-state-error">
          <span>{{ sceneErrorText }}</span>
          <el-button type="danger" plain @click="loadScenes">重试</el-button>
        </div>

        <el-table
          v-loading="sceneLoading"
          :data="scenes"
          size="small"
          highlight-current-row
          empty-text="暂无地图场景"
          @row-click="selectScene"
        >
          <el-table-column prop="name" label="名称" min-width="120" />
          <el-table-column label="状态" width="78">
            <template #default="{ row }">
              <el-tag :type="sceneStatusTagType[row.status] || 'info'">{{ sceneStatusText[row.status] || row.status }}</el-tag>
            </template>
          </el-table-column>
        </el-table>
      </aside>

      <main class="map-workbench">
        <div class="map-toolbar">
          <span>{{ selectedScene?.name || '标注画布' }}</span>
          <el-tag type="info">第一期支持矩形和点位</el-tag>
        </div>
        <div class="map-canvas">
          <img v-if="sceneForm.backgroundUrl" class="map-background" :src="sceneForm.backgroundUrl" alt="地图底图" />
          <span v-else>底图上传后可在此标注档口和配套点位</span>
        </div>
      </main>

      <aside class="object-panel">
        <el-tabs v-model="activePanel">
          <el-tab-pane label="场景" name="scene">
            <el-form label-position="top">
              <el-form-item label="场景编码">
                <el-input v-model="sceneForm.code" placeholder="zhili_lijilu_middle" />
              </el-form-item>
              <el-form-item label="场景名称">
                <el-input v-model="sceneForm.name" placeholder="利济路中段" />
              </el-form-item>
              <el-form-item label="场景类型">
                <el-select v-model="sceneForm.type">
                  <el-option label="总览" value="overview" />
                  <el-option label="街区" value="street" />
                  <el-option label="路段" value="street_segment" />
                  <el-option label="商场" value="mall" />
                  <el-option label="楼层" value="floor" />
                </el-select>
              </el-form-item>
              <el-form-item label="城市站">
                <el-select v-model="sceneForm.cityCode">
                  <el-option v-for="station in cityStationOptions" :key="station.value" :label="station.label" :value="station.value" />
                </el-select>
              </el-form-item>
              <el-form-item label="底图">
                <el-upload :show-file-list="false" :http-request="uploadBackground">
                  <el-button>上传底图</el-button>
                </el-upload>
                <el-input v-model="sceneForm.backgroundUrl" class="background-url-input" placeholder="底图 URL" />
              </el-form-item>
              <div class="scene-size-grid">
                <el-form-item label="宽度">
                  <el-input-number v-model="sceneForm.width" :min="1" />
                </el-form-item>
                <el-form-item label="高度">
                  <el-input-number v-model="sceneForm.height" :min="1" />
                </el-form-item>
              </div>
              <div class="scene-size-grid">
                <el-form-item label="默认缩放">
                  <el-input v-model="sceneForm.defaultScale" />
                </el-form-item>
                <el-form-item label="状态">
                  <el-select v-model="sceneForm.status">
                    <el-option label="草稿" value="draft" />
                    <el-option label="已发布" value="published" />
                    <el-option label="已归档" value="archived" />
                  </el-select>
                </el-form-item>
              </div>
              <div class="drawer-actions">
                <el-button type="primary" :loading="sceneSaving" @click="submitScene">保存场景</el-button>
                <el-button :disabled="!sceneForm.code" :loading="scenePublishing" @click="publishScene">发布</el-button>
              </div>
            </el-form>
          </el-tab-pane>
          <el-tab-pane label="点位" name="object">
            <div class="table-state">点击画布对象后编辑</div>
          </el-tab-pane>
        </el-tabs>
      </aside>
    </section>
  </section>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listMapScenes, publishMapScene, saveMapScene } from '../api/sourcingMap'
import { uploadMapBackgroundImage } from '../api/upload'
import { cityStationOptions, defaultCityCode } from '../common/cityStations'

const sceneStatusText = { draft: '草稿', published: '已发布', archived: '已归档' }
const sceneStatusTagType = { draft: 'info', published: 'success', archived: 'warning' }
const activePanel = ref('scene')
const batchDrawerVisible = ref(false)
const sceneLoading = ref(false)
const sceneSaving = ref(false)
const scenePublishing = ref(false)
const sceneErrorText = ref('')
const scenes = ref([])
const selectedSceneCode = ref('')
const sceneForm = reactive(defaultSceneForm())
const selectedScene = computed(() => scenes.value.find((scene) => scene.code === selectedSceneCode.value) || null)

onMounted(loadScenes)

function defaultSceneForm() {
  return {
    cityCode: defaultCityCode,
    code: '',
    name: '',
    type: 'street_segment',
    parentCode: '',
    backgroundUrl: '',
    width: 3000,
    height: 1800,
    defaultScale: '1',
    defaultCenterX: '',
    defaultCenterY: '',
    status: 'draft',
  }
}

function resetSceneForm(data = {}) {
  Object.assign(sceneForm, defaultSceneForm(), data)
}

async function loadScenes() {
  sceneLoading.value = true
  sceneErrorText.value = ''
  try {
    const resp = await listMapScenes({ cityCode: defaultCityCode })
    scenes.value = resp.items || []
    if (!selectedSceneCode.value && scenes.value.length) {
      selectScene(scenes.value[0])
    }
  } catch {
    sceneErrorText.value = '地图场景加载失败，请重试'
  } finally {
    sceneLoading.value = false
  }
}

function selectScene(row) {
  selectedSceneCode.value = row.code
  resetSceneForm(row)
  activePanel.value = 'scene'
}

function newScene() {
  selectedSceneCode.value = ''
  resetSceneForm()
  activePanel.value = 'scene'
}

async function uploadBackground(options) {
  try {
    const url = await uploadMapBackgroundImage(options.file)
    sceneForm.backgroundUrl = url
    ElMessage.success('底图已上传')
  } catch (err) {
    ElMessage.error(err.message || '底图上传失败，请重试')
  }
}

async function submitScene() {
  sceneSaving.value = true
  try {
    const resp = await saveMapScene({ ...sceneForm })
    ElMessage.success('场景已保存')
    await loadScenes()
    if (resp.item?.code) {
      selectedSceneCode.value = resp.item.code
      resetSceneForm(resp.item)
    }
  } catch (err) {
    ElMessage.error(err.message || '场景保存失败，请重试')
  } finally {
    sceneSaving.value = false
  }
}

async function publishScene() {
  try {
    await ElMessageBox.confirm('发布后小程序将可读取该地图场景，确认发布吗？', '确认发布', {
      type: 'warning',
      confirmButtonText: '确认发布',
      cancelButtonText: '取消',
    })
  } catch {
    return
  }
  scenePublishing.value = true
  try {
    const resp = await publishMapScene(sceneForm.code)
    ElMessage.success(resp.message || '地图场景已发布')
    await loadScenes()
  } catch (err) {
    ElMessage.error(err.message || '地图场景发布失败，请重试')
  } finally {
    scenePublishing.value = false
  }
}

function createBooth() {}

function createPoi() {}
</script>

<style scoped>
.title-actions {
  display: flex;
  gap: 8px;
}

.sourcing-map-shell {
  display: grid;
  grid-template-columns: 280px minmax(0, 1fr) 360px;
  gap: 16px;
  min-height: calc(100vh - 150px);
}

.scene-sidebar,
.object-panel,
.map-workbench {
  min-width: 0;
}

.panel-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.panel-heading h3 {
  margin: 0;
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

.map-background {
  max-width: 100%;
  max-height: 560px;
  object-fit: contain;
}

.background-url-input {
  margin-top: 8px;
}

.scene-size-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 12px;
}
</style>
