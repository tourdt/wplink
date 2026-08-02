import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const componentPath = path.resolve(new URL('.', import.meta.url).pathname, 'ResourceFeedCard.vue')
const source = fs.existsSync(componentPath) ? fs.readFileSync(componentPath, 'utf8') : ''

test('resource feed card renders one compact structure from the normalized model', () => {
  assert.equal(fs.existsSync(componentPath), true)
  assert.match(source, /import \{ buildResourceFeedCardModel \} from '\.\/resourceFeedCardState\.js'/)
  assert.match(source, /const cardModel = computed\(\(\) => buildResourceFeedCardModel\(props\.resource\)\)/)
  assert.match(source, /:class="\['resource-feed-card', \{ demand: cardModel\.isDemand \}\]"/)
  assert.match(source, /\{\{ cardModel\.resourceTypeLabel \|\| cardModel\.directionLabel \}\}/)
  assert.match(source, /\{\{ cardModel\.titleText \}\}/)
  assert.match(source, /\{\{ cardModel\.merchantName \}\}/)
  assert.doesNotMatch(source, /<template v-if="cardModel\.isDemand">/)
})

test('resource feed card preserves the click contract and compact visual hierarchy', () => {
  assert.match(source, /defineEmits\(\['open'\]\)/)
  assert.match(source, /role="button"/)
  assert.match(source, /@click="\$emit\('open', resource\)"/)
  assert.match(source, /\.resource-feed-card \{[\s\S]*transition: transform 120ms ease-out;/)
  assert.match(source, /\.resource-feed-card:active \{[\s\S]*transform: translateY\(1rpx\);/)
  assert.match(source, /\.feed-thumb-wrap \{[\s\S]*width: 152rpx;[\s\S]*height: 152rpx;/)
  assert.match(source, /\.feed-card-main \{[\s\S]*align-content: space-between;/)
  assert.match(source, /\.feed-type-badge\.demand \{[\s\S]*background: \$wplink-warning;/)
  assert.match(source, /\.feed-type-badge:not\(\.demand\) \{[\s\S]*background: \$wplink-primary;/)
})

test('resource feed card overlays completed resources with a readable non-interactive watermark', () => {
  assert.match(
    source,
    /<text v-if="cardModel\.isCompleted" class="feed-completed-watermark">已完成<\/text>/,
  )
  assert.match(source, /\.resource-feed-card \{[\s\S]*position: relative;/)
  assert.match(source, /\.feed-completed-watermark \{[\s\S]*position: absolute;[\s\S]*pointer-events: none;/)
  assert.match(source, /\.feed-completed-watermark \{[\s\S]*font-size: 76rpx;[\s\S]*transform: translate\(-50%, -50%\) rotate\(-18deg\);/)
  assert.match(source, /\.feed-thumb-wrap,[\s\S]*\.feed-card-main \{[\s\S]*position: relative;[\s\S]*z-index: 1;/)
})
