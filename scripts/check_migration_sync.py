#!/usr/bin/env python3
"""Enforce goose/GORM schema consistency (tables, columns and column types).

Every GORM model table must exist in the goose SQL migrations with all of its
columns, and each column's base type must match (e.g. bigint vs int, text vs
varchar). Without this check the two migration tracks can drift silently (as
they did for transcode_dead_letters v23 and transcode_outbox v26-28):

  - a model renamed/added without a goose migration -> table missing;
  - a model column added without a goose migration -> column missing;
  - a column type changed on one track only -> type mismatch.

The check is deliberately one-directional (model -> SQL): historical SQL may
contain extra columns that models do not map, which is not drift. Type
comparison uses the normalized base type (size/unsigned/null are ignored),
so varchar(64) vs varchar(255) is not reported as drift.

Usage:
  python3 scripts/check_migration_sync.py
"""

import os
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent

# ---------------------------------------------------------------- SQL parser

CREATE_TABLE_RE = re.compile(
    r"(?i)CREATE TABLE (?:IF NOT EXISTS )?`?([A-Za-z0-9_]+)`?\s*\("
)
ALTER_TABLE_RE = re.compile(r"(?i)ALTER TABLE\s+`?([A-Za-z0-9_]+)`?")
ADD_COLUMN_RE = re.compile(
    r"(?i)ADD COLUMN\s+`?([A-Za-z_][A-Za-z0-9_]*)`?\s+([A-Za-z]+(?:\([^)]*\))?(?:\s+unsigned)?)"
)
MODIFY_COLUMN_RE = re.compile(
    r"(?i)MODIFY COLUMN\s+`?([A-Za-z_][A-Za-z0-9_]*)`?\s+([A-Za-z]+(?:\([^)]*\))?(?:\s+unsigned)?)"
)
CHANGE_COLUMN_RE = re.compile(
    r"(?i)CHANGE COLUMN\s+`?([A-Za-z_][A-Za-z0-9_]*)`?\s+`?([A-Za-z_][A-Za-z0-9_]*)`?\s+"
    r"([A-Za-z]+(?:\([^)]*\))?(?:\s+unsigned)?)"
)
RENAME_COLUMN_RE = re.compile(
    r"(?i)RENAME COLUMN\s+`?([A-Za-z_][A-Za-z0-9_]*)`?\s+TO\s+`?([A-Za-z_][A-Za-z0-9_]*)`?"
)
COLUMN_RE = re.compile(
    r"(?im)^\s*`?([A-Za-z_][A-Za-z0-9_]*)`?\s+"
    r"([A-Za-z]+(?:\([^)]*\))?(?:\s+unsigned)?)\b"
)


def type_base(sql_type):
    """Normalize a MySQL type to its base name, ignoring size/unsigned/NULL."""
    return re.split(r"[(\s]", sql_type.strip(), maxsplit=1)[0].lower()


def parse_sql_migrations(migrations_dir):
    """Return {table: {column: base_type}} from the Up sections of goose migrations."""
    tables = {}

    def ensure(table):
        return tables.setdefault(table, {})

    for path in sorted(Path(migrations_dir).glob("*.sql")):
        in_up = True
        body = ""
        pending_alter = None
        for raw_line in path.read_text(encoding="utf-8").splitlines():
            line = raw_line.strip()
            if line.startswith("-- +goose Up"):
                in_up = True
                continue
            if line.startswith("-- +goose Down"):
                in_up = False
                continue
            if not in_up:
                continue

            m = ALTER_TABLE_RE.match(line)
            if m:
                pending_alter = m.group(1).strip("`")
                if "ADD COLUMN" in line or "MODIFY COLUMN" in line:
                    cm = ADD_COLUMN_RE.search(line) or MODIFY_COLUMN_RE.search(line)
                    if cm:
                        ensure(pending_alter)[cm.group(1).strip("`")] = type_base(cm.group(2))
                    pending_alter = None
                    continue
                if "CHANGE COLUMN" in line:
                    cm = CHANGE_COLUMN_RE.search(line)
                    if cm:
                        cols = ensure(pending_alter)
                        cols.pop(cm.group(1).strip("`"), None)
                        cols[cm.group(2).strip("`")] = type_base(cm.group(3))
                    pending_alter = None
                    continue
                if "RENAME COLUMN" in line:
                    cm = RENAME_COLUMN_RE.search(line)
                    if cm:
                        cols = ensure(pending_alter)
                        old = cols.pop(cm.group(1).strip("`"), None)
                        if old is not None:
                            cols[cm.group(2).strip("`")] = old
                    pending_alter = None
                    continue
                continue
            if pending_alter is not None:
                cm = ADD_COLUMN_RE.match(line)
                if cm:
                    ensure(pending_alter)[cm.group(1).strip("`")] = type_base(cm.group(2))
                mm = MODIFY_COLUMN_RE.match(line)
                if mm:
                    ensure(pending_alter)[mm.group(1).strip("`")] = type_base(mm.group(2))
                if line.endswith(";"):
                    pending_alter = None
                continue

            m = CREATE_TABLE_RE.search(line)
            if m:
                body = raw_line + "\n"
                continue
            if body:
                body += raw_line + "\n"
                if line.startswith(")") and line.endswith(";"):
                    tm = CREATE_TABLE_RE.search(body)
                    if tm:
                        cols = ensure(tm.group(1).strip("`"))
                        for cm in COLUMN_RE.finditer(body):
                            cols[cm.group(1).strip("`")] = type_base(cm.group(2))
                    body = ""
    return tables


