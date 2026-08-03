import assert from 'node:assert/strict'
import test from 'node:test'

import {
  buildQuotaPurchaseUrl,
  QUOTA_TYPE_PUBLISH,
  QUOTA_TYPE_REFRESH,
} from './entitlementPurchase.js'

test('buildQuotaPurchaseUrl routes publish quota purchases to the publish add-on tab', () => {
  assert.equal(
    buildQuotaPurchaseUrl(QUOTA_TYPE_PUBLISH),
    '/pages/vip/index?tab=addons&quotaType=publish_quota',
  )
})

test('buildQuotaPurchaseUrl routes refresh quota purchases to the refresh add-on tab', () => {
  assert.equal(
    buildQuotaPurchaseUrl(QUOTA_TYPE_REFRESH),
    '/pages/vip/index?tab=addons&quotaType=refresh_quota',
  )
})
