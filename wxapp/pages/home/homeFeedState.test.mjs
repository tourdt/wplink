import assert from 'node:assert/strict'
import test from 'node:test'

import { getHomeFeedState } from './homeFeedState.js'

test('both feeds show the switcher and prefer recent merchants', () => {
  assert.deepEqual(
    getHomeFeedState({ merchantCount: 6, resourceCount: 8, hasRecommendCard: true }),
    {
      hasMerchants: true,
      hasResources: true,
      hasAnyContent: true,
      showSwitcher: true,
      defaultTab: 'merchants',
    }
  )
})

test('resources become the only visible feed when merchants are unavailable', () => {
  assert.deepEqual(
    getHomeFeedState({ merchantCount: 0, resourceCount: 1, hasRecommendCard: false }),
    {
      hasMerchants: false,
      hasResources: true,
      hasAnyContent: true,
      showSwitcher: false,
      defaultTab: 'resources',
    }
  )
})

test('a recommendation card counts as resource content', () => {
  assert.equal(
    getHomeFeedState({ merchantCount: 0, resourceCount: 0, hasRecommendCard: true }).defaultTab,
    'resources'
  )
})

test('empty feeds hide the whole home feed section', () => {
  assert.deepEqual(
    getHomeFeedState({ merchantCount: 0, resourceCount: 0, hasRecommendCard: false }),
    {
      hasMerchants: false,
      hasResources: false,
      hasAnyContent: false,
      showSwitcher: false,
      defaultTab: '',
    }
  )
})
