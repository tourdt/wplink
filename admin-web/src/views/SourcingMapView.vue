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
        <div class="map-canvas" @click="handleCanvasClick">
          <div v-if="sceneForm.backgroundUrl" class="map-canvas-stage" :style="stageStyle">
            <img class="map-background" :src="sceneForm.backgroundUrl" alt="地图底图" />
            <button
              v-for="object in objects"
              :key="object.id || object.code"
              type="button"
              :class="['map-object', object.layer === 'booth' ? 'booth' : 'poi', { selected: isObjectSelected(object) }]"
              :style="objectStyle(object)"
              @click.stop="selectObject(object)"
              @mousedown.stop="startDragObject($event, object)"
            >
              <span class="object-label">{{ object.name }}</span>
            </button>
          </div>
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
            <div class="panel-heading object-heading">
              <h3>点位对象</h3>
              <el-tag type="info">{{ objects.length }} 个</el-tag>
            </div>

            <div v-if="objectErrorText" class="table-state table-state-error">
              <span>{{ objectErrorText }}</span>
              <el-button type="danger" plain :disabled="!selectedScene" @click="loadObjects(selectedScene.code)">重试</el-button>
            </div>

            <el-table
              v-loading="objectLoading"
              :data="objects"
              size="small"
              height="168"
              highlight-current-row
              empty-text="暂无点位对象"
              @row-click="selectObject"
            >
              <el-table-column prop="code" label="编码" width="88" />
              <el-table-column prop="name" label="名称" min-width="110" />
              <el-table-column label="形状" width="58">
                <template #default="{ row }">
                  {{ row.geometryType === 'rect' ? '矩形' : '点位' }}
                </template>
              </el-table-column>
              <el-table-column label="状态" width="58">
                <template #default="{ row }">
                  {{ objectStatusText[row.status] || row.status }}
                </template>
              </el-table-column>
            </el-table>

            <el-form class="object-form" label-position="top">
              <el-form-item label="点位编码">
                <el-input v-model="objectForm.code" placeholder="A001" />
              </el-form-item>
              <el-form-item label="点位名称">
                <el-input v-model="objectForm.name" placeholder="A001 档口" />
              </el-form-item>
              <div class="scene-size-grid">
                <el-form-item label="类型">
                  <el-select v-model="objectForm.type">
                    <el-option label="档口" value="booth" />
                    <el-option label="打包站" value="packing_station" />
                    <el-option label="停车场" value="parking" />
                    <el-option label="餐饮" value="restaurant" />
                  </el-select>
                </el-form-item>
                <el-form-item label="图层">
                  <el-select v-model="objectForm.layer">
                    <el-option label="档口" value="booth" />
                    <el-option label="配套" value="poi" />
                  </el-select>
                </el-form-item>
              </div>
              <div class="scene-size-grid">
                <el-form-item label="形状">
                  <el-select v-model="objectForm.geometryType" @change="syncGeometryType">
                    <el-option label="矩形" value="rect" />
                    <el-option label="点位" value="point" />
                  </el-select>
                </el-form-item>
                <el-form-item label="状态">
                  <el-select v-model="objectForm.status">
                    <el-option label="正常" value="normal" />
                    <el-option label="隐藏" value="hidden" />
                    <el-option label="歇业" value="closed" />
                  </el-select>
                </el-form-item>
              </div>
              <div class="geometry-grid">
                <el-form-item label="X">
                  <el-input-number v-model="objectForm.geometry.x" :min="0" controls-position="right" />
                </el-form-item>
                <el-form-item label="Y">
                  <el-input-number v-model="objectForm.geometry.y" :min="0" controls-position="right" />
                </el-form-item>
                <el-form-item v-if="objectForm.geometryType === 'rect'" label="宽">
                  <el-input-number v-model="objectForm.geometry.width" :min="1" controls-position="right" />
                </el-form-item>
                <el-form-item v-if="objectForm.geometryType === 'rect'" label="高">
                  <el-input-number v-model="objectForm.geometry.height" :min="1" controls-position="right" />
                </el-form-item>
              </div>
              <el-form-item label="地址">
                <el-input v-model="objectForm.address" placeholder="市场/路段/门牌" />
              </el-form-item>
              <div class="scene-size-grid">
                <el-form-item label="电话">
                  <el-input v-model="objectForm.phone" />
                </el-form-item>
                <el-form-item label="微信">
                  <el-input v-model="objectForm.wechat" />
                </el-form-item>
              </div>
              <div class="drawer-actions">
                <el-button type="primary" :disabled="!selectedScene" :loading="objectSaving" @click="submitObject">保存点位</el-button>
              </div>
            </el-form>
          </el-tab-pane>
        </el-tabs>
      </aside>
    </section>

    <el-drawer v-model="batchDrawerVisible" title="批量生成档口" size="420px">
      <el-form label-position="top">
        <el-form-item label="起始编号">
          <el-input v-model="batchForm.startCode" placeholder="A001" />
        </el-form-item>
        <div class="scene-size-grid">
          <el-form-item label="数量">
            <el-input-number v-model="batchForm.count" :min="1" :max="200" controls-position="right" />
          </el-form-item>
          <el-form-item label="方向">
            <el-select v-model="batchForm.direction">
              <el-option label="横向" value="horizontal" />
              <el-option label="纵向" value="vertical" />
            </el-select>
          </el-form-item>
        </div>
        <div class="scene-size-grid">
          <el-form-item label="起始 X">
            <el-input-number v-model="batchForm.startX" :min="0" controls-position="right" />
          </el-form-item>
          <el-form-item label="起始 Y">
            <el-input-number v-model="batchForm.startY" :min="0" controls-position="right" />
          </el-form-item>
        </div>
        <div class="scene-size-grid">
          <el-form-item label="档口宽">
            <el-input-number v-model="batchForm.width" :min="1" controls-position="right" />
          </el-form-item>
          <el-form-item label="档口高">
            <el-input-number v-model="batchForm.height" :min="1" controls-position="right" />
          </el-form-item>
        </div>
        <div class="scene-size-grid">
          <el-form-item label="间距">
            <el-input-number v-model="batchForm.gap" :min="0" controls-position="right" />
          </el-form-item>
          <el-form-item label="类型">
            <el-select v-model="batchForm.type">
              <el-option label="档口" value="booth" />
              <el-option label="打包站" value="packing_station" />
            </el-select>
          </el-form-item>
        </div>
        <el-form-item label="图层">
          <el-select v-model="batchForm.layer">
            <el-option label="档口" value="booth" />
            <el-option label="配套" value="poi" />
          </el-select>
        </el-form-item>
        <el-form-item label="分类编码">
          <el-select v-model="batchForm.categoryCodes" multiple filterable allow-create default-first-option placeholder="输入后回车">
          </el-select>
        </el-form-item>
        <el-form-item label="服务标签">
          <el-select v-model="batchForm.serviceTags" multiple filterable allow-create default-first-option placeholder="输入后回车">
          </el-select>
        </el-form-item>
        <div class="drawer-actions">
          <el-button @click="batchDrawerVisible = false">取消</el-button>
          <el-button type="primary" :loading="batchSaving" :disabled="!selectedScene" @click="submitBatchGenerate">生成</el-button>
        </div>
      </el-form>
    </el-drawer>
  </section>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  batchGenerateMapObjects,
  listMapObjects,
  listMapScenes,
  publishMapScene,
  saveMapObject,
  saveMapScene,
} from '../api/sourcingMap'
import { uploadMapBackgroundImage } from '../api/upload'
import { cityStationOptions, defaultCityCode } from '../common/cityStations'

