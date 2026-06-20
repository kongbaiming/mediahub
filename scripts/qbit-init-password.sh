#!/command/with-contenv bash
# qBittorrent 启动前从环境变量写入 WebUI 密码（挂载到 /config/custom-cont-init.d/）
#
# linuxserver/qbittorrent 不会读 WEBUI_PASS 环境变量；若 conf 里没有
# WebUI\Password_PBKDF2，每次重启都会生成临时密码。
# 本脚本在 qBit 进程启动前把 .env 中的 QBIT_PASSWORD 写入 qBittorrent.conf。
#
# docker-compose.yml 需传入：
#   QBIT_PASSWORD / QBIT_USER
# 并挂载：
#   ./scripts/qbit-init-password.sh:/config/custom-cont-init.d/99-set-password:ro

set -euo pipefail

CONF="/config/qBittorrent/qBittorrent.conf"
USER="${QBIT_USER:-admin}"
PASS="${QBIT_PASSWORD:-}"

if [ -z "$PASS" ]; then
  echo "[qbit-init] QBIT_PASSWORD 未设置，跳过（将使用 qBit 临时密码，见 docker logs）"
  exit 0
fi

# init-qbittorrent-config 应先于 custom-cont-init.d 创建默认 conf
for i in $(seq 1 30); do
  if [ -f "$CONF" ]; then
    break
  fi
  sleep 1
done
if [ ! -f "$CONF" ]; then
  echo "[qbit-init] 未找到 $CONF，跳过"
  exit 0
fi

# 与 qBittorrent Password::PBKDF2::generate 一致：SHA512 / 100000 轮 / 16 字节 salt / 64 字节 hash
gen_pbkdf2() {
  local password="$1"
  local salt_file hash_file salt_hex salt_b64 hash_b64
  salt_file=$(mktemp)
  hash_file=$(mktemp)
  trap 'rm -f "$salt_file" "$hash_file"' RETURN

  openssl rand 16 >"$salt_file"
  salt_b64=$(base64 <"$salt_file" | tr -d '\n')
  salt_hex=$(od -An -tx1 "$salt_file" | tr -d ' \n')
  openssl kdf -keylen 64 -iter 100000 -md sha512 \
    -salt "hex:${salt_hex}" -pass "pass:${password}" >"$hash_file"
  hash_b64=$(base64 <"$hash_file" | tr -d '\n')

  printf '@ByteArray(%s:%s)' "$salt_b64" "$hash_b64"
}

PBKDF2=$(gen_pbkdf2 "$PASS")

set_kv() {
  local key="$1"
  local value="$2"
  local escaped
  escaped=$(printf '%s' "$value" | sed 's/[\\&|]/\\&/g')
  if grep -q "^${key}=" "$CONF"; then
    sed -i "s|^${key}=.*|${key}=${escaped}|" "$CONF"
  else
    printf '%s=%s\n' "$key" "$value" >>"$CONF"
  fi
}

set_kv 'WebUI\\Username' "$USER"
set_kv 'WebUI\\Password_PBKDF2' "\"${PBKDF2}\""

# 清除旧版明文/sha256 密码项，避免冲突
sed -i '/^WebUI\\Password_ha256=/d' "$CONF" 2>/dev/null || true
sed -i '/^WebUI\\Password=$/d' "$CONF" 2>/dev/null || true

# 清除因 API 密码错误累积的 IP 封禁
if grep -q '^WebUI\\BannedIPs=' "$CONF" 2>/dev/null; then
  sed -i 's|^WebUI\\BannedIPs=.*|WebUI\\BannedIPs=|' "$CONF"
  echo "[qbit-init] 已清除 WebUI BannedIPs"
fi

echo "[qbit-init] 已从环境变量设置 WebUI 用户: ${USER}"
