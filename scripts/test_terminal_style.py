import unittest

from terminal_style import decorative_background, malformed_ansi


class TransparencyTest(unittest.TestCase):
    def test_background_and_inverse(self):
        for sequence in (b"\x1b[41m", b"\x1b[104m", b"\x1b[48;5;2m", b"\x1b[48;2;1;2;3m", b"\x1b[48:2::1:2:3m", b"\x1b[7m"):
            with self.subTest(sequence=sequence):
                self.assertIsNotNone(decorative_background(sequence))

    def test_default_and_foreground(self):
        for sequence in (b"\x1b[0m", b"\x1b[49m", b"\x1b[38;2;48;100;40m", b"\x1b[38;5;104m", b"\x1b[38:2:0:48:100:40m", b"\x1b[38:2::48:100:40m", b"\x1b[38:5:104;49m"):
            with self.subTest(sequence=sequence):
                self.assertIsNone(decorative_background(sequence))


class ANSIIntegrityTest(unittest.TestCase):
    def test_valid_controls(self):
        self.assertIsNone(malformed_ansi(b"\x1b[38;2;100;150;200mtext\x1b[0m\x1b[2J\x1b[H\x1b[?25l"))

    def test_leaked_and_incomplete(self):
        for sequence in (b"[38;2;100;150;200m", b"[38;5;81m", b"\x1b[38;2;100;", b"\x1b[38;2;"):
            with self.subTest(sequence=sequence):
                self.assertIsNotNone(malformed_ansi(sequence))

    def test_literal_user_text(self):
        literal = b"example [38;2;100;150;200m"
        self.assertIsNone(malformed_ansi(literal, (literal,)))
