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

        <div class="scene-filter-bar">
          <el-select v-model="sceneFilters.type" placeholder="全部场景类型" clearable @change="loadScenes">
            <el-option label="全部场景类型" value="" />
            <el-option v-for="item in sceneTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
          <el-select v-model="sceneFilters.status" placeholder="全部场景状态" clearable @change="loadScenes">
            <el-option label="全部场景状态" value="" />
            <el-option v-for="item in sceneStatusOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
          <div class="scene-filter-actions">
            <el-button :loading="sceneLoading" @click="loadScenes">筛选场景</el-button>
            <el-button @click="clearSceneFilters">清空</el-button>
          </div>
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
        <div ref="mapCanvasRef" class="map-canvas" @click="handleCanvasClick">
          <div v-if="sceneForm.backgroundUrl" class="map-canvas-stage" :style="stageStyle">
            <img class="map-background" :src="sceneForm.backgroundUrl" alt="地图底图" />
            <button
              v-for="object in objects"
              :key="object.id || object.code"
              type="button"
              :class="['map-object', object.layer === 'booth' ? 'booth' : 'poi', objectStatusClass(object.status), { selected: isObjectSelected(object) }]"
              :style="objectStyle(object)"
              :title="objectTitle(object)"
              @click.stop="selectObject(object)"
              @mousedown.stop="startDragObject($event, object)"
            >
              <span class="object-label">{{ object.name }}</span>
              <span v-if="object.status && object.status !== 'normal'" class="object-status-badge">
                {{ objectStatusText[object.status] || object.status }}
              </span>
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
                  <el-option v-for="item in sceneTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
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
              <div class="scene-default-viewport">
                <div class="section-subtitle">
                  <span>默认视野</span>
                  <el-button type="primary" link :disabled="!sceneForm.backgroundUrl" @click="setSceneDefaultCenterFromCanvas">设为当前画布中心</el-button>
                </div>
                <div class="scene-size-grid">
                  <el-form-item label="默认缩放">
                    <el-input v-model="sceneForm.defaultScale" placeholder="1" />
                  </el-form-item>
                  <el-form-item label="默认中心 X">
                    <el-input v-model="sceneForm.defaultCenterX" placeholder="1500" />
                  </el-form-item>
                </div>
                <div class="scene-size-grid">
                  <el-form-item label="默认中心 Y">
                    <el-input v-model="sceneForm.defaultCenterY" placeholder="900" />
                  </el-form-item>
                  <el-form-item label="状态">
                    <el-select v-model="sceneForm.status">
                      <el-option v-for="item in sceneStatusOptions" :key="item.value" :label="item.label" :value="item.value" />
                    </el-select>
                  </el-form-item>
                </div>
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

            <div class="object-filter-bar">
              <el-input
                v-model.trim="objectFilters.keyword"
                class="object-filter-keyword"
                placeholder="搜索编码/名称"
                clearable
                :disabled="!selectedScene"
                @keyup.enter="applyObjectFilters"
                @clear="applyObjectFilters"
              />
              <el-select v-model="objectFilters.type" placeholder="全部类型" clearable :disabled="!selectedScene" @change="applyObjectFilters">
                <el-option label="全部类型" value="" />
                <el-option v-for="item in objectTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
              <el-select v-model="objectFilters.status" placeholder="全部状态" clearable :disabled="!selectedScene" @change="applyObjectFilters">
                <el-option label="全部状态" value="" />
                <el-option v-for="item in objectStatusOptions" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
              <div class="object-filter-actions">
                <el-button :disabled="!selectedScene" :loading="objectLoading" @click="applyObjectFilters">筛选点位</el-button>
                <el-button :disabled="!selectedScene" @click="clearObjectFilters">清空</el-button>
              </div>
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
                  <el-tag size="small" :type="objectStatusTagType[row.status] || 'info'">{{ objectStatusText[row.status] || row.status }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="操作" width="126">
                <template #default="{ row }">
                  <div class="object-row-actions">
                    <el-button type="primary" link @click.stop="locateObject(row)">定位</el-button>
                    <el-dropdown trigger="click" @command="(status) => changeObjectStatus(row, status)">
                      <el-button type="primary" link :loading="objectStatusSavingId === row.id" @click.stop>状态操作</el-button>
                      <template #dropdown>
                        <el-dropdown-menu>
                          <el-dropdown-item command="normal" :disabled="row.status === 'normal'">设为正常</el-dropdown-item>
                          <el-dropdown-item command="hidden" :disabled="row.status === 'hidden'">设为隐藏</el-dropdown-item>
                          <el-dropdown-item command="closed" :disabled="row.status === 'closed'">设为歇业</el-dropdown-item>
                        </el-dropdown-menu>
                      </template>
                    </el-dropdown>
                  </div>
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
                    <el-option v-for="item in objectTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
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
                    <el-option v-for="item in objectStatusOptions" :key="item.value" :label="item.label" :value="item.value" />
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
              <el-form-item label="主营分类">
                <el-select v-model="objectForm.categoryCodes" multiple filterable allow-create default-first-option placeholder="选择或输入分类">
                  <el-option v-for="item in mergedCategoryOptions" :key="item.value" :label="item.label" :value="item.value" />
                </el-select>
              </el-form-item>
              <el-form-item label="档口服务">
                <el-select v-model="objectForm.serviceTags" multiple filterable allow-create default-first-option placeholder="选择或输入服务标签">
                  <el-option v-for="item in mergedServiceTagOptions" :key="item.value" :label="item.label" :value="item.value" />
                </el-select>
              </el-form-item>
              <el-form-item label="平台标签">
                <el-select v-model="objectForm.platformTags" multiple filterable allow-create default-first-option placeholder="运营侧推荐/认证标签">
                  <el-option v-for="item in mergedPlatformTagOptions" :key="item.value" :label="item.label" :value="item.value" />
                </el-select>
              </el-form-item>
              <el-form-item label="配套服务">
                <el-select v-model="objectForm.poiServiceTags" multiple filterable allow-create default-first-option placeholder="打包/物流/快递服务">
                  <el-option v-for="item in mergedPoiServiceTagOptions" :key="item.value" :label="item.label" :value="item.value" />
                </el-select>
              </el-form-item>
              <el-form-item label="营业时间">
                <el-input v-model="objectForm.extra.openHours" placeholder="08:00-22:00" />
              </el-form-item>
              <el-form-item label="支持服务">
                <el-select v-model="objectForm.extra.services" multiple filterable allow-create default-first-option placeholder="打包/贴单/纸箱/胶带">
                  <el-option v-for="item in extraServiceOptions" :key="item.value" :label="item.label" :value="item.value" />
                </el-select>
              </el-form-item>
              <el-form-item label="物流线路">
                <el-select v-model="objectForm.extra.lines" multiple filterable allow-create default-first-option placeholder="杭州/上海/江苏/全国">
                  <el-option v-for="item in logisticsLineOptions" :key="item.value" :label="item.label" :value="item.value" />
                </el-select>
              </el-form-item>
              <el-form-item label="发货方式">
                <el-select v-model="objectForm.extra.deliveryTypes" multiple filterable allow-create default-first-option placeholder="零担/整车/到付">
                  <el-option v-for="item in deliveryTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
                </el-select>
              </el-form-item>
              <div class="scene-size-grid">
                <el-form-item label="发车时间">
                  <el-input v-model="objectForm.extra.departureTime" placeholder="每天 18:00 前" />
                </el-form-item>
                <el-form-item label="收费说明">
                  <el-input v-model="objectForm.extra.priceNote" placeholder="按件计费" />
                </el-form-item>
              </div>
              <el-form-item label="快递品牌">
                <el-select v-model="objectForm.extra.brands" multiple filterable allow-create default-first-option placeholder="中通/圆通/极兔/顺丰">
                  <el-option v-for="item in expressBrandOptions" :key="item.value" :label="item.label" :value="item.value" />
                </el-select>
              </el-form-item>
              <div class="drawer-actions">
                <el-button type="primary" :disabled="!selectedScene" :loading="objectSaving" @click="submitObject">保存点位</el-button>
              </div>
            </el-form>
          </el-tab-pane>
          <el-tab-pane label="标签" name="category">
            <div class="panel-heading object-heading">
              <h3>标准标签</h3>
              <el-button type="primary" link @click="newCategory">新增标签</el-button>
            </div>
            <div class="category-filter-bar">
              <el-select v-model="categoryFilters.type" placeholder="全部类型" clearable @change="loadCategories">
                <el-option label="全部类型" value="" />
                <el-option v-for="item in categoryTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
              <el-select v-model="categoryFilters.status" placeholder="全部状态" clearable @change="loadCategories">
                <el-option label="全部状态" value="" />
                <el-option v-for="item in categoryStatusOptions" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
              <el-button :loading="categoryLoading" @click="loadCategories">筛选标签</el-button>
            </div>

            <div v-if="categoryErrorText" class="table-state table-state-error">
              <span>{{ categoryErrorText }}</span>
              <el-button type="danger" plain @click="loadCategories">重试</el-button>
            </div>

            <el-table
              v-loading="categoryLoading"
              :data="mapCategories"
              size="small"
              height="168"
              highlight-current-row
              empty-text="暂无标准标签"
              @row-click="selectCategory"
            >
              <el-table-column prop="code" label="编码" width="92" />
              <el-table-column prop="name" label="名称" min-width="100" />
              <el-table-column label="类型" width="86">
                <template #default="{ row }">
                  {{ categoryTypeText[row.type] || row.type }}
                </template>
              </el-table-column>
              <el-table-column label="状态" width="58">
                <template #default="{ row }">
                  {{ categoryStatusText[row.status] || row.status }}
                </template>
              </el-table-column>
            </el-table>

            <el-form class="object-form" label-position="top">
              <el-form-item label="标签编码">
                <el-input v-model.trim="categoryForm.code" placeholder="girl" />
              </el-form-item>
              <el-form-item label="标签名称">
                <el-input v-model.trim="categoryForm.name" placeholder="女童" />
              </el-form-item>
              <div class="scene-size-grid">
                <el-form-item label="标签类型">
                  <el-select v-model="categoryForm.type">
                    <el-option v-for="item in categoryTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
                  </el-select>
                </el-form-item>
                <el-form-item label="排序">
                  <el-input-number v-model="categoryForm.sort" :min="0" controls-position="right" />
                </el-form-item>
              </div>
              <el-form-item label="图标 URL">
                <el-input v-model.trim="categoryForm.iconUrl" placeholder="可选" />
              </el-form-item>
              <div class="scene-size-grid">
                <el-form-item label="前端展示">
                  <el-switch v-model="categoryForm.isVisible" />
                </el-form-item>
                <el-form-item label="状态">
                  <el-select v-model="categoryForm.status">
                    <el-option v-for="item in categoryStatusOptions" :key="item.value" :label="item.label" :value="item.value" />
                  </el-select>
                </el-form-item>
              </div>
              <div class="drawer-actions">
                <el-button @click="newCategory">新增标签</el-button>
                <el-button type="primary" :loading="categorySaving" @click="submitCategory">保存标签</el-button>
              </div>
            </el-form>
          </el-tab-pane>
        </el-tabs>
      </aside>
    </section>

    <el-drawer v-model="batchDrawerVisible" title="批量生成点位" size="420px">
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
          <el-form-item label="点位宽">
            <el-input-number v-model="batchForm.width" :min="1" controls-position="right" />
          </el-form-item>
          <el-form-item label="点位高">
            <el-input-number v-model="batchForm.height" :min="1" controls-position="right" />
          </el-form-item>
        </div>
        <div class="scene-size-grid">
          <el-form-item label="间距">
            <el-input-number v-model="batchForm.gap" :min="0" controls-position="right" />
          </el-form-item>
          <el-form-item label="类型">
            <el-select v-model="batchForm.type" @change="syncBatchLayerByType">
              <el-option v-for="item in objectTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
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
            <el-option v-for="item in mergedCategoryOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="服务标签">
          <el-select v-model="batchForm.serviceTags" multiple filterable allow-create default-first-option placeholder="输入后回车">
            <el-option v-for="item in mergedServiceTagOptions" :key="item.value" :label="item.label" :value="item.value" />
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
  listMapCategories,
  listMapObjects,
  listMapScenes,
  publishMapScene,
  saveMapCategory,
  saveMapObject,
  saveMapScene,
  updateMapObjectStatus,
} from '../api/sourcingMap'
import { uploadMapBackgroundImage } from '../api/upload'
import { cityStationOptions, defaultCityCode } from '../common/cityStations'

