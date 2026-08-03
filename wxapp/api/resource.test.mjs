import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'
import { pathToFileURL } from 'node:url'
import vm from 'node:vm'

const resourceApiPath = path.resolve(new URL('.', import.meta.url).pathname, 'resource.js')

test('resource detail APIs preserve backend presentation field order and layout', async () => {
  const fields = [
    { key: 'quantity', label: '数量/面积', value: '5000 件', layout: 'half' },
    { key: 'deliveryTime', label: '交期要求', value: '20 天内完成', layout: 'full' },
  ]
  const tags = ['现货', '可混批']
  const calls = []
  const api = await loadResourceApi(async (options) => {
    calls.push(options)
    return { id: 'resource-1', presentation: { fields, tags } }
  })

  const publicDetail = await api.getResource('resource-1', { suppressErrorToast: true })
  const ownDetail = await api.getOwnResource('resource-1', 'merchant-1', { suppressErrorToast: true })

  assert.deepEqual(publicDetail.presentation.fields, fields)
  assert.deepEqual(ownDetail.presentation.fields, fields)
  assert.equal(publicDetail.presentation.fields, fields)
  assert.equal(ownDetail.presentation.fields, fields)
  assert.deepEqual(publicDetail.presentation.tags, tags)
  assert.deepEqual(ownDetail.presentation.tags, tags)
  assert.equal(publicDetail.presentation.tags, tags)
  assert.equal(ownDetail.presentation.tags, tags)
  assert.deepEqual(calls, [
    {
      url: '/api/v1/resources/resource-1',
      method: 'GET',
      suppressErrorToast: true,
    },
    {
      url: '/api/v1/me/resources/resource-1/detail?merchantId=merchant-1',
      method: 'GET',
      suppressErrorToast: true,
      requireAuth: true,
    },
  ])
})

test('resource detail APIs standardize missing presentation lists to empty arrays', async () => {
  const api = await loadResourceApi(async () => ({ id: 'resource-1' }))

  const detail = await api.getResource('resource-1')

  assert.deepEqual(detail, {
    id: 'resource-1',
    presentation: { fields: [], tags: [] },
  })
})

async function loadResourceApi(requestImpl) {
  const source = fs.readFileSync(resourceApiPath, 'utf8')
  const module = new vm.SourceTextModule(source, {
    identifier: pathToFileURL(resourceApiPath).href,
  })
  await module.link(async (specifier) => {
    assert.equal(specifier, './request')
    return new vm.SyntheticModule(['default'], function setRequestExport() {
      this.setExport('default', requestImpl)
    })
  })
  await module.evaluate()
  return module.namespace
}
