#!/usr/bin/env python3
"""Clean Go build cache from C: drive after compilation.

Usage:
  python scripts/clean_go_cache.py              # clean all (local + system)
  python scripts/clean_go_cache.py --local-only  # only clean project-local .gocache/.gotmp
  python scripts/clean_go_cache.py --system-only # only clean C: system cache
"""
import shutil, os, sys, pathlib

BASE = pathlib.Path(__file__).resolve().parent.parent
LOCAL_CACHE = BASE / ".gocache"
LOCAL_TMP   = BASE / ".gotmp"

def clean_local():
    for p in [LOCAL_CACHE, LOCAL_TMP]:
        if p.exists():
            try:
                shutil.rmtree(str(p), ignore_errors=True)
                p.mkdir(parents=True, exist_ok=True)
                print("  Cleaned " + p.name)
            except Exception as e:
                print("  SKIP " + p.name + ": " + str(e))

def clean_system():
    localappdata = os.environ.get("LOCALAPPDATA", "")
    if not localappdata:
        print("  SKIP: LOCALAPPDATA not set")
        return
    go_build = pathlib.Path(localappdata) / "go-build"
    if go_build.exists():
        try:
            shutil.rmtree(str(go_build), ignore_errors=True)
            print("  Cleaned go-build (C:)")
        except Exception as e:
            print("  SKIP go-build: " + str(e))
    temp = pathlib.Path(localappdata) / "Temp"
    for d in list(temp.glob("go-*")) + list(temp.glob("gc-*")) + list(temp.glob("gm-*")):
        if d.is_dir():
            try:
                shutil.rmtree(str(d), ignore_errors=True)
                print("  Cleaned " + d.name + " (C: Temp)")
            except PermissionError:
                print("  SKIP " + d.name + ": in use")

if __name__ == "__main__":
    args = set(sys.argv[1:])
    do_local = "--local-only" in args or (not args)
    do_system = "--system-only" in args or (not args)
    if do_local:
        clean_local()
    if do_system:
        clean_system()
    print("OK")
