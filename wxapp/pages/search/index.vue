<template>
  <view class="search-page" :style="searchPageStyle" @click="closeTypePanel">
    <view class="search-nav" :style="searchNavStyle">
      <view class="search-title-bar" :style="searchTitleBarStyle">
        <button class="nav-back-button" @click="goBack">
          <view class="nav-back-icon"></view>
        </button>
        <button class="channel-title-button" @click="openGroupDrawer">
          <text class="channel-title-text">{{ channelTitle }}</text>
          <text class="channel-title-arrow"></text>
        </button>
      </view>
    </view>

    <view class="search-toolbar" :style="searchToolbarStyle">
      <view class="search-bar">
        <input v-model="keyword" class="search-input" :placeholder="searchPlaceholder" @confirm="search" />
        <button class="search-button" :disabled="loading" :loading="loading" @click="search">{{ loading ? '搜索中' : '搜索' }}</button>
      </view>

      <view :class="['filter-shell', showAllTypeButton ? 'has-type-panel-button' : '']" @click.stop>
        <scroll-view
          class="filter-row"
          scroll-x
          scroll-with-animation
          enhanced
          :show-scrollbar="false"
          :scroll-into-view="scrollIntoTypeId"
        >
          <button
            v-for="item in visibleResourceTypes"
            :key="item.value"
            :id="getTypeButtonId(item.value)"
            :class="['filter-button', item.value === filters.typeCode ? 'active' : '']"
            @click="selectType(item.value)"
          >
            {{ item.label }}
          </button>
        </scroll-view>
        <button
          v-if="showAllTypeButton"
          class="type-panel-toggle"
          :aria-label="showTypePanel ? '收起全部分类' : '展开全部分类'"
          :aria-expanded="showTypePanel"
          @click.stop="toggleTypePanel"
        >
          <text :class="['type-panel-arrow', showTypePanel ? 'expanded' : '']"></text>
        </button>
      </view>
      <scroll-view
        v-if="showTypePanel"
        class="type-panel"
        scroll-y
        enhanced
        :show-scrollbar="false"
        @click.stop
      >
        <view class="type-panel-content">
          <view class="type-panel-head">
            <text class="type-panel-title">选择分类</text>
            <text class="type-panel-current">当前：{{ selectedTypeName }}</text>
          </view>
          <view class="type-panel-grid">
            <button
              v-for="item in resourceTypes"
              :key="item.value"
              :class="['type-panel-button', item.value === filters.typeCode ? 'active' : '']"
              @click="selectType(item.value)"
            >
              <text class="type-panel-button-text">{{ item.label }}</text>
              <text
                v-if="item.value === filters.typeCode"
                class="type-panel-check"
                aria-hidden="true"
              ></text>
            </button>
          </view>
        </view>
      </scroll-view>
      <scroll-view
        v-if="searchTagOptions.length"
        class="tag-filter-row"
        scroll-x
        enhanced
        :show-scrollbar="false"
      >
        <button
          v-for="tag in searchTagOptions"
          :key="tag"
          :class="['tag-filter-button', isSearchTagSelected(tag) ? 'active' : '']"
          @click="toggleSearchTag(tag)"
        >
          {{ tag }}
        </button>
      </scroll-view>
    </view>

    <view v-if="showGroupDrawer" class="group-drawer-mask" @click="closeGroupDrawer">
      <view class="group-drawer-panel" @click.stop>
        <view class="type-drawer-head">
          <text class="type-drawer-title">选择类目</text>
          <button class="type-drawer-close" @click="closeGroupDrawer">关闭</button>
        </view>
        <view class="drawer-group-list">
          <button
            v-for="item in groupFilterOptions"
            :key="item.code"
            :class="['drawer-group-button', item.code === filters.groupCode ? 'active' : '']"
            @click="selectGroup(item.code)"
          >
            {{ item.name }}
          </button>
        </view>
      </view>
    </view>

    <view v-if="hotKeywords.length" class="hot-row">
      <text class="hot-label">热门：</text>
      <button v-for="item in hotKeywords" :key="item" class="hot-button" @click="searchHotKeyword(item)">
        {{ item }}
      </button>
    </view>

    <view v-if="rows.length" class="result-list">
      <template v-for="item in rows" :key="item.id">
        <ResourceExposure :resource-id="item.id" source="search">
          <ResourceFeedCard :resource="item" @open="openResource" />
        </ResourceExposure>
      </template>
      <text class="load-more-text">{{ loading ? '加载中...' : hasMore ? '上拉加载更多' : '没有更多了' }}</text>
    </view>

    <view v-else-if="searched" class="empty-card">
      <view class="empty-visual">
        <view class="empty-sheet">
          <text></text>
          <text></text>
          <text></text>
        </view>
        <view class="empty-magnifier"></view>
      </view>
      <text class="empty-title">{{ emptyTitle }}</text>
      <text class="empty-desc">{{ emptyDesc }}</text>
      <view v-if="emptySuggestions.length" class="empty-suggestions">
        <button
          v-for="item in emptySuggestions"
          :key="item"
          class="empty-suggestion"
          @click="searchHotKeyword(item)"
        >
          {{ item }}
        </button>
      </view>
      <view v-if="emptyPrimaryActionLabel" class="empty-actions">
        <button v-if="emptyPrimaryActionLabel" class="primary-empty-button" @click="handleEmptyPrimaryAction">
          {{ emptyPrimaryActionLabel }}
        </button>
      </view>
    </view>

  </view>
