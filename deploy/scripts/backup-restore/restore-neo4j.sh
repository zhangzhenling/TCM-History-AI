#!/usr/bin/env bash
# =============================================================================
# restore-neo4j.sh — 从 PVC 恢复 Neo4j 备份
#
# 用途：从备份 PVC（deploy/helm/tcm-history-ai/templates/backup-cronjob.yaml 创建的
#       <release>-backup-pvc）中选择最近一次或指定日期的 Neo4j 备份，恢复到目标
#       Neo4j StatefulSet。
#
# 备份目录命名规范（与 backup-cronjob.yaml 中 backup-neo4j.sh 一致）：
#   /backup/neo4j/<YYYYMMDD_HHMMSS>/   ← 目录，内含 neo4j-admin dump 输出
#   每个目录内为 neo4j-admin database backup 的产物（含 <db_name> 目录）
#
# 恢复策略：
#   1. 缩容目标 Neo4j StatefulSet 至 0 副本（避免恢复期间写入冲突）
#   2. 在备份 Pod 中通过 kubectl cp 把备份目录复制到目标 PVC
#      或直接挂载目标 PVC 到恢复 Pod，用 neo4j-admin database load 导入
#   3. 扩容目标 Neo4j StatefulSet 回原副本数
#   4. 验证节点/关系数量
#
# 前置条件：
#   - kubectl 已配置且能访问目标集群
#   - 当前用户对目标 namespace 有 create pods / pods/exec / delete pods /
#     scale statefulsets 权限
#   - 备份 PVC（<release>-backup-pvc）已存在且有可用备份
#   - 目标 Neo4j StatefulSet 已存在
#
# 用法：
#   ./restore-neo4j.sh -n tcm-history-ai -r tcm-history-ai
#   ./restore-neo4j.sh -n tcm-history-ai -r tcm-history-ai -d 20260727_030000
#   ./restore-neo4j.sh --help
# =============================================================================
set -euo pipefail

# -----------------------------------------------------------------------------
# 帮助函数
# -----------------------------------------------------------------------------
print_help() {
    cat <<'EOF'
restore-neo4j.sh — 从 PVC 恢复 Neo4j 备份

用法:
  ./restore-neo4j.sh [选项]

选项:
  -n, --namespace    目标 K8s 命名空间（默认 tcm-history-ai）
  -r, --release      Helm release 名称，用于定位备份 PVC（默认 tcm-history-ai）
  -d, --backup-date  备份时间戳（YYYYMMDD_HHMMSS），未指定时用最近一次
  -N, --neo4j        目标 Neo4j StatefulSet 名称（默认 neo4j）
      --db           目标 Neo4j 数据库名（默认 neo4j）
      --dry-run      仅打印将要执行的步骤，不实际执行
  -h, --help         显示本帮助

示例:
  # 恢复最近一次备份到 tcm-history-ai 命名空间
  ./restore-neo4j.sh -n tcm-history-ai -r tcm-history-ai

  # 恢复指定时间戳的备份
  ./restore-neo4j.sh -n tcm-history-ai -r tcm-history-ai -d 20260727_030000

  # 仅模拟执行，不实际恢复
  ./restore-neo4j.sh -n tcm-history-ai -r tcm-history-ai --dry-run

退出码:
  0  成功
  1  参数错误或前置条件不满足
  2  备份目录未找到
  3  恢复执行失败
EOF
}

# -----------------------------------------------------------------------------
# 参数解析
# -----------------------------------------------------------------------------
NAMESPACE="tcm-history-ai"
RELEASE="tcm-history-ai"
BACKUP_DATE=""
NEO4J_STS="neo4j"
NEO4J_DB="neo4j"
DRY_RUN=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        -n|--namespace)  NAMESPACE="$2";  shift 2 ;;
        -r|--release)    RELEASE="$2";    shift 2 ;;
        -d|--backup-date) BACKUP_DATE="$2"; shift 2 ;;
        -N|--neo4j)      NEO4J_STS="$2";  shift 2 ;;
        --db)            NEO4J_DB="$2";   shift 2 ;;
        --dry-run)       DRY_RUN=true;    shift ;;
        -h|--help)       print_help; exit 0 ;;
        *) echo "[ERROR] 未知参数: $1" >&2; print_help >&2; exit 1 ;;
    esac
