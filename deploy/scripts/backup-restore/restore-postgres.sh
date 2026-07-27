#!/usr/bin/env bash
# =============================================================================
# restore-postgres.sh — 从 PVC 恢复 PostgreSQL 备份
#
# 用途：从备份 PVC（deploy/helm/tcm-history-ai/templates/backup-cronjob.yaml 创建的
#       <release>-backup-pvc）中选择最近一次或指定日期的 PostgreSQL 备份，恢复到目标
#       PostgreSQL StatefulSet。
#
# 备份文件命名规范（与 backup-cronjob.yaml 中 backup-postgres.sh 一致）：
#   /backup/postgres/backup_YYYYMMDD_HHMMSS.sql.gz
#
# 恢复策略：
#   1. 创建临时恢复 Pod（挂载备份 PVC + 目标 PG 数据卷）
#   2. 解压 gzip → 通过 pg_restore / psql 导入目标库
#   3. 验证记录数（与备份时对比）
#   4. 清理临时 Pod
#
# 前置条件：
#   - kubectl 已配置且能访问目标集群
#   - 当前用户对目标 namespace 有 create pods / pods/exec / delete pods 权限
#   - 备份 PVC（<release>-backup-pvc）已存在且有可用备份
#   - 目标 PostgreSQL StatefulSet 已就绪
#
# 用法：
#   ./restore-postgres.sh -n tcm-history-ai -r tcm-history-ai
#   ./restore-postgres.sh -n tcm-history-ai -r tcm-history-ai -d 20260727_020000
#   ./restore-postgres.sh --help
# =============================================================================
set -euo pipefail

# -----------------------------------------------------------------------------
# 帮助函数
# -----------------------------------------------------------------------------
print_help() {
    cat <<'EOF'
restore-postgres.sh — 从 PVC 恢复 PostgreSQL 备份

用法:
  ./restore-postgres.sh [选项]

选项:
  -n, --namespace    目标 K8s 命名空间（默认 tcm-history-ai）
  -r, --release      Helm release 名称，用于定位备份 PVC（默认 tcm-history-ai）
  -d, --backup-date  备份时间戳（YYYYMMDD_HHMMSS），未指定时用最近一次
  -p, --postgres     目标 PostgreSQL StatefulSet 名称（默认 postgres）
  -D, --database     目标数据库名（默认从 ConfigMap tcm-config 的 TCM_DB_DBNAME 读取）
  -U, --user         目标数据库用户（默认从 ConfigMap tcm-config 的 TCM_DB_USER 读取）
      --dry-run      仅打印将要执行的步骤，不实际执行
  -h, --help         显示本帮助

示例:
  # 恢复最近一次备份到 tcm-history-ai 命名空间
  ./restore-postgres.sh -n tcm-history-ai -r tcm-history-ai

  # 恢复指定时间戳的备份
  ./restore-postgres.sh -n tcm-history-ai -r tcm-history-ai -d 20260727_020000

  # 仅模拟执行，不实际恢复
  ./restore-postgres.sh -n tcm-history-ai -r tcm-history-ai --dry-run

退出码:
  0  成功
  1  参数错误或前置条件不满足
  2  备份文件未找到
  3  恢复执行失败
EOF
}

# -----------------------------------------------------------------------------
# 参数解析
# -----------------------------------------------------------------------------
NAMESPACE="tcm-history-ai"
RELEASE="tcm-history-ai"
BACKUP_DATE=""
POSTGRES_STS="postgres"
DATABASE=""
USER=""
DRY_RUN=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        -n|--namespace) NAMESPACE="$2"; shift 2 ;;
        -r|--release)   RELEASE="$2";   shift 2 ;;
        -d|--backup-date) BACKUP_DATE="$2"; shift 2 ;;
        -p|--postgres)  POSTGRES_STS="$2"; shift 2 ;;
        -D|--database)  DATABASE="$2";  shift 2 ;;
        -U|--user)      USER="$2";      shift 2 ;;
        --dry-run)      DRY_RUN=true;   shift ;;
        -h|--help)      print_help; exit 0 ;;
        *) echo "[ERROR] 未知参数: $1" >&2; print_help >&2; exit 1 ;;
    esac
done

BACKUP_PVC="${RELEASE}-backup-pvc"
BACKUP_PATH="/backup/postgres"
RESTORE_POD="${RELEASE}-pg-restore-$$"
IMAGE="${POSTGRES_IMAGE:-postgres:16-alpine}"

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
        kubectl -n "${NAMESPACE}" delete pod "${RESTORE_POD}" --ignore-not-found >/dev/null 2>&1 || true
    fi
    exit $rc
}
trap cleanup EXIT INT TERM

