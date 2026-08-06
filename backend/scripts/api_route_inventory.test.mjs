import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import test from 'node:test'

import {
  checkContractGeneratedParity,
  checkNoMigrationStubs,
  parseAPIContracts,
  parseGeneratedRoutes,
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

test('keeps an unindented legacy route immediately after a line comment', () => {
  withFixture({
    'legacy_routes.go': `package server
// mux.HandleFunc("GET /api/v1/retired", retired)
mux.HandleFunc("GET /api/v1/active", active)
`,
  }, (fixtureDir) => {
    const routes = parseLegacyRoutes([path.join(fixtureDir, 'legacy_routes.go')])

    assert.deepEqual(routes.map(routeFingerprint), [
      'GET /api/v1/active',
    ])
    assert.equal(routes[0].line, 3)
  })
})

test('parses generated routes with import aliases and applies each AddRoutes prefix', () => {
  withFixture({
    'routes.go': `package handler

import (
  "net/http"
  maphandler "wplink/backend/app/internal/handler/map"
  public "wplink/backend/app/internal/handler/public"
)

func RegisterHandlers(server *rest.Server, serverCtx *svc.ServiceContext) {
  // server.AddRoutes([]rest.Route{{Method: http.MethodDelete, Path: "/fake", Handler: public.FakeHandler(serverCtx)}})
  server.AddRoutes(
    []rest.Route{
      {Method: http.MethodGet, Path: "/scenes/:sceneId", Handler: maphandler.GetSceneHandler(serverCtx)},
    },
    rest.WithPrefix("/api/v1/map"),
  )
  server.AddRoutes(
    []rest.Route{
      {Method: http.MethodPost, Path: "/search", Handler: public.SearchHandler(serverCtx)},
    },
    rest.WithPrefix("/api/v1"),
  )
}
`,
  }, (fixtureDir) => {
    const routes = parseGeneratedRoutes(path.join(fixtureDir, 'routes.go'))

    assert.deepEqual(routes.map((route) => ({
      fingerprint: routeFingerprint(route),
      handler: route.handler,
      group: route.group,
    })), [
      { fingerprint: 'GET /api/v1/map/scenes/:sceneId', handler: 'GetScene', group: 'map' },
      { fingerprint: 'POST /api/v1/search', handler: 'Search', group: 'public' },
    ])
  })
})

test('reports sorted contract and generated route parity differences with exact handlers', () => {
  const contractRoutes = [
    { method: 'POST', path: '/api/v1/z', handler: 'CreateZ' },
    { method: 'GET', path: '/api/v1/a', handler: 'ReadA' },
  ]
  const generatedRoutes = [
    { method: 'GET', path: '/api/v1/a', handler: 'WrongReadA' },
    { method: 'DELETE', path: '/api/v1/b', handler: 'DeleteB' },
  ]

  assert.throws(
    () => checkContractGeneratedParity(contractRoutes, generatedRoutes),
    (error) => {
      assert.equal(error.message, `API 契约与生成路由不一致:
缺失:
- GET /api/v1/a -> ReadA
- POST /api/v1/z -> CreateZ
多余:
- DELETE /api/v1/b -> DeleteB
- GET /api/v1/a -> WrongReadA`)
      return true
    },
  )
})

test('reports sorted duplicate fingerprints in contracts and generated routes', () => {
  const contractRoutes = [
    { method: 'POST', path: '/api/v1/z', handler: 'CreateZAgain' },
    { method: 'GET', path: '/api/v1/a', handler: 'ReadA' },
    { method: 'POST', path: '/api/v1/z', handler: 'CreateZ' },
  ]
  const generatedRoutes = [
    { method: 'GET', path: '/api/v1/a', handler: 'ReadA' },
    { method: 'GET', path: '/api/v1/a', handler: 'ReadA' },
    { method: 'POST', path: '/api/v1/z', handler: 'CreateZ' },
    { method: 'POST', path: '/api/v1/z', handler: 'CreateZAgain' },
  ]

  assert.throws(
    () => checkContractGeneratedParity(contractRoutes, generatedRoutes),
    (error) => {
      assert.equal(error.message, `API 路由存在重复:
契约重复:
- POST /api/v1/z -> CreateZ
- POST /api/v1/z -> CreateZAgain
生成路由重复:
- GET /api/v1/a -> ReadA
- GET /api/v1/a -> ReadA
- POST /api/v1/z -> CreateZ
- POST /api/v1/z -> CreateZAgain`)
      return true
    },
  )
})

test('rejects a same-named Handler imported from another package', () => {
  const contractRoutes = [
    { method: 'GET', path: '/api/v1/map/scenes', handler: 'ListMapScenes', group: 'map' },
  ]
  const generatedRoutes = [
    {
      method: 'GET',
      path: '/api/v1/map/scenes',
      handler: 'ListMapScenes',
      group: 'map',
      handlerPackage: 'example.com/decoy/app/internal/handler/map',
    },
  ]

  assert.throws(
    () => checkContractGeneratedParity(contractRoutes, generatedRoutes),
    /example\.com\/decoy\/app\/internal\/handler\/map\.ListMapScenes/,
  )
})

test('detects real handlerx NotMigrated calls without matching comments strings or another package', () => {
  withFixture({
    'real/handler.go': `package real

import migrated "wplink/backend/app/internal/handler/handlerx"

func build() http.HandlerFunc {
  return migrated.NotMigrated("RealHandler")
}
`,
    'decoy/comment.go': `package decoy

// handlerx.NotMigrated("CommentHandler")
const text = "handlerx.NotMigrated(\\\"StringHandler\\\")"
`,
    'decoy/other.go': `package decoy

import other "example.com/other/app/internal/handler/handlerx"

func build() http.HandlerFunc {
  return other.NotMigrated("OtherHandler")
}
`,
    'real/handler_test.go': `package real

import handlerx "wplink/backend/app/internal/handler/handlerx"

func fixture() http.HandlerFunc {
  return handlerx.NotMigrated("TestOnlyHandler")
}
`,
  }, (fixtureDir) => {
    assert.throws(
      () => checkNoMigrationStubs(fixtureDir),
      (error) => {
        assert.match(error.message, /real\/handler\.go:6/)
        assert.match(error.message, /RealHandler/)
        assert.doesNotMatch(error.message, /CommentHandler|StringHandler|OtherHandler|TestOnlyHandler/)
        return true
      },
    )
  })
})

test('detects direct alias and function-value references to the real project NotMigrated symbol', () => {
  withFixture({
    'direct.go': `package fixture

import hx "wplink/backend/app/internal/handler/handlerx"

var forbidden = hx.NotMigrated
var parenthesized = (hx).NotMigrated
`,
    'indirect.go': `package fixture

import hx "wplink/backend/app/internal/handler/handlerx"

func build() http.HandlerFunc {
  stub := hx.NotMigrated
  return stub("IndirectHandler")
}
`,
  }, (fixtureDir) => {
    assert.throws(
      () => checkNoMigrationStubs(fixtureDir),
      (error) => {
        assert.match(error.message, /direct\.go:5:\d+ -> handlerx\.NotMigrated/)
        assert.match(error.message, /direct\.go:6:\d+ -> handlerx\.NotMigrated/)
        assert.match(error.message, /indirect\.go:6:\d+ -> handlerx\.NotMigrated/)
        return true
      },
    )
  })
})

test('does not report a project import alias shadowed by parameters or short declarations', () => {
  withFixture({
    'shadow.go': `package fixture

import hx "wplink/backend/app/internal/handler/handlerx"

var _ = hx.ClientIP

type localStub struct{}

func (localStub) NotMigrated(string) http.HandlerFunc { return nil }

func parameterShadow(hx localStub) http.HandlerFunc {
  return hx.NotMigrated("ParameterShadow")
}

func shortDeclarationShadow() http.HandlerFunc {
  hx := localStub{}
  return hx.NotMigrated("ShortDeclarationShadow")
}

func varDeclarationShadow() http.HandlerFunc {
  var hx localStub
  return hx.NotMigrated("VarDeclarationShadow")
}
`,
  }, (fixtureDir) => {
    assert.doesNotThrow(() => checkNoMigrationStubs(fixtureDir))
  })
})

test('restores the project import binding after a nested alias shadow ends', () => {
  withFixture({
    'nested.go': `package fixture

import hx "wplink/backend/app/internal/handler/handlerx"

type localStub struct{}

func (localStub) NotMigrated(string) http.HandlerFunc { return nil }

func build() http.HandlerFunc {
  {
    hx := localStub{}
    _ = hx.NotMigrated("NestedShadow")
  }
  return hx.NotMigrated("RestoredImport")
}
`,
  }, (fixtureDir) => {
    assert.throws(
      () => checkNoMigrationStubs(fixtureDir),
      (error) => {
        assert.match(error.message, /nested\.go:14:\d+ -> RestoredImport/)
        assert.doesNotMatch(error.message, /NestedShadow/)
        return true
      },
    )
  })
})

test('detects a dot-imported NotMigrated reference while ignoring a local shadow', () => {
  withFixture({
    'dot.go': `package fixture

import . "wplink/backend/app/internal/handler/handlerx"

var forbidden = NotMigrated
`,
    'dot_shadow.go': `package fixture

import . "wplink/backend/app/internal/handler/handlerx"

var _ = ClientIP

func build(NotMigrated func(string) http.HandlerFunc) http.HandlerFunc {
  return NotMigrated("LocalParameter")
}
`,
  }, (fixtureDir) => {
    assert.throws(
      () => checkNoMigrationStubs(fixtureDir),
      (error) => {
        assert.match(error.message, /dot\.go:5:\d+ -> handlerx\.NotMigrated/)
        assert.doesNotMatch(error.message, /LocalParameter/)
        return true
      },
    )
  })
})