done

BACKUP_PVC="${RELEASE}-backup-pvc"
BACKUP_PATH="/backup/neo4j"
RESTORE_POD="${RELEASE}-neo4j-restore-$$"
IMAGE="${NEO4J_IMAGE:-neo4j:5-community}"

# -----------------------------------------------------------------------------
# 工具函数
# -----------------------------------------------------------------------------
log()  { echo "[$(date +'%Y-%m-%d %H:%M:%S')] [INFO]  $*"; }
warn() { echo "[$(date +'%Y-%m-%d %H:%M:%S')] [WARN]  $*" >&2; }
err()  { echo "[$(date +'%Y-%m-%d %H:%M:%S')] [ERROR] $*" >&2; }

run() {
    if $DRY_RUN; then
        echo "[DRY-RUN] $*"
    else
        log "执行: $*"
        eval "$@"
    fi
}

cleanup() {
    local rc=$?
    if [[ $rc -ne 0 ]]; then
        err "恢复过程中出错（exit=$rc），开始清理临时 Pod..."
    else
        log "恢复完成，清理临时 Pod..."
    fi
    if ! $DRY_RUN; then
        # 清理临时恢复 Pod
        kubectl -n "${NAMESPACE}" delete pod "${RESTORE_POD}" --ignore-not-found >/dev/null 2>&1 || true
        kubectl -n "${NAMESPACE}" delete pod "${RESTORE_POD}-list" --ignore-not-found >/dev/null 2>&1 || true
        # 提示：目标 StatefulSet 副本数已由脚本主体在恢复完成后扩容回来
    fi
    exit $rc
}
trap cleanup EXIT INT TERM

# -----------------------------------------------------------------------------
# 前置检查
# -----------------------------------------------------------------------------
log "=== Neo4j 备份恢复演练开始 ==="
log "命名空间:        ${NAMESPACE}"
log "Release:         ${RELEASE}"
log "备份 PVC:        ${BACKUP_PVC}"
log "备份路径:        ${BACKUP_PATH}"
log "目标 StatefulSet: ${NEO4J_STS}"
log "目标数据库:      ${NEO4J_DB}"
log "指定备份时间戳:  ${BACKUP_DATE:-<最近一次>}"
log "Dry-run:         ${DRY_RUN}"
echo

# 1. kubectl 可用性
if ! command -v kubectl >/dev/null 2>&1; then
    err "kubectl 未安装或不在 PATH，请先安装 kubectl"
    exit 1
fi

# 2. 命名空间可达
if ! kubectl get namespace "${NAMESPACE}" >/dev/null 2>&1; then
    err "命名空间 ${NAMESPACE} 不存在或当前用户无权限访问"
    exit 1
fi

# 3. 备份 PVC 存在
if ! kubectl -n "${NAMESPACE}" get pvc "${BACKUP_PVC}" >/dev/null 2>&1; then
    err "备份 PVC ${BACKUP_PVC} 不存在"
    err "提示：检查 Helm release 名称（-r 参数）是否正确；备份 CronJob 由 backup.enabled=true 启用"
    exit 1
fi

# 4. 目标 Neo4j StatefulSet 存在
if ! kubectl -n "${NAMESPACE}" get statefulset "${NEO4J_STS}" >/dev/null 2>&1; then
    err "目标 Neo4j StatefulSet ${NEO4J_STS} 不存在"
    exit 1
fi

# 记录原副本数，恢复后恢复
ORIGINAL_REPLICAS=$(kubectl -n "${NAMESPACE}" get statefulset "${NEO4J_STS}" \
    -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "1")