# ---------------------------------------------------------------- model parser

STRUCT_RE = re.compile(r"^type\s+(\w+)\s+struct\s*\{")
FIELD_RE = re.compile(
    r"^\s+(\w+)\s+([\w.*\[\]]+)\s*(?:`gorm:\"([^\"]*)\"`)?"
)
TABLE_NAME_RE = re.compile(
    r"func\s+\(\*?\s*(\w+)\s*\)\s+TableName\(\)\s+string\s*\{\s*return\s+\"([A-Za-z0-9_]+)\"\s*\}"
)


def snake_case(name):
    out = []
    prev_upper = False
    for i, ch in enumerate(name):
        if ch.isupper():
            prev_lower = i > 0 and (name[i - 1].islower() or name[i - 1].isdigit())
            next_lower = i + 1 < len(name) and name[i + 1].islower()
            if prev_lower or (prev_upper and next_lower):
                out.append("_")
            out.append(ch.lower())
            prev_upper = True
        else:
            out.append(ch)
            prev_upper = False
    return "".join(out)


def plural(name):
    """GORM's inflection.Pluralize for the patterns used in this repo."""
    if name.endswith("s"):
        return name
    if name.endswith("y") and len(name) > 1 and name[-2] not in "aeiou":
        return name[:-1] + "ies"
    return name + "s"


def gorm_type_base(go_type, gorm_tag):
    """Derive the MySQL base type GORM would use for a Go field."""
    if gorm_tag:
        for part in gorm_tag.split(";"):
            if part.startswith("type:"):
                return type_base(part.split(":", 1)[1])
            if part.startswith("serializer:"):
                return "json"
    go_type = go_type.strip()
    if go_type.startswith("*"):
        go_type = go_type[1:]
    if go_type.startswith("[]"):
        return "blob" if go_type == "[]byte" else "json"
    return {
        "uint64": "bigint",
        "uint": "bigint",
        "int64": "bigint",
        "int": "bigint",
        "int32": "int",
        "int16": "smallint",
        "int8": "tinyint",
        "uint32": "int",
        "uint16": "smallint",
        "uint8": "tinyint",
        "string": "varchar",
        "bool": "tinyint",
        "time.Time": "datetime",
        "float64": "double",
        "float32": "float",
        "json.RawMessage": "json",
    }.get(go_type, go_type)


def parse_models(model_dir):
    """Return [Model(table=..., columns={col: base_type})] for table structs."""
    table_names = {}  # "pkg.Type" -> explicit TableName
    structs = {}  # "pkg.Type" -> (type_name, fields)
    for root, _, files in os.walk(model_dir):
        pkg = Path(root).name
        for fn in files:
            if not fn.endswith(".go") or fn.endswith("_test.go"):
                continue
            text = (Path(root) / fn).read_text(encoding="utf-8")
            for m in TABLE_NAME_RE.finditer(text):
                table_names[f"{pkg}.{m.group(1)}"] = m.group(2)
            current = None
            depth = 0
            for line in text.splitlines():
                sm = STRUCT_RE.match(line)
                if sm:
                    current = sm.group(1)
                    structs.setdefault(f"{pkg}.{current}", [])
                    depth = 1
                    continue
                if current is not None:
                    if line.strip() == "}":
                        depth -= 1
                        if depth == 0:
                            current = None
                        continue
                    if line.strip().startswith("}") and depth == 1:
                        current = None
                        continue
                    fm = FIELD_RE.match(line)
                    if fm:
                        structs[f"{pkg}.{current}"].append(
                            (fm.group(1), fm.group(2), fm.group(3) or "")
                        )

    models = []
    for key, fields in structs.items():
        pkg, type_name = key.rsplit(".", 1)
        table = table_names.get(key)
        if table is None:
            if not any(name == "ID" or "primaryKey" in tag for name, _, tag in fields):
                continue  # helper struct, not a table
            table = plural(snake_case(type_name))
        columns = {}
        for name, go_type, tag in fields:
            if tag == "-":
                continue
            if name in ("ID", "CreatedAt", "UpdatedAt", "DeletedAt"):
                columns[snake_case(name)] = gorm_type_base(go_type, tag)
                continue
            col = None
            for part in tag.split(";"):
                if part.startswith("column:"):
                    col = part.split(":", 1)[1].strip()
                    break
            columns[col or snake_case(name)] = gorm_type_base(go_type, tag)
        models.append({"type": f"{pkg}.{type_name}", "table": table, "columns": columns})
    return models


def main():
    sql_tables = parse_sql_migrations(ROOT / "migrations")
    models = parse_models(ROOT / "internal" / "model")

    problems = []
    for model in models:
        table = model["table"]
        if table not in sql_tables:
            problems.append(
                f"model {model['type']}: table {table!r} missing from goose SQL migrations"
            )
            continue
        for col, want_type in sorted(model["columns"].items()):
            if col not in sql_tables[table]:
                problems.append(
                    f"model {model['type']}: column {table}.{col} missing from goose SQL"
                )
                continue
            if sql_tables[table][col] != want_type:
                problems.append(
                    f"model {model['type']}: column {table}.{col} type mismatch "
                    f"(goose {sql_tables[table][col]} vs GORM {want_type})"
                )

    if problems:
        print(f"goose/GORM drift detected ({len(problems)}):", file=sys.stderr)
        for p in sorted(problems):
            print("  -", p, file=sys.stderr)
        return 1
    print(f"OK: {len(models)} GORM models match goose SQL migrations")
    return 0


if __name__ == "__main__":
    sys.exit(main())
