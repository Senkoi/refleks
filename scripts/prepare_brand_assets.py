"""Rebuild local brand sizes and Windows ICO from the approved PNG masters."""
from pathlib import Path

from PIL import Image

ROOT = Path(__file__).resolve().parents[1]
ASSETS = ROOT / "frontend/src/assets/brand"


def main():
    mascot = Image.open(ASSETS / "aimmeow-mascot-master.png").convert("RGBA")
    mascot.resize((512, 512), Image.Resampling.LANCZOS).save(
        ASSETS / "aimmeow-mascot-512.png", optimize=True
    )
    icon = Image.open(ASSETS / "aimmeow-icon-master.png").convert("RGBA")
    for size in (32, 128, 256, 512):
        icon.resize((size, size), Image.Resampling.LANCZOS).save(
            ASSETS / f"aimmeow-icon-{size}.png", optimize=True
        )
    icon.resize((1024, 1024), Image.Resampling.LANCZOS).save(
        ROOT / "build/appicon.png", optimize=True
    )
    icon.resize((256, 256), Image.Resampling.LANCZOS).save(
        ROOT / "build/windows/icon.ico",
        format="ICO",
        sizes=[(size, size) for size in (16, 24, 32, 48, 64, 128, 256)],
    )


if __name__ == "__main__":
    main()
