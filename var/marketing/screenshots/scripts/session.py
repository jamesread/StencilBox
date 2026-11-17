"""Logs in as the throwaway admin and pins the picocrank theme.

repo-helper reuses one browser session for every capture, so cookies and
localStorage carry over between sections.
"""
import time

LOGIN_JS = """
const done = arguments[arguments.length - 1];
fetch('/api/StencilBox.clientapi.v1.StencilBoxApiService/LoginWithUsernameAndPassword', {
  method: 'POST',
  credentials: 'same-origin',
  headers: {'Content-Type': 'application/json'},
  body: JSON.stringify({username: 'admin', password: 'marketing-demo'}),
}).then((r) => done(r.ok)).catch(() => done(false));
"""

ME_JS = """
const done = arguments[arguments.length - 1];
fetch('/api/StencilBox.clientapi.v1.StencilBoxApiService/GetCurrentUser', {
  method: 'POST',
  credentials: 'same-origin',
  headers: {'Content-Type': 'application/json'},
  body: '{}',
}).then((r) => r.json()).then((j) => done(!!(j.isAuthenticated || j.username))).catch(() => done(false));
"""


def prepare_session(driver, theme):
    target = driver.execute_script("return performance.getEntriesByType('navigation')[0].name;")
    changed = False
    if not driver.execute_async_script(ME_JS):
        if not driver.execute_async_script(LOGIN_JS):
            raise RuntimeError("marketing login failed")
        changed = True
    if driver.execute_script("return localStorage.getItem('picocrank-theme');") != theme:
        driver.execute_script("localStorage.setItem('picocrank-theme', arguments[0]);", theme)
        changed = True
    if changed:
        driver.get(target)
        time.sleep(3)
