<template>
  <view class="redirect-page"><text>正在打开找档口</text><button @click="openDirectory">进入找档口</button></view>
</template>
<script setup>
import { onLoad } from '@dcloudio/uni-app'
function openDirectory() { uni.switchTab({ url: '/pages/sourcing-map/index' }) }
// 保留旧首页路径，历史分享和登录回跳统一落到当前找档口入口。
onLoad(openDirectory)
</script>
<style scoped>
.redirect-page { padding: 100rpx 40rpx; text-align: center; color: #6b7280; }
button { margin-top: 30rpx; color: #c23a00; }
</style>