const sceneStatusText = { draft: '草稿', published: '已发布', archived: '已归档' }
const sceneStatusTagType = { draft: 'info', published: 'success', archived: 'warning' }
const objectStatusText = { normal: '正常', hidden: '隐藏', closed: '歇业' }
const activePanel = ref('scene')
const batchDrawerVisible = ref(false)
const sceneLoading = ref(false)
const sceneSaving = ref(false)
const scenePublishing = ref(false)
const objectLoading = ref(false)
const objectSaving = ref(false)
const batchSaving = ref(false)
const sceneErrorText = ref('')
const objectErrorText = ref('')
const scenes = ref([])
const objects = ref([])
const selectedSceneCode = ref('')
const selectedObjectId = ref('')
const sceneForm = reactive(defaultSceneForm())
const objectForm = reactive(defaultObjectForm())
const batchForm = reactive(defaultBatchForm())
const selectedScene = computed(() => scenes.value.find((scene) => scene.code === selectedSceneCode.value) || null)
const stageStyle = computed(() => ({
  width: `${toPositiveNumber(sceneForm.width, 1200)}px`,
  height: `${toPositiveNumber(sceneForm.height, 720)}px`,
}))

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

function defaultGeometry(geometryType = 'rect') {
  if (geometryType === 'point') {
    return { x: 160, y: 160 }
  }
  return { x: 100, y: 100, width: 80, height: 50 }
}

