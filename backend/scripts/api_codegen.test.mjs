import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { spawnSync } from 'node:child_process'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

const backendDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')

function copyIfExists(source, destination) {
  if (!fs.existsSync(source)) {
    return
  }
  fs.mkdirSync(path.dirname(destination), { recursive: true })
  fs.cpSync(source, destination, { recursive: true })
}

function createBackendFixture() {
  const fixtureDir = fs.mkdtempSync(path.join(os.tmpdir(), 'api-codegen-'))
  const fixtureBackendDir = path.join(fixtureDir, 'backend')

  copyIfExists(path.join(backendDir, 'app/api'), path.join(fixtureBackendDir, 'app/api'))
  copyIfExists(path.join(backendDir, 'app/goctl'), path.join(fixtureBackendDir, 'app/goctl'))
  copyIfExists(path.join(backendDir, 'app/internal/handler'), path.join(fixtureBackendDir, 'app/internal/handler'))
  copyIfExists(path.join(backendDir, 'app/internal/types/types.go'), path.join(fixtureBackendDir, 'app/internal/types/types.go'))
  copyIfExists(path.join(backendDir, 'scripts/api_route_inventory.mjs'), path.join(fixtureBackendDir, 'scripts/api_route_inventory.mjs'))
  copyIfExists(path.join(backendDir, 'scripts/api_not_migrated_scanner.go'), path.join(fixtureBackendDir, 'scripts/api_not_migrated_scanner.go'))
  copyIfExists(path.join(backendDir, 'scripts/api_codegen.mjs'), path.join(fixtureBackendDir, 'scripts/api_codegen.mjs'))

  return { fixtureDir, fixtureBackendDir }
}

function runGenerator(fixtureBackendDir, userGoctlHome, mode) {
  return spawnSync(process.execPath, [`scripts/api_codegen.mjs`, mode], {
    cwd: fixtureBackendDir,
    env: {
      ...process.env,
      GOCTL_HOME: userGoctlHome,
    },
    encoding: 'utf8',
  })
}

function writeFixtureFile(fixtureBackendDir, relativePath, source) {
  const filePath = path.join(fixtureBackendDir, relativePath)
  fs.mkdirSync(path.dirname(filePath), { recursive: true })
  fs.writeFileSync(filePath, source)
  return filePath
}

function withGeneratedFixture(callback) {
  const { fixtureDir, fixtureBackendDir } = createBackendFixture()
  try {
    const userGoctlHome = path.join(fixtureDir, 'unused-user-home')
    const writeResult = runGenerator(fixtureBackendDir, userGoctlHome, '--write')
    assert.equal(writeResult.status, 0, `${writeResult.stdout}\n${writeResult.stderr}`)
    return callback({ fixtureBackendDir, userGoctlHome })
  } finally {
    fs.rmSync(fixtureDir, { recursive: true, force: true })
  }
}

function assertMissingMerchantHandler(checkResult) {
  const output = `${checkResult.stdout}\n${checkResult.stderr}`
  assert.notEqual(checkResult.status, 0, output)
  assert.match(output, /merchant\/get_merchant_handler\.go/)
  assert.match(output, /缺少 GetMerchantHandler/)
}

function readGoFiles(directory) {
  const sources = []
  for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
    const entryPath = path.join(directory, entry.name)
    if (entry.isDirectory()) {
      sources.push(...readGoFiles(entryPath))
    } else if (entry.name.endsWith('.go')) {
      sources.push(fs.readFileSync(entryPath, 'utf8'))
    }
  }
  return sources
}

function mutateGeneratedRoutes(fixtureBackendDir, mutate) {
  const routesPath = path.join(fixtureBackendDir, 'app/internal/handler/routes.go')
  const source = fs.readFileSync(routesPath, 'utf8')
  const mutated = mutate(source)
  assert.notEqual(mutated, source, '路由篡改夹具未命中生成源码')
  fs.writeFileSync(routesPath, mutated)
}

function assertParityFailure(checkResult, patterns) {
  const output = `${checkResult.stdout}\n${checkResult.stderr}`
  assert.notEqual(checkResult.status, 0, output)
  assert.match(output, /API 契约与生成路由不一致|API 路由存在重复/)
  for (const pattern of patterns) {
    assert.match(output, pattern)
  }
}

