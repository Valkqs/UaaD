#!/usr/bin/env bash
# JMeter 无界面跑计划并生成 HTML 报告（macOS / Linux）。
# 生成物写入 out/（该目录已加入 .gitignore）。
#
# 压测前自动：
#   1. 用 MERCHANT 账号创建并发布新活动（触发 Redis 库存初始化）
#   2. 调用 gen_jmeter_data 生成对应的 jmeter_users.csv
# 用法：
#   bash run-jmeter-report.sh              # 默认 1000 线程
#   bash run-jmeter-report.sh 1000
#   bash run-jmeter-report.sh 3000
#   bash run-jmeter-report.sh 5000
#   bash run-jmeter-report.sh 10000       # 万级：10000 独立用户各报名一次
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$here"

# ── thread count ──────────────────────────────────────────────────────────
thread_count="${1:-1000}"
case "$thread_count" in
  1000)  rampup=30 ;;
  3000)  rampup=60 ;;
  5000)  rampup=90 ;;
  10000) rampup=60 ;;
  *)
    echo "Usage: $0 [1000|3000|5000|10000]"
    echo "  Invalid thread count: ${thread_count}"
    exit 1
    ;;
esac

jmx="enrollment-load.jmx"
out_root="$here/out"
mkdir -p "$out_root"

# ── config ────────────────────────────────────────────────────────────────
env_file="$here/../../.env"
PORT=8080
if [[ -f "$env_file" ]]; then
  PORT=$(grep -E '^PORT=' "$env_file" | tr -d '\r' | cut -d= -f2)
  PORT="${PORT:-8080}"
fi
BASE_URL="http://localhost:${PORT}"
MERCHANT_PHONE="${MERCHANT_PHONE:-13800000004}"
MERCHANT_PASS="${MERCHANT_PASS:-test123456}"

# ── helpers ───────────────────────────────────────────────────────────────
json_val() {
  python3 -c "import json,sys; d=json.loads(sys.argv[1]); print($2)" "$1" 2>/dev/null
}

echo "=== Pre-flight: create & publish a fresh activity ==="
echo "    threads=${thread_count}  ramp-up=${rampup}s"

# ── 1. login MERCHANT ─────────────────────────────────────────────────────
login_resp=$(curl -s -X POST "${BASE_URL}/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"phone\":\"${MERCHANT_PHONE}\",\"password\":\"${MERCHANT_PASS}\"}")
token=""
token=$(json_val "$login_resp" "d['data']['token']") || true

if [[ -z "$token" ]]; then
  echo "[FAIL] MERCHANT login failed (${MERCHANT_PHONE})"
  echo "  response: $login_resp"
  echo "  Make sure the backend is running and the MERCHANT account exists."
  exit 1
fi
echo "[OK] MERCHANT login succeeded"

# ── 2. create activity ────────────────────────────────────────────────────
open_at=$(python3 -c "from datetime import datetime,timedelta,timezone as tz; print((datetime.now(tz.utc)-timedelta(minutes=5)).strftime('%Y-%m-%dT%H:%M:%SZ'))")
close_at=$(python3 -c "from datetime import datetime,timedelta,timezone as tz; print((datetime.now(tz.utc)+timedelta(days=3)).strftime('%Y-%m-%dT%H:%M:%SZ'))")
activity_at=$(python3 -c "from datetime import datetime,timedelta,timezone as tz; print((datetime.now(tz.utc)+timedelta(days=30)).strftime('%Y-%m-%dT%H:%M:%SZ'))")

capacity=$(( thread_count * 5 ))

create_resp=$(curl -s -X POST "${BASE_URL}/api/v1/activities" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${token}" \
  -d "{
    \"title\": \"loadtest-$(date +%Y%m%d-%H%M%S)\",
    \"description\": \"JMeter auto-created\",
    \"category\": \"CONCERT\",
    \"location\": \"auto\",
    \"activity_at\": \"${activity_at}\",
    \"enroll_open_at\": \"${open_at}\",
    \"enroll_close_at\": \"${close_at}\",
    \"max_capacity\": ${capacity},
    \"price\": 0
  }")

new_id=""
new_id=$(json_val "$create_resp" "d['data']['activity_id']") || true

if [[ -z "$new_id" ]]; then
  echo "[FAIL] Activity creation failed"
  echo "  response: ${create_resp}"
  exit 1
fi
echo "[OK] Activity created: id=${new_id}, capacity=${capacity}"

# ── 3. publish via API (triggers Redis WarmUp) ────────────────────────────
pub_resp=$(curl -s -X PUT "${BASE_URL}/api/v1/activities/${new_id}/publish" \
  -H "Authorization: Bearer ${token}")

stock=""
stock=$(json_val "$pub_resp" "d['data']['stock_in_cache']") || true

if [[ -z "$stock" ]]; then
  echo "[FAIL] Activity ${new_id} publish failed"
  echo "  response: ${pub_resp}"
  exit 1
fi
echo "[OK] Activity published: Redis stock=${stock}"

# ── 4. generate jmeter_users.csv ──────────────────────────────────────────
echo "[..] Generating ${thread_count} user tokens..."
(cd "$here/../.." && go run ./scripts/gen_jmeter_data -count "$thread_count")
echo "[OK] CSV generated"

# ── run JMeter ─────────────────────────────────────────────────────────────
echo ""
echo "=== JMeter load test (threads=${thread_count}, ramp-up=${rampup}s) ==="
ts="$(date +%Y%m%d-%H%M%S)"
report_dir="${out_root}/report-${thread_count}t-${ts}"
jtl="${out_root}/results-${thread_count}t-${ts}.jtl"
jm_log="${out_root}/jmeter-${thread_count}t-${ts}.log"

[[ -e "$report_dir" ]] && rm -rf "$report_dir"

echo "JTL:    ${jtl}"
echo "Log:    ${jm_log}"
echo "Report: ${report_dir}"

jmeter -n -t "$jmx" \
  -Jthreads="${thread_count}" \
  -Jrampup="${rampup}" \
  -l "$jtl" -j "$jm_log" -e -o "$report_dir"
echo "Done. Open ${report_dir}/index.html"