</template>

<script setup>
import { computed, nextTick, reactive, ref } from 'vue'
import { onLoad, onPageScroll, onReachBottom, onShow } from '@dcloudio/uni-app'
import ResourceFeedCard from '../../components/ResourceFeedCard.vue'
import ResourceExposure from '../../components/ResourceExposure.vue'
import { DEFAULT_CITY_CODE } from '../../common/constants'
import { loadHotSearchKeywords } from '../../common/hotSearchKeywords'
import { groupResourceTypes } from '../../common/resourceCategories'
import { listCityResourceTypes } from '../../api/city'
import { searchResources } from '../../api/resource'

const resourceTypes = ref([{ label: '全部', value: '' }])
const categoryGroups = ref([])
const hotKeywords = ref([])
const SEARCH_KEY = 'wplink_pending_search_keyword'
const NAV_BOTTOM_RPX = 12
const MAX_SEARCH_TAGS = 8
const headerMetrics = ref({
  statusBarHeight: 44,
  navBarHeight: 44,
  headerHeight: 94,
})
const keyword = ref('')
const rows = ref([])
const searched = ref(false)
const page = ref(1)
const pageSize = 20
const total = ref(0)
const hasMore = ref(true)
const loading = ref(false)
const searchRequestSeq = ref(0)
const filters = reactive({
  cityCode: DEFAULT_CITY_CODE,
  groupCode: '',
  typeCode: '',
  tags: [],
})
const showGroupDrawer = ref(false)
const showTypePanel = ref(false)
const scrollIntoTypeId = ref('')
const pageScrollTop = ref(0)
const groupFilterOptions = computed(() => [{ code: '', name: '全部类目' }, ...categoryGroups.value])
const currentGroupResourceTypeItems = computed(() => {
  const selectedGroup = categoryGroups.value.find((item) => item.code === filters.groupCode)
  return selectedGroup
    ? selectedGroup.items || []
    : categoryGroups.value.flatMap((item) => item.items || [])
})
const selectedResourceType = computed(() => currentGroupResourceTypeItems.value.find((item) => item.typeCode === filters.typeCode))
const searchTagOptions = computed(() => {
  // 标签筛选只跟具体二级分类绑定；停留在“全部分类”时隐藏，避免用户被跨分类标签误导。
  if (!selectedResourceType.value) return []
  return normalizeSearchTagOptions(selectedResourceType.value.fieldSchema?.tagOptions || [])
})
const selectedGroupName = computed(() => {
  const selectedGroup = groupFilterOptions.value.find((item) => item.code === filters.groupCode)
  return selectedGroup?.name || '全部类目'
})
const channelTitle = computed(() => filters.groupCode ? selectedGroupName.value : '供需搜索')
const visibleResourceTypes = computed(() => resourceTypes.value)
const selectedTypeName = computed(() => resourceTypes.value.find((item) => item.value === filters.typeCode)?.label || '全部')
const showAllTypeButton = computed(() => resourceTypes.value.length - 1 > 3)
const trimmedKeyword = computed(() => keyword.value.trim())
const searchNavStyle = computed(() => `padding-top: ${headerMetrics.value.statusBarHeight}px;`)
const searchTitleBarStyle = computed(() => `height: ${headerMetrics.value.navBarHeight}px;`)
const searchPageStyle = computed(() => `padding-top: calc(${headerMetrics.value.headerHeight}px + 24rpx);`)
const searchToolbarStyle = computed(() => `top: ${headerMetrics.value.headerHeight}px;`)
const searchPlaceholder = computed(() => filters.groupCode
  ? `在${selectedGroupName.value}中搜索`
  : '搜供应、需求、场地或服务')
