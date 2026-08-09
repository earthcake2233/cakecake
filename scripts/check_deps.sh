#!/usr/bin/env bash
# Dependency direction guard: keeps the Kratos-ward layering intact.
#
# Rules (add future ones here, e.g. for cakecake/api when it exists):
#   handler must not import internal/data
#   service must not import internal/handler
#   client must not import gorm
#   model must not import internal/service or internal/data
set -euo pipefail

cd "$(dirname "$0")/.."

go list -f '{{.ImportPath}}|{{join .Imports " "}}' ./internal/... 2>&1 |
	awk -F'|' '
	{
		n = split($2, imps, " ")
		for (i = 1; i <= n; i++) {
			imp = imps[i]
			if ($1 ~ /^cakecake\/internal\/handler/ && imp ~ /^cakecake\/internal\/data/) {
				print "handler -> data: " $1 " imports " imp; bad = 1
			}
			if ($1 ~ /^cakecake\/internal\/service/ && imp ~ /^cakecake\/internal\/handler/) {
				print "service -> handler: " $1 " imports " imp; bad = 1
			}
			if ($1 == "cakecake/internal/client" && imp ~ /^gorm\.io\/gorm/) {
				print "client -> gorm: " $1 " imports " imp; bad = 1
			}
			if ($1 ~ /^cakecake\/internal\/model/ && imp ~ /^cakecake\/internal\/(service|data)/) {
				print "model -> service/data: " $1 " imports " imp; bad = 1
			}
			# Future: transport must not import model directly once api/ DTOs exist.
			# if ($1 ~ /^cakecake\/internal\/handler/ && imp ~ /^cakecake\/api\//) { ... }
		}
	}
	END { if (bad) exit 1 }
	'
