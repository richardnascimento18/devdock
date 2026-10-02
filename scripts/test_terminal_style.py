import unittest

from terminal_style import decorative_background


class TransparencyTest(unittest.TestCase):
    def test_background_and_inverse(self):
        for sequence in (b"\x1b[41m", b"\x1b[104m", b"\x1b[48;5;2m", b"\x1b[48;2;1;2;3m", b"\x1b[48:2::1:2:3m", b"\x1b[7m"):
            with self.subTest(sequence=sequence):
                self.assertIsNotNone(decorative_background(sequence))

    def test_default_and_foreground(self):
        for sequence in (b"\x1b[0m", b"\x1b[49m", b"\x1b[38;2;48;100;40m", b"\x1b[38;5;104m", b"\x1b[38:2:0:48:100:40m", b"\x1b[38:2::48:100:40m", b"\x1b[38:5:104;49m"):
            with self.subTest(sequence=sequence):
                self.assertIsNone(decorative_background(sequence))