const emptyTitle = '暂无匹配内容'
const emptyDesc = computed(() => {
  if (trimmedKeyword.value) return '当前关键词暂无匹配。'
  if (filters.tags.length) return '当前标签暂无结果。'
  if (filters.groupCode && filters.typeCode) {
    return '当前分类暂无结果。'
  }
  if (filters.groupCode) return '该频道暂无内容。'
  return '换个关键词或分类试试。'
})
const emptyPrimaryActionLabel = computed(() => {
  if (trimmedKeyword.value) return '清空关键词'
  if (filters.tags.length) return '清空标签'
  if (filters.groupCode && filters.typeCode) {
    return `查看${selectedGroupName.value}全部`
  }
  return ''
})
const emptySuggestions = computed(() => {
  if (filters.groupCode && !filters.typeCode && !trimmedKeyword.value) return []
  return hotKeywords.value
    .filter((item) => item !== trimmedKeyword.value)
    .slice(0, 3)
})

onLoad(async (options = {}) => {
  updateHeaderMetrics()
  loadHotKeywordOptions()
  await loadResourceTypes()
  const routeSearched = await applyRouteSearch(options)
  if (!routeSearched && !hasPendingSearch()) {
    await search()
  }
})
onShow(applyPendingKeyword)
onPageScroll(handlePageScroll)

// 搜索结果使用接口分页；上拉只追加下一页，切换关键词或分类时重置到第一页。
onReachBottom(() => {
  search({ reset: false })
})

function updateHeaderMetrics() {
  try {
    const systemInfo = uni.getSystemInfoSync()
    const statusBarHeight = Number(systemInfo.statusBarHeight) || 44
    let menuTop = statusBarHeight + 6
    let menuHeight = 32

    if (typeof uni.getMenuButtonBoundingClientRect === 'function') {
      const menuButton = uni.getMenuButtonBoundingClientRect()
      menuTop = Number(menuButton.top) || menuTop
      menuHeight = Number(menuButton.height) || menuHeight
    }

    const navBarHeight = Math.max(44, (menuTop - statusBarHeight) * 2 + menuHeight)
    const bottomPadding = typeof uni.upx2px === 'function' ? uni.upx2px(NAV_BOTTOM_RPX) : 6
    headerMetrics.value = {
      statusBarHeight,
      navBarHeight,
      headerHeight: statusBarHeight + navBarHeight + bottomPadding,
    }
  } catch {
    headerMetrics.value = {
      statusBarHeight: 44,
      navBarHeight: 44,
      headerHeight: 94,
    }
  }
}

async function loadResourceTypes() {
  const resp = await listCityResourceTypes(filters.cityCode)
  categoryGroups.value = groupResourceTypes(resp.items || [])
  applyCurrentGroupTypes()
  await scrollToSelectedType(filters.typeCode)
}

async function loadHotKeywordOptions() {
  hotKeywords.value = await loadHotSearchKeywords(filters.cityCode)
}

