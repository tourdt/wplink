<template>
  <view v-if="width" :style="itemStyle" class="uni-grid-item">
    <view
      class="uni-grid-item__box"
      :class="{
        'uni-grid-item--border': showBorder,
        'uni-grid-item--border-top': showBorder && index < column,
        'uni-highlight': highlight,
      }"
      :style="{
        'border-right-color': borderColor,
        'border-bottom-color': borderColor,
        'border-top-color': borderColor,
      }"
      @click="onClick"
    >
      <slot />
    </view>
  </view>
</template>

<script>
export default {
  name: 'UniGridItem',
  inject: ['grid'],
  props: {
    index: {
      type: Number,
      default: 0,
    },
  },
  data() {
    return {
      column: 0,
      showBorder: true,
      square: true,
      highlight: true,
      width: 0,
      borderColor: '#e5e5e5',
    }
  },
  computed: {
    itemStyle() {
      return `width:${this.width};${this.square ? `height:${this.width}` : ''}`
    },
  },
  created() {
    this.column = this.grid.column
    this.showBorder = this.grid.showBorder
    this.square = this.grid.square
    this.highlight = this.grid.highlight
    this.borderColor = this.grid.borderColor
    this.grid.children.push(this)
    this.width = this.grid.width
  },
  beforeUnmount() {
    this.grid.children = this.grid.children.filter((item) => item !== this)
  },
  methods: {
    onClick() {
      this.grid.change({
        detail: {
          index: this.index,
        },
      })
    },
  },
}
</script>

<style lang="scss">
.uni-grid-item {
  display: flex;
  height: 100%;
}

.uni-grid-item__box {
  position: relative;
  display: flex;
  flex: 1;
  flex-direction: column;
  width: 100%;
}

.uni-grid-item--border {
  position: relative;
  z-index: 0;
  border-right: 1px #d2d2d2 solid;
  border-bottom: 1px #d2d2d2 solid;
}

.uni-grid-item--border-top {
  position: relative;
  z-index: 0;
  border-top: 1px #d2d2d2 solid;
}

.uni-highlight:active {
  background-color: #f1f1f1;
}
</style>
