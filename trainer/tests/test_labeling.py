import pandas as pd

from fusionguard_trainer.labeling import label_windows


def test_label_windows():
    df = pd.DataFrame({"time_to_disruption": [10, 60, 210]})
    labeled = label_windows(df, [50, 200])
    assert labeled.loc[0, "label_h50"] == 1
    assert labeled.loc[2, "label_h50"] == 0
    assert labeled.loc[2, "label_h200"] == 0