const sceneTypeOptions = [
  { label: '总览', value: 'overview' },
  { label: '街区', value: 'street' },
  { label: '路段', value: 'street_segment' },
  { label: '商场', value: 'mall' },
  { label: '楼层', value: 'floor' },
]
const sceneStatusOptions = [
  { label: '草稿', value: 'draft' },
  { label: '已发布', value: 'published' },
  { label: '已归档', value: 'archived' },
]
const sceneStatusText = Object.fromEntries(sceneStatusOptions.map((item) => [item.value, item.label]))
const sceneStatusTagType = { draft: 'info', published: 'success', archived: 'warning' }
const objectTypeOptions = [
  { label: '档口', value: 'booth' },
  { label: '源头工厂', value: 'factory_booth' },
  { label: '仓库', value: 'warehouse' },
  { label: '打包站', value: 'packing_station' },
  { label: '物流点', value: 'logistics_point' },
  { label: '快递点', value: 'express_point' },
  { label: '停车场', value: 'parking' },
  { label: '餐饮', value: 'restaurant' },
]
const batchPoiTypeValues = new Set(['packing_station', 'logistics_point', 'express_point', 'parking', 'restaurant'])
const objectStatusOptions = [
  { label: '正常', value: 'normal' },
  { label: '隐藏', value: 'hidden' },
  { label: '歇业', value: 'closed' },
]
const objectStatusText = Object.fromEntries(objectStatusOptions.map((item) => [item.value, item.label]))
const objectStatusTagType = { normal: 'success', hidden: 'info', closed: 'warning' }
const categoryTypeOptions = [
  { label: '主营分类', value: 'booth_category' },
  { label: '档口服务', value: 'booth_service' },
  { label: '平台标签', value: 'platform_tag' },
  { label: '配套服务', value: 'poi_service' },
  { label: 'POI 类型', value: 'poi_type' },
]
const categoryTypeText = Object.fromEntries(categoryTypeOptions.map((item) => [item.value, item.label]))
const categoryStatusText = { normal: '正常', hidden: '隐藏', closed: '停用' }
const categoryStatusOptions = [
  { label: '正常', value: 'normal' },
  { label: '隐藏', value: 'hidden' },
  { label: '停用', value: 'closed' },
]
const categoryOptions = [
  { label: '女童', value: 'girl' },
  { label: '男童', value: 'boy' },
  { label: '婴童', value: 'baby' },
  { label: '中大童', value: 'middle_child' },
  { label: '套装', value: 'suit' },
  { label: '裙装', value: 'dress' },
  { label: '外套', value: 'coat' },
  { label: '校服', value: 'school_uniform' },
]
const serviceTagOptions = [
  { label: '现货', value: 'spot' },
  { label: '源头工厂', value: 'factory' },
  { label: '支持打样', value: 'sample' },
  { label: '一件代发', value: 'drop_shipping' },
  { label: '可小单', value: 'small_order' },
  { label: '支持混批', value: 'mixed_batch' },
]
const platformTagOptions = [
  { label: '实地认证', value: 'verified' },
  { label: '热门推荐', value: 'hot' },
  { label: '新手推荐', value: 'newbie_friendly' },
  { label: '优质档口', value: 'quality_booth' },
  { label: '平台精选', value: 'recommended' },
]
const poiServiceTagOptions = [
  { label: '打包', value: 'packing' },
  { label: '贴单', value: 'labeling' },
  { label: '纸箱', value: 'carton' },
  { label: '胶带', value: 'tape' },
  { label: '零担', value: 'less_than_truckload' },
  { label: '全国物流', value: 'national' },
  { label: '批量发货', value: 'bulk_shipping' },
]
const extraServiceOptions = [
  { label: '打包', value: 'packing' },
  { label: '贴单', value: 'labeling' },
  { label: '纸箱', value: 'carton' },
  { label: '胶带', value: 'tape' },
  { label: '临时寄存', value: 'storage' },
]
const logisticsLineOptions = [
  { label: '杭州', value: '杭州' },
  { label: '上海', value: '上海' },
  { label: '江苏', value: '江苏' },
  { label: '全国', value: '全国' },
]
const deliveryTypeOptions = [
  { label: '零担', value: 'less_than_truckload' },
  { label: '整车', value: 'full_truckload' },
  { label: '到付', value: 'cod' },
  { label: '代收', value: 'collection' },
]
const expressBrandOptions = [
  { label: '中通', value: 'zto' },
  { label: '圆通', value: 'yto' },
  { label: '申通', value: 'sto' },
  { label: '韵达', value: 'yunda' },
  { label: '极兔', value: 'jtexpress' },
  { label: '顺丰', value: 'sf' },
]
const activePanel = ref('scene')
const batchDrawerVisible = ref(false)
const sceneLoading = ref(false)
const sceneSaving = ref(false)
const scenePublishing = ref(false)
const objectLoading = ref(false)
const objectSaving = ref(false)
const objectStatusSavingId = ref('')
const batchSaving = ref(false)
const categoryLoading = ref(false)
const categorySaving = ref(false)
const sceneErrorText = ref('')
const objectErrorText = ref('')
const categoryErrorText = ref('')
const scenes = ref([])
const objects = ref([])
const mapCategories = ref([])
const categoryOptionItems = ref([])
const mapCanvasRef = ref(null)
const selectedSceneCode = ref('')
const selectedObjectId = ref('')
const sceneForm = reactive(defaultSceneForm())
const objectForm = reactive(defaultObjectForm())
const batchForm = reactive(defaultBatchForm())
const categoryForm = reactive(defaultCategoryForm())
const sceneFilters = reactive(defaultSceneFilters())
const categoryFilters = reactive(defaultCategoryFilters())
const objectFilters = reactive(defaultObjectFilters())
const selectedScene = computed(() => scenes.value.find((scene) => scene.code === selectedSceneCode.value) || null)
const mergedCategoryOptions = computed(() => mergeCategoryOptions(categoryOptions, mapCategoryOptions('booth_category')))
const mergedServiceTagOptions = computed(() => mergeCategoryOptions(serviceTagOptions, mapCategoryOptions('booth_service')))
const mergedPlatformTagOptions = computed(() => mergeCategoryOptions(platformTagOptions, mapCategoryOptions('platform_tag')))
const mergedPoiServiceTagOptions = computed(() => mergeCategoryOptions(poiServiceTagOptions, mapCategoryOptions('poi_service')))
const stageStyle = computed(() => ({
  width: `${toPositiveNumber(sceneForm.width, 1200)}px`,
  height: `${toPositiveNumber(sceneForm.height, 720)}px`,
}))

