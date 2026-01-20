from pathlib import Path

from fusionguard_trainer.export_onnx import export_model


def test_export_model(tmp_path: Path):
    out = tmp_path / "model.onnx"
    export_model(out, {"model_version": "dev"})
    assert out.exists()
