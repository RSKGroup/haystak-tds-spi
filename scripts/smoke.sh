#!/usr/bin/env bash
# Live smoke: a real go-sqlcmd client runs queries through the gateway (in-mem backend),
# over both plaintext and TLS. Usage: scripts/smoke.sh [port]
set -uo pipefail

PORT="${1:-21433}"
TLSPORT=$((PORT + 1))
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if ! command -v sqlcmd >/dev/null 2>&1; then
  echo "SKIP: sqlcmd not found (brew install sqlcmd)"
  exit 0
fi

BIN="$(mktemp -t haystak-gw.XXXXXX)"
GW=""
cleanup() { [ -n "$GW" ] && kill "$GW" 2>/dev/null; rm -f "$BIN"; }
trap cleanup EXIT
go build -o "$BIN" ./examples/gateway || { echo "build failed"; exit 1; }

fail=0
CUR=0
q() { sqlcmd -S "127.0.0.1,$CUR" -U sa -P x -C -h-1 -Q "$1" 2>&1; }
present() { local out; out="$(q "$2")"; if printf '%s' "$out" | grep -q -- "$3"; then echo "  PASS  $1"; else echo "  FAIL  $1 (want /$3/)"; printf '%s\n' "$out" | sed 's/^/        /'; fail=1; fi; }
absent()  { local out; out="$(q "$2")"; if printf '%s' "$out" | grep -q -- "$3"; then echo "  FAIL  $1 (unexpected /$3/)"; fail=1; else echo "  PASS  $1"; fi; }

