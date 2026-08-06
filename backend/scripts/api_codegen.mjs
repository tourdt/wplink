import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

import {
  checkContractGeneratedParity,
  checkNoDuplicateRoutes,
  checkNoMigrationStubs,
  parseAPIContracts,
  parseGeneratedRoutes,
  routeFingerprint,
} from './api_route_inventory.mjs'

const requiredGoctlVersion = '1.7.5'
const scriptDir = path.dirname(fileURLToPath(import.meta.url))
const backendDir = path.resolve(scriptDir, '..')
const apiDir = path.join(backendDir, 'app/api')
const templateHome = path.join(backendDir, 'app/goctl')
const handlerDir = path.join(backendDir, 'app/internal/handler')
const goKeywords = new Set([
  'break', 'default', 'func', 'interface', 'select',
  'case', 'defer', 'go', 'map', 'struct',
  'chan', 'else', 'goto', 'package', 'switch',
  'const', 'fallthrough', 'if', 'range', 'type',
  'continue', 'for', 'import', 'return', 'var',
])

function runCommand(command, args, options = {}) {
  const result = spawnSync(command, args, {
    cwd: options.cwd || backendDir,
    env: options.env || process.env,
    encoding: 'utf8',
  })
  if (result.error) {
    throw new Error(`无法执行 ${command}: ${result.error.message}`)
  }
  if (result.status !== 0) {
    const detail = [result.stdout, result.stderr].filter(Boolean).join('\n').trim()
    throw new Error(`${command} 执行失败${detail ? `:\n${detail}` : ''}`)
  }
  return `${result.stdout || ''}${result.stderr || ''}`.trim()
}

function verifyGoctlVersion() {
  const output = runCommand('goctl', ['--version'])
  const actualVersion = output.match(/\bgoctl version ([^\s]+)/)?.[1]
  if (actualVersion !== requiredGoctlVersion) {
    throw new Error(`goctl 版本不匹配，需要 ${requiredGoctlVersion}，当前为 ${actualVersion || '未知版本'}`)
  }
}

function listGoFiles(directory) {
  if (!fs.existsSync(directory)) {
    return []
  }
  const files = []
  for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
    const entryPath = path.join(directory, entry.name)
    if (entry.isDirectory()) {
      files.push(...listGoFiles(entryPath))
    } else if (entry.name.endsWith('.go')) {
      files.push(entryPath)
    }
  }
  return files.sort()
}

function maskGoCommentsAndLiterals(source) {
  // split('') 与后续基于 UTF-16 下标的 source[index] 保持一致，避免非 BMP 字符导致掩码错位。
  const result = source.split('')
  let state = 'code'
  let escaped = false

  for (let index = 0; index < source.length; index += 1) {
    const current = source[index]
    const next = source[index + 1]
    if (state === 'code') {
      if (current === '/' && next === '/') {
        result[index] = ' '
        result[index + 1] = ' '
        index += 1
        state = 'line-comment'
      } else if (current === '/' && next === '*') {
        result[index] = ' '
        result[index + 1] = ' '
        index += 1
        state = 'block-comment'
      } else if (current === '"') {
        result[index] = ' '
        state = 'string'
        escaped = false
      } else if (current === '\'') {
        result[index] = ' '
        state = 'rune'
        escaped = false
      } else if (current === '`') {
        result[index] = ' '
        state = 'raw-string'
      }
      continue
    }

    if (current !== '\n' && current !== '\r') {
      result[index] = ' '
    }
    if (state === 'line-comment' && (current === '\n' || current === '\r')) {
      state = 'code'
    } else if (state === 'block-comment' && current === '*' && next === '/') {
      result[index + 1] = ' '
      index += 1
      state = 'code'
    } else if ((state === 'string' || state === 'rune') && current === '\\' && !escaped) {
      escaped = true
    } else if (state === 'string' && current === '"' && !escaped) {
      state = 'code'
    } else if (state === 'rune' && current === '\'' && !escaped) {
      state = 'code'
    } else if (state === 'raw-string' && current === '`') {
      state = 'code'
    } else {
      escaped = false
    }
  }
  return result.join('')
}

