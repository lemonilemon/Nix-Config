"""Eww bar backend.

Intentionally empty: re-exporting the submodules here made every consumer —
including the eww-barctl click path — import the whole daemon. Import submodules
directly (`from eww_bar_backend import collectors`).
"""