onMounted(() => {
  loadScenes()
  loadCategories()
  loadCategoryOptions()
})

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
  const form = {
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
  form.extra = normalizeExtraForm(form.extra)
  return form
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

function defaultCategoryForm(data = {}) {
  return {
    code: '',
    name: '',
    type: 'booth_category',
    iconUrl: '',
    sort: 0,
    isVisible: true,
    status: 'normal',
    ...data,
  }
}

function defaultCategoryFilters() {
  return {
    type: '',
    status: '',
  }
}

function defaultSceneFilters() {
  return {
    type: '',
    status: '',
  }
}

function defaultObjectFilters() {
  return {
    keyword: '',
    type: '',
    status: '',
  }
}

function resetSceneForm(data = {}) {
  Object.assign(sceneForm, defaultSceneForm(), data)
}

function resetObjectForm(data = {}) {
  const next = defaultObjectForm(data)
  next.geometry = { ...defaultGeometry(next.geometryType), ...(data.geometry || {}) }
  next.extra = normalizeExtraForm(next.extra)
  Object.assign(objectForm, next)
}

function resetCategoryForm(data = {}) {
  Object.assign(categoryForm, defaultCategoryForm(data))
}

async function loadScenes() {
  sceneLoading.value = true
  sceneErrorText.value = ''
  try {
    const resp = await listMapScenes({
      cityCode: defaultCityCode,
      type: sceneFilters.type,
      status: sceneFilters.status,
    })
    scenes.value = resp.items || []
    if (selectedSceneCode.value && !scenes.value.some((scene) => scene.code === selectedSceneCode.value)) {
      selectedSceneCode.value = ''
      selectedObjectId.value = ''
      objects.value = []
      resetSceneForm()
      resetObjectForm()
    }
    if (!selectedSceneCode.value && scenes.value.length) {
      selectScene(scenes.value[0])
    }
  } catch {
    sceneErrorText.value = '地图场景加载失败，请重试'
  } finally {
    sceneLoading.value = false
  }
}

function clearSceneFilters() {
  Object.assign(sceneFilters, defaultSceneFilters())
  loadScenes()
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
    const resp = await listMapObjects(sceneCode, {
      types: objectFilters.type,
      status: objectFilters.status,
      keyword: objectFilters.keyword,
    })
    objects.value = resp.items || []
  } catch {
    objectErrorText.value = '地图点位加载失败，请重试'
  } finally {
    objectLoading.value = false
  }
}

