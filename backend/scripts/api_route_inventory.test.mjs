import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import test from 'node:test'

import {
  parseAPIContracts,
  parseLegacyRoutes,
  routeFingerprint,
} from './api_route_inventory.mjs'

function withFixture(files, callback) {
  const fixtureDir = fs.mkdtempSync(path.join(os.tmpdir(), 'api-route-inventory-'))
  try {
    for (const [relativePath, source] of Object.entries(files)) {
      const filePath = path.join(fixtureDir, relativePath)
      fs.mkdirSync(path.dirname(filePath), { recursive: true })
      fs.writeFileSync(filePath, source)
    }
    return callback(fixtureDir)
  } finally {
    fs.rmSync(fixtureDir, { recursive: true, force: true })
  }
}

test('parses imported API contracts with prefixes and retains duplicate handlers', () => {
  withFixture({
    'app.api': 'import "public.api"\nimport "admin.api"\n',
    'public.api': `
@server (
  prefix: /api/v1/
  group: public
)
service example {
  @handler ReadItem
  get /items/:itemId/ returns (ItemResp)
}
`,
    'admin.api': `
@server (
  prefix: /api/v1/admin
  group: admin
)
service example {
  @handler ReadItem
  post /items returns (ItemResp)
}
`,
  }, (fixtureDir) => {
    const routes = parseAPIContracts(fixtureDir)

    assert.deepEqual(routes.map(routeFingerprint), [
      'GET /api/v1/items/:itemId',
      'POST /api/v1/admin/items',
    ])
    assert.deepEqual(routes.map((route) => route.handler), ['ReadItem', 'ReadItem'])
    assert.deepEqual(routes.map((route) => route.group), ['public', 'admin'])
  })
})

test('ignores commented legacy routes and normalizes Go path parameters', () => {
  withFixture({
    'legacy_routes.go': `package server

func register(mux *http.ServeMux) {
  /*
  mux.HandleFunc("GET /api/v1/retired/{itemId}", retired)
  */
  // mux.HandleFunc("GET /api/v1/also-retired/{itemId}", retired)
  mux.HandleFunc("get //api/v1/items/{itemId}/", active)
}
`,
  }, (fixtureDir) => {
    const legacyFile = path.join(fixtureDir, 'legacy_routes.go')
    const routes = parseLegacyRoutes([legacyFile])

    assert.deepEqual(routes.map(routeFingerprint), [
      'GET /api/v1/items/:itemId',
    ])
    assert.equal(routes[0].line, 8)
    assert.equal(routes[0].source, legacyFile)
  })
})

test('keeps legacy routes that share a path but use different methods', () => {
  withFixture({
    'legacy_routes.go': `package server
func register(mux *http.ServeMux) {
  mux.HandleFunc("GET /api/v1/items/{itemId}", readItem)
  mux.HandleFunc("POST /api/v1/items/{itemId}", updateItem)
}
`,
  }, (fixtureDir) => {
    const routes = parseLegacyRoutes([path.join(fixtureDir, 'legacy_routes.go')])

    assert.deepEqual(routes.map(routeFingerprint), [
      'GET /api/v1/items/:itemId',
      'POST /api/v1/items/:itemId',
    ])
  })
})