test('write handles keyword API groups, isolates GOCTL_HOME, and preserves existing handlers', () => {
  const { fixtureDir, fixtureBackendDir } = createBackendFixture()
  try {
    const poisonedHome = path.join(fixtureDir, 'poisoned-goctl-home')
    fs.mkdirSync(path.join(poisonedHome, 'api'), { recursive: true })
    fs.writeFileSync(path.join(poisonedHome, 'api/handler.tpl'), `package {{.PkgName}}

import "zmall/common/response"
`)

    const adminHandlerPath = path.join(fixtureBackendDir, 'app/internal/handler/adminauth/admin_login_handler.go')
    const cityHandlerPath = path.join(fixtureBackendDir, 'app/internal/handler/city/city_handler.go')
    const adminHandlerBefore = fs.readFileSync(adminHandlerPath, 'utf8')
    const cityHandlerBefore = fs.readFileSync(cityHandlerPath, 'utf8')

    const result = runGenerator(fixtureBackendDir, poisonedHome, '--write')

    assert.equal(result.status, 0, `${result.stdout}\n${result.stderr}`)
    assert.equal(fs.readFileSync(adminHandlerPath, 'utf8'), adminHandlerBefore)
    assert.equal(fs.readFileSync(cityHandlerPath, 'utf8'), cityHandlerBefore)

    const handlerSources = readGoFiles(path.join(fixtureBackendDir, 'app/internal/handler'))
    assert.ok(handlerSources.every((source) => !source.includes('zmall/')))
    assert.ok(fs.existsSync(path.join(fixtureBackendDir, 'app/internal/handler/routes.go')))
    assert.ok(fs.existsSync(path.join(fixtureBackendDir, 'app/internal/handler/routes_manifest_gen.go')))
    assert.ok(fs.existsSync(path.join(fixtureBackendDir, 'app/internal/types/types.go')))
    assert.equal(fs.existsSync(path.join(fixtureBackendDir, 'app/internal/config/config.go')), false)
    assert.equal(fs.existsSync(path.join(fixtureBackendDir, 'app/internal/svc/service_context.go')), false)
    assert.equal(fs.existsSync(path.join(fixtureBackendDir, 'app/app.go')), false)
  } finally {
    fs.rmSync(fixtureDir, { recursive: true, force: true })
  }
})

test('write assigns a deterministic unique alias when map and maphandler groups coexist', () => {
  const { fixtureDir, fixtureBackendDir } = createBackendFixture()
  try {
    fs.appendFileSync(path.join(fixtureBackendDir, 'app/api/app.api'), '\nimport "maphandler_collision.api"\n')
    writeFixtureFile(
      fixtureBackendDir,
      'app/api/maphandler_collision.api',
      `syntax = "v1"

@server (
  prefix: /api/v1/alias-collision
  group: maphandler
)
service wplink-api {
  @handler AliasCollisionProbe
  get /probe
}
`,
    )

    const userGoctlHome = path.join(fixtureDir, 'unused-user-home')
    const firstResult = runGenerator(fixtureBackendDir, userGoctlHome, '--write')
    assert.equal(firstResult.status, 0, `${firstResult.stdout}\n${firstResult.stderr}`)

    const routesPath = path.join(fixtureBackendDir, 'app/internal/handler/routes.go')
    const firstRoutes = fs.readFileSync(routesPath, 'utf8')
    assert.match(firstRoutes, /maphandler2 "wplink\/backend\/app\/internal\/handler\/map"/)
    assert.match(firstRoutes, /maphandler "wplink\/backend\/app\/internal\/handler\/maphandler"/)
    assert.match(firstRoutes, /maphandler2\.ListMapScenesHandler/)
    assert.match(firstRoutes, /maphandler\.AliasCollisionProbeHandler/)
    assert.match(
      fs.readFileSync(path.join(fixtureBackendDir, 'app/internal/handler/map/list_map_scenes_handler.go'), 'utf8'),
      /^package maphandler$/m,
    )

    const secondResult = runGenerator(fixtureBackendDir, userGoctlHome, '--write')
    assert.equal(secondResult.status, 0, `${secondResult.stdout}\n${secondResult.stderr}`)
    assert.equal(fs.readFileSync(routesPath, 'utf8'), firstRoutes)
  } finally {
    fs.rmSync(fixtureDir, { recursive: true, force: true })
  }
})

