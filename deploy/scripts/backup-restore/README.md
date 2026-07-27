# 备份恢复演练手册

本目录提供 TCM-History-AI 平台 PostgreSQL 与 Neo4j 数据库的恢复脚本与演练流程，
与 `deploy/helm/tcm-history-ai/templates/backup-cronjob.yaml` 中定义的备份 CronJob 配套使用。

> 关联文档：[doc/16-Kubernetes部署.md](../../../doc/16-Kubernetes部署.md) 第 5 节「备份与恢复」

---

## 1. 备份存储位置

### 1.1 备份 PVC

| 项 | 值 |
| -- | -- |
| PVC 名称 | `<release>-backup-pvc`（Helm release 名默认 `tcm-history-ai`，全名 `tcm-history-ai-backup-pvc`） |
| StorageClass | `global.storageClass`（dev: `standard`，prod: `fast-ssd`） |
| 容量 | `backup.storage`（dev: `100Gi`，prod: `200Gi`） |
| 访问模式 | `ReadWriteOnce` |
| 创建者 | `deploy/helm/tcm-history-ai/templates/backup-cronjob.yaml` 末尾的 `PersistentVolumeClaim` 段 |

### 1.2 备份目录结构

PVC 内的目录结构如下（由 `backup-cronjob.yaml` 中的 `backup-postgres.sh` / `backup-neo4j.sh` 写入）：

```
/backup/
├── postgres/
│   ├── backup_20260727_020000.sql.gz     # pg_dump | gzip，命名 YYYYMMDD_HHMMSS
│   ├── backup_20260728_020000.sql.gz
│   └── ...
└── neo4j/
    ├── 20260727_030000/                  # neo4j-admin database backup 输出目录
    │   └── neo4j/                        # <db_name> 子目录
    │       ├── ...
    ├── 20260728_030000/
    └── ...
```

### 1.3 备份调度

| 数据库 | CronJob 名称 | Schedule（prod） | 说明 |
| ------ | ------------ | ---------------- | ---- |
| PostgreSQL | `<release>-backup-postgres` | `0 2 * * *`（每日 02:00） | 全库 pg_dump + gzip |
| Neo4j | `<release>-backup-neo4j` | `0 3 * * *`（每日 03:00） | neo4j-admin database backup 全量 |

保留策略：`backup.retentionDays`（dev: 7 天，prod: 14 天），由备份脚本内 `find ... -mtime +N -delete` 自动清理。

---

## 2. 恢复前置条件

执行恢复前必须确认：

- [ ] **集群可达**：`kubectl cluster-info` 正常返回
- [ ] **kubectl 权限**：当前用户在目标 namespace 有以下权限：
  - `create pods` / `delete pods` / `get pods` / `pods/exec`
  - `get statefulsets` / `scale statefulsets`（Neo4j 恢复需要）
  - `get pvc` / `get configmap` / `get secret`
  - 如使用最小权限 RBAC，建议绑定到 `<release>-backup` Role（见 `backup-cronjob.yaml` 中的 Role 定义，可按需扩展）
- [ ] **备份 PVC 可读**：`kubectl -n <ns> get pvc <release>-backup-pvc` 返回 Bound 状态
- [ ] **目标 PVC 可写**：
  - PostgreSQL：目标 StatefulSet 的 `data` PVC（`data-<sts>-0`）可被恢复 Pod 挂载
  - Neo4j：目标 StatefulSet 的 `data` PVC（同上）可被恢复 Pod 挂载（恢复期间需缩容至 0 副本释放 PVC）
- [ ] **目标数据库可达**：
  - PostgreSQL StatefulSet 已就绪（`readyReplicas ≥ 1`），用于恢复后验证
  - Neo4j StatefulSet 已就绪（恢复前会自动缩容，恢复后扩容回来）
- [ ] **维护窗口**：恢复期间业务会有数据中断（Neo4j 需缩容，PostgreSQL 直接覆盖导入），建议在低峰期执行

---

## 3. PostgreSQL 恢复演练

### 3.1 恢复脚本

脚本路径：`deploy/scripts/backup-restore/restore-postgres.sh`

### 3.2 完整演练步骤

