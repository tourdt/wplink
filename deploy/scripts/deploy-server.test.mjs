import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

const scriptDir = path.dirname(fileURLToPath(import.meta.url))
const repoRoot = path.resolve(scriptDir, '../..')
const deployScriptPath = path.join(scriptDir, 'deploy-server.sh')
const migrationsDir = path.join(repoRoot, 'backend/migrations')

test('deploy script covers local build, remote install, migrations, restart, and health checks', () => {
  const script = fs.readFileSync(deployScriptPath, 'utf8')

  assert.match(script, /WPLINK_DEPLOY_TARGET/)
  assert.match(script, /WPLINK_SSH_KEY/)
  assert.match(script, /ssh_cmd=\(ssh/)
  assert.match(script, /scp_cmd=\(scp/)
  assert.match(script, /-i "\$WPLINK_SSH_KEY"/)
  assert.match(script, /build-release\.sh/)
  assert.match(script, /scp\b/)
  assert.match(script, /ssh\b/)
  assert.match(script, /systemctl restart "\$REMOTE_SERVICE"/)
  assert.match(script, /\/healthz/)
  assert.match(script, /\/readyz/)
})

test('deploy script migrates every current up migration in sorted order', () => {
  const script = fs.readFileSync(deployScriptPath, 'utf8')
  const migrationFiles = fs
    .readdirSync(migrationsDir)
    .filter((fileName) => fileName.endsWith('.up.sql'))
    .sort()

  assert(migrationFiles.length > 0)
  for (const migrationFile of migrationFiles) {
    assert(
      script.includes(migrationFile),
      `deploy script should include ${migrationFile}`,
    )
  }
})