test('write creates a compilable marked handler skeleton without the retired NotMigrated helper', () => {
  const { fixtureDir, fixtureBackendDir } = createBackendFixture()
  try {
    const handlerPath = path.join(fixtureBackendDir, 'app/internal/handler/merchant/get_merchant_handler.go')
    fs.rmSync(handlerPath)

    const writeResult = runGenerator(fixtureBackendDir, path.join(fixtureDir, 'unused-user-home'), '--write')

    assert.equal(writeResult.status, 0, `${writeResult.stdout}\n${writeResult.stderr}`)
    const generated = fs.readFileSync(handlerPath, 'utf8')
    assert.match(generated, /WPLINK_API_HANDLER_STUB/)
    assert.doesNotMatch(generated, /NotMigrated/)
    assert.match(generated, /response\.JSON/)
    assert.match(generated, /CodeInternalError/)
  } finally {
    fs.rmSync(fixtureDir, { recursive: true, force: true })
  }
})

test('check rejects a handler mutated with the generated skeleton marker', () => {
  withGeneratedFixture(({ fixtureBackendDir, userGoctlHome }) => {
    const handlerPath = path.join(fixtureBackendDir, 'app/internal/handler/merchant/get_merchant_handler.go')
    fs.appendFileSync(handlerPath, '\nconst WPLINK_API_HANDLER_STUB = "GetMerchantHandler"\n')

    const checkResult = runGenerator(fixtureBackendDir, userGoctlHome, '--check')
    const output = `${checkResult.stdout}\n${checkResult.stderr}`
    assert.notEqual(checkResult.status, 0, output)
    assert.match(output, /merchant\/get_merchant_handler\.go:\d+/)
    assert.match(output, /WPLINK_API_HANDLER_STUB/)
  })
})

test('check reports a tampered generated prefix with exact missing and extra fingerprints', () => {
  withGeneratedFixture(({ fixtureBackendDir, userGoctlHome }) => {
    mutateGeneratedRoutes(fixtureBackendDir, (source) => source.replace(
      'rest.WithPrefix("/api/v1/admin")',
      'rest.WithPrefix("/api/v2/admin")',
    ))

    assertParityFailure(runGenerator(fixtureBackendDir, userGoctlHome, '--check'), [
      /POST \/api\/v1\/admin\/auth\/login -> AdminLogin/,
      /POST \/api\/v2\/admin\/auth\/login -> AdminLogin/,
    ])
  })
})

test('check reports a tampered generated method with exact missing and extra fingerprints', () => {
  withGeneratedFixture(({ fixtureBackendDir, userGoctlHome }) => {
    mutateGeneratedRoutes(fixtureBackendDir, (source) => source.replace(
      'Method:  http.MethodPost,\n\t\t\t\tPath:    "/auth/login",',
      'Method:  http.MethodGet,\n\t\t\t\tPath:    "/auth/login",',
    ))

    assertParityFailure(runGenerator(fixtureBackendDir, userGoctlHome, '--check'), [
      /POST \/api\/v1\/admin\/auth\/login -> AdminLogin/,
      /GET \/api\/v1\/admin\/auth\/login -> AdminLogin/,
    ])
  })
})

test('check reports a tampered generated handler with its exact route fingerprint', () => {
  withGeneratedFixture(({ fixtureBackendDir, userGoctlHome }) => {
    mutateGeneratedRoutes(fixtureBackendDir, (source) => source.replace(
      'adminauth.AdminLoginHandler(serverCtx)',
      'adminauth.AdminLoginHandlerRenamed(serverCtx)',
    ))

    assertParityFailure(runGenerator(fixtureBackendDir, userGoctlHome, '--check'), [
      /POST \/api\/v1\/admin\/auth\/login -> AdminLogin/,
      /POST \/api\/v1\/admin\/auth\/login -> AdminLoginHandlerRenamed/,
    ])
  })
})

test('check reports a duplicated generated route with its exact fingerprint', () => {
  withGeneratedFixture(({ fixtureBackendDir, userGoctlHome }) => {
    mutateGeneratedRoutes(fixtureBackendDir, (source) => source.replace(
      `\t\t\t{\n\t\t\t\tMethod:  http.MethodPost,\n\t\t\t\tPath:    "/auth/login",\n\t\t\t\tHandler: adminauth.AdminLoginHandler(serverCtx),\n\t\t\t},`,
      `\t\t\t{\n\t\t\t\tMethod:  http.MethodPost,\n\t\t\t\tPath:    "/auth/login",\n\t\t\t\tHandler: adminauth.AdminLoginHandler(serverCtx),\n\t\t\t},\n\t\t\t{\n\t\t\t\tMethod:  http.MethodPost,\n\t\t\t\tPath:    "/auth/login",\n\t\t\t\tHandler: adminauth.AdminLoginHandler(serverCtx),\n\t\t\t},`,
    ))

    assertParityFailure(runGenerator(fixtureBackendDir, userGoctlHome, '--check'), [
      /POST \/api\/v1\/admin\/auth\/login -> AdminLogin/,
    ])
  })
})