async function applyRouteSearch(options = {}) {
  const routeKeyword = decodeSearchValue(options.keyword || options.q || '')
  const routeGroupCode = decodeSearchValue(options.groupCode || '')
  const routeTypeCode = decodeSearchValue(options.typeCode || '')
  const routeTags = parseSearchTags(options.tags || '')
  const routeCityCode = decodeSearchValue(options.cityCode || '') || DEFAULT_CITY_CODE
  if (!routeKeyword && !routeGroupCode && !routeTypeCode && !routeTags.length && routeCityCode === DEFAULT_CITY_CODE) return false
  keyword.value = routeKeyword
  filters.groupCode = routeGroupCode
  filters.typeCode = routeTypeCode
  filters.tags = routeTags
  filters.cityCode = routeCityCode
  await loadResourceTypes()
  await scrollToSelectedType(routeTypeCode)
  await search()
  return true
}

async function applyPendingKeyword() {
  const pendingSearch = uni.getStorageSync(SEARCH_KEY)
  if (!pendingSearch) return
  uni.removeStorageSync(SEARCH_KEY)
  if (typeof pendingSearch === 'string') {
    keyword.value = pendingSearch
  } else {
    keyword.value = pendingSearch.keyword || ''
    filters.groupCode = pendingSearch.groupCode || ''
    filters.typeCode = pendingSearch.typeCode || ''
    filters.tags = parseSearchTags(pendingSearch.tags || [])
    filters.cityCode = pendingSearch.cityCode || DEFAULT_CITY_CODE
  }
  await loadResourceTypes()
  await scrollToSelectedType(filters.typeCode)
  await search()
}

function hasPendingSearch() {
  return Boolean(uni.getStorageSync(SEARCH_KEY))
}

async function search({ reset = true, force = false } = {}) {
  if (loading.value && !force) return
  if (!reset && !hasMore.value) return
  const requestSeq = searchRequestSeq.value + 1
  searchRequestSeq.value = requestSeq
  loading.value = true
  try {
    const nextPage = reset ? 1 : page.value + 1
    const resp = await searchResources({
      ...filters,
      tags: filters.tags.join(','),
      keyword: keyword.value.trim(),
      page: nextPage,
      pageSize,
    })
    if (requestSeq !== searchRequestSeq.value) return
    const items = resp.items || []
    rows.value = reset ? items : [...rows.value, ...items]
    page.value = nextPage
    total.value = resp.total || rows.value.length
    hasMore.value = rows.value.length < total.value
    searched.value = true
  } finally {
    if (requestSeq === searchRequestSeq.value) {
      loading.value = false
    }
  }
}

function searchHotKeyword(value) {
  keyword.value = value
  search()
}

async function resetSearchConditions() {
  keyword.value = ''
  filters.typeCode = ''
  filters.tags = []
  showGroupDrawer.value = false
  showTypePanel.value = false
  applyCurrentGroupTypes()
  await scrollToSelectedType('')
  await search()
}

async function clearSearchKeyword() {
  keyword.value = ''
  await search()
}

async function handleEmptyPrimaryAction() {
  if (trimmedKeyword.value) {
    await clearSearchKeyword()
    return
  }
  if (filters.tags.length) {
    await clearSearchTags()
    return
  }
  if (filters.groupCode && filters.typeCode) {
    await resetSearchConditions()
  }
}

async function selectGroup(groupCode) {
  showGroupDrawer.value = false
  if (filters.groupCode === groupCode) return
  filters.groupCode = groupCode
  filters.typeCode = ''
  showTypePanel.value = false
  applyCurrentGroupTypes()
  await scrollToSelectedType('')
  await search()
}

async function selectType(typeCode) {
  filters.typeCode = typeCode
  showGroupDrawer.value = false
  showTypePanel.value = false
  syncSelectedTagsWithOptions()
  await scrollToSelectedType(typeCode)
  await search()
}

function getTypeButtonId(typeCode) {
  const key = typeCode || 'all'
  return `search-type-${String(key).replace(/[^a-zA-Z0-9_-]/g, '-')}`
}