function topLevelFunctionNames(source) {
  const masked = maskGoCommentsAndLiterals(source)
  const names = []
  let braceDepth = 0

  for (let index = 0; index < masked.length;) {
    const current = masked[index]
    if (current === '{') {
      braceDepth += 1
      index += 1
      continue
    }
    if (current === '}') {
      braceDepth = Math.max(0, braceDepth - 1)
      index += 1
      continue
    }
    if (braceDepth !== 0 || !masked.startsWith('func', index)) {
      index += 1
      continue
    }
    const before = masked[index - 1] || ''
    const after = masked[index + 4] || ''
    if (/[A-Za-z0-9_]/.test(before) || /[A-Za-z0-9_]/.test(after)) {
      index += 4
      continue
    }

    let cursor = index + 4
    while (/\s/.test(masked[cursor] || '')) {
      cursor += 1
    }
    // receiver 紧跟在 func 后，出现左括号即为方法而非顶层函数声明。
    if (masked[cursor] === '(') {
      index = cursor + 1
      continue
    }
    const nameMatch = masked.slice(cursor).match(/^([A-Za-z_][A-Za-z0-9_]*)\s*\(/)
    if (nameMatch) {
      names.push(nameMatch[1])
      index = cursor + nameMatch[0].length
      continue
    }
    index += 4
  }
  return names
}

function handlerFunctionName(source) {
  return topLevelFunctionNames(source).find((name) => name.endsWith('Handler')) || ''
}

function handlerKey(packagePath, functionName) {
  return `${packagePath.split(path.sep).join('/')}\u0000${functionName}`
}

function existingHandlerFunctions() {
  const definitions = new Map()
  for (const filePath of listGoFiles(handlerDir)) {
    if (filePath.endsWith('_test.go')) {
      continue
    }
    const packagePath = path.relative(handlerDir, path.dirname(filePath))
    const source = fs.readFileSync(filePath, 'utf8')
    for (const functionName of topLevelFunctionNames(source).filter((name) => name.endsWith('Handler'))) {
      const key = handlerKey(packagePath, functionName)
      const files = definitions.get(key) || []
      files.push(filePath)
      definitions.set(key, files)
    }
  }

  for (const [key, files] of definitions) {
    if (files.length < 2) {
      continue
    }
    const [packagePath, functionName] = key.split('\u0000')
    throw new Error(`Handler 重复定义: ${packagePath}.${functionName}\n- ${files.map((filePath) => path.relative(backendDir, filePath)).join('\n- ')}`)
  }
  return new Set(definitions.keys())
}

function goString(value) {
  return JSON.stringify(String(value))
}

function buildRoutesManifest() {
  const routes = parseAPIContracts(apiDir).sort((left, right) => {
    const leftKey = `${routeFingerprint(left)}\u0000${left.handler}\u0000${left.group}`
    const rightKey = `${routeFingerprint(right)}\u0000${right.handler}\u0000${right.group}`
    return leftKey < rightKey ? -1 : leftKey > rightKey ? 1 : 0
  })
  const entries = routes.map((route) => `\t\t{Method: ${goString(route.method)}, Path: ${goString(route.path)}, Handler: ${goString(route.handler)}, Group: ${goString(route.group)}},`).join('\n')

  return `// Code generated by scripts/api_codegen.mjs. DO NOT EDIT.

package handler

// ContractRoute 描述 app/api 中声明的单条 HTTP 路由，供运行时门禁核对注册结果。
type ContractRoute struct {
\tMethod  string
\tPath    string
\tHandler string
\tGroup   string
}

// ContractRoutes 返回按 method、path、handler、group 稳定排序的契约路由副本。
func ContractRoutes() []ContractRoute {
\treturn []ContractRoute{
${entries}
\t}
}
`
}

function formatGoFiles(files, cwd) {
  if (files.length === 0) {
    return
  }
  runCommand('gofmt', ['-w', ...files], { cwd })
}

function normalizeKeywordHandlerPackages(generatedHandlerDir) {
  const routesPath = path.join(generatedHandlerDir, 'routes.go')
  let routesSource = fs.readFileSync(routesPath, 'utf8')
  const entries = fs.readdirSync(generatedHandlerDir, { withFileTypes: true })
    .filter((entry) => entry.isDirectory())
    .sort((left, right) => left.name.localeCompare(right.name))
  // 非关键字 group 会直接成为 Go package 标识符，先预占它们，避免 map -> maphandler
  // 与真实 group: maphandler 冲突；后缀从 2 起递增，保证无碰撞场景的既有输出不变。
  const occupiedPackageNames = new Set(
    entries.filter((entry) => !goKeywords.has(entry.name)).map((entry) => entry.name),
  )

  for (const entry of entries) {
    if (!goKeywords.has(entry.name)) {
      continue
    }
    const basePackageName = `${entry.name}handler`
    let safePackageName = basePackageName
    for (let suffix = 2; occupiedPackageNames.has(safePackageName); suffix += 1) {
      safePackageName = `${basePackageName}${suffix}`
    }
    occupiedPackageNames.add(safePackageName)
    for (const filePath of listGoFiles(path.join(generatedHandlerDir, entry.name))) {
      const source = fs.readFileSync(filePath, 'utf8')
      fs.writeFileSync(filePath, source.replace(
        new RegExp(`^package\\s+${entry.name}\\b`, 'm'),
        `package ${basePackageName}`,
      ))
    }
    routesSource = routesSource
      .replace(
        new RegExp(`^\\s*${entry.name}\\s+("[^"]+/handler/${entry.name}")`, 'm'),
        `\t${safePackageName} $1`,
      )
      .replace(new RegExp(`\\b${entry.name}\\.`, 'g'), `${safePackageName}.`)
  }
  fs.writeFileSync(routesPath, routesSource)
}

function generateIntoTemporaryModule() {
  const temporaryDir = fs.mkdtempSync(path.join(os.tmpdir(), 'wplink-api-codegen-'))
  try {
    fs.writeFileSync(path.join(temporaryDir, 'go.mod'), 'module wplink/backend\n\ngo 1.23.0\n')
    fs.mkdirSync(path.join(temporaryDir, 'app'), { recursive: true })
    fs.cpSync(apiDir, path.join(temporaryDir, 'app/api'), { recursive: true })

    // --home 与隔离后的 GOCTL_HOME 双重限定模板来源，避免本机用户模板影响生成结果。
    const isolatedEnv = {
      ...process.env,
      GOCTL_HOME: templateHome,
    }
    runCommand('goctl', ['api', 'validate', '--api', 'app/api/app.api'], {
      cwd: temporaryDir,
      env: isolatedEnv,
    })
    runCommand('goctl', [
      'api', 'go',
      '--api', 'app/api/app.api',
      '--dir', 'app',
      '--home', templateHome,
      '--style', 'go_zero',
    ], {
      cwd: temporaryDir,
      env: isolatedEnv,
    })

    const generatedHandlerDir = path.join(temporaryDir, 'app/internal/handler')
    const manifestPath = path.join(generatedHandlerDir, 'routes_manifest_gen.go')
    fs.writeFileSync(manifestPath, buildRoutesManifest())
    // goctl 1.7.5 会直接把 group 当作包名；map 等关键字需在同步前统一转成合法别名。
    normalizeKeywordHandlerPackages(generatedHandlerDir)
    const generatedGoFiles = [
      path.join(temporaryDir, 'app/internal/types/types.go'),
      ...listGoFiles(generatedHandlerDir),
    ]
    formatGoFiles(generatedGoFiles, temporaryDir)

    for (const filePath of generatedGoFiles) {
      if (fs.readFileSync(filePath, 'utf8').includes('zmall/')) {
        throw new Error(`仓库模板生成了禁止的 zmall import: ${path.relative(temporaryDir, filePath)}`)
      }
    }
    return temporaryDir
  } catch (error) {
    fs.rmSync(temporaryDir, { recursive: true, force: true })
    throw error
  }
}

function synchronizeFile(sourcePath, destinationPath, mode, staleFiles) {
  const expected = fs.readFileSync(sourcePath)
  const current = fs.existsSync(destinationPath) ? fs.readFileSync(destinationPath) : null
  if (current && current.equals(expected)) {
    return
  }
  if (mode === '--check') {
    staleFiles.push(path.relative(backendDir, destinationPath))
    return
  }
  fs.mkdirSync(path.dirname(destinationPath), { recursive: true })
  fs.writeFileSync(destinationPath, expected)
}

function synchronizeMissingHandlers(generatedHandlerDir, mode, staleFiles) {
  const existingFunctions = existingHandlerFunctions()
  for (const generatedPath of listGoFiles(generatedHandlerDir)) {
    if (path.dirname(generatedPath) === generatedHandlerDir) {
      continue
    }
    const source = fs.readFileSync(generatedPath, 'utf8')
    const functionName = handlerFunctionName(source)
    const relativePath = path.relative(generatedHandlerDir, generatedPath)
    const packagePath = path.dirname(relativePath)
    const key = handlerKey(packagePath, functionName)
    if (!functionName || existingFunctions.has(key)) {
      continue
    }

    const destinationPath = path.join(handlerDir, relativePath)
    if (mode === '--check') {
      staleFiles.push(`${path.relative(backendDir, destinationPath)}（缺少 ${functionName}）`)
      continue
    }
    if (fs.existsSync(destinationPath)) {
      throw new Error(`无法生成 ${functionName}：目标文件已存在 ${path.relative(backendDir, destinationPath)}`)
    }
    fs.mkdirSync(path.dirname(destinationPath), { recursive: true })
    fs.writeFileSync(destinationPath, source)
    existingFunctions.add(key)
  }
}

function run(mode) {
  if (mode !== '--write' && mode !== '--check') {
    throw new Error('用法: node scripts/api_codegen.mjs --write|--check')
  }
  verifyGoctlVersion()
  const contractRoutes = parseAPIContracts(apiDir)
  // 在调用 goctl 前给出稳定、可定位的重复指纹，避免上游错误格式变化削弱门禁诊断。
  checkNoDuplicateRoutes(contractRoutes, '契约')
  if (mode === '--check') {
    const generatedRoutesPath = path.join(handlerDir, 'routes.go')
    if (!fs.existsSync(generatedRoutesPath)) {
      throw new Error(`生成路由不存在: ${path.relative(backendDir, generatedRoutesPath)}`)
    }
    checkContractGeneratedParity(contractRoutes, parseGeneratedRoutes(generatedRoutesPath))
    checkNoMigrationStubs(handlerDir)
  }
  const temporaryDir = generateIntoTemporaryModule()
  try {
    const generatedAppDir = path.join(temporaryDir, 'app')
    const staleFiles = []
    synchronizeFile(
      path.join(generatedAppDir, 'internal/types/types.go'),
      path.join(backendDir, 'app/internal/types/types.go'),
      mode,
      staleFiles,
    )
    synchronizeFile(
      path.join(generatedAppDir, 'internal/handler/routes.go'),
      path.join(handlerDir, 'routes.go'),
      mode,
      staleFiles,
    )
    synchronizeFile(
      path.join(generatedAppDir, 'internal/handler/routes_manifest_gen.go'),
      path.join(handlerDir, 'routes_manifest_gen.go'),
      mode,
      staleFiles,
    )
    synchronizeMissingHandlers(path.join(generatedAppDir, 'internal/handler'), mode, staleFiles)

    if (staleFiles.length > 0) {
      throw new Error(`API 生成文件已过期，请运行 node scripts/api_codegen.mjs --write:\n- ${staleFiles.join('\n- ')}`)
    }
    console.log(mode === '--write' ? 'API 生成文件已更新' : 'API 生成文件为最新')
  } finally {
    fs.rmSync(temporaryDir, { recursive: true, force: true })
  }
}

try {
  run(process.argv[2])
} catch (error) {
  console.error(`API 代码生成失败: ${error.message}`)
  process.exitCode = 1
}