test('check reports an indirect real NotMigrated reference with source fingerprint', () => {
  withGeneratedFixture(({ fixtureBackendDir, userGoctlHome }) => {
    writeFixtureFile(
      fixtureBackendDir,
      'app/internal/handler/city/not_migrated_probe.go',
      `package city

import migrated "wplink/backend/app/internal/handler/handlerx"

func routeGateProbe() http.HandlerFunc {
  stub := migrated.NotMigrated
  return stub("RouteGateProbe")
}
`,
    )

    const checkResult = runGenerator(fixtureBackendDir, userGoctlHome, '--check')
    const output = `${checkResult.stdout}\n${checkResult.stderr}`
    assert.notEqual(checkResult.status, 0, output)
    assert.match(output, /city\/not_migrated_probe\.go:6:\d+/)
    assert.match(output, /handlerx\.NotMigrated/)
  })
})

test('check reports a tampered generated route file as stale', () => {
  const { fixtureDir, fixtureBackendDir } = createBackendFixture()
  try {
    const writeResult = runGenerator(fixtureBackendDir, path.join(fixtureDir, 'unused-user-home'), '--write')
    assert.equal(writeResult.status, 0, `${writeResult.stdout}\n${writeResult.stderr}`)

    const routesPath = path.join(fixtureBackendDir, 'app/internal/handler/routes.go')
    fs.appendFileSync(routesPath, '\n// stale fixture\n')

    const checkResult = runGenerator(fixtureBackendDir, path.join(fixtureDir, 'unused-user-home'), '--check')

    assert.notEqual(checkResult.status, 0)
    assert.match(`${checkResult.stdout}\n${checkResult.stderr}`, /routes\.go/)
    assert.match(`${checkResult.stdout}\n${checkResult.stderr}`, /过期/)
  } finally {
    fs.rmSync(fixtureDir, { recursive: true, force: true })
  }
})

for (const fixture of [
  {
    name: 'same-named handler in another package',
    relativePath: 'app/internal/handler/decoy/get_merchant_handler.go',
    source: `package decoy

func GetMerchantHandler() {}
`,
  },
  {
    name: 'same-named handler only in a test file',
    relativePath: 'app/internal/handler/merchant/get_merchant_handler_test.go',
    source: `package merchant

func GetMerchantHandler() {}
`,
  },
  {
    name: 'same-named handler text in a comment',
    relativePath: 'app/internal/handler/merchant/decoy_comment.go',
    source: `package merchant

// func GetMerchantHandler() {}
`,
  },
  {
    name: 'same-named handler text in a string',
    relativePath: 'app/internal/handler/merchant/decoy_string.go',
    source: `package merchant

const decoy = "func GetMerchantHandler()"
`,
  },
]) {
  test(`check reports a missing target-package handler despite ${fixture.name}`, () => {
    withGeneratedFixture(({ fixtureBackendDir, userGoctlHome }) => {
      fs.rmSync(path.join(fixtureBackendDir, 'app/internal/handler/merchant/get_merchant_handler.go'))
      writeFixtureFile(fixtureBackendDir, fixture.relativePath, fixture.source)

      assertMissingMerchantHandler(runGenerator(fixtureBackendDir, userGoctlHome, '--check'))
    })
  })
}

test('check rejects duplicate top-level handlers in the target package', () => {
  withGeneratedFixture(({ fixtureBackendDir, userGoctlHome }) => {
    writeFixtureFile(
      fixtureBackendDir,
      'app/internal/handler/merchant/get_merchant_handler_duplicate.go',
      `package merchant

func GetMerchantHandler() {}
`,
    )

    const checkResult = runGenerator(fixtureBackendDir, userGoctlHome, '--check')
    const output = `${checkResult.stdout}\n${checkResult.stderr}`
    assert.notEqual(checkResult.status, 0, output)
    assert.match(output, /重复/)
    assert.match(output, /GetMerchantHandler/)
  })
})
