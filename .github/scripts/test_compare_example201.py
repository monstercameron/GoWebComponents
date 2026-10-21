import importlib.util
import unittest
from pathlib import Path


SPEC = importlib.util.spec_from_file_location("compare_example201", Path(__file__).with_name("compare-example201.py"))
MODULE = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
SPEC.loader.exec_module(MODULE)


class CompareExample201Test(unittest.TestCase):
    def test_capture_median_ignores_single_browser_scheduling_spike(self):
        result = {
            "primitive-render": {
                "getDomReadySamplesMs": [1.7, 1.8, 1.9, 2.0, 2.1, 2.2, 23.5]
            }
        }
        self.assertEqual(MODULE.capture_medians(result), {"primitive-render": 2.0})

    def test_geometric_mean_is_available_for_median_rows(self):
        self.assertAlmostEqual(MODULE.geometric_mean([1.0, 4.0]), 2.0)


if __name__ == "__main__":
    unittest.main()
