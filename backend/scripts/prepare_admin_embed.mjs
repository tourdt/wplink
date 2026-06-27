import fs from 'node:fs'
import path from 'node:path'
import { spawnSync } from 'node:child_process'

const repoRoot = path.resolve(new URL('../..', import.meta.url).pathname)
const adminRoot = path.join(repoRoot, 'admin-web')
const adminDist = path.join(adminRoot, 'dist')
const embedDist = path.join(repoRoot, 'backend/app/internal/adminweb/dist')

function run(command, args, options = {}) {
  const result = spawnSync(command, args, {
    cwd: options.cwd || repoRoot,
    stdio: 'inherit',
    env: { ...process.env, ...(options.env || {}) },
  })
  if (result.status !== 0) {
    throw new Error(`${command} ${args.join(' ')} 执行失败`)
  }
}

run('npm', ['run', 'build'], {
  cwd: adminRoot,
  env: { VITE_ADMIN_BASE: '/admin/' },
})

fs.rmSync(embedDist, { recursive: true, force: true })
fs.mkdirSync(path.dirname(embedDist), { recursive: true })
fs.cpSync(adminDist, embedDist, { recursive: true })

console.log(`admin web dist copied to ${embedDist}`)