# -----------------------------------------------------------------------------
# 前置检查
# -----------------------------------------------------------------------------
log "=== PostgreSQL 备份恢复演练开始 ==="
log "命名空间:        ${NAMESPACE}"
log "Release:         ${RELEASE}"
log "备份 PVC:        ${BACKUP_PVC}"
log "备份路径:        ${BACKUP_PATH}"
log "目标 StatefulSet: ${POSTGRES_STS}"
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

# 4. 目标 PostgreSQL StatefulSet 存在且就绪
if ! kubectl -n "${NAMESPACE}" get statefulset "${POSTGRES_STS}" >/dev/null 2>&1; then
    err "目标 PostgreSQL StatefulSet ${POSTGRES_STS} 不存在"
    exit 1
fi
READY_REPLICAS=$(kubectl -n "${NAMESPACE}" get statefulset "${POSTGRES_STS}" \
    -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
if [[ "${READY_REPLICAS}" == "0" || -z "${READY_REPLICAS}" ]]; then
    err "目标 PostgreSQL StatefulSet ${POSTGRES_STS} 没有就绪的副本（readyReplicas=${READY_REPLICAS}）"
    err "提示：先 kubectl -n ${NAMESPACE} rollout status statefulset/${POSTGRES_STS}"
    exit 1
fi
log "目标 PostgreSQL StatefulSet 就绪（readyReplicas=${READY_REPLICAS}）"

# 5. 读取 DB 名称与用户（若未通过参数指定）
if [[ -z "${DATABASE}" ]]; then
    DATABASE=$(kubectl -n "${NAMESPACE}" get configmap "${RELEASE}-config" \
        -o jsonpath='{.data.TCM_DB_DBNAME}' 2>/dev/null || echo "")
    if [[ -z "${DATABASE}" ]]; then
        DATABASE=$(kubectl -n "${NAMESPACE}" get configmap tcm-config \
            -o jsonpath='{.data.TCM_DB_DBNAME}' 2>/dev/null || echo "tcm_history")
        warn "未从 ${RELEASE}-config 读取到 TCM_DB_DBNAME，回退到 tcm-config / 默认值: ${DATABASE}"
    fi
fi
if [[ -z "${USER}" ]]; then
    USER=$(kubectl -n "${NAMESPACE}" get configmap "${RELEASE}-config" \
        -o jsonpath='{.data.TCM_DB_USER}' 2>/dev/null || echo "")
    if [[ -z "${USER}" ]]; then
        USER=$(kubectl -n "${NAMESPACE}" get configmap tcm-config \
            -o jsonpath='{.data.TCM_DB_USER}' 2>/dev/null || echo "tcm")
        warn "未从 ${RELEASE}-config 读取到 TCM_DB_USER，回退到 tcm-config / 默认值: ${USER}"
    fi
fi
log "目标数据库: ${DATABASE}"
log "目标用户:   ${USER}"
echo

# -----------------------------------------------------------------------------
# 列出可用备份并选择
# -----------------------------------------------------------------------------
log "列出 PVC ${BACKUP_PVC} 中的可用 PostgreSQL 备份..."

LIST_CMD=$(cat <<'EOF'
set -e
ls -1 /backup/postgres/backup_*.sql.gz 2>/dev/null | sort -r || echo "NO_BACKUP_FOUND"
EOF
)

LIST_OUTPUT=$(kubectl -n "${NAMESPACE}" run "${RESTORE_POD}-list" \
    --image="${IMAGE}" \
    --restart=Never \
    --rm \
    -i \
    --overrides="$(
        cat <<JSON
{
  "spec": {
    "containers": [{
      "name": "list",
      "image": "${IMAGE}",
      "command": ["/bin/sh"],
      "args": ["-c", $(echo "${LIST_CMD}" | jq -Rs .)],
      "volumeMounts": [{"name": "backup", "mountPath": "/backup"}]
    }],
    "volumes": [{
      "name": "backup",
      "persistentVolumeClaim": {"claimName": "${BACKUP_PVC}"}
    }]
  }
}
JSON
    )" 2>/dev/null) || true

# 上面 jq 可能不可用，回退到更简单的临时 Pod 创建方式
if [[ -z "${LIST_OUTPUT}" || "${LIST_OUTPUT}" == "NO_BACKUP_FOUND" ]]; then
    log "尝试用 ConfigMap 注入脚本方式列出备份..."
    # 创建一次性 Pod 挂载 PVC 并列文件
    kubectl -n "${NAMESPACE}" delete pod "${RESTORE_POD}-list" --ignore-not-found >/dev/null 2>&1 || true
    kubectl -n "${NAMESPACE}" run "${RESTORE_POD}-list" \
        --image="${IMAGE}" \
        --restart=Never \
        --overrides="$(
            cat <<JSON
{
  "spec": {
    "containers": [{
      "name": "list",
      "image": "${IMAGE}",
      "command": ["/bin/sh", "-c", "ls -1 ${BACKUP_PATH}/backup_*.sql.gz 2>/dev/null | sort -r || echo NO_BACKUP_FOUND"],
      "volumeMounts": [{"name": "backup", "mountPath": "/backup"}]
    }],
    "volumes": [{
      "name": "backup",
      "persistentVolumeClaim": {"claimName": "${BACKUP_PVC}"}
    }]
  }
}
JSON
        )" >/dev/null

    # 等待 Pod 完成并获取日志
    log "等待 list Pod 完成..."
    kubectl -n "${NAMESPACE}" wait --for=condition=Ready pod/"${RESTORE_POD}-list" --timeout=60s >/dev/null 2>&1 || true
    kubectl -n "${NAMESPACE}" wait --for=jsonpath='{.status.phase}=Succeeded}' pod/"${RESTORE_POD}-list" --timeout=120s >/dev/null 2>&1 || true
    LIST_OUTPUT=$(kubectl -n "${NAMESPACE}" logs "${RESTORE_POD}-list" 2>/dev/null || echo "")
    kubectl -n "${NAMESPACE}" delete pod "${RESTORE_POD}-list" --ignore-not-found >/dev/null 2>&1 || true