checks() {
  present "select by id"      "SELECT name FROM users WHERE id = 2"                                         "alan"
  absent  "filter > excludes" "SELECT name FROM users WHERE id > 1"                                         "ada"
  present "order desc + top"  "SELECT TOP 1 id FROM users ORDER BY id DESC"                                 "2"
  present "catalog tables"    "SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES"                            "users"
  present "catalog col type"  "SELECT DATA_TYPE FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME = 'users'" "bigint"
  present "@@version probe"   "SELECT @@VERSION"                                                            "Microsoft SQL Server"
  absent  "set no-op"         "SET NOCOUNT ON"                                                             "Msg"
  present "where IN"          "SELECT name FROM users WHERE id IN (2)"                                      "alan"
  present "where OR"          "SELECT name FROM users WHERE id = 1 OR id = 2"                               "ada"
  present "where LIKE"        "SELECT name FROM users WHERE name LIKE 'al%'"                                "alan"
  absent  "where LIKE excl"   "SELECT name FROM users WHERE name LIKE 'al%'"                                "ada"
  present "where BETWEEN"     "SELECT name FROM users WHERE id BETWEEN 2 AND 9"                             "alan"
  present "distinct"          "SELECT DISTINCT name FROM users"                                             "alan"
  present "alias"             "SELECT name AS who FROM users WHERE id = 2"                                  "alan"
  present "count(*)"          "SELECT COUNT(*) FROM users"                                                  "2"
  present "max(id)"           "SELECT MAX(id) FROM users"                                                   "2"
  present "sum(id)"           "SELECT SUM(id) FROM users"                                                   "3"
  present "group by"          "SELECT name, COUNT(*) AS c FROM users GROUP BY name"                         "ada"
  present "having"            "SELECT name, COUNT(*) AS c FROM users GROUP BY name HAVING c >= 1"           "ada"
  present "table alias"       "SELECT name FROM users u WHERE id = 2"                                       "alan"
  present "qualified col"     "SELECT u.name FROM users u WHERE u.id = 2"                                   "alan"
  present "inner join"        "SELECT u.name, o.amount FROM users u JOIN orders o ON u.id = o.user_id"      "alan"
  present "join + where"      "SELECT u.name FROM users u JOIN orders o ON u.id = o.user_id WHERE o.amount = 100" "ada"
  absent  "join where excl"   "SELECT u.name FROM users u JOIN orders o ON u.id = o.user_id WHERE o.amount = 100" "alan"
  present "join + group sum"  "SELECT u.name, SUM(o.amount) AS total FROM users u JOIN orders o ON u.id = o.user_id GROUP BY u.name" "250"
  present "offset/fetch"      "SELECT id FROM orders ORDER BY id OFFSET 1 ROWS FETCH NEXT 1 ROWS ONLY"       "11"
  absent  "offset skips"      "SELECT id FROM orders ORDER BY id OFFSET 1 ROWS FETCH NEXT 1 ROWS ONLY"       "10"
  present "@@servername"      "SELECT @@SERVERNAME"                                                         "haystak"
  present "db_name()"         "SELECT DB_NAME()"                                                            "master"
  present "system_user"       "SELECT SYSTEM_USER"                                                          "haystak"
  present "serverproperty"    "SELECT SERVERPROPERTY('ProductVersion')"                                     "16.0"
  absent  "use no-op"         "USE master"                                                                 "Msg"
  present "multi-stmt batch"  "SET NOCOUNT ON; SELECT name FROM users WHERE id = 2"                         "alan"
  present "sys.tables"        "SELECT name FROM sys.tables"                                                 "users"
  present "sys.columns"       "SELECT name FROM sys.columns WHERE object_id = 100"                          "id"
  present "sys.databases"     "SELECT name FROM sys.databases WHERE database_id = 5"                        "haystak"
  present "sys.schemas"       "SELECT name FROM sys.schemas WHERE schema_id = 1"                            "dbo"
  present "sys.types"         "SELECT name FROM sys.types WHERE system_type_id = 127"                       "bigint"
  present "sys.foreign_keys"  "SELECT name FROM sys.foreign_keys"                                           "FK_orders_users"
  present "ref constraints"   "SELECT CONSTRAINT_NAME FROM INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS"       "FK_emps_depts"
  present "key col usage"     "SELECT COLUMN_NAME FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE WHERE CONSTRAINT_NAME = 'FK_orders_users'" "user_id"
  present "table constraints" "SELECT CONSTRAINT_TYPE FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS WHERE TABLE_NAME = 'orders'" "FOREIGN KEY"
  present "expr arith"        "SELECT id + 10 FROM users WHERE id = 2"                                      "12"
  present "expr upper"        "SELECT UPPER(name) FROM users WHERE id = 2"                                   "ALAN"
  present "expr len"          "SELECT LEN(name) FROM users WHERE id = 1"                                     "3"
  present "expr concat"       "SELECT name + '!' AS greet FROM users WHERE id = 2"                           "alan!"
  present "expr isnull"       "SELECT ISNULL(name, 'none') FROM users WHERE id = 1"                          "ada"
  present "case searched"     "SELECT CASE WHEN id = 1 THEN 'one' ELSE 'other' END FROM users WHERE id = 1"  "one"
  present "case else"         "SELECT CASE WHEN id = 99 THEN 'x' ELSE 'other' END FROM users WHERE id = 2"   "other"
  present "case simple"       "SELECT CASE id WHEN 2 THEN 'two' ELSE 'no' END FROM users WHERE id = 2"       "two"
  present "cast concat"       "SELECT CAST(id AS VARCHAR) + '!' FROM users WHERE id = 2"                      "2!"
  present "convert"           "SELECT CONVERT(VARCHAR, id) FROM users WHERE id = 2"                          "2"
  present "where fn-left"     "SELECT name FROM users WHERE LEN(name) = 4"                                  "alan"
  absent  "where fn-left ex"  "SELECT name FROM users WHERE LEN(name) = 4"                                  "ada"
  present "where arith-left"  "SELECT name FROM users WHERE id + 1 = 2"                                     "ada"
  present "where upper-left"  "SELECT name FROM users WHERE UPPER(name) = 'ALAN'"                           "alan"
  present "union"             "SELECT name FROM users WHERE id = 1 UNION SELECT name FROM users WHERE id = 2" "alan"
  present "union cross-tbl"   "SELECT id FROM users UNION SELECT id FROM orders ORDER BY id"                 "12"
  present "union all dup"     "SELECT id FROM users UNION ALL SELECT id FROM users ORDER BY id"              "1"
  present "intersect"         "SELECT user_id FROM orders INTERSECT SELECT id FROM users ORDER BY user_id"   "1"
  present "except"            "SELECT id FROM orders EXCEPT SELECT id FROM users ORDER BY id"                "10"
  absent  "except empties"    "SELECT id FROM users EXCEPT SELECT id FROM users"                            "1"
  present "in subquery"       "SELECT name FROM users WHERE id IN (SELECT user_id FROM orders WHERE amount = 100)" "ada"
  absent  "in subquery excl"  "SELECT name FROM users WHERE id IN (SELECT user_id FROM orders WHERE amount = 100)" "alan"
  present "exists"            "SELECT name FROM users WHERE EXISTS (SELECT 1 FROM orders WHERE amount = 100)" "alan"
  absent  "exists false"      "SELECT name FROM users WHERE EXISTS (SELECT 1 FROM orders WHERE amount = 999)" "ada"
  present "scalar subquery"   "SELECT name FROM users WHERE id = (SELECT MIN(user_id) FROM orders)"          "ada"
  absent  "scalar subq excl"  "SELECT name FROM users WHERE id = (SELECT MIN(user_id) FROM orders)"          "alan"
  present "derived table"     "SELECT name FROM (SELECT name, id FROM users) AS t WHERE id = 2"              "alan"
  present "derived qualified" "SELECT t.name FROM (SELECT name, id FROM users WHERE id = 1) AS t"            "ada"
  present "derived agg"       "SELECT COUNT(*) AS c FROM (SELECT id FROM orders) AS t"                       "3"
  present "cte"               "WITH t AS (SELECT name, id FROM users) SELECT name FROM t WHERE id = 2"       "alan"
  present "cte agg"           "WITH big AS (SELECT id FROM orders WHERE amount > 60) SELECT COUNT(*) AS c FROM big" "2"
  present "decimal type"      "SELECT price FROM items WHERE id = 1"                                        "19.99"
  present "uuid type"         "SELECT ref FROM items WHERE id = 1"                                          "12345678"
  present "datetime2 type"    "SELECT made FROM items WHERE id = 1"                                         "2026-06-10"
  present "varbinary type"    "SELECT data FROM items WHERE id = 1"                                         "DEADBEEF"
  present "left join edge"    "SELECT e.name FROM emps e LEFT JOIN depts d ON e.dept_id = d.id"             "orphan"
  absent  "inner no orphan"   "SELECT e.name FROM emps e JOIN depts d ON e.dept_id = d.id"                  "orphan"
  present "right join edge"   "SELECT d.name AS dn FROM emps e RIGHT JOIN depts d ON e.dept_id = d.id"      "ops"
  present "full join left"    "SELECT e.name FROM emps e FULL JOIN depts d ON e.dept_id = d.id"             "orphan"
  present "full join right"   "SELECT d.name AS dn FROM emps e FULL JOIN depts d ON e.dept_id = d.id"       "ops"
  present "correlated exists" "SELECT name FROM depts d WHERE EXISTS (SELECT 1 FROM emps e WHERE e.dept_id = d.id)"     "eng"
  absent  "corr exists excl"  "SELECT name FROM depts d WHERE EXISTS (SELECT 1 FROM emps e WHERE e.dept_id = d.id)"     "ops"
  present "corr not exists"   "SELECT name FROM depts d WHERE NOT EXISTS (SELECT 1 FROM emps e WHERE e.dept_id = d.id)" "ops"
  absent  "corr not exists x" "SELECT name FROM depts d WHERE NOT EXISTS (SELECT 1 FROM emps e WHERE e.dept_id = d.id)" "eng"
  present "scalar no-from"    "SELECT 1 + 2 AS three"                                                       "3"
  present "recursive cte"     "WITH nums AS (SELECT 1 AS n UNION ALL SELECT n + 1 AS n FROM nums WHERE n < 5) SELECT COUNT(*) AS c FROM nums" "5"
  present "recursive cte max" "WITH nums AS (SELECT 1 AS n UNION ALL SELECT n + 1 AS n FROM nums WHERE n < 5) SELECT MAX(n) AS m FROM nums" "5"
  present "year()"            "SELECT YEAR(made) FROM items WHERE id = 1"                                   "2026"
  present "month()"           "SELECT MONTH(made) FROM items WHERE id = 1"                                  "6"
  present "day()"             "SELECT DAY(made) FROM items WHERE id = 1"                                    "10"
  present "getdate()"         "SELECT YEAR(GETDATE()) FROM items WHERE id = 1"                              "20"
  present "order by ordinal"  "SELECT TOP 1 id, name FROM users ORDER BY 1 DESC"                            "alan"
  absent  "order ordinal exc" "SELECT TOP 1 id, name FROM users ORDER BY 1 DESC"                            "ada"
  present "alias eq form"     "SELECT upper_name = UPPER(name) FROM users WHERE id = 2"                     "ALAN"
  present "top percent"       "SELECT TOP 50 PERCENT id FROM orders ORDER BY id"                            "11"
  absent  "top percent excl"  "SELECT TOP 50 PERCENT id FROM orders ORDER BY id"                            "12"
  present "insert + select"   "INSERT INTO users (id, name) VALUES (9, 'zoe'); SELECT name FROM users WHERE id = 9"                  "zoe"
  present "update"            "INSERT INTO users (id, name) VALUES (7, 'old'); UPDATE users SET name = 'new' WHERE id = 7; SELECT name FROM users WHERE id = 7" "new"
  absent  "delete"            "INSERT INTO users (id, name) VALUES (8, 'tmp'); DELETE FROM users WHERE id = 8; SELECT name FROM users WHERE id = 8" "tmp"
  present "create table+dml"  "CREATE TABLE t9 (id INT, label NVARCHAR(50)); INSERT INTO t9 (id, label) VALUES (1, 'made'); SELECT label FROM t9 WHERE id = 1" "made"
}

start_gw() { # $1=port  $2=tls(0|1)
  local env_tls=""
  [ "$2" = 1 ] && env_tls="HAYSTAK_TLS=1"
  env $env_tls "$BIN" "127.0.0.1:$1" >/dev/null 2>&1 &
  GW=$!
  for _ in $(seq 1 300); do nc -z 127.0.0.1 "$1" 2>/dev/null && break || sleep 0.05; done
}

echo "haystak-tds-spi smoke (go-sqlcmd)"
echo "[plaintext :$PORT]"; start_gw "$PORT" 0; CUR=$PORT; checks; kill "$GW" 2>/dev/null; GW=""
echo "[TLS :$TLSPORT]"; start_gw "$TLSPORT" 1; CUR=$TLSPORT; checks; kill "$GW" 2>/dev/null; GW=""

if [ "$fail" -eq 0 ]; then echo "SMOKE PASSED"; else echo "SMOKE FAILED"; fi
exit "$fail"
