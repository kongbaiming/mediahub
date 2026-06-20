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
# 关键：用容器内 URL，不用宿主机的 localhost。
# Synology DSM 自带的 Reverse Proxy 会拦截 localhost:8080 返回 502，
# 容器内 localhost 才是 qBit 真正的 Web UI。
# 脚本通过 docker exec mediahub-qbittorrent 在 qBit 容器内发起 curl，
# 既绕过 DSM 反代，也保证 localhost 直达 qBit。
QBIT_URL="${QBIT_URL:-http://localhost:8080}"
QBIT_CURL="docker exec $CONTAINER curl -sS"
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

# 立刻验证一次——避免 docker logs 显示旧密码、qBit 实际已经重启过的情况
# 用 docker exec 在 qBit 容器内调 localhost:8080，绕过 DSM 的反向代理
info "立即验证临时密码（通过 docker exec 进 qBit 容器内，绕过 DSM 反代）..."
VERIFY=$($QBIT_CURL -m 5 -X POST "$QBIT_URL/api/v2/auth/login" \
  --data-urlencode "username=admin" \
  --data-urlencode "password=$TEMP_PW" 2>&1 || echo "CURL_ERR")
if [ "$VERIFY" != "Ok." ]; then
  err "logs 里的临时密码 $TEMP_PW 已失效（响应: $VERIFY）"
  err "可能 qBit 刚刚重启过，临时密码已经更新。再跑一次脚本试试；"
  err "或者直接用 WebUI 登录（http://nas.local:8080），在 UI 里改密码。"
  exit 1
fi
info "临时密码验证通过 ✓"

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

# ---------- 4. 登录拿 SID cookie（通过 qBit 容器内 curl 绕过 DSM 反代）----------
info "登录 qBit Web API（容器内 curl，绕开 DSM 反代）..."
COOKIE_JAR=$(mktemp)
SID_FILE="$COOKIE_JAR.sid"
trap "rm -f $COOKIE_JAR $SID_FILE" EXIT

# qBit 不返回 Set-Cookie（它的 SID 用的是请求里的特殊头），
# 实际是直接通过 cookie jar 管理 -b/-c。我们用 -c 存 cookie 到 host 文件，
# 然后 docker exec 读 host 文件，加载到容器内 curl 的 -b 参数。
# 简化做法：直接把 SID cookie 提取出来硬塞到 -H 头里。
#
# 实际上更简单——qBit 的 auth 不需要保存 SID cookie，
# 它的 setPreferences 接受任何有效的用户名密码重新登录就够了。
# 我们直接传 username + password，不依赖 cookie。

LOGIN_RESP=$($QBIT_CURL -X POST "$QBIT_URL/api/v2/auth/login" \
  --data-urlencode "username=admin" \
  --data-urlencode "password=$TEMP_PW" || echo "CURL_FAILED")

if [ "$LOGIN_RESP" != "Ok." ]; then
  err "登录失败: $LOGIN_RESP"
  exit 1
fi
info "登录成功"

# ---------- 5. 改密码（这一调用才会写 conf）----------
# qBit 5.x CSRF 流程：
#   1. login → response Set-Cookie: SID + Set-Cookie: X-XSRF-TOKEN
#   2. setPreferences → 必须带 SID cookie + X-XSRF-TOKEN 头
# 单 Referer 头不够，必须完整走 CSRF 流程。
# 所有操作都在 qBit 容器内进行（cookie 文件存在容器内 /tmp）。
info "在 qBit 容器内走完整 CSRF 流程：login → 拿 X-XSRF-TOKEN → setPreferences..."

SET_RESP=$(docker exec "$CONTAINER" sh -c "
  set -e
  # 1. login 拿 SID + X-XSRF-TOKEN cookie
  curl -sS -c /tmp/qbit.cookies -X POST '$QBIT_URL/api/v2/auth/login' \
    --data-urlencode 'username=admin' \
    --data-urlencode 'password=$NEW_PW'
  echo
  # 2. 从 Netscape 格式 cookie 文件提取 X-XSRF-TOKEN
  XSRF=\$(awk '/X-XSRF-TOKEN/ {print \$NF}' /tmp/qbit.cookies)
  if [ -z \"\$XSRF\" ]; then
    echo 'NO_XSRF_TOKEN' >&2
    exit 1
  fi
  echo \"[XSRF] \$XSRF\" >&2
  # 3. setPreferences 带 cookie + X-XSRF-TOKEN 头
  curl -sS -b /tmp/qbit.cookies \
    -H \"X-XSRF-TOKEN: \$XSRF\" \
    -H 'Referer: $QBIT_URL' \
    -X POST '$QBIT_URL/api/v2/app/setPreferences' \
    --data-urlencode 'json={\"webui_username\":\"admin\",\"webui_password\":\"$NEW_PW\"}'
" 2>&1)

# SET_RESP 现在混了 stderr + stdout，提取最后一行（curl 的响应体）
SET_BODY=$(echo "$SET_RESP" | grep -v '^\[' | tail -1)
if [ "$SET_BODY" != "Ok." ]; then
  err "改密码失败: $SET_BODY"
  echo "$SET_RESP" | sed 's/^/    /' >&2
  err ""
  err "备选方案：在 WebUI (http://nas.local:8080) 用临时密码登录、改密码、保存。"
  err "然后跑这个脚本最后一步同步 .env + 重启 api。"
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
info "用新密码验证 qBit 登录（容器内 curl）..."
VERIFY=$($QBIT_CURL -X POST "$QBIT_URL/api/v2/auth/login" \
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