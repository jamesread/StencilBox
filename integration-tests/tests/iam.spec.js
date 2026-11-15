import assert from 'node:assert'
import { Builder, By, until } from 'selenium-webdriver'
import chrome from 'selenium-webdriver/chrome.js'

const base = process.env.STENCILBOX_BASE_URL || 'http://localhost:18080'

// run-tests.js gives each run a fresh database, so the default admin account exists.
const adminUser = 'admin'
const adminPassword = 'admin'

async function textOf(driver) {
  return driver.findElement(By.css('body')).getText()
}

async function waitForText(driver, pattern) {
  await driver.wait(async () => pattern.test(await textOf(driver)), 10000, `waiting for ${pattern}`)
}

async function login(driver, username, password) {
  await driver.get(base + '/webui/')
  const user = await driver.wait(until.elementLocated(By.css('#username')), 10000)
  await user.clear()
  await user.sendKeys(username)
  const pass = await driver.findElement(By.css('#password'))
  await pass.clear()
  await pass.sendKeys(password)
  await driver.findElement(By.css('.local-login-form button[type="submit"]')).click()
}

describe('StencilBox IAM', function () {
  this.timeout(60000)
  let driver

  before(async function () {
    const options = new chrome.Options()
    options.addArguments('--headless=new', '--no-sandbox', '--disable-dev-shm-usage')
    driver = await new Builder().forBrowser('chrome').setChromeOptions(options).build()
  })

  after(async function () {
    if (driver) {
      await driver.quit()
    }
  })

  it('shows the login form to guests', async function () {
    await driver.manage().deleteAllCookies()
    await driver.get(base + '/webui/build-configs')
    await driver.wait(until.elementLocated(By.css('#username')), 10000)
    assert.doesNotMatch(await textOf(driver), /My links homepage/, 'build configs are hidden from guests')
  })

  it('rejects a wrong password with a generic error', async function () {
    await driver.manage().deleteAllCookies()
    await login(driver, adminUser, 'not-the-password')
    await waitForText(driver, /Invalid username or password/)
  })

  it('logs in and reaches the IAM pages', async function () {
    await driver.manage().deleteAllCookies()
    await login(driver, adminUser, adminPassword)
    await waitForText(driver, /default password/)

    for (const [path, expected] of [
      ['/webui/control-panel', /Identity & Access/],
      ['/webui/control-panel/iam', /Permission Catalog/],
      ['/webui/control-panel/iam/users', /admin/],
      ['/webui/control-panel/iam/groups', /Administrators/],
      ['/webui/control-panel/iam/roles', /superuser/],
      ['/webui/control-panel/iam/permissions', /usergroups\.manage/],
      ['/webui/account/permissions', /all permissions are granted|every permission is granted/],
      ['/webui/account/api-keys', /API Keys/],
    ]) {
      await driver.get(base + path)
      await waitForText(driver, expected)
    }
  })

  it('creates an API key limited to one build config', async function () {
    await driver.get(base + '/webui/account/api-keys')
    await waitForText(driver, /API Keys/)
    await driver.findElement(By.css('button[aria-label="Create API key"]')).click()

    const name = await driver.wait(until.elementLocated(By.css('#new-key-name')), 10000)
    await driver.wait(until.elementIsVisible(name), 10000)
    await name.sendKeys('homepage builder')

    const radio = async (group, value) => {
      const label = await driver.findElement(By.xpath(`//input[@type="radio"][@name="${group}"]/following-sibling::span[contains(., "${value}")]/..`))
      await label.click()
    }
    const check = async (text) => {
      await driver.findElement(By.xpath(`//label[contains(., "${text}")]`)).click()
    }

    await radio('api-key-permission-mode', 'Only the permissions I choose')
    await check('builds.run:')
    await radio('api-key-build-config-mode', 'Only the build configs I choose')
    await check('My links homepage')

    await driver.findElement(By.css('button[form="api-key-form"]')).click()
    await waitForText(driver, /It will not be shown again/)
    const key = await driver.findElement(By.css('code.api-key')).getText()
    assert.match(key, /^sbk_/)

    // Use the key directly against the API.
    const call = (method, body) => fetch(`${base}/api/StencilBox.clientapi.v1.StencilBoxApiService/${method}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${key}` },
      body: JSON.stringify(body),
    })
    assert.strictEqual((await call('GetBuildConfigs', {})).status, 403, 'no buildconfigs.view')
    const other = await call('ClearBuildCache', { configName: 'My sidebar of apps' })
    assert.strictEqual(other.status, 403, 'another build config')
    const own = await call('ClearBuildCache', { configName: 'My links homepage' })
    assert.strictEqual(own.status, 200)
  })
})
