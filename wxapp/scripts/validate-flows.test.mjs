import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import test from 'node:test'

import { validateFlows } from './validate-flows.mjs'

test('current wxapp pages satisfy MVP flow checks', () => {
  assert.deepEqual(validateFlows(path.resolve(new URL('..', import.meta.url).pathname)), [])
})

test('launch UI hides matching feature copy', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const files = [
    'pages/messages/index.vue',
    'pages/demand-success/index.vue',
    'pages/my/index.vue',
    'pages/search/index.vue',
    'pages/my-demands/index.vue',
  ]

  const visibleSource = files.map((file) => fs.readFileSync(path.join(root, file), 'utf8')).join('\n')

  assert.equal(visibleSource.includes('撮合'), false)
})

test('home banner only overlays labels and title on image', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/home/index.vue'), 'utf8')

  assert.equal(source.includes('banner-pill'), false)
  assert.equal(source.includes('banner-subtitle'), false)
  assert.match(source, /<image[^>]+class="banner-image"/)
  assert.match(source, /banner-kicker/)
  assert.match(source, /banner-title/)
})

test('home page keeps custom brand first screen structure', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/home/index.vue'), 'utf8')

  assert.equal(source.includes('search-divider'), false)
  assert.equal(source.includes('voice-icon'), false)
  assert.equal(source.includes('search-action-icon'), false)

  for (const token of [
    'home-fixed-header',
    'custom-title-bar',
    'home-brand',
    'brand-icon',
    '衣货通',
    'getMenuButtonBoundingClientRect',
    'homeContentStyle',
    '搜索现货、厂家或求购需求',
    'factory-hero',
    '织里站 · 精选工厂',
    '童装产业带数字化撮合中心',
    'quick-action-grid',
    '我要找货',
    '我要清货',
    '我要找厂',
    '我要接单',
  ]) {
    assert.match(source, new RegExp(token))
  }
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