function applyObjectFilters() {
  if (!selectedScene.value?.code) {
    return
  }
  loadObjects(selectedScene.value.code)
}

function clearObjectFilters() {
  Object.assign(objectFilters, defaultObjectFilters())
  applyObjectFilters()
}

async function loadCategories() {
  categoryLoading.value = true
  categoryErrorText.value = ''
  try {
    const resp = await listMapCategories({
      type: categoryFilters.type,
      status: categoryFilters.status,
    })
    mapCategories.value = resp.items || []
  } catch {
    categoryErrorText.value = '标准标签加载失败，请重试'
  } finally {
    categoryLoading.value = false
  }
}

async function loadCategoryOptions() {
  try {
    const resp = await listMapCategories({ status: 'normal' })
    categoryOptionItems.value = resp.items || []
  } catch {
    categoryOptionItems.value = []
  }
}

function selectCategory(row) {
  resetCategoryForm(row)
  activePanel.value = 'category'
}

function newCategory() {
  resetCategoryForm()
  activePanel.value = 'category'
}

async function submitCategory() {
  if (!categoryForm.code || !categoryForm.name) {
    ElMessage.error('请填写标签编码和名称')
    return
  }
  categorySaving.value = true
  try {
    const resp = await saveMapCategory({
      ...categoryForm,
      sort: toPositiveInteger(categoryForm.sort, 0),
    })
    ElMessage.success('标准标签已保存')
    await loadCategories()
    await loadCategoryOptions()
    if (resp.item?.code) {
      resetCategoryForm(resp.item)
    }
  } catch (err) {
    ElMessage.error(err.message || '标准标签保存失败，请重试')
  } finally {
    categorySaving.value = false
  }
}

