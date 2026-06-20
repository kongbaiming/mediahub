#!/usr/bin/env bash
# scripts/fix-qbit-password.sh
#
# 一次性脚本：固化 qBittorrent WebUI 密码，解决"每次容器重启密码都变"的问题。
#
# 根因：linuxserver/qbittorrent 镜像启动时如果 /config/qBittorrent/qBittorrent.conf
# 里没有 WebUI\Password_PBKDF2 行，就生成一次性临时密码（只在内存，重启即丢）。
# 必须在 WebUI/API 里**显式改一次密码**才会写进 conf。
#
# 做法：用 qBit 自己的 Web API 改密码（比点 WebUI 更稳，绕过任何前端缓存问题）
#
# 用法：
#   1. cd /volume1/progect/mediahub
#   2. bash scripts/fix-qbit-password.sh
#   3. 输入新密码（或回车用默认 MyNAS2026!qBit）

set -euo pipefail

CONTAINER="${QBIT_CONTAINER:-mediahub-qbittorrent}"
QBIT_URL="${QBIT_URL:-http://localhost:8080}"
CONF_PATH="${CONF_PATH:-/volume1/docker/mediahub/qbittorrent/config/qBittorrent/qBittorrent.conf}"
ENV_FILE="${ENV_FILE:-/volume1/progect/mediahub/.env}"

# ---------- 颜色 ----------
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'

info()  { printf "${GREEN}[INFO]${NC}  %s\n" "$*"; }
warn()  { printf "${YELLOW}[WARN]${NC}  %s\n" "$*"; }
err()   { printf "${RED}[ERR]${NC}   %s\n" "$*" >&2; }

# ---------- 1. 拿最新临时密码 ----------
info "从 docker logs 抓最新临时密码..."
TEMP_PW=$(docker logs --tail 80 "$CONTAINER" 2>&1 \
  | grep -oP 'temporary password is provided for this session: \K\S+' \
  | tail -1 || true)

if [ -z "$TEMP_PW" ]; then
  err "没找到临时密码。可能 qBit 已经在用持久密码（grep ${CONF_PATH} 验证），或者容器没运行。"
  exit 1
fi
info "TEMP_PW=$TEMP_PW"

# ---------- 2. 如果 conf 已经有密码，跳过 ----------
if grep -q '^WebUI\\Password_PBKDF2' "$CONF_PATH" 2>/dev/null; then
  info "conf 已经有 Password_PBKDF2 —— 密码已经固化，不需要改"
  CURRENT=$(grep '^QBIT_PASSWORD=' "$ENV_FILE" | cut -d= -f2- || echo "")
  info "当前 .env 里 QBIT_PASSWORD=$CURRENT"
  exit 0
fi

# ---------- 3. 默认用临时密码作为永久密码 ----------
# 不需要询问用户——直接拿当前临时密码固化为永久密码。
# 临时密码本身已经是随机生成的、强度足够（10 字符 [a-zA-Z0-9]），
# 没有必要再换一个。省掉交互步骤也避免用户输入错误。
#
# 如果想指定别的密码，传环境变量即可：
#   FIX_QBIT_PASSWORD="MyStrongPwd!" bash scripts/fix-qbit-password.sh
NEW_PW="${FIX_QBIT_PASSWORD:-$TEMP_PW}"
info "新密码（直接固化临时密码）: $NEW_PW"

# ---------- 4. 登录拿 SID cookie ----------
info "登录 qBit Web API..."
COOKIE_JAR=$(mktemp)
trap "rm -f $COOKIE_JAR" EXIT

LOGIN_RESP=$(curl -sS -c "$COOKIE_JAR" -X POST "$QBIT_URL/api/v2/auth/login" \
  --data-urlencode "username=admin" \
  --data-urlencode "password=$TEMP_PW" || echo "CURL_FAILED")

if [ "$LOGIN_RESP" != "Ok." ]; then
  err "登录失败: $LOGIN_RESP"
  exit 1
fi
info "登录成功"

# ---------- 5. 改密码（这一调用才会写 conf） ----------
info "调用 /api/v2/app/setPreferences 改密码（这一步才会写 conf）..."
SET_RESP=$(curl -sS -b "$COOKIE_JAR" -X POST "$QBIT_URL/api/v2/app/setPreferences" \
  --data-urlencode "json={\"webui_username\":\"admin\",\"webui_password\":\"$NEW_PW\"}")

if [ "$SET_RESP" != "Ok." ]; then
  err "改密码失败: $SET_RESP"
  exit 1
fi
info "改密码 API 返回 Ok."

# ---------- 6. 验证 conf ----------
sleep 1
if ! grep -q '^WebUI\\Password_PBKDF2' "$CONF_PATH"; then
  err "conf 里仍然没有 Password_PBKDF2 —— qBit 内部写盘失败，方案失败"
  exit 1
fi
info "conf 已写入 Password_PBKDF2 ✓"

# ---------- 7. 同步到 .env ----------
info "更新 .env..."
if grep -q '^QBIT_PASSWORD=' "$ENV_FILE"; then
  sed -i "s|^QBIT_PASSWORD=.*|QBIT_PASSWORD=${NEW_PW}|" "$ENV_FILE"
else
  echo "QBIT_PASSWORD=${NEW_PW}" >> "$ENV_FILE"
fi
info "QBIT_PASSWORD=${NEW_PW}"

# ---------- 8. 重启 API 容器（让它用新密码连 qBit）----------
info "重启 api 容器..."
( cd "$(dirname "$ENV_FILE")" && docker compose restart api )
info "api 已重启"

# ---------- 9. 验证 ----------
sleep 3
info "验证 qBit 登录..."
VERIFY=$(curl -sS -X POST "$QBIT_URL/api/v2/auth/login" \
  --data-urlencode "username=admin" \
  --data-urlencode "password=$NEW_PW" || echo "VERIFY_FAILED")

if [ "$VERIFY" = "Ok." ]; then
  printf "${GREEN}[DONE]${NC}  qBit 密码已固化：%s\n" "$NEW_PW"
else
  err "用新密码登录失败: $VERIFY"
  exit 1
fi

# ---------- 10. 清理 test.txt（之前诊断留下的）----------
docker exec "$CONTAINER" rm -f /config/qBittorrent/config/test.txt 2>/dev/null || true
rm -f /volume1/docker/mediahub/qbittorrent/config/qBittorrent/config/test.txt 2>/dev/null || true

echo
echo "接下来："
echo "  1. 浏览器访问 http://nas.local:8080 用 admin / ${NEW_PW} 登录 qBit WebUI"
echo "  2. 重启 qBit 验证密码持久化："
echo "       docker restart mediahub-qbittorrent"
echo "       docker logs --tail 30 mediahub-qbittorrent | grep -i password"
echo "     不应该再出现 'temporary password is provided'"