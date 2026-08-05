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
  copyIfExists(path.join(backendDir, 'app/internal/handler/adminauth'), path.join(fixtureBackendDir, 'app/internal/handler/adminauth'))
  copyIfExists(path.join(backendDir, 'app/internal/handler/city'), path.join(fixtureBackendDir, 'app/internal/handler/city'))
  copyIfExists(path.join(backendDir, 'app/internal/types/types.go'), path.join(fixtureBackendDir, 'app/internal/types/types.go'))
  copyIfExists(path.join(backendDir, 'scripts/api_route_inventory.mjs'), path.join(fixtureBackendDir, 'scripts/api_route_inventory.mjs'))
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
    assert.ok(handlerSources.some((source) => source.includes('handlerx.NotMigrated(')))
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