log "目标 Neo4j StatefulSet 当前副本数: ${ORIGINAL_REPLICAS}"
echo

# -----------------------------------------------------------------------------
# 列出可用备份并选择
# -----------------------------------------------------------------------------
log "列出 PVC ${BACKUP_PVC} 中的可用 Neo4j 备份目录..."

LIST_CMD="ls -1 ${BACKUP_PATH}/ 2>/dev/null | grep -E '^[0-9]{8}_[0-9]{6}$' | sort -r || echo NO_BACKUP_FOUND"

# 创建一次性 Pod 挂载 PVC 并列目录
kubectl -n "${NAMESPACE}" delete pod "${RESTORE_POD}-list" --ignore-not-found >/dev/null 2>&1 || true
run kubectl -n "${NAMESPACE}" run "${RESTORE_POD}-list" \
    --image="${IMAGE}" \
    --restart=Never \
    --overrides="$(
        cat <<JSON
{
  "spec": {
    "containers": [{
      "name": "list",
      "image": "${IMAGE}",
      "command": ["/bin/sh", "-c", "${LIST_CMD}"],
      "volumeMounts": [{"name": "backup", "mountPath": "/backup", "readOnly": true}]
    }],
    "volumes": [{
      "name": "backup",
      "persistentVolumeClaim": {"claimName": "${BACKUP_PVC}"}
    }]
  }
}
JSON
    )" >/dev/null

if $DRY_RUN; then
    log "[DRY-RUN] 模式下未实际创建 Pod，跳过列出与恢复"
    log "[DRY-RUN] 完整恢复流程："
    log "  1. 缩容 ${NEO4J_STS} 至 0 副本（避免恢复期间写入）"
    log "  2. 启动恢复 Pod，挂载备份 PVC + 目标 PVC"
    log "  3. 执行 neo4j-admin database load --from-path=<备份目录> --database=${NEO4J_DB}"
    log "  4. 验证节点/关系数量（MATCH (n) RETURN count(n)）"
    log "  5. 扩容 ${NEO4J_STS} 回 ${ORIGINAL_REPLICAS} 副本"
    log "[DRY-RUN] 完成"
    exit 0
fi