function mapCategoryOptions(type) {
  return categoryOptionItems.value
    .filter((item) => item.type === type && item.isVisible !== false && item.status !== 'hidden' && item.status !== 'closed')
    .sort((left, right) => toNumber(left.sort, 0) - toNumber(right.sort, 0))
    .map((item) => ({ label: item.name, value: item.code }))
}

function mergeCategoryOptions(defaultOptions, configuredOptions) {
  const seen = new Set()
  return [...configuredOptions, ...defaultOptions].filter((item) => {
    if (!item.value || seen.has(item.value)) {
      return false
    }
    seen.add(item.value)
    return true
  })
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

function locateObject(object) {
  selectObject(object)
  scrollCanvasToObject(object)
}

function scrollCanvasToObject(object) {
  const canvas = mapCanvasRef.value
  if (!canvas) {
    return
  }
  const geometry = object.geometry || {}
  const x = toNumber(geometry.x, 0)
  const y = toNumber(geometry.y, 0)
  const centerX = object.geometryType === 'rect' ? x + toPositiveNumber(geometry.width, 80) / 2 : x
  const centerY = object.geometryType === 'rect' ? y + toPositiveNumber(geometry.height, 50) / 2 : y
  const maxLeft = Math.max(0, canvas.scrollWidth - canvas.clientWidth)
  const maxTop = Math.max(0, canvas.scrollHeight - canvas.clientHeight)
  canvas.scrollLeft = clampNumber(Math.round(centerX - canvas.clientWidth / 2), 0, maxLeft)
  canvas.scrollTop = clampNumber(Math.round(centerY - canvas.clientHeight / 2), 0, maxTop)
}

function setSceneDefaultCenterFromCanvas() {
  const canvas = mapCanvasRef.value
  const stage = canvas?.querySelector('.map-canvas-stage')
  if (!canvas || !stage) {
    ElMessage.error('请先上传底图并打开画布')
    return
  }
  const centerX = clampNumber(Math.round(canvas.scrollLeft + canvas.clientWidth / 2), 0, toPositiveNumber(sceneForm.width, stage.clientWidth))
  const centerY = clampNumber(Math.round(canvas.scrollTop + canvas.clientHeight / 2), 0, toPositiveNumber(sceneForm.height, stage.clientHeight))
  sceneForm.defaultCenterX = String(centerX)
  sceneForm.defaultCenterY = String(centerY)
  ElMessage.success('当前画布中心已写入默认视野')
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

function objectStatusClass(status) {
  return status && status !== 'normal' ? `status-${status}` : ''
}

function objectTitle(object) {
  return `${object.name} · ${objectStatusText[object.status] || object.status || '正常'}`
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

async function changeObjectStatus(row, status) {
  if (!row?.id) {
    ElMessage.error('点位缺少 ID，无法更新状态')
    return
  }
  if (row.status === status) {
    return
  }
  const sceneCode = selectedScene.value?.code || row.sceneCode
  objectStatusSavingId.value = row.id
  try {
    const resp = await updateMapObjectStatus(row.id, status)
    ElMessage.success('点位状态已更新')
    if (resp.item && selectedObjectId.value === objectIdentity(row)) {
      resetObjectForm(resp.item)
    }
    if (sceneCode) {
      await loadObjects(sceneCode)
    }
  } catch (err) {
    ElMessage.error(err.message || '点位状态更新失败，请重试')
  } finally {
    objectStatusSavingId.value = ''
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
    extra: normalizedExtra(),
    sort: objectForm.sort,
    status: objectForm.status,
  }
}

function normalizeExtraForm(extra = {}) {
  return {
    ...(extra || {}),
    openHours: extra?.openHours || '',
    services: Array.isArray(extra?.services) ? [...extra.services] : [],
    lines: Array.isArray(extra?.lines) ? [...extra.lines] : [],
    deliveryTypes: Array.isArray(extra?.deliveryTypes) ? [...extra.deliveryTypes] : [],
    departureTime: extra?.departureTime || '',
    brands: Array.isArray(extra?.brands) ? [...extra.brands] : [],
    priceNote: extra?.priceNote || '',
  }
}

function normalizedExtra() {
  const extra = normalizeExtraForm(objectForm.extra)
  return Object.fromEntries(
    Object.entries(extra).filter(([, value]) => {
      if (Array.isArray(value)) return value.length > 0
      return value !== ''
    }),
  )
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

function syncBatchLayerByType() {
  batchForm.layer = batchPoiTypeValues.has(batchForm.type) ? 'poi' : 'booth'
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

function clampNumber(value, min, max) {
  return Math.min(max, Math.max(min, value))
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

.scene-filter-bar {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 8px;
  margin-bottom: 12px;
}

.scene-filter-actions {
  grid-column: 1 / -1;
  display: flex;
  gap: 8px;
}

.scene-filter-actions .el-button {
  flex: 1;
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

.map-object.status-hidden {
  border-style: dashed;
  opacity: 0.5;
}

.map-object.status-closed {
  border-color: #64748b;
  background: rgba(100, 116, 139, 0.22);
  color: #334155;
}

.map-object.poi.status-closed {
  background: #64748b;
  color: #fff;
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

.object-status-badge {
  position: absolute;
  top: -10px;
  right: -10px;
  min-width: 28px;
  padding: 1px 5px;
  border-radius: 999px;
  background: #475569;
  color: #fff;
  font-size: 11px;
  line-height: 16px;
  white-space: nowrap;
  pointer-events: none;
}

.background-url-input {
  margin-top: 8px;
}

.scene-default-viewport {
  margin-bottom: 18px;
  padding: 12px;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  background: #f8fafc;
}

.section-subtitle {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
  color: #334155;
  font-size: 13px;
  font-weight: 600;
}

.scene-size-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 12px;
}

.object-heading {
  margin-top: 4px;
}

.object-filter-bar {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 8px;
  margin-bottom: 12px;
}

.object-filter-keyword,
.object-filter-actions {
  grid-column: 1 / -1;
}

.object-filter-actions {
  display: flex;
  gap: 8px;
}

.object-filter-actions .el-button {
  flex: 1;
}

.object-row-actions {
  display: flex;
  align-items: center;
  gap: 6px;
  white-space: nowrap;
}

.category-filter-bar {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) auto;
  gap: 8px;
  margin-bottom: 12px;
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
