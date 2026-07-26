<template>
  <view class="legal-page">
    <text class="legal-title">{{ document.title }}</text>
    <text class="legal-version">生效日期：2026年7月25日</text>
    <view v-for="section in document.sections" :key="section.title" class="legal-section">
      <text class="section-title">{{ section.title }}</text>
      <text class="section-body">{{ section.body }}</text>
    </view>
    <button class="service-button" open-type="contact">对政策有疑问，联系小程序客服</button>
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'

const type = ref('privacy')

const documents = {
  privacy: {
    title: '隐私政策',
    sections: [
      {
        title: '一、我们收集的信息',
        body: '为提供登录、发布、联系、收藏和消息功能，我们会处理微信登录标识、账号资料、您主动填写的联系电话与微信号、发布内容、操作记录及必要的设备和网络日志。使用地图选点或附近功能时，经您授权后处理位置信息。',
      },
      {
        title: '二、使用目的与公开范围',
        body: '这些信息用于账号识别、供需撮合、内容审核、安全防护、客户服务和统计分析。您主动发布的标题、描述、图片、商家资料及联系方式可能向其他用户展示；请勿填写与交易无关的个人敏感信息。',
      },
      {
        title: '三、第三方服务',
        body: '业务可能使用微信登录与内容安全、地图定位、短信验证、对象存储及微信支付等服务。我们仅在实现对应功能所需的范围内传递必要信息，并要求服务方依其规则保护数据。',
      },
      {
        title: '四、保存、安全与您的权利',
        body: '我们按业务和合规所需期限保存数据，并采取访问控制、审计和异常登录防护。您可以在“我的—账号与隐私”查看政策、退出登录或申请注销。注销后账号停用，个人资料会匿名化；依法需要保留的交易、安全和审计记录将在期限届满后处理。',
      },
      {
        title: '五、未成年人和政策更新',
        body: '未成年人应在监护人指导下使用。政策发生重要变化时，我们会通过页面提示并重新取得必要同意。若不同意更新，可停止使用并申请注销账号。',
      },
    ],
  },
  agreement: {
    title: '用户协议',
    sections: [
      {
        title: '一、服务说明',
        body: '衣货通提供本地供需信息发布、浏览、联系和商家展示服务。平台不直接成为线下交易当事人，不对用户自行约定的价格、交付和质量作保证。',
      },
      {
        title: '二、账号与内容责任',
        body: '您应提供真实、合法、与业务相关的信息并妥善保管账号。不得发布违法、侵权、虚假、欺诈、骚扰或引流至非法交易的内容，不得绕过审核、限流或安全措施。',
      },
      {
        title: '三、审核与处置',
        body: '平台可通过自动化规则和人工复核处理内容。对存在风险的内容，平台可限制展示、要求修改、下架或封禁账号；对明显误判可通过客服或举报反馈入口申请处理。',
      },
      {
        title: '四、交易提示',
        body: '联系电话和商家信息由发布者提供，现阶段不代表平台已核验。联系或交易前请自行核实对方身份、货品、场地、资质和付款条件，谨防私下转账及异常低价。',
      },
      {
        title: '五、终止服务',
        body: '您可以停止使用或申请注销。严重违反协议、危害平台或他人权益时，平台可限制或终止服务，并按法律要求保存必要记录。',
      },
    ],
  },
}

const document = computed(() => documents[type.value] || documents.privacy)

onLoad((options = {}) => {
  type.value = options.type === 'agreement' ? 'agreement' : 'privacy'
  uni.setNavigationBarTitle({ title: document.value.title })
})
</script>

<style lang="scss" scoped>
.legal-page {
  min-height: 100vh;
  padding: 32rpx 28rpx 56rpx;
  background: $wplink-bg;
}

.legal-title {
  display: block;
  color: $wplink-primary;
  font-size: 42rpx;
  font-weight: 700;
}

.legal-version {
  display: block;
  margin: 10rpx 0 30rpx;
  color: $wplink-muted;
  font-size: 24rpx;
}

.legal-section {
  margin-bottom: 18rpx;
  padding: 26rpx;
  border-radius: 12rpx;
  background: $wplink-card;
}

.section-title,
.section-body {
  display: block;
}

.section-title {
  margin-bottom: 12rpx;
  color: $wplink-primary;
  font-size: 30rpx;
  font-weight: 700;
}

.section-body {
  color: $wplink-text;
  font-size: 27rpx;
  line-height: 1.8;
}

.service-button {
  margin-top: 20rpx;
  border: 1rpx solid $wplink-line;
  background: $wplink-card;
  color: $wplink-accent;
  font-size: 27rpx;
}
</style>
