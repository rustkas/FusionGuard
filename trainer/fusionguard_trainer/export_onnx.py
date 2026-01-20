from __future__ import annotations

import json
from pathlib import Path


def export_model(path: Path, metadata: dict[str, str]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", encoding="utf-8") as fh:
        json.dump(metadata, fh, indent=2)
