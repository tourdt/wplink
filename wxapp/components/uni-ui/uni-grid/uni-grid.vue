<template>
  <view class="uni-grid-wrap">
    <view
      :id="elId"
      ref="uni-grid"
      class="uni-grid"
      :class="{ 'uni-grid--border': showBorder }"
      :style="{ 'border-left-color': borderColor }"
    >
      <slot />
    </view>
  </view>
</template>

<script>
export default {
  name: 'UniGrid',
  emits: ['change'],
  props: {
    column: {
      type: Number,
      default: 3,
    },
    showBorder: {
      type: Boolean,
      default: true,
    },
    borderColor: {
      type: String,
      default: '#D2D2D2',
    },
    square: {
      type: Boolean,
      default: true,
    },
    highlight: {
      type: Boolean,
      default: true,
    },
  },
  provide() {
    return {
      grid: this,
    }
  },
  data() {
    const elId = `Uni_${Math.ceil(Math.random() * 10e5).toString(36)}`
    return {
      elId,
      width: 0,
    }
  },
  created() {
    this.children = []
  },
  mounted() {
    this.$nextTick(() => {
      this.init()
    })
  },
  methods: {
    init() {
      setTimeout(() => {
        this.getSize((width) => {
          this.children.forEach((item) => {
            item.width = width
          })
        })
      }, 50)
    },
    change(e) {
      this.$emit('change', e)
    },
    getSize(fn) {
      uni
        .createSelectorQuery()
        .in(this)
        .select(`#${this.elId}`)
        .boundingClientRect()
        .exec((ret) => {
          this.width = `${parseInt((ret[0].width - 1) / this.column, 10)}px`
          fn(this.width)
        })
    },
  },
}
</script>

<style lang="scss">
.uni-grid-wrap {
  display: flex;
  flex: 1;
  flex-direction: column;
}

.uni-grid {
  display: flex;
  flex-direction: row;
  flex-wrap: wrap;
}

.uni-grid--border {
  position: relative;
  z-index: 1;
  border-left: 1px #d2d2d2 solid;
}
</style>
