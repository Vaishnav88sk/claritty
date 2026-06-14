# -*- mode: python ; coding: utf-8 -*-


a = Analysis(
    ['claritty_sre/cli.py'],
    pathex=[],
    binaries=[],
    datas=[('claritty_sre/runbooks', 'claritty_sre/runbooks')],
    hiddenimports=[],
    hookspath=[],
    hooksconfig={},
    runtime_hooks=[],
    excludes=[
        'matplotlib', 'scipy', 'pandas', 'tkinter', 'unittest', 
        'cv2', 'PyQt5', 'PyQt6', 'IPython', 'jupyter', 'notebook', 
        'PIL', 'numpy.core.tests', 'numpy.testing', 'pydoc_data', 
        'setuptools', 'pip', 'wheel'
    ],
    noarchive=False,
    optimize=0,
)
pyz = PYZ(a.pure)

exe = EXE(
    pyz,
    a.scripts,
    a.binaries,
    a.datas,
    [],
    name='clarctl',
    debug=False,
    bootloader_ignore_signals=False,
    strip=False,
    upx=True,
    upx_exclude=[],
    runtime_tmpdir=None,
    console=True,
    disable_windowed_traceback=False,
    argv_emulation=False,
    target_arch=None,
    codesign_identity=None,
    entitlements_file=None,
)
