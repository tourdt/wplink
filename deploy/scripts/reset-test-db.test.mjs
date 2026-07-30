import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { spawnSync } from 'node:child_process'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

const scriptDir = path.dirname(fileURLToPath(import.meta.url))
const wrapperPath = path.join(scriptDir, 'reset-test-db.sh')

test('reset test database wrapper uses the configured test server, private key and demo seed', () => {
  const tempRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'wplink-reset-test-db-'))
  const fakeBin = path.join(tempRoot, 'bin')
  const fakeHome = path.join(tempRoot, 'home')
  const sshDir = path.join(fakeHome, '.ssh')
  const logPath = path.join(tempRoot, 'remote-tools.log')
  fs.mkdirSync(fakeBin, { recursive: true })
  fs.mkdirSync(sshDir, { recursive: true })
  fs.writeFileSync(path.join(sshDir, 'ebyby.pem'), 'test-only-private-key-placeholder\n', { mode: 0o600 })

  for (const command of ['scp', 'ssh']) {
    fs.writeFileSync(
      path.join(fakeBin, command),
      `#!/usr/bin/env bash\nprintf '${command}:%s\\n' "$*" >> "$WPLINK_TEST_LOG"\ncat >/dev/null || true\n`,
      { mode: 0o755 },
    )
  }

  const result = spawnSync(wrapperPath, [], {
    cwd: path.resolve(scriptDir, '../..'),
    encoding: 'utf8',
    env: {
      ...process.env,
      HOME: fakeHome,
      PATH: `${fakeBin}:${process.env.PATH}`,
      TMPDIR: tempRoot,
      WPLINK_TEST_LOG: logPath,
      WPLINK_DEPLOY_TARGET: '',
      WPLINK_SSH_KEY: '',
    },
  })

  assert.equal(result.status, 0, result.stderr || result.stdout)
  const remoteToolLog = fs.readFileSync(logPath, 'utf8')
  assert.match(remoteToolLog, new RegExp(`scp:-P 22 -i ${escapeRegExp(path.join(sshDir, 'ebyby.pem'))}`))
  assert.match(remoteToolLog, /root@124\.223\.186\.63:\/tmp\/wplink-test-db-/)
  assert.match(remoteToolLog, new RegExp(`ssh:-p 22 -i ${escapeRegExp(path.join(sshDir, 'ebyby.pem'))}`))
  assert.match(result.stdout, /packaging migrations and optional seed files/)
  assert.doesNotMatch(result.stdout, /skipping demo seed data/)
})

function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}