```bash
cd deploy/scripts/backup-restore/

# 1. 查看帮助
./restore-postgres.sh --help

# 2. Dry-run 预演（不实际修改数据，强烈推荐首次执行）
./restore-postgres.sh \
  -n tcm-history-ai \
  -r tcm-history-ai \
  --dry-run

# 3. 列出可用备份（脚本会自动列出，无需单独命令；输出形如）：
#    /backup/postgres/backup_20260727_020000.sql.gz
#    /backup/postgres/backup_20260728_020000.sql.gz

# 4. 恢复最近一次备份
./restore-postgres.sh \
  -n tcm-history-ai \
  -r tcm-history-ai

# 5. 恢复指定时间戳的备份（YYYYMMDD_HHMMSS）
./restore-postgres.sh \
  -n tcm-history-ai \
  -r tcm-history-ai \
  -d 20260727_020000

# 6. 指定目标数据库与用户（覆盖默认从 ConfigMap 读取的值）
./restore-postgres.sh \
  -n tcm-history-ai \
  -r tcm-history-ai \
  -D tcm_history \
  -U tcm
```

### 3.3 恢复流程详解

脚本内部执行步骤（无需手动干预）：

1. **前置检查**：kubectl 可用 → namespace 存在 → 备份 PVC 存在 → 目标 PG StatefulSet 就绪 → 读取 DB 名/用户/密码
2. **列出备份**：启动一次性 Pod 挂载备份 PVC，`ls -1 /backup/postgres/backup_*.sql.gz | sort -r`
3. **选择备份**：按 `-d` 指定时间戳，否则取最近一次
4. **创建恢复 Pod**：挂载备份 PVC（只读），通过 `kubectl run --overrides` 注入恢复脚本
5. **执行恢复**：Pod 内 `zcat <backup>.sql.gz | psql -v ON_ERROR_STOP=1`
6. **验证**：
   - 恢复前后 public schema 表数量对比
   - 抽样查询前 10 张表的行数（`pg_stat_user_tables`）
7. **清理**：trap EXIT 自动删除临时 Pod

### 3.4 手动验证命令

恢复完成后，建议手动执行以下验证：

```bash
# 进入目标 PostgreSQL Pod
kubectl -n tcm-history-ai exec -it postgres-0 -- psql -U tcm -d tcm_history

# 查看所有表
\dt

# 抽样查询关键表行数
SELECT 'history_dynasty' AS tbl, count(*) FROM history_dynasty
UNION ALL SELECT 'history_person', count(*) FROM history_person
UNION ALL SELECT 'history_book', count(*) FROM history_book
UNION ALL SELECT 'users', count(*) FROM users
UNION ALL SELECT 'learning_courses', count(*) FROM learning_courses;

# 退出
\q
```

### 3.5 业务侧冒烟测试

```bash
# 通过 Gateway 暴露的端口访问
kubectl -n tcm-history-ai port-forward svc/gateway 8080:8080

# 健康检查
curl http://127.0.0.1:8080/health

# 关键路由冒烟（需先登录获取 token）
curl -X POST http://127.0.0.1:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"<test-password>"}'

# 查询历史人物
curl http://127.0.0.1:8080/api/v1/history/persons?page=1&page_size=10 \
  -H 'Authorization: Bearer <token>'
```

---

## 4. Neo4j 恢复演练

### 4.1 恢复脚本

脚本路径：`deploy/scripts/backup-restore/restore-neo4j.sh`

### 4.2 完整演练步骤

```bash
cd deploy/scripts/backup-restore/

# 1. 查看帮助
./restore-neo4j.sh --help

# 2. Dry-run 预演
./restore-neo4j.sh \
  -n tcm-history-ai \
  -r tcm-history-ai \
  --dry-run

# 3. 恢复最近一次备份
./restore-neo4j.sh \
  -n tcm-history-ai \
  -r tcm-history-ai

# 4. 恢复指定时间戳的备份
./restore-neo4j.sh \
  -n tcm-history-ai \
  -r tcm-history-ai \
  -d 20260727_030000

# 5. 指定目标数据库（默认 neo4j）
./restore-neo4j.sh \
  -n tcm-history-ai \
  -r tcm-history-ai \
  --db neo4j
```

### 4.3 恢复流程详解

脚本内部执行步骤：

1. **前置检查**：kubectl 可用 → namespace 存在 → 备份 PVC 存在 → 目标 Neo4j StatefulSet 存在
2. **列出备份目录**：`ls -1 /backup/neo4j/ | grep -E '^[0-9]{8}_[0-9]{6}$' | sort -r`
3. **选择备份**：按 `-d` 指定时间戳，否则取最近一次
4. **缩容目标**：`kubectl scale statefulset neo4j --replicas=0`，等待 Pod 完全终止（释放数据 PVC）
5. **创建恢复 Pod**：同时挂载备份 PVC（只读）+ 目标 Neo4j 数据 PVC（读写）
6. **执行恢复**：Pod 内 `neo4j-admin database load --from-path=<备份目录> --database=<db> --overwrite-destination=true`
   - 失败回退：直接 `cp -r` 复制备份目录到数据卷（适用于 backup 目录格式）
