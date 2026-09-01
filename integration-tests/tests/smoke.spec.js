import assert from 'node:assert'
import { Builder, By, until } from 'selenium-webdriver'
import chrome from 'selenium-webdriver/chrome.js'

const base = process.env.STENCILBOX_BASE_URL || 'http://localhost:18080'

describe('StencilBox smoke', function () {
  this.timeout(30000)
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

  it('serves the web UI with StencilBox title', async function () {
    await driver.get(base + '/webui/')
    await driver.wait(until.elementLocated(By.css('body')), 10000)
    const title = await driver.getTitle()
    assert.strictEqual(title, 'StencilBox')
  })

  it('serves SPA deep links to build configs', async function () {
    await driver.get(base + '/webui/build-config/homepage')
    await driver.wait(until.elementLocated(By.css('body')), 10000)
    const title = await driver.getTitle()
    assert.strictEqual(title, 'StencilBox')
  })

  it('renders the SPA shell', async function () {
    await driver.get(base + '/webui/')
    await driver.wait(until.elementLocated(By.css('#app, [id="layout"], main')), 10000)
    const body = await driver.findElement(By.css('body')).getText()
    assert.match(body, /StencilBox|Welcome|Build/)
  })
})
