import assert from 'node:assert/strict'
import test from 'node:test'

import { buildTencentMapMarkers, fallbackMapCenter, placeIdFromMarker } from './tencentMapState.js'

test('native map markers distinguish claimed prelisted and selected places', () => {
  const places = [
    { objectId: 'a', claimed: true, name: '已入驻', lat: '30.90', lng: '120.20' },
    { objectId: 'b', claimed: false, name: '待认领', lat: '30.91', lng: '120.21' },
    { objectId: 'c', claimed: false, name: '缺坐标', lat: '', lng: '' },
  ]

  const markers = buildTencentMapMarkers(places, 'b')

  assert.equal(markers.length, 2)
  assert.match(markers[0].iconPath, /marker-claimed\.png$/)
  assert.match(markers[1].iconPath, /marker-selected\.png$/)
  assert.equal(placeIdFromMarker(markers[1].id, markers), 'b')
})

test('native map falls back to the first valid merchant coordinate', () => {
  assert.deepEqual(fallbackMapCenter([
    { lat: '', lng: '' },
    { lat: '30.90', lng: '120.20' },
  ], { latitude: 30.87, longitude: 120.12 }), {
    latitude: 30.9,
    longitude: 120.2,
  })
})
