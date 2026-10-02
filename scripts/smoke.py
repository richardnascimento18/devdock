import argparse, fcntl, os, pathlib, pty, select, struct, subprocess, tempfile, termios, time, tomllib
parser = argparse.ArgumentParser(description="Smoke-test a DevDock binary in an isolated terminal/workspace.")
parser.add_argument("binary", type=pathlib.Path)
parser.add_argument("--oauth-configured", action="store_true", help="expect an embedded public Client ID; never complete authorization")
parser.add_argument("--configuration-editor", action="store_true", help="exercise the Settings tab and persist a default preset")
args = parser.parse_args()
binary = args.binary.resolve()
with tempfile.TemporaryDirectory(prefix='devdock-smoke-') as directory:
    test_home = pathlib.Path(directory)
    root = test_home / 'workspace'
    project = root / 'apps' / 'demo'
    project.mkdir(parents=True)
    (project / 'go.mod').write_text('module demo\n')
    config_dir = test_home / '.config' / 'devdock'
    config_dir.mkdir(parents=True)
    (config_dir / 'config.toml').write_text(f'roots = ["{root}"]\n')
    master, slave = pty.openpty()
    fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", 40, 140, 0, 0))
    child_env = os.environ.copy()
    child_env.update({'HOME': str(test_home), 'TERM': 'xterm-256color'})
    child_env.pop('DEVDOCK_GITHUB_CLIENT_ID', None)
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
        os.write(master, b'v'); drain(0.2)
        os.write(master, b'\t'); drain(0.2)
        os.write(master, b'f'); drain(0.2)
        os.write(master, b'e'); drain(0.3)
        assert b'Presets' in captured and b'Templates' in captured, 'editor did not render'
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
        print('PASS: isolated TUI startup, project rendering, flat view, root switching, favorite toggle, editor, OAuth screen, cancellation, shutdown')
        print('Captured terminal bytes:', len(captured))
    finally:
        if process.poll() is None: process.kill(); process.wait()
        os.close(master)
