import unittest
from pathlib import Path

from app.storage import paths


class StoragePathsTests(unittest.TestCase):
    def test_model_cache_path(self) -> None:
        cache_dir = Path("/tmp/autoinspect-cache")
        path = paths.model_cache_path(cache_dir, "models/general/parts_segmentation.pt")
        self.assertEqual(path.name, "models_general_parts_segmentation.pt")
        self.assertEqual(path.parent, cache_dir / "models")

    def test_job_file_path(self) -> None:
        cache_dir = Path("/tmp/autoinspect-cache")
        path = paths.job_file_path(cache_dir, "corr-1", "uploads/image_1.jpg")
        self.assertEqual(path.name, "image_1.jpg")
        self.assertEqual(path.parent, cache_dir / "jobs" / "corr-1")


if __name__ == "__main__":
    unittest.main()

