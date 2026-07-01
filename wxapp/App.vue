<script>
import { DEFAULT_CITY_CODE } from './common/constants'
import { loadHotSearchKeywords } from './common/hotSearchKeywords'

export default {
  onLaunch() {
    // 小程序启动时只保留本地会话，不在入口处做强制登录，避免阻断游客搜索和提交需求路径。
    loadHotSearchKeywords(DEFAULT_CITY_CODE)
  },
}
</script>

<style lang="scss">
@import './uni.scss';
</style>
