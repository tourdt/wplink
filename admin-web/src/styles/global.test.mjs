import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const source = fs.readFileSync(path.join(root, 'src/styles/global.css'), 'utf8')

function cssBlock(selector) {
  const escapedSelector = selector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const match = source.match(new RegExp(`${escapedSelector} \\{([\\s\\S]*?)\\n\\}`))
  return match?.[1] || ''
}

test('admin table empty state is centered in the table display area', () => {
  assert.match(cssBlock('.panel .el-table__empty-block'), /display:\s*flex;/)
  assert.match(cssBlock('.panel .el-table__empty-block'), /align-items:\s*center;/)
  assert.match(cssBlock('.panel .el-table__empty-block'), /justify-content:\s*center;/)
  assert.match(cssBlock('.panel .el-table__empty-block'), /min-height:\s*clamp\(180px, 32vh, 320px\);/)
  assert.match(cssBlock('.panel .el-table__empty-block'), /padding-bottom:\s*24px;/)
  assert.match(cssBlock('.panel .el-table--small .el-table__empty-block'), /min-height:\s*160px;/)
})