7. **扩容目标**：`kubectl scale statefulset neo4j --replicas=<原副本数>`，等待就绪
8. **Cypher 验证**：
   - 节点总数：`MATCH (n) RETURN count(n)`
   - 关系总数：`MATCH ()-[r]->() RETURN count(r)`
   - 各 label 节点数：`MATCH (n) RETURN labels(n)[0], count(n)`
9. **清理**：trap EXIT 自动删除临时 Pod

### 4.4 手动验证命令

```bash
# 进入目标 Neo4j Pod
kubectl -n tcm-history-ai exec -it neo4j-0 -- cypher-shell -u neo4j -p <password> -d neo4j

# 查询节点总数
MATCH (n) RETURN count(n) AS nodes;

# 查询关系总数
MATCH ()-[r]->() RETURN count(r) AS rels;

# 各 label 节点数
MATCH (n)
RETURN labels(n)[0] AS label, count(n) AS cnt
ORDER BY cnt DESC LIMIT 10;

# 抽样查询关键节点（Person）
MATCH (p:Person) RETURN p.name, p.dynasty LIMIT 10;

# 退出
:exit
```

### 4.5 业务侧冒烟测试

```bash
# Graph Service 健康检查
kubectl -n tcm-history-ai port-forward svc/graph-service 8085:8085
curl http://127.0.0.1:8085/health

# 节点查询
curl 'http://127.0.0.1:8085/api/v1/graph/nodes?label=Person&page=1&page_size=10'

# 子图查询
curl 'http://127.0.0.1:8085/api/v1/graph/query?node_id=<some-node-id>&depth=2'
```

---

## 5. 故障排查常见问题

### 5.1 备份 PVC 不存在

**现象**：脚本报错 `备份 PVC <release>-backup-pvc 不存在`

**排查**：
```bash
# 1. 确认 Helm release 名称
helm list -n tcm-history-ai

# 2. 检查 backup.enabled 是否为 true（生产默认 true，dev 默认 false）
helm get values <release> -n tcm-history-ai | grep -A2 backup

# 3. 手动检查 PVC
kubectl -n tcm-history-ai get pvc
```

**解决**：
```bash
# 启用备份（如未启用）
helm upgrade <release> deploy/helm/tcm-history-ai \
  -f deploy/helm/tcm-history-ai/values.prod.yaml \
  --set backup.enabled=true
```

### 5.2 备份目录为空

**现象**：脚本报错 `未找到任何 backup_*.sql.gz 文件` 或 `未找到任何 YYYYMMDD_HHMMSS 的备份目录`

**排查**：
```bash
# 1. 查看 CronJob 是否执行过
kubectl -n tcm-history-ai get jobs -l app.kubernetes.io/component=backup

# 2. 查看 CronJob 最近一次 Job 日志
kubectl -n tcm-history-ai logs job/<latest-backup-job>

# 3. 手动触发一次备份
kubectl -n tcm-history-ai create job --from=cronjob/<release>-backup-postgres manual-backup-pg
kubectl -n tcm-history-ai wait --for=condition=complete job/manual-backup-pg --timeout=600s
```

### 5.3 恢复 Pod 卡在 ContainerCreating

**现象**：`kubectl get pod <restore-pod>` 长时间处于 `ContainerCreating`

**排查**：
```bash
# 查看事件
kubectl -n tcm-history-ai describe pod <restore-pod>

# 常见原因：
# - 备份 PVC 与目标 PVC 在不同节点，无法同时挂载（RWO 模式）
# - 镜像拉取失败（networkpolicy / imagePullSecret）
# - 资源不足
```

**解决**：
- 对于 Neo4j 恢复，确保先缩容目标 StatefulSet 至 0 副本（脚本会自动执行）
- 对于 PostgreSQL 恢复，备份 PVC 是 RWO，恢复 Pod 会调度到与备份 PVC 相同的节点；若目标 PG 在不同节点，恢复 Pod 仍可访问 PG Service（ClusterIP）
- 手动指定调度节点：`--overrides` 中加 `nodeSelector`

### 5.4 psql 恢复报错

**现象**：恢复 Pod 日志中 `psql: error: FATAL: password authentication failed`

**排查**：
```bash
# 检查 Secret 中的密码是否与目标 PG 一致
kubectl -n tcm-history-ai get secret <release>-secret -o jsonpath='{.data.TCM_DB_PASSWORD}' | base64 -d

# 进入 PG Pod 验证
kubectl -n tcm-history-ai exec -it postgres-0 -- psql -U tcm -d tcm_history -c '\l'
```

