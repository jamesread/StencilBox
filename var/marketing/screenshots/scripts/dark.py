"""Logs in and forces the dark theme before capture."""
import os
import sys

sys.path.insert(0, os.path.dirname(__file__))
from session import prepare_session  # noqa: E402


def run(driver):
    prepare_session(driver, "dark")