function defaultObjectForm(data = {}) {
  const geometryType = data.geometryType || 'rect'
  return {
    id: '',
    code: '',
    name: '',
    type: 'booth',
    layer: 'booth',
    geometryType,
    geometry: { ...defaultGeometry(geometryType), ...(data.geometry || {}) },
    minZoom: 3,
    maxZoom: 5,
    categoryCodes: [],
    serviceTags: [],
    platformTags: [],
    poiServiceTags: [],
    address: '',
    phone: '',
    wechat: '',
    lat: '',
    lng: '',
    extra: {},
    sort: 0,
    status: 'normal',
    ...data,
  }
}

function defaultBatchForm() {
  return {
    startCode: 'A001',
    count: 10,
    direction: 'horizontal',
    startX: 100,
    startY: 100,
    width: 80,
    height: 50,
    gap: 8,
    type: 'booth',
    layer: 'booth',
    categoryCodes: [],
    serviceTags: [],
  }
}

function resetSceneForm(data = {}) {
  Object.assign(sceneForm, defaultSceneForm(), data)
}

function resetObjectForm(data = {}) {
  const next = defaultObjectForm(data)
  next.geometry = { ...defaultGeometry(next.geometryType), ...(data.geometry || {}) }
  Object.assign(objectForm, next)
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
  selectedObjectId.value = ''
  resetObjectForm()
  loadObjects(row.code)
}

function newScene() {
  selectedSceneCode.value = ''
  selectedObjectId.value = ''
  objects.value = []
  resetSceneForm()
  resetObjectForm()
  activePanel.value = 'scene'
}