**解决**：通过 `-U` / `-D` 参数显式指定用户与数据库，或修复 Secret。

### 5.5 neo4j-admin load 报错

**现象**：`neo4j-admin database load` 报 `Failed to load database` 或 `Database already exists`

**排查**：
- 确保目标 Neo4j 已缩容至 0 副本（脚本自动执行，但若失败需手动 `kubectl scale statefulset neo4j --replicas=0`）
- 确保备份目录格式与 `neo4j-admin load` 期望一致（5.x 备份为目录，含 `<db_name>/` 子目录）
- 检查数据卷权限：Neo4j 容器以 `fsGroup: 7474` 运行，恢复 Pod 需要相同 fsGroup（脚本未显式设置，必要时在 `--overrides` 中加 `spec.securityContext.fsGroup: 7474`）

**回退方案**：脚本已内置 `cp -r` 回退逻辑；若仍失败，可手动操作：
```bash
# 1. 缩容
kubectl -n tcm-history-ai scale statefulset neo4j --replicas=0

# 2. 启动一个调试 Pod 挂载两个 PVC
kubectl -n tcm-history-ai run debug --image=busybox --restart=Never \
  --overrides='{
    "spec": {
      "containers": [{
        "name": "debug", "image": "busybox",
        "command": ["sh"],
        "stdin": true, "tty": true,
        "volumeMounts": [
          {"name": "backup", "mountPath": "/backup", "readOnly": true},
          {"name": "data", "mountPath": "/data"}
        ]
      }],
      "volumes": [
        {"name": "backup", "persistentVolumeClaim": {"claimName": "<release>-backup-pvc"}},
        {"name": "data", "persistentVolumeClaim": {"claimName": "data-neo4j-0"}}
      ]
    }
  }'

# 3. 进入调试 Pod 手动复制
kubectl -n tcm-history-ai attach debug

# 4. 清理
kubectl -n tcm-history-ai delete pod debug
kubectl -n tcm-history-ai scale statefulset neo4j --replicas=1
```

### 5.6 恢复后业务异常

**现象**：恢复成功但业务接口返回错误数据

**可能原因**：
- 恢复的是较旧备份，业务期望较新数据（检查备份时间戳）
- 恢复后未重启业务 Pod（缓存陈旧）

**解决**：
```bash
# 重启所有后端微服务 Pod（强制重新加载配置与缓存）
kubectl -n tcm-history-ai rollout restart deployment gateway user-service history-service \
  knowledge-service graph-service ai-service learning-service

# 等待滚动完成
kubectl -n tcm-history-ai rollout status deployment/gateway
```

---

## 6. 恢复演练 Checklist

每次执行恢复演练建议按下表逐项确认：

| 阶段 | 检查项 | 命令 / 操作 |
| ---- | ------ | ----------- |
| 演练前 | 备份 PVC 存在且有可用备份 | `kubectl -n <ns> get pvc <release>-backup-pvc` |
| 演练前 | 目标数据库就绪 | `kubectl -n <ns> rollout status statefulset/postgres` |
| 演练前 | 业务侧无活跃流量（或已切换到备用环境） | `kubectl -n <ns> get pods -l app.kubernetes.io/component=microservice` |
| 演练中 | Dry-run 通过 | `./restore-*.sh --dry-run` |
| 演练中 | 恢复 Pod 成功完成（Succeeded） | `kubectl -n <ns> get pod <restore-pod>` |
| 演练后 | 表/节点数量符合预期 | 见 §3.4 / §4.4 验证命令 |
| 演练后 | 业务冒烟测试通过 | `curl http://<gateway>/health` + 关键路由 |
| 演练后 | 临时 Pod 已清理 | `kubectl -n <ns> get pods \| grep restore` 应为空 |
| 演练后 | 演练记录归档 | 记录备份时间戳、恢复耗时、异常情况 |

---

## 7. 自动化建议

- **定期演练**：建议每季度执行一次完整恢复演练，纳入 SRE 例行巡检
- **CI 集成**：可在 CI 流水线中加 `--dry-run` 步骤，验证脚本可用性（不实际恢复）
- **监控告警**：备份失败应在 CronJob `failedJobsHistoryLimit` 内被发现，配置 Prometheus 告警：
  ```promql
  increase(kube_job_failed{job_name=~".*backup.*"}[1d]) > 0
  ```
- **异地备份**：生产环境建议额外配置 Velero / k8up 把备份 PVC 同步到对象存储（S3 / OSS），防集群级故障
