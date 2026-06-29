<template>
  <el-container class="admin-shell">
    <el-aside class="admin-aside" width="220px">
      <div class="brand">
        <span class="brand-mark">衣</span>
        <span class="brand-name">衣货通</span>
      </div>
      <el-menu router :default-active="$route.path" class="side-menu">
        <el-menu-item index="/dashboard">
          <el-icon><DataLine /></el-icon>
          <span>数据概览</span>
        </el-menu-item>
        <el-menu-item index="/resources/pending">
          <el-icon><Tickets /></el-icon>
          <span>资源审核</span>
        </el-menu-item>
        <el-menu-item index="/merchants">
          <el-icon><Shop /></el-icon>
          <span>商家管理</span>
        </el-menu-item>
        <el-menu-item index="/demands">
          <el-icon><Search /></el-icon>
          <span>采购需求</span>
        </el-menu-item>
        <el-menu-item index="/verifications">
          <el-icon><CircleCheck /></el-icon>
          <span>认证审核</span>
        </el-menu-item>
        <el-menu-item index="/entitlements">
          <el-icon><Ticket /></el-icon>
          <span>权益发放</span>
        </el-menu-item>
        <el-menu-item index="/banner-topics">
          <el-icon><Picture /></el-icon>
          <span>Banner 配置</span>
        </el-menu-item>
        <el-menu-item index="/resource-type-configs">
          <el-icon><Setting /></el-icon>
          <span>资源配置</span>
        </el-menu-item>
        <el-menu-item index="/operation-logs">
          <el-icon><Document /></el-icon>
          <span>操作日志</span>
        </el-menu-item>
        <el-menu-item index="/search-logs">
          <el-icon><Search /></el-icon>
          <span>搜索日志</span>
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
import { useRouter } from 'vue-router'
import { CircleCheck, DataLine, Document, Picture, Search, Setting, Shop, Ticket, Tickets, User } from '@element-plus/icons-vue'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const router = useRouter()

function handleCommand(command) {
  if (command === 'logout') {
    auth.logout()
    router.push({ name: 'login' })
  }
}
</script>