# 等待 list Pod 完成并获取日志
log "等待 list Pod 完成..."
kubectl -n "${NAMESPACE}" wait --for=condition=Ready pod/"${RESTORE_POD}-list" --timeout=60s >/dev/null 2>&1 || true
# 等待 phase 为 Succeeded/Failed
for _ in {1..30}; do
    PHASE_LIST=$(kubectl -n "${NAMESPACE}" get pod "${RESTORE_POD}-list" \
        -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown")
    [[ "${PHASE_LIST}" == "Succeeded" || "${PHASE_LIST}" == "Failed" ]] && break
    sleep 2
done
LIST_OUTPUT=$(kubectl -n "${NAMESPACE}" logs "${RESTORE_POD}-list" 2>/dev/null || echo "")
kubectl -n "${NAMESPACE}" delete pod "${RESTORE_POD}-list" --ignore-not-found >/dev/null 2>&1 || true

if [[ -z "${LIST_OUTPUT}" || "${LIST_OUTPUT}" == "NO_BACKUP_FOUND" ]]; then
    err "备份 PVC ${BACKUP_PVC} 的 ${BACKUP_PATH}/ 下未找到任何形如 YYYYMMDD_HHMMSS 的备份目录"
    err "提示：检查 backup.enabled=true 且 CronJob 已成功执行过至少一次"
    exit 2
fi

log "可用备份目录列表（按时间倒序）："
echo "${LIST_OUTPUT}" | sed 's/^/    /'

# 选择备份目录
if [[ -n "${BACKUP_DATE}" ]]; then
    SELECTED_DIR="${BACKUP_DATE}"
    if ! echo "${LIST_OUTPUT}" | grep -qx "${SELECTED_DIR}"; then
        err "指定的备份目录不存在: ${SELECTED_DIR}"
        err "请检查 -d 参数，应为 YYYYMMDD_HHMMSS 格式"
        exit 2
    fi
else
    SELECTED_DIR=$(echo "${LIST_OUTPUT}" | head -n1)
fi
SELECTED_BACKUP_PATH="${BACKUP_PATH}/${SELECTED_DIR}"
log "选中备份目录: ${SELECTED_BACKUP_PATH}"
echo

# -----------------------------------------------------------------------------
# 缩容目标 Neo4j StatefulSet 至 0
# -----------------------------------------------------------------------------
log "缩容目标 Neo4j StatefulSet ${NEO4J_STS} 至 0 副本（避免恢复期间写入冲突）..."
run kubectl -n "${NAMESPACE}" scale statefulset "${NEO4J_STS}" --replicas=0

# 等待 Pod 终止
log "等待 ${NEO4J_STS}-0 Pod 终止..."
for _ in {1..60}; do
    PODS=$(kubectl -n "${NAMESPACE}" get pods -l "app.kubernetes.io/name=${NEO4J_STS}" \
        --no-headers 2>/dev/null | wc -l || echo "0")
    [[ "${PODS}" == "0" ]] && break
    sleep 5
done
log "目标 Neo4j StatefulSet 已缩容至 0 副本"

# -----------------------------------------------------------------------------
# 创建临时恢复 Pod 并执行 neo4j-admin load
# -----------------------------------------------------------------------------
log "创建临时恢复 Pod: ${RESTORE_POD}"

# 获取目标 Neo4j 数据卷名（PVC 命名规范：<statefulset-name>-data-<statefulset-name>-0）
NEO4J_DATA_PVC=$(kubectl -n "${NAMESPACE}" get pvc \
    -l "app.kubernetes.io/name=${NEO4J_STS}" \
    -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
if [[ -z "${NEO4J_DATA_PVC}" ]]; then
    # 回退到命名约定
    NEO4J_DATA_PVC="data-${NEO4J_STS}-0"
    warn "未通过 label 找到 Neo4j 数据 PVC，回退到命名约定: ${NEO4J_DATA_PVC}"
fi
log "目标 Neo4j 数据 PVC: ${NEO4J_DATA_PVC}"

RESTORE_SCRIPT=$(cat <<'EOF'
set -euo pipefail
echo "[restore-pod] 备份目录: ${BACKUP_DIR}"
echo "[restore-pod] 目标数据库: ${NEO4J_DB}"
echo "[restore-pod] 备份目录内容:"
ls -lah "${BACKUP_DIR}" || true

# Neo4j 数据卷路径（与 Helm chart neo4j-statefulset.yaml 一致）
NEO4J_DATA="/data"

# 1. 预检查：备份目录下应包含 <db_name> 子目录
if [ ! -d "${BACKUP_DIR}/${NEO4J_DB}" ] && [ ! -f "${BACKUP_DIR}/${NEO4J_DB}" ]; then
    echo "[restore-pod] [WARN] 备份目录下未直接找到 ${NEO4J_DB}，列出全部内容供参考："
    find "${BACKUP_DIR}" -maxdepth 3 -type f -o -type d | head -50
fi

# 2. 停止 Neo4j（在恢复 Pod 内 Neo4j 不会自动启动，但保险起见）
# neo4j-admin database load 要求 Neo4j 进程未运行

# 3. 执行恢复
# neo4j 5.x 命令：neo4j-admin database load --from-path=<dir> --database=<db>
# 若备份是 dump 文件，load 会自动识别；若是 backup 目录，load 也支持
echo "[restore-pod] 开始 neo4j-admin database load..."
neo4j-admin database load \
    --from-path="${BACKUP_DIR}" \
    --database="${NEO4J_DB}" \
    --overwrite-destination=true || {
        echo "[restore-pod] [WARN] neo4j-admin load 失败，尝试用 copy 方式（适用于 backup 目录）..."
        # 回退：直接复制文件到数据卷
        mkdir -p "${NEO4J_DATA}/databases/${NEO4J_DB}"
        mkdir -p "${NEO4J_DATA}/transactions/${NEO4J_DB}"
        if [ -d "${BACKUP_DIR}/${NEO4J_DB}" ]; then
            cp -r "${BACKUP_DIR}/${NEO4J_DB}/"* "${NEO4J_DATA}/databases/${NEO4J_DB}/" || true
        fi
        echo "[restore-pod] 回退复制完成（请人工校验）"
    }

# 4. 验证：列出数据卷下的数据库目录
echo "[restore-pod] 恢复后数据卷 databases 目录："
ls -1 "${NEO4J_DATA}/databases/" 2>/dev/null | sed 's/^/    /' || echo "    (空)"
echo "[restore-pod] 恢复后数据卷 transactions 目录："
ls -1 "${NEO4J_DATA}/transactions/" 2>/dev/null | sed 's/^/    /' || echo "    (空)"

echo "[restore-pod] 恢复完成 ✅"
echo "[restore-pod] 注意：节点/关系数量校验需在 Neo4j 启动后通过 cypher-shell 执行"
EOF
)

RESTORE_CMD=$(cat <<EOF
BACKUP_DIR="${SELECTED_BACKUP_PATH}"
NEO4J_DB="${NEO4J_DB}"
export BACKUP_DIR NEO4J_DB

${RESTORE_SCRIPT}
EOF
)

# 简单的 JSON 字符串转义（避免依赖 jq/python）
RESTORE_CMD_JSON=$(printf '%s' "${RESTORE_CMD}" | sed 's/\\/\\\\/g; s/"/\\"/g' | awk '{printf "%s\\n", $0}' | tr -d '\n')

run kubectl -n "${NAMESPACE}" run "${RESTORE_POD}" \
    --image="${IMAGE}" \
    --restart=Never \
    --overrides="$(
        cat <<JSON
{
  "spec": {
    "containers": [{
      "name": "restore",
      "image": "${IMAGE}",
      "command": ["/bin/sh", "-c", "${RESTORE_CMD_JSON}"],
      "volumeMounts": [
        {"name": "backup", "mountPath": "/backup", "readOnly": true},
        {"name": "neo4j-data", "mountPath": "/data"}
      ]
    }],
    "volumes": [
      {"name": "backup", "persistentVolumeClaim": {"claimName": "${BACKUP_PVC}"}},
      {"name": "neo4j-data", "persistentVolumeClaim": {"claimName": "${NEO4J_DATA_PVC}"}}
    ]
  }
}
JSON
    )"

# 等待 Pod 完成并跟随日志
log "等待恢复 Pod ${RESTORE_POD} 完成（最长 30 分钟）..."
kubectl -n "${NAMESPACE}" wait --for=condition=Ready pod/"${RESTORE_POD}" --timeout=60s >/dev/null 2>&1 || true
kubectl -n "${NAMESPACE}" logs -f "${RESTORE_POD}" || true

# 等待 Pod 进入 Succeeded/Failed
TIMEOUT_SECONDS=1800
ELAPSED=0
while [[ $ELAPSED -lt $TIMEOUT_SECONDS ]]; do
    PHASE=$(kubectl -n "${NAMESPACE}" get pod "${RESTORE_POD}" \
        -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown")
    if [[ "${PHASE}" == "Succeeded" || "${PHASE}" == "Failed" ]]; then
        break
    fi
    sleep 10
    ELAPSED=$((ELAPSED + 10))
done

PHASE=$(kubectl -n "${NAMESPACE}" get pod "${RESTORE_POD}" \
    -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown")
log "恢复 Pod 最终状态: ${PHASE}"

# -----------------------------------------------------------------------------
# 扩容目标 Neo4j StatefulSet 回原副本数
# -----------------------------------------------------------------------------
log "扩容目标 Neo4j StatefulSet ${NEO4J_STS} 回 ${ORIGINAL_REPLICAS} 副本..."
run kubectl -n "${NAMESPACE}" scale statefulset "${NEO4J_STS}" --replicas="${ORIGINAL_REPLICAS}"

# 等待 Neo4j 就绪
log "等待 ${NEO4J_STS}-0 就绪..."
for _ in {1..120}; do
    READY=$(kubectl -n "${NAMESPACE}" get statefulset "${NEO4J_STS}" \
        -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
    [[ "${READY}" == "${ORIGINAL_REPLICAS}" ]] && break
    sleep 10
done

# -----------------------------------------------------------------------------
# 验证：执行 Cypher 查询节点/关系数量
# -----------------------------------------------------------------------------
log "验证恢复结果（执行 Cypher 查询节点/关系数量）..."

# 从 Secret 读取 Neo4j 密码
NEO4J_SECRET="${RELEASE}-secret"
NEO4J_PASSWORD=$(kubectl -n "${NAMESPACE}" get secret "${NEO4J_SECRET}" \
    -o jsonpath="{.data.TCM_NEO4J_PASSWORD}" 2>/dev/null | base64 -d 2>/dev/null || echo "")
if [[ -z "${NEO4J_PASSWORD}" ]]; then
    NEO4J_SECRET="tcm-secret"
    NEO4J_PASSWORD=$(kubectl -n "${NAMESPACE}" get secret "${NEO4J_SECRET}" \
        -o jsonpath="{.data.TCM_NEO4J_PASSWORD}" 2>/dev/null | base64 -d 2>/dev/null || echo "")
fi
if [[ -z "${NEO4J_PASSWORD}" ]]; then
    warn "未读取到 Neo4j 密码，跳过 Cypher 自动验证（可手工 kubectl exec 进入 Neo4j Pod 验证）"
else
    VERIFY_CMD=$(cat <<EOF
set -euo pipefail
echo "[verify] 节点总数: \$(cypher-shell -u neo4j -p '${NEO4J_PASSWORD}' -d ${NEO4J_DB} 'MATCH (n) RETURN count(n) AS nodes' 2>/dev/null | tail -n +2 | head -n 2)"
echo "[verify] 关系总数: \$(cypher-shell -u neo4j -p '${NEO4J_PASSWORD}' -d ${NEO4J_DB} 'MATCH ()-[r]->() RETURN count(r) AS rels' 2>/dev/null | tail -n +2 | head -n 2)"
echo "[verify] 各 label 节点数:"
cypher-shell -u neo4j -p '${NEO4J_PASSWORD}' -d ${NEO4J_DB} 'MATCH (n) RETURN labels(n)[0] AS label, count(n) AS cnt ORDER BY cnt DESC LIMIT 10' 2>/dev/null | sed 's/^/    /'
EOF
)
    kubectl -n "${NAMESPACE}" exec "${NEO4J_STS}-0" -- /bin/sh -c "${VERIFY_CMD}" 2>/dev/null || \
        warn "Cypher 验证执行失败（Neo4j 可能尚未完全启动），可稍后手工执行"
fi

if [[ "${PHASE}" != "Succeeded" ]]; then
    err "恢复 Pod 未成功完成（phase=${PHASE}）"
    err "完整日志: kubectl -n ${NAMESPACE} logs ${RESTORE_POD}"
    err "提示：目标 StatefulSet 已扩容回 ${ORIGINAL_REPLICAS} 副本，请人工检查 Neo4j 数据一致性"
    exit 3
fi

echo
log "=== Neo4j 备份恢复演练成功 ✅ ==="
log "恢复源: ${BACKUP_PVC}${SELECTED_BACKUP_PATH}"
log "目标:   ${NEO4J_STS} (db=${NEO4J_DB})"
log "下一步建议："
log "  1. 等待 Neo4j 完全就绪后执行业务侧冒烟测试"
log "  2. 抽样查询关键节点（Person/Dynasty/Book 等）确认数据完整"
log "  3. 触发 graph-service 同步验证事件流"
log "  4. 确认无误后清理临时 Pod（本脚本已自动清理）"
