"""Cross-platform cleanup script (replaces rm/find in Makefile)."""
import pathlib, shutil, os

ROOT = pathlib.Path(__file__).parent

# Coverage artifacts
for name in ["coverage.out", "coverage_total", "coverage_total.out",
             "cov_out", "covprofile", "tmp_cov"]:
    (ROOT / name).unlink(missing_ok=True)

# Go build cache
gocache = ROOT / ".gocache"
if gocache.exists():
    shutil.rmtree(gocache, ignore_errors=True)

# Frontend coverage
vue_cov = ROOT / "cakecake-vue" / "bilibili-vue" / "coverage"
if vue_cov.exists():
    for f in vue_cov.glob("*"):
        if f.is_file():
            f.unlink(missing_ok=True)

# Python artifacts
for pyc in ROOT.rglob("*.pyc"):
    pyc.unlink(missing_ok=True)
for cache in ROOT.rglob("__pycache__"):
    shutil.rmtree(cache, ignore_errors=True)

# Temp / debug scripts (R-CLEAN-1)
patterns = ["_*.py", "_fix*.py", "_gen*.py", "_debug*",
            "fix_*.py", "make_*.py", "write_*.py", "test_a*.py"]
for p in patterns:
    for f in ROOT.glob(p):
        f.unlink(missing_ok=True)
        print(f"  removed {f.name}")

print("Cleanup done.")