function applyCurrentGroupTypes() {
  const items = currentGroupResourceTypeItems.value.map((item) => ({
    label: item.typeName,
    value: item.typeCode,
  }))
  resourceTypes.value = [{ label: '全部', value: '' }, ...items]
  if (filters.typeCode && !items.some((item) => item.value === filters.typeCode)) {
    filters.typeCode = ''
  }
  syncSelectedTagsWithOptions()
}

function normalizeSearchTagOptions(options = []) {
  const normalized = []
  const seen = new Set()
  for (const option of options) {
    const tag = normalizeSearchTagText(option)
    if (!tag || seen.has(tag)) continue
    seen.add(tag)
    normalized.push(tag)
  }
  return normalized
}

function normalizeSearchTags(tags = [], options = searchTagOptions.value) {
  const values = Array.isArray(tags) ? tags : [tags]
  const optionSet = new Set(options)
  const normalized = []
  const seen = new Set()
  for (const value of values) {
    const tag = normalizeSearchTagText(value)
    if (!tag || seen.has(tag) || (optionSet.size && !optionSet.has(tag))) continue
    if (normalized.length >= MAX_SEARCH_TAGS) break
    seen.add(tag)
    normalized.push(tag)
  }
  return normalized
}

function normalizeSearchTagText(value) {
  return String(value || '').trim()
}

function parseSearchTags(value) {
  if (Array.isArray(value)) {
    return value.flatMap(parseSearchTags)
  }
  return String(decodeSearchValue(value) || '')
    .split(/[,，]/)
    .map(normalizeSearchTagText)
    .filter(Boolean)
}

function syncSelectedTagsWithOptions() {
  filters.tags = normalizeSearchTags(filters.tags, searchTagOptions.value)
}

function isSearchTagSelected(tag) {
  return filters.tags.includes(tag)
}

async function toggleSearchTag(tag) {
  tag = normalizeSearchTagText(tag)
  if (!tag) return
  const currentTags = normalizeSearchTags(filters.tags, searchTagOptions.value)
  if (currentTags.includes(tag)) {
    filters.tags = currentTags.filter((item) => item !== tag)
  } else {
    if (currentTags.length >= MAX_SEARCH_TAGS) {
      uni.showToast({ title: `最多选择${MAX_SEARCH_TAGS}个标签`, icon: 'none' })
      return
    }
    filters.tags = normalizeSearchTags([...currentTags, tag], searchTagOptions.value)
  }
  await search()
}

async function clearSearchTags() {
  if (!filters.tags.length) return
  filters.tags = []
  await search()
}

async function scrollToSelectedType(typeCode = filters.typeCode) {
  // 横滑分类不回写 scroll-left，避免滚动中重新渲染抢占用户手势；只在选择分类后定位选中项。
  const nextId = getTypeButtonId(typeCode)
  if (scrollIntoTypeId.value === nextId) {
    scrollIntoTypeId.value = ''
    await nextTick()
  }
  scrollIntoTypeId.value = nextId
}

function openGroupDrawer() {
  showTypePanel.value = false
  showGroupDrawer.value = true
}

function closeGroupDrawer() {
  showGroupDrawer.value = false
}

function openTypePanel() {
  showGroupDrawer.value = false
  showTypePanel.value = true
}

function closeTypePanel() {
  showTypePanel.value = false
}

// 箭头只表达二级分类面板的展开状态；打开一级类目或选择分类时都会同步收起，避免两个选择层同时出现。
function toggleTypePanel() {
  if (showTypePanel.value) {
    closeTypePanel()
    return
  }
  openTypePanel()
}

function handlePageScroll(event = {}) {
  const scrollTop = Number(event.scrollTop) || 0
  pageScrollTop.value = scrollTop
}

async function restorePageScroll(scrollTop = pageScrollTop.value) {
  pageScrollTop.value = Number(scrollTop) || 0
  await nextTick()
  if (typeof uni.pageScrollTo === 'function') {
    uni.pageScrollTo({ scrollTop, duration: 0 })
  }
}

function goBack() {
  uni.navigateBack()
}

