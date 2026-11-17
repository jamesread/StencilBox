"""Waits for the admin SPA to render data before capture."""
import time

from selenium.webdriver.common.by import By


def run(driver):
    deadline = time.time() + 20
    while time.time() < deadline:
        if driver.find_elements(By.CSS_SELECTOR, "main table tbody tr, main section, main form, main article"):
            break
        time.sleep(0.25)
    driver.execute_script("window.scrollTo(0, 0);")
