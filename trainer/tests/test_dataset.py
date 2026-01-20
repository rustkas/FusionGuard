import pandas as pd

from fusionguard_trainer.dataset import build_dataset


def test_build_dataset():
    samples = [{"a": 1}, {"a": 2}]
    df = build_dataset(samples)
    assert isinstance(df, pd.DataFrame)
    assert df.shape[0] == 2
