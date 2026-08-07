// Starts the StencilBox service with -configdir and runs mocha.

import { spawn } from 'node:child_process'
import { existsSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import waitOn from 'wait-on'

const __dirname = dirname(fileURLToPath(import.meta.url))
const repoRoot = join(__dirname, '..')
const testsDir = join(__dirname, 'tests')
const serviceBin = join(repoRoot, 'service', 'StencilBox')
const frontendDist = join(repoRoot, 'frontend', 'dist')
const testPort = process.env.STENCILBOX_TEST_PORT || process.env.PORT || '18080'
const baseUrl = process.env.STENCILBOX_BASE_URL || `http://localhost:${testPort}`

async function main() {
  if (!existsSync(serviceBin)) {
    throw new Error(`StencilBox binary not found at ${serviceBin}. Run "make service" from the repository root first.`)
  }

  if (!existsSync(join(frontendDist, 'index.html'))) {
    throw new Error(`Frontend build not found at ${frontendDist}. Run "make frontend" from the repository root first.`)
  }

  const env = {
    ...process.env,
    PORT: testPort,
  }

  const child = spawn(serviceBin, ['-configdir', testsDir], {
    cwd: join(repoRoot, 'service'),
    env,
    stdio: ['ignore', 'inherit', 'inherit'],
  })

  try {
    await waitOn({
      resources: [`http-get://${new URL(baseUrl).host}/webui/`],
      timeout: 30000,
      validateStatus: (status) => status >= 200 && status < 500,
    })

    const mocha = spawn(
      'npx',
      ['mocha', '--timeout', '30000', 'tests/**/*.spec.js'],
      {
        cwd: __dirname,
        env: {
          ...env,
          STENCILBOX_BASE_URL: baseUrl,
        },
        stdio: 'inherit',
      },
    )

    const code = await new Promise((resolve) => mocha.on('close', resolve))
    child.kill('SIGTERM')
    process.exit(code ?? 1)
  } catch (err) {
    child.kill('SIGTERM')
    throw err
  }
}

main().catch((err) => {
  console.error(err)
  process.exit(1)
})
