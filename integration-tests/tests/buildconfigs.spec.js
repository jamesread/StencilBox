import assert from 'node:assert'
import { Builder, By, Key, until } from 'selenium-webdriver'
import chrome from 'selenium-webdriver/chrome.js'

const base = process.env.STENCILBOX_BASE_URL || 'http://localhost:18080'

async function bodyText(driver) {
  return driver.findElement(By.css('body')).getText()
}

async function waitForText(driver, pattern) {
  await driver.wait(async () => pattern.test(await bodyText(driver)), 10000, `waiting for ${pattern}`)
}

async function visible(driver, css) {
  const el = await driver.wait(until.elementLocated(By.css(css)), 10000)
  await driver.wait(until.elementIsVisible(el), 10000)
  return el
}

// Click after scrolling the element clear of the fixed header.
async function click(driver, el) {
  await driver.executeScript('arguments[0].scrollIntoView({block: "center"})', el)
  await el.click()
}

// Replace a textarea's whole value the way a user would.
async function replaceText(el, text) {
  await el.click()
  await el.sendKeys(Key.chord(Key.CONTROL, 'a'), Key.DELETE)
  await el.sendKeys(text)
}

describe('StencilBox build configs and data files', function () {
  this.timeout(90000)
  let driver
  const name = `UI site ${Date.now()}`

  before(async function () {
    const options = new chrome.Options()
    options.addArguments('--headless=new', '--no-sandbox', '--disable-dev-shm-usage')
    driver = await new Builder().forBrowser('chrome').setChromeOptions(options).build()

    await driver.get(base + '/webui/')
    await (await visible(driver, '#username')).sendKeys('admin')
    await driver.findElement(By.css('#password')).sendKeys('admin')
    await driver.findElement(By.css('.local-login-form button[type="submit"]')).click()
    await waitForText(driver, /Generate super speedy/) // only shown once logged in
  })

  after(async function () {
    if (driver) {
      await driver.quit()
    }
  })

  it('creates a build config, adds a data file, and edits it', async function () {
    await driver.get(base + '/webui/build-configs')
    await waitForText(driver, /Build Configurations/)
    await driver.findElement(By.xpath('//button[normalize-space(.)="Create"]')).click()

    await (await visible(driver, '#new-config-name')).sendKeys(name)
    const template = await visible(driver, '#new-config-template')
    await driver.wait(async () => (await template.findElements(By.css('option[value="links-homepage"]'))).length > 0, 10000)
    await template.findElement(By.css('option[value="links-homepage"]')).click()
    await driver.findElement(By.css('button[form="create-build-config-form"]')).click()

    await waitForText(driver, new RegExp('Build Config: ' + name))
    await click(driver, await driver.findElement(By.xpath('//button[contains(., "Add data file")]')))
    await (await visible(driver, '#new-datafile-name')).sendKeys('links')

    const editor = await visible(driver, '.add-data-file-dialog .yaml-editor textarea')
    await replaceText(editor, 'links: [unclosed')
    await waitForText(driver, /Line 1/)
    const submit = await driver.findElement(By.css('button[form="add-data-file-form"]'))
    assert.strictEqual(await submit.isEnabled(), false, 'invalid YAML cannot be submitted')

    await replaceText(editor, 'links:\n  - title: GitHub\n    url: https://github.com\n')
    await waitForText(driver, /Valid YAML/)
    await submit.click()

    // Lands on the data file page, editable.
    await waitForText(driver, /Data File: links/)
    const content = await visible(driver, '.yaml-editor textarea')
    assert.match(await content.getAttribute('value'), /title: GitHub/)

    await replaceText(content, 'links:\n  - title: StencilBox\n    url: https://github.com/jamesread/StencilBox\n')
    await waitForText(driver, /unsaved changes/i)
    await click(driver, await driver.findElement(By.xpath('//button[normalize-space(.)="Save"]')))
    await waitForText(driver, /Saved\. Rebuilding:/)

    // The saved content is what the API now returns.
    await driver.navigate().refresh()
    const reloaded = await visible(driver, '.yaml-editor textarea')
    await driver.wait(async () => /StencilBox/.test(await reloaded.getAttribute('value')), 10000)
  })
})