function decodeSearchValue(value) {
  if (!value) return ''
  try {
    return decodeURIComponent(value)
  } catch (err) {
    return value
  }
}

function openResource(item) {
  uni.navigateTo({ url: `/pages/resource/detail?id=${item.id}` })
}
</script>

<style lang="scss" scoped>
.search-page {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  padding: 24rpx;
  background: $wplink-bg;
}

.search-nav {
  position: fixed;
  top: 0;
  right: 0;
  left: 0;
  z-index: 30;
  box-sizing: border-box;
  padding-right: 24rpx;
  padding-bottom: 12rpx;
  padding-left: 24rpx;
  background: $wplink-bg;
  box-shadow: 0 8rpx 20rpx rgba(15, 23, 42, 0.04);
}

.search-title-bar {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  box-sizing: border-box;
}

.nav-back-button {
  position: absolute;
  left: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 64rpx;
  min-width: 64rpx;
  height: 64rpx;
  margin: 0;
  padding: 0;
  background: transparent;
  color: #111827;
}

.nav-back-button::after {
  border: 0;
}

.nav-back-icon {
  box-sizing: border-box;
  width: 18rpx;
  height: 18rpx;
  border-bottom: 4rpx solid currentColor;
  border-left: 4rpx solid currentColor;
  transform: rotate(45deg);
}

.search-title {
  color: $wplink-primary;
  font-size: 30rpx;
  font-weight: 800;
}

.channel-title-button {
  display: flex;
  align-items: center;
  justify-content: center;
  max-width: 420rpx;
  height: 64rpx;
  margin: 0;
  padding: 0 18rpx;
  background: transparent;
  color: $wplink-primary;
  line-height: 1;
}

.channel-title-button::after {
  border: 0;
}

.channel-title-text {
  min-width: 0;
  overflow: hidden;
  font-size: 30rpx;
  font-weight: 800;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.channel-title-arrow {
  box-sizing: border-box;
  width: 14rpx;
  height: 14rpx;
  margin-left: 10rpx;
  border-right: 3rpx solid currentColor;
  border-bottom: 3rpx solid currentColor;
  transform: rotate(45deg) translateY(-2rpx);
}

.search-toolbar {
  position: sticky;
  position: -webkit-sticky;
  z-index: 20;
  margin: -24rpx -24rpx 16rpx;
  padding: 24rpx 24rpx 16rpx;
  background: $wplink-bg;
  box-shadow: 0 10rpx 18rpx rgba(32, 42, 68, 0.06);
}

.search-bar {
  display: grid;
  grid-template-columns: 1fr 116rpx;
  align-items: center;
  gap: 16rpx;
  margin-bottom: 20rpx;
}

.search-input {
  height: 80rpx;
  border-radius: 10rpx;
}

.search-input {
  padding: 0 20rpx;
  border: 1rpx solid $wplink-line;
  background: $wplink-card;
}

.search-button {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 76rpx;
  padding: 0;
  border-radius: 10rpx;
  background: $wplink-primary;
  color: $wplink-card;
  font-size: 26rpx;
  font-weight: 700;
  line-height: 1;
}

.filter-shell {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  align-items: center;
  gap: 4rpx;
  height: 82rpx;
  padding: 4rpx;
  border: 1rpx solid $wplink-line;
  border-radius: 12rpx;
  background: $wplink-card;
  box-shadow: 0 2rpx 8rpx rgba(6, 22, 37, 0.03);
  margin-bottom: 0;
}

.filter-shell.has-type-panel-button {
  grid-template-columns: minmax(0, 1fr) 72rpx;
}

.filter-row {
  flex: 1;
  min-width: 0;
  width: 100%;
  height: 72rpx;
  white-space: nowrap;
  overflow-x: auto;
  overflow-y: hidden;
  -webkit-overflow-scrolling: touch;
}

.filter-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 112rpx;
  height: 72rpx;
  margin-right: 8rpx;
  padding: 0 20rpx;
  border-radius: 8rpx;
  background: transparent;
  color: #364152;
  font-size: 26rpx;
}

.filter-button.active {
  background: $wplink-primary;
  color: $wplink-card;
  font-weight: 700;
}

