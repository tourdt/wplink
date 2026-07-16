<template>
  <el-container class="admin-shell">
    <el-aside class="admin-aside" width="220px">
      <div class="brand">
        <span class="brand-mark">衣</span>
        <span class="brand-name">衣货通</span>
      </div>
      <el-menu router :default-active="$route.path" class="side-menu">
        <el-menu-item v-for="item in visibleMenuItems" :key="item.index" :index="item.index">
          <el-icon><component :is="item.icon" /></el-icon>
          <span>{{ item.label }}</span>
        </el-menu-item>
      </el-menu>
    </el-aside>

    <el-container>
      <el-header class="admin-header">
        <div>
          <strong>织里站</strong>
          <span class="header-subtitle">运营后台</span>
        </div>
        <el-dropdown @command="handleCommand">
          <button class="user-button" type="button">
            <el-icon><User /></el-icon>
            <span>{{ auth.displayName }}</span>
          </button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="logout">退出登录</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </el-header>

      <el-main class="admin-main">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup>
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { CircleCheck, DataLine, Document, Lock, MapLocation, Medal, Picture, Search, Setting, Shop, Ticket, Tickets, User, Warning } from '@element-plus/icons-vue'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const router = useRouter()
const menuItems = [
  { index: '/dashboard', label: '数据概览', icon: DataLine, moduleCode: 'dashboard' },
  { index: '/resources/pending', label: '供需信息审核', icon: Tickets, moduleCode: 'resource_review' },
  { index: '/resource-reports', label: '举报审核', icon: Warning, moduleCode: 'resource_reports' },
  { index: '/merchants', label: '商家管理', icon: Shop, moduleCode: 'merchants' },
  { index: '/verifications', label: '认证审核', icon: CircleCheck, moduleCode: 'verification_review' },
  { index: '/entitlements', label: '权益发放', icon: Ticket, moduleCode: 'entitlements' },
  { index: '/banner-topics', label: '首页运营位', icon: Picture, moduleCode: 'banner_topics' },
  { index: '/hot-search-keywords', label: '热门搜索词', icon: Search, moduleCode: 'hot_search_keywords' },
  { index: '/vip-configs', label: 'VIP 配置', icon: Medal, moduleCode: 'vip_configs' },
  { index: '/growth-campaigns', label: '增长活动', icon: Tickets, moduleCode: 'growth_campaigns' },
  { index: '/sourcing-map', label: '拿货地图', icon: MapLocation, moduleCode: 'sourcing_map' },
  { index: '/resource-type-configs', label: '供需类型配置', icon: Setting, moduleCode: 'resource_type_configs' },
  { index: '/admin-permissions', label: '管理员权限', icon: Lock, moduleCode: 'admin_permissions' },
  { index: '/operation-logs', label: '操作日志', icon: Document, moduleCode: 'operation_logs' },
  { index: '/search-logs', label: '搜索日志', icon: Search, moduleCode: 'search_logs' },
]
const visibleMenuItems = computed(() =>
  menuItems.filter((item) => (item.moduleCode === 'admin_permissions' ? auth.isSuperAdmin : auth.canAccessModule(item.moduleCode))),
)

function handleCommand(command) {
  if (command === 'logout') {
    auth.logout()
    router.push({ name: 'login' })
  }
}
</script>