async function loadObjects(sceneCode) {
  if (!sceneCode) {
    objects.value = []
    return
  }
  objectLoading.value = true
  objectErrorText.value = ''
  try {
    const resp = await listMapObjects(sceneCode)
    objects.value = resp.items || []
  } catch {
    objectErrorText.value = '地图点位加载失败，请重试'
  } finally {
    objectLoading.value = false
  }
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

function createBooth() {
  const index = objects.value.length + 1
  selectedObjectId.value = ''
  resetObjectForm({
    code: `B${String(index).padStart(3, '0')}`,
    name: `档口 ${index}`,
    type: 'booth',
    layer: 'booth',
    geometryType: 'rect',
    geometry: { x: 100, y: 100, width: 80, height: 50 },
  })
  activePanel.value = 'object'
}

function createPoi() {
  const index = objects.value.length + 1
  selectedObjectId.value = ''
  resetObjectForm({
    code: `P${String(index).padStart(3, '0')}`,
    name: `配套 ${index}`,
    type: 'packing_station',
    layer: 'poi',
    geometryType: 'point',
    geometry: { x: 160, y: 160 },
  })
  activePanel.value = 'object'
}

function selectObject(object) {
  selectedObjectId.value = objectIdentity(object)
  resetObjectForm(object)
  activePanel.value = 'object'
}

function syncGeometryType() {
  objectForm.geometry = { ...defaultGeometry(objectForm.geometryType), ...objectForm.geometry }
}

function handleCanvasClick(event) {
  if (!selectedScene.value || !objectForm.code) {
    return
  }
  const stage = event.currentTarget.querySelector('.map-canvas-stage')
  if (!stage) {
    return
  }
  const rect = stage.getBoundingClientRect()
  const x = Math.max(0, Math.round(event.clientX - rect.left))
  const y = Math.max(0, Math.round(event.clientY - rect.top))
  Object.assign(objectForm.geometry, { x, y })
  activePanel.value = 'object'
}

function objectIdentity(object) {
  return object.id || object.code
}

function isObjectSelected(object) {
  return objectIdentity(object) === selectedObjectId.value
}

function objectStyle(object) {
  const geometry = object.geometry || {}
  const x = toNumber(geometry.x, 0)
  const y = toNumber(geometry.y, 0)
  if (object.geometryType === 'point') {
    return {
      left: `${Math.max(0, x - 11)}px`,
      top: `${Math.max(0, y - 11)}px`,
      width: '22px',
      height: '22px',
    }
  }
  return {
    left: `${x}px`,
    top: `${y}px`,
    width: `${toPositiveNumber(geometry.width, 80)}px`,
    height: `${toPositiveNumber(geometry.height, 50)}px`,
  }
}

let dragState = null

function startDragObject(event, object) {
  event.preventDefault()
  selectObject(object)
  const geometry = object.geometry || {}
  dragState = {
    id: objectIdentity(object),
    startX: event.clientX,
    startY: event.clientY,
    originX: toNumber(geometry.x, 0),
    originY: toNumber(geometry.y, 0),
  }
  window.addEventListener('mousemove', dragObject)
  window.addEventListener('mouseup', stopDragObject, { once: true })
}

function dragObject(event) {
  if (!dragState) {
    return
  }
  const x = Math.max(0, Math.round(dragState.originX + event.clientX - dragState.startX))
  const y = Math.max(0, Math.round(dragState.originY + event.clientY - dragState.startY))
  updateObjectGeometry(dragState.id, { x, y })
}

function stopDragObject() {
  window.removeEventListener('mousemove', dragObject)
  dragState = null
}

function updateObjectGeometry(identity, patch) {
  const object = objects.value.find((item) => objectIdentity(item) === identity)
  if (object) {
    object.geometry = { ...(object.geometry || {}), ...patch }
  }
  if (identity === selectedObjectId.value) {
    Object.assign(objectForm.geometry, patch)
  }
}

async function submitObject() {
  if (!selectedScene.value) {
    ElMessage.error('请先选择地图场景')
    return
  }
  objectSaving.value = true
  try {
    const resp = await saveMapObject(selectedScene.value.code, buildObjectPayload())
    ElMessage.success('点位已保存')
    await loadObjects(selectedScene.value.code)
    if (resp.item?.id || resp.item?.code) {
      selectedObjectId.value = objectIdentity(resp.item)
      resetObjectForm(resp.item)
    }
  } catch (err) {
    ElMessage.error(err.message || '点位保存失败，请重试')
  } finally {
    objectSaving.value = false
  }
}

function buildObjectPayload() {
  return {
    id: objectForm.id,
    code: objectForm.code,
    name: objectForm.name,
    type: objectForm.type,
    layer: objectForm.layer,
    geometryType: objectForm.geometryType,
    geometry: normalizedObjectGeometry(),
    minZoom: objectForm.minZoom,
    maxZoom: objectForm.maxZoom,
    categoryCodes: objectForm.categoryCodes || [],
    serviceTags: objectForm.serviceTags || [],
    platformTags: objectForm.platformTags || [],
    poiServiceTags: objectForm.poiServiceTags || [],
    address: objectForm.address,
    phone: objectForm.phone,
    wechat: objectForm.wechat,
    lat: objectForm.lat,
    lng: objectForm.lng,
    extra: objectForm.extra || {},
    sort: objectForm.sort,
    status: objectForm.status,
  }
}

async function submitBatchGenerate() {
  if (!selectedScene.value) {
    ElMessage.error('请先选择地图场景')
    return
  }
  batchSaving.value = true
  try {
    const resp = await batchGenerateMapObjects(selectedScene.value.code, buildBatchPayload())
    const count = resp.items?.length || 0
    ElMessage.success(`已生成 ${count} 个点位`)
    batchDrawerVisible.value = false
    await loadObjects(selectedScene.value.code)
  } catch (err) {
    ElMessage.error(err.message || '批量生成失败，请重试')
  } finally {
    batchSaving.value = false
  }
}

function buildBatchPayload() {
  return {
    startCode: batchForm.startCode,
    count: toPositiveInteger(batchForm.count, 1),
    direction: batchForm.direction,
    startX: String(toNumber(batchForm.startX, 0)),
    startY: String(toNumber(batchForm.startY, 0)),
    width: String(toPositiveNumber(batchForm.width, 80)),
    height: String(toPositiveNumber(batchForm.height, 50)),
    gap: String(toNumber(batchForm.gap, 0)),
    type: batchForm.type,
    layer: batchForm.layer,
    categoryCodes: batchForm.categoryCodes || [],
    serviceTags: batchForm.serviceTags || [],
  }
}

function normalizedObjectGeometry() {
  if (objectForm.geometryType === 'point') {
    return {
      x: toNumber(objectForm.geometry.x, 0),
      y: toNumber(objectForm.geometry.y, 0),
    }
  }
  return {
    x: toNumber(objectForm.geometry.x, 0),
    y: toNumber(objectForm.geometry.y, 0),
    width: toPositiveNumber(objectForm.geometry.width, 80),
    height: toPositiveNumber(objectForm.geometry.height, 50),
  }
}

function toNumber(value, fallback) {
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : fallback
}

function toPositiveNumber(value, fallback) {
  const parsed = toNumber(value, fallback)
  return parsed > 0 ? parsed : fallback
}

function toPositiveInteger(value, fallback) {
  const parsed = Math.floor(toNumber(value, fallback))
  return parsed > 0 ? parsed : fallback
}
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
  display: flex;
  min-height: 560px;
  align-items: flex-start;
  justify-content: flex-start;
  overflow: auto;
  border: 1px dashed #cdd5df;
  border-radius: 8px;
  background: #f8fafc;
  color: #697586;
}

.map-canvas > span {
  margin: auto;
}

.map-canvas-stage {
  position: relative;
  flex: 0 0 auto;
  min-width: 480px;
  min-height: 320px;
  background: #fff;
}

.map-background {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: fill;
  pointer-events: none;
  user-select: none;
}

.map-object {
  position: absolute;
  z-index: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  box-sizing: border-box;
  padding: 0 4px;
  border: 2px solid #2563eb;
  background: rgba(37, 99, 235, 0.18);
  color: #0f172a;
  cursor: move;
  font: inherit;
}

.map-object.poi {
  padding: 0;
  border-color: #f97316;
  border-radius: 999px;
  background: #f97316;
  color: #fff;
}

.map-object.selected {
  border-color: #16a34a;
  box-shadow: 0 0 0 3px rgba(22, 163, 74, 0.2);
}

.object-label {
  max-width: 100%;
  overflow: hidden;
  font-size: 12px;
  line-height: 1.2;
  text-overflow: ellipsis;
  white-space: nowrap;
  pointer-events: none;
}

.background-url-input {
  margin-top: 8px;
}

.scene-size-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 12px;
}

.object-heading {
  margin-top: 4px;
}

.object-form {
  margin-top: 16px;
}

.geometry-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}
</style>