fi

if [[ -z "${LIST_OUTPUT}" || "${LIST_OUTPUT}" == "NO_BACKUP_FOUND" ]]; then
    err "备份 PVC ${BACKUP_PVC} 的 ${BACKUP_PATH}/ 下未找到任何 backup_*.sql.gz 文件"
    err "提示：检查 backup.enabled=true 且 CronJob 已成功执行过至少一次"
    exit 2
fi

log "可用备份列表（按时间倒序）："
echo "${LIST_OUTPUT}" | sed 's/^/    /'

# 选择备份
if [[ -n "${BACKUP_DATE}" ]]; then
    SELECTED_BACKUP="${BACKUP_PATH}/backup_${BACKUP_DATE}.sql.gz"
    if ! echo "${LIST_OUTPUT}" | grep -q "${SELECTED_BACKUP}$\|${SELECTED_BACKUP##*/}"; then
        err "指定的备份文件不存在: ${SELECTED_BACKUP}"
        err "请检查 -d 参数，应为 YYYYMMDD_HHMMSS 格式"
        exit 2
    fi
else
    SELECTED_BACKUP=$(echo "${LIST_OUTPUT}" | head -n1)
fi
log "选中备份: ${SELECTED_BACKUP}"
echo

# -----------------------------------------------------------------------------
# 创建临时恢复 Pod 并执行 pg_restore
# -----------------------------------------------------------------------------
log "创建临时恢复 Pod: ${RESTORE_POD}"

# 获取 PG Service 名（默认与 StatefulSet 同名）
PG_SERVICE="${POSTGRES_STS}"

# 从 Secret 读取 PG 密码
PG_SECRET="${RELEASE}-secret"
PG_PASSWORD=$(kubectl -n "${NAMESPACE}" get secret "${PG_SECRET}" \
    -o jsonpath="{.data.TCM_DB_PASSWORD}" 2>/dev/null | base64 -d 2>/dev/null || echo "")
if [[ -z "${PG_PASSWORD}" ]]; then
    warn "未从 Secret ${PG_SECRET} 读取到 TCM_DB_PASSWORD，尝试 tcm-secret..."
    PG_SECRET="tcm-secret"
    PG_PASSWORD=$(kubectl -n "${NAMESPACE}" get secret "${PG_SECRET}" \
        -o jsonpath="{.data.TCM_DB_PASSWORD}" 2>/dev/null | base64 -d 2>/dev/null || echo "")
