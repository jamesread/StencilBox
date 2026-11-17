"""Expands the iframe-sidebar strip so icon labels are visible."""
import time

from selenium.webdriver.common.by import By


def run(driver):
    deadline = time.time() + 15
    while time.time() < deadline:
        if driver.find_elements(By.CSS_SELECTOR, "#sidebar"):
            break
        time.sleep(0.25)
    driver.execute_script(
        """
        const sidebar = document.getElementById('sidebar');
        if (sidebar) {
          sidebar.style.width = '10em';
          sidebar.style.overflow = 'visible';
        }
        """
    )
    time.sleep(0.5)
    driver.execute_script("window.scrollTo(0, 0);")
