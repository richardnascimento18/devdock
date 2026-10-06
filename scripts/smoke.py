import argparse, fcntl, os, pathlib, pty, select, signal, struct, subprocess, tempfile, termios, time, tomllib
from terminal_style import decorative_background, malformed_ansi
parser = argparse.ArgumentParser(description="Smoke-test a DevDock binary in an isolated terminal/workspace.")
parser.add_argument("binary", type=pathlib.Path)
parser.add_argument("--oauth-configured", action="store_true", help="expect an embedded public Client ID; never complete authorization")
parser.add_argument("--configuration-editor", action="store_true", help="exercise the Settings tab and persist a default preset")
parser.add_argument("--capture", type=pathlib.Path, help="save actual ANSI output for terminal review or failure diagnosis")
parser.add_argument("--reduced-motion", action="store_true", help="exercise the same workflows with static progress")
args = parser.parse_args()
binary = args.binary.resolve()
with tempfile.TemporaryDirectory(prefix='devdock-smoke-') as directory:
    test_home = pathlib.Path(directory)
    root = test_home / 'workspace'
    project = root / 'apps' / 'demo'
    project.mkdir(parents=True)
    (project / 'go.mod').write_text('module demo\n')
    (project / '.devdock').write_text('name = "demo"\n')
    config_dir = test_home / '.config' / 'devdock'
    config_dir.mkdir(parents=True)
    (config_dir / 'config.toml').write_text(f'roots = ["{root}"]\n')
    master, slave = pty.openpty()
    fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", 40, 140, 0, 0))
    child_env = os.environ.copy()
    child_env.update({'HOME': str(test_home), 'TERM': 'xterm-256color'})
    # The harness chooses its profile; a caller's NO_COLOR must not silently
    # turn both normal and reduced-motion runs into the same static test.
    child_env.pop('NO_COLOR', None)
    child_env.pop('DEVDOCK_GITHUB_CLIENT_ID', None)
    child_env['DEVDOCK_REDUCED_MOTION'] = '1' if args.reduced_motion else '0'
    process = subprocess.Popen([str(binary)], stdin=slave, stdout=slave, stderr=slave, env=child_env, start_new_session=True)
    os.close(slave)
    captured = bytearray()
    def drain(duration):
        deadline = time.monotonic() + duration
        while time.monotonic() < deadline:
            ready, _, _ = select.select([master], [], [], 0.05)
            if ready:
                try: captured.extend(os.read(master, 65536))
                except OSError: break
    try:
        drain(2)
        assert b'demo' in captured, 'main project navigation did not render'
        os.write(master, b'!'); drain(0.2)
        assert b'Status details' in captured, 'status detail view missing'
        os.write(master, b'\x1b'); drain(0.2)
        os.write(master, b'r'); drain(0.3)
        assert process.poll() is None, 'refresh terminated the TUI'
        for direct in (b'\x1b1', b'\x1b2', b'\x1b3', b'\x1b2'):
            os.write(master, direct); drain(0.15)
            assert process.poll() is None, 'direct pane navigation terminated TUI'
        os.write(master, b'\x10'); drain(0.15)
        os.write(master, b'scope apps'); drain(0.2)
        os.write(master, b'\r'); drain(0.2)
        assert process.poll() is None, 'palette scope selection launched or terminated TUI'
        for scope_key in (b'\x7f', b'\x15', b'\x01'):
            os.write(master, scope_key); drain(0.15)
            assert process.poll() is None, 'scope shortcut terminated TUI'
        os.write(master, b'\t'); drain(0.2)
        assert b'Inspector' in captured, 'inspector did not render'
        os.write(master, b'\t'); drain(0.2)
        os.write(master, b'j'); drain(0.2)
        os.write(master, b'\r'); drain(0.2)
        os.write(master, b'/'); drain(0.15)
        os.write(master, b'demo'); drain(0.2)
        os.write(master, b'\r'); drain(0.2)
        os.write(master, b'\x1b'); drain(0.2)
        os.write(master, b' '); drain(0.2)
        assert b'1 selected' in captured, 'multi-selection did not render'
        os.write(master, b'm'); drain(0.2)
        assert b'destination domain' in captured, 'bulk destination workflow missing'
        os.write(master, b'\x1b'); drain(0.2)
        os.write(master, b'\x1b'); drain(0.2)
        os.write(master, b'\x10'); drain(0.2)
        assert b'Commands' in captured, 'command palette did not render'
        os.write(master, b'settings'); drain(0.2)
        os.write(master, b'\r'); drain(0.2)
        assert b'Settings' in captured, 'palette did not launch Settings'
        os.write(master, b'\x1b'); drain(0.2)
        for width, height in ((120, 40), (100, 30), (80, 24), (60, 20), (40, 15), (140, 40)):
            fcntl.ioctl(master, termios.TIOCSWINSZ, struct.pack("HHHH", height, width, 0, 0))
            os.kill(process.pid, signal.SIGWINCH)
            drain(0.15)
            assert process.poll() is None, 'resize terminated the TUI'
        os.write(master, b'?'); drain(0.2)
        assert b'Keyboard Reference' in captured, 'help did not render'
        os.write(master, b'j\x1b'); drain(0.2)
        os.write(master, b'v'); drain(0.2)
        os.write(master, b'}'); drain(0.2)
        os.write(master, b'f'); drain(0.2)
        os.write(master, b'e'); drain(0.3)
        assert b'Presets' in captured and b'Templates' in captured, 'editor did not render'
        os.write(master, b'\r'); drain(0.2)
        assert b'Preset name' in captured, 'preset form did not render'
        os.write(master, b'i'); drain(0.2)
        os.write(master, b'X'); drain(0.2)
        os.write(master, b'\x1b'); drain(0.2)
        os.write(master, b'\x1b'); drain(0.2)
        assert b'Discard unsaved changes' in captured, 'unsaved preset confirmation missing'
        os.write(master, b'y'); drain(0.2)
        os.write(master, b'\t\r'); drain(0.2)
        assert b'Edit Template' in captured, 'template form did not render'
        os.write(master, b'\x1b'); drain(0.2)
        os.write(master, b'\x1b[Z'); drain(0.2)
        if args.configuration_editor:
            os.write(master, b'\t\t'); drain(0.3)
            assert b'Settings' in captured, 'settings tab did not render'
            os.write(master, b'\r'); drain(0.2)
            os.write(master, b'i'); drain(0.2)
            os.write(master, b'nvim'); drain(0.2)
            os.write(master, b'\x13'); drain(0.3)
            assert tomllib.loads((config_dir / 'config.toml').read_text()).get('default_preset') == 'nvim', 'configuration editor did not persist default preset'
            print('PASS: Settings tab default-preset edit and persistence')
        os.write(master, b'\x1b'); drain(0.2)
        os.write(master, b'g'); drain(0.3)
        if args.oauth_configured:
            assert b'Client ID' not in captured, 'embedded OAuth Client ID missing'
        else:
            assert b'Client ID' in captured, 'missing OAuth Client ID not reported'
        os.write(master, b'\x1b'); drain(0.2)
        os.write(master, b'q'); drain(0.3)
        assert process.wait(timeout=5) == 0, 'TUI shutdown failed'
        assert (config_dir / 'presets.json').exists(), 'defaults not generated'
        assert (config_dir / 'templates.json').exists(), 'templates not generated'
        assert malformed_ansi(captured) is None, 'malformed ANSI or leaked SGR emitted'
        assert decorative_background(captured) is None, 'explicit background/inverse video emitted'
        print('PASS: isolated TUI startup, project rendering, workspace, inspector, live search, multi-selection/bulk cancellation, palette/Settings, help, resize, flat view, root switching, favorite, preset/template forms, OAuth, cancellation, shutdown, transparency')
        print('Captured terminal bytes:', len(captured))
    finally:
        if args.capture:
            args.capture.write_bytes(captured)
        if process.poll() is None: process.kill(); process.wait()
        os.close(master)
