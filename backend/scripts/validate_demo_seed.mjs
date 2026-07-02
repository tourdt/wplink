import fs from 'node:fs'
import path from 'node:path'

const scriptPath = path.resolve(new URL('.', import.meta.url).pathname, 'seed_demo_data.sql')

if (!fs.existsSync(scriptPath)) {
  throw new Error('缺少 backend/scripts/seed_demo_data.sql')
}

const sql = fs.readFileSync(scriptPath, 'utf8')
const requiredSnippets = [
  '认证工厂',
  '认证库存商',
  '服务商',
  '采购商',
  'type_code',
  'inventory',
  'goods',
  'factory',
  'order',
  'job',
  'rental',
  'service',
  'pending',
  'rejected',
  'expired',
  'match_cases',
  'messages',
  'resource_metrics_daily',
  'map_scene',
  'map_object',
  'zhili_lijilu_demo',
  '利济路童装拿货示范图',
  '晨星童装 A 区',
  '云仓尾货 B 区',
  '利济路打包点',
  'A006',
  'C306',
  '童装城快递集包点',
  '利济路样衣中心',
  '早餐简餐补给点',
  'published',
]

for (const snippet of requiredSnippets) {
  if (!sql.includes(snippet)) {
    throw new Error(`演示种子缺少关键内容: ${snippet}`)
  }
}

const demoSceneRefs = sql.match(/'zhili_lijilu_demo'/g) || []
if (demoSceneRefs.length < 13) {
  throw new Error(`拿货地图演示数据对象偏少: scene refs=${demoSceneRefs.length}, want >= 13`)
}

if (sql.includes('8020000000000002,')) {
  throw new Error('演示种子包含疑似截断的商家 ID: 8020000000000002')
}

console.log('demo seed static check ok')