.type-panel-toggle {
  width: 72rpx;
  height: 72rpx;
  padding: 0;
  border-left: 1rpx solid $wplink-line;
  border-radius: 0 8rpx 8rpx 0;
  background: transparent;
  color: $wplink-primary;
}

.type-panel-toggle:active {
  background: rgba($wplink-primary, 0.04);
}

.type-panel-arrow {
  display: block;
  width: 14rpx;
  height: 14rpx;
  border-right: 3rpx solid currentColor;
  border-bottom: 3rpx solid currentColor;
  transform: rotate(45deg) translate(-2rpx, -2rpx);
}

.type-panel-arrow.expanded {
  transform: rotate(225deg) translate(-1rpx, -1rpx);
}

.type-panel {
  max-height: 360rpx;
  margin-top: 10rpx;
  padding: 0;
  border: 1rpx solid $wplink-line;
  border-radius: 12rpx;
  background: $wplink-card;
  box-shadow: 0 10rpx 24rpx rgba(6, 22, 37, 0.08);
  overflow: hidden;
}

.type-panel-content {
  padding: 16rpx;
}

.type-panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16rpx;
  margin-bottom: 14rpx;
}

.type-panel-title {
  flex: 0 0 auto;
  color: $wplink-text;
  font-size: 24rpx;
  font-weight: 700;
}

.type-panel-current {
  min-width: 0;
  overflow: hidden;
  color: $wplink-muted;
  font-size: 22rpx;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.type-panel-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12rpx;
}

.type-panel-button {
  position: relative;
  height: 72rpx;
  padding: 0 10rpx;
  border: 1rpx solid transparent;
  border-radius: 10rpx;
  background: $wplink-bg;
  color: #364152;
  font-size: 24rpx;
}

.type-panel-button.active {
  border-color: $wplink-primary;
  background: $wplink-primary;
  color: $wplink-card;
  font-weight: 700;
}

.type-panel-button-text {
  display: -webkit-box;
  overflow: hidden;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.type-panel-check {
  position: absolute;
  top: 10rpx;
  right: 12rpx;
  width: 10rpx;
  height: 6rpx;
  border-left: 2rpx solid currentColor;
  border-bottom: 2rpx solid currentColor;
  transform: rotate(-45deg);
}

.tag-filter-row {
  width: 100%;
  margin-top: 14rpx;
  white-space: nowrap;
  overflow-x: auto;
  overflow-y: hidden;
  -webkit-overflow-scrolling: touch;
}

.tag-filter-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 120rpx;
  height: 60rpx;
  margin-right: 12rpx;
  padding: 0 18rpx;
  border: 1rpx solid $wplink-line;
  border-radius: 10rpx;
  background: $wplink-card;
  color: #475569;
  font-size: 24rpx;
}

.tag-filter-button.active {
  border-color: rgba($wplink-primary, 0.28);
  background: $wplink-primary-soft;
  color: $wplink-primary;
  font-weight: 700;
}

.hot-row {
  display: flex;
  align-items: center;
  gap: 12rpx;
  margin-bottom: 20rpx;
  overflow-x: auto;
}

.hot-label {
  flex: 0 0 auto;
  color: $wplink-muted;
  font-size: 26rpx;
}

.hot-button {
  flex: 0 0 auto;
  min-width: 136rpx;
  height: 62rpx;
  padding: 0 18rpx;
  border-radius: 10rpx;
  background: $wplink-card;
  color: #364152;
  font-size: 24rpx;
}

.result-list {
  display: grid;
  gap: 18rpx;
}

.load-more-text {
  padding: 8rpx 0 18rpx;
  color: $wplink-muted;
  font-size: 24rpx;
  text-align: center;
}

.empty-card {
  display: grid;
  flex: 1;
  align-content: center;
  justify-items: center;
  gap: 12rpx;
  min-height: 420rpx;
  padding: 32rpx 24rpx;
  padding-bottom: 88rpx;
  border-radius: 12rpx;
  background: $wplink-card;
  text-align: center;
}

