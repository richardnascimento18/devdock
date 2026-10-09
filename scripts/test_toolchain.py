import unittest
from toolchain import supported


class ToolchainTests(unittest.TestCase):
    def test_security_floor(self):
        for version in ["go1.26.9", "go1.26.10", "go1.27.2", "go1.27.2-X:nodwarf5"]:
            self.assertTrue(supported(version), version)
        for version in ["go1.26.0", "go1.26.8", "go1.27.1", "go1.25.20", "go1.28rc1", "devel"]:
            self.assertFalse(supported(version), version)