fi
if [[ -z "${PG_PASSWORD}" ]]; then
    err "无法读取 PostgreSQL 密码（Secret: ${PG_SECRET}，key: TCM_DB_PASSWORD）"
    err "提示：检查 helm release 名称（-r 参数）或确认 Secret 已创建"
    exit 1
fi

RESTORE_SCRIPT=$(cat <<'EOF'
set -euo pipefail
echo "[restore-pod] 备份文件: ${BACKUP_FILE}"
echo "[restore-pod] 文件大小: $(ls -lh ${BACKUP_FILE} | awk '{print $5}')"
echo "[restore-pod] 目标库:   ${PGDATABASE} @ ${PGHOST}:${PGPORT}"

# 1. 解压并预检查
echo "[restore-pod] 预检查备份内容（前 5 行 SQL）..."
zcat "${BACKUP_FILE}" | head -n5

# 2. 恢复前记录目标库现有表数量
BEFORE_TABLES=$(psql -tAc "SELECT count(*) FROM information_schema.tables WHERE table_schema='public'" 2>/dev/null || echo "0")
echo "[restore-pod] 恢复前 public schema 表数量: ${BEFORE_TABLES}"

# 3. 执行恢复（gzip 解压 → psql 导入）
#    注意：本脚本采用「直接覆盖导入」策略，适合空库或可重置的恢复演练场景
#    生产正式恢复前建议：
#      a. 先 dump 当前库做二次备份（防误恢复）
#      b. 评估是否需要 DROP DATABASE 再 CREATE（避免冲突）
echo "[restore-pod] 开始恢复（zcat | psql）..."
zcat "${BACKUP_FILE}" | psql -v ON_ERROR_STOP=1 -q

# 4. 验证恢复结果
AFTER_TABLES=$(psql -tAc "SELECT count(*) FROM information_schema.tables WHERE table_schema='public'")
echo "[restore-pod] 恢复后 public schema 表数量: ${AFTER_TABLES}"

# 抽样验证：列出前 10 张表与各自行数
echo "[restore-pod] 抽样验证（前 10 张表的行数）："
psql -tAc "SELECT relname, n_live_tup FROM pg_stat_user_tables ORDER BY n_live_tup DESC LIMIT 10" | sed 's/^/    /'

echo "[restore-pod] 恢复完成 ✅"
EOF
)

RESTORE_CMD=$(cat <<EOF
BACKUP_FILE="${SELECTED_BACKUP}"
PGHOST="${PG_SERVICE}"
PGPORT="5432"
PGUSER="${USER}"
PGDATABASE="${DATABASE}"
PGPASSWORD="${PG_PASSWORD}"
export PGHOST PGPORT PGUSER PGDATABASE PGPASSWORD

${RESTORE_SCRIPT}
EOF
)

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
      "command": ["/bin/sh", "-c", $(printf '%s' "${RESTORE_CMD}" | python3 -c 'import json,sys; print(json.dumps(sys.stdin.read()))' 2>/dev/null || printf '%s' "${RESTORE_CMD}" | sed 's/"/\\"/g' | awk '{printf "%s\\n", $0}' | tr -d "\n" | sed 's/^/"/;s/$/"/')],
      "volumeMounts": [{"name": "backup", "mountPath": "/backup", "readOnly": true}]
    }],
    "volumes": [{
      "name": "backup",
      "persistentVolumeClaim": {"claimName": "${BACKUP_PVC}"}
    }]
  }
}
JSON
    )"

if $DRY_RUN; then
    log "[DRY-RUN] 模式下未实际创建 Pod，恢复脚本预览如下："
    echo "${RESTORE_CMD}" | sed 's/^/    /'
    log "[DRY-RUN] 完成"
    exit 0
fi

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

if [[ "${PHASE}" != "Succeeded" ]]; then
    err "恢复 Pod 未成功完成（phase=${PHASE}）"
    err "完整日志: kubectl -n ${NAMESPACE} logs ${RESTORE_POD}"
    exit 3
fi

echo
log "=== PostgreSQL 备份恢复演练成功 ✅ ==="
log "恢复源: ${BACKUP_PVC}${SELECTED_BACKUP}"
log "目标库: ${PG_SERVICE}:${PGPORT}/${DATABASE}"
log "下一步建议："
log "  1. 业务侧冒烟测试（API 健康检查、关键路由联调）"
log "  2. 抽样数据校验（行数对比、关键字段抽样查询）"
log "  3. 确认无误后清理临时 Pod（本脚本已自动清理）"