.empty-visual {
  position: relative;
  display: grid;
  place-items: center;
  width: 168rpx;
  height: 112rpx;
  margin-bottom: 2rpx;
  border-radius: 12rpx;
  background:
    linear-gradient(140deg, rgba(255, 255, 255, 0.28), transparent 42%),
    linear-gradient(135deg, rgba($wplink-primary, 0.32), rgba($wplink-blue, 0.18));
  overflow: hidden;
}

.empty-visual::before {
  position: absolute;
  top: 16rpx;
  right: 20rpx;
  width: 42rpx;
  height: 8rpx;
  border-radius: 999rpx;
  background: rgba(255, 255, 255, 0.34);
  content: '';
}

.empty-sheet {
  position: absolute;
  left: 26rpx;
  bottom: 18rpx;
  display: grid;
  gap: 7rpx;
  width: 64rpx;
  padding: 12rpx 10rpx;
  border-radius: 8rpx;
  background: rgba(255, 255, 255, 0.92);
  box-shadow: 0 8rpx 16rpx rgba(15, 23, 42, 0.08);
}

.empty-sheet text {
  display: block;
  height: 5rpx;
  border-radius: 999rpx;
  background: rgba($wplink-primary, 0.18);
}

.empty-sheet text:first-child {
  width: 44rpx;
  background: rgba($wplink-warning, 0.46);
}

.empty-magnifier {
  position: absolute;
  right: 32rpx;
  bottom: 28rpx;
  width: 42rpx;
  height: 42rpx;
  border: 6rpx solid rgba(255, 255, 255, 0.9);
  border-radius: 999rpx;
}

.empty-magnifier::after {
  position: absolute;
  right: -20rpx;
  bottom: -9rpx;
  width: 28rpx;
  height: 6rpx;
  border-radius: 999rpx;
  background: rgba(255, 255, 255, 0.9);
  content: '';
  transform: rotate(45deg);
}

.empty-title {
  color: $wplink-text;
  font-size: 28rpx;
  font-weight: 700;
}

.empty-desc {
  max-width: 560rpx;
  color: $wplink-muted;
  font-size: 24rpx;
  line-height: 1.5;
}

.empty-suggestions {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 12rpx;
  width: 100%;
  margin-top: 2rpx;
}

.empty-suggestion {
  height: 58rpx;
  padding: 0 18rpx;
  border-radius: 999rpx;
  background: $wplink-primary-soft;
  color: $wplink-primary;
  font-size: 24rpx;
}

.empty-actions {
  display: flex;
  justify-content: center;
  gap: 12rpx;
  flex-wrap: wrap;
  width: 100%;
  margin-top: 4rpx;
}

.primary-empty-button {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  max-width: 300rpx;
  height: 72rpx;
  border-radius: 12rpx;
  background: $wplink-primary;
  color: $wplink-card;
  font-size: 26rpx;
  font-weight: 700;
  box-shadow: none;
}

.group-drawer-mask {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 30;
  display: flex;
  align-items: flex-end;
  background: rgba(15, 23, 42, 0.38);
}

.group-drawer-panel {
  width: 100%;
  max-height: 72vh;
  padding: 26rpx 24rpx calc(30rpx + env(safe-area-inset-bottom));
  border-radius: 16rpx 16rpx 0 0;
  background: $wplink-bg;
}

.drawer-group-list {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12rpx;
}

.drawer-group-button {
  height: 76rpx;
  padding: 0 16rpx;
  border-radius: 10rpx;
  background: $wplink-card;
  color: #364152;
  font-size: 25rpx;
}

.drawer-group-button.active {
  background: $wplink-warning-soft;
  color: $wplink-primary;
  font-weight: 700;
}

.type-drawer-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16rpx;
}

.type-drawer-title {
  color: $wplink-primary;
  font-size: 32rpx;
  font-weight: 700;
}

.type-drawer-close {
  width: 112rpx;
  height: 58rpx;
  border-radius: 10rpx;
  background: $wplink-card;
  color: $wplink-muted;
  font-size: 24rpx;
}

</style>
