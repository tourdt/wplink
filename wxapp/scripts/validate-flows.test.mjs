import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import test from 'node:test'

import { validateFlows } from './validate-flows.mjs'

test('current wxapp pages satisfy MVP flow checks', () => {
  assert.deepEqual(validateFlows(path.resolve(new URL('..', import.meta.url).pathname)), [])
})

test('reports missing API call in required page flow', () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'wplink-wxapp-flow-'))
  fs.mkdirSync(path.join(root, 'pages/home'), { recursive: true })
  fs.writeFileSync(path.join(root, 'pages/home/index.vue'), '<script setup>function noop(){}</script>')

  const issues = validateFlows(root, [
    {
      file: 'pages/home/index.vue',
      checks: ['listHomeBanners'],
      description: '首页加载 Banner',
    },
  ])

  assert.deepEqual(issues, ['pages/home/index.vue 缺少 首页加载 Banner: listHomeBanners'])
})
