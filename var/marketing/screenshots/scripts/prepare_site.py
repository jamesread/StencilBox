"""Waits for a built static site page to render before capture."""
import time

from selenium.webdriver.common.by import By


def run(driver):
    deadline = time.time() + 15
    while time.time() < deadline:
        body = driver.find_elements(By.CSS_SELECTOR, "body")
        if body and body[0].text.strip():
            break
        time.sleep(0.25)
    driver.execute_script("window.scrollTo(0, 0);")
