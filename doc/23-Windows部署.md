# 23 Windows 原生部署指南

本章提供 TCM-History-AI 在 Windows 服务器/桌面环境（Windows Server 2022 / Windows 11）上的部署方案，包含两种路径：

| 方案 | 推荐度 | 适用场景 | 复杂度 |
|------|--------|----------|--------|
| **WSL2 方案** | 强烈推荐 | 开发测试、轻量级生产 | 低 |
| **原生 Windows 方案** | 可选 | 无外网 WSL 安装条件、必须与现有 Windows 生态集成 | 中 |

**强烈建议优先使用 WSL2 方案**：WSL2 提供完整的 Linux 内核，各组件（PostgreSQL、Redis、Neo4j 等）的 Linux 版本远比 Windows 版本成熟稳定，且与本项目的 Linux 原生部署方案（第二十二章）完全一致。

---

## 1 WSL2 部署方案（推荐）

### 1.1 启用 WSL2

以管理员身份打开 PowerShell：

```powershell
# 启用 WSL 和虚拟机平台
wsl --install
# 安装完成后重启计算机

# 如果已启用 WSL1，设置为默认 WSL2
wsl --set-default-version 2

# 安装 Ubuntu 22.04
wsl --install -d Ubuntu-22.04
```

重启后，Ubuntu 22.04 会自动启动并要求设置用户名和密码。

### 1.2 WSL2 资源配置

在 Windows 用户目录创建 `.wslconfig` 文件，限制 WSL2 内存使用，避免拖垮 Windows：

```powershell
# 文件路径：C:\Users\<你的用户名>\.wslconfig
notepad $env:USERPROFILE\.wslconfig
```

写入以下内容：
```ini
[wsl2]
memory=12GB
processors=6
swap=2GB
localhostForwarding=true
autoProxy=true
```

保存后在 PowerShell 执行：
```powershell
wsl --shutdown
```

### 1.3 在 WSL2 Ubuntu 中部署

启动 Ubuntu：
```powershell
wsl -d Ubuntu-22.04
```

进入 WSL2 后，所有操作与标准 Linux 完全一致，直接按照 [第二十二章 Linux 部署指南](./22-Linux部署.md) 执行即可：

```bash
# 更新系统
sudo apt update && sudo apt upgrade -y

# 安装基础依赖
sudo apt install -y git curl wget build-essential nginx

# 后续步骤与 Linux 部署完全一致：
# 1. 安装 Go 1.22+、Node 20、pnpm 9
# 2. 安装 PostgreSQL 16、Redis 7、Neo4j 5、MinIO、Meilisearch、RabbitMQ
# 3. 编译后端服务
# 4. 配置 systemd 服务
# 5. 编译前端并配置 Nginx
```

### 1.4 Windows 访问 WSL2 服务

WSL2 自动将 Linux 端口转发到 Windows localhost，无需额外配置：

| WSL2 服务 | Windows 访问地址 |
|-----------|-----------------|
| 前端（Nginx 80） | `http://localhost` |
| Gateway（8080） | `http://localhost:8080` |
| PostgreSQL（5432） | `localhost:5432` |
| Redis（6379） | `localhost:6379` |
| Neo4j Browser（7474） | `http://localhost:7474` |
| MinIO Console（9001） | `http://localhost:9001` |
| RabbitMQ Mgmt（15672） | `http://localhost:15672` |

若需局域网其他设备访问，需配置 Windows 防火墙放行端口，或使用 WSL 端口转发：

```powershell
# 在 PowerShell（管理员）中，将 WSL2 端口暴露给局域网
$wslIp = (wsl hostname -I).Trim().Split()[0]
netsh interface portproxy add v4tov4 listenport=80 connectport=80 connectaddress=$wslIp
netsh interface portproxy add v4tov4 listenport=8080 connectport=8080 connectaddress=$wslIp
netsh advfirewall firewall add rule name="TCM-WSL2-80" dir=in action=allow protocol=tcp localport=80
netsh advfirewall firewall add rule name="TCM-WSL2-8080" dir=in action=allow protocol=tcp localport=8080
```

### 1.5 文件共享

WSL2 挂载 Windows 磁盘到 `/mnt/`：
```bash
# Windows C 盘在 WSL2 中的路径
cd /mnt/c/

# 将项目放在 Windows 磁盘上，在 WSL2 中访问
cd /mnt/c/dev/TCM-History-AI
```

> **注意**：WSL2 访问 Windows 磁盘（`/mnt/c`）的 IO 性能比访问 WSL2 原生文件系统（`~`）慢约 3-5 倍。建议将项目代码放在 WSL2 文件系统中（如 `~/TCM-History-AI`），仅在需要时通过 Windows 文件管理器访问 `\\wsl$\Ubuntu-22.04\home\<username>`。

### 1.6 WSL2 开机自启

创建 Windows 计划任务，使 WSL2 在系统启动时自动运行服务：

```powershell
# 创建启动脚本
$script = @'
@echo off
wsl -d Ubuntu-22.04 -u root /bin/bash -c "systemctl start postgresql redis-server neo4j nginx minio meilisearch rabbitmq-server etcd; systemctl start 'tcm-*'"
'@
$script | Out-File -FilePath "$env:USERPROFILE\wsl2-startup.bat" -Encoding ASCII

# 创建计划任务
$action = New-ScheduledTaskAction -Execute "$env:USERPROFILE\wsl2-startup.bat"
$trigger = New-ScheduledTaskTrigger -AtLogon
$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
Register-ScheduledTask -TaskName "TCM-WSL2-Startup" -Action $action -Trigger $trigger -Settings $settings -RunLevel Highest
```

---

## 2 原生 Windows 部署方案

如确实无法使用 WSL2，以下提供纯 Windows 环境的部署步骤。

### 2.1 环境准备

#### 2.1.1 Go 1.22+

1. 下载 Windows 安装包：https://go.dev/dl/go1.22.5.windows-amd64.msi
2. 双击安装，默认路径 `C:\Program Files\Go`
3. 验证：
```powershell
go version
# go version go1.22.5 windows/amd64

# 配置 Go 代理
$env:GOPROXY = "https://goproxy.cn,direct"
[Environment]::SetEnvironmentVariable("GOPROXY", "https://goproxy.cn,direct", "User")
```

#### 2.1.2 Git

```powershell
winget install Git.Git
```

#### 2.1.3 Node.js 20 + pnpm

```powershell
# 使用 winget 安装 Node
winget install OpenJS.NodeJS.LTS

# 安装 pnpm
npm install -g pnpm@9
```

#### 2.1.4 Make（可选）

```powershell
winget install GnuWin32.Make
# 或安装完整的 MinGW/MSYS2
```

### 2.2 安装数据库与中间件

#### 2.2.1 PostgreSQL 16

```powershell
# 使用 Chocolatey 安装
choco install postgresql16 --params '/Password:tcm_pass'

# 或使用 EnterpriseDB 安装包下载安装
# https://www.enterprisedb.com/downloads/postgres-postgresql-downloads

# 安装后创建数据库
& "C:\Program Files\PostgreSQL\16\bin\psql.exe" -U postgres -c "CREATE USER tcm WITH PASSWORD 'tcm_pass';"
& "C:\Program Files\PostgreSQL\16\bin\psql.exe" -U postgres -c "CREATE DATABASE tcm_history OWNER tcm;"
```

PostgreSQL Windows 服务默认开机自启，服务名为 `postgresql-x64-16`。

#### 2.2.2 Redis 7

```powershell
# 下载 Windows 版 Redis（微软维护的 Fork）
# https://github.com/microsoftarchive/redis/releases
# 或使用 Memurai（商业兼容版）

# 或使用 Chocolatey
choco install redis-64

# 配置密码：编辑 C:\ProgramData\chocolatey\lib\redis-64\redis.windows.conf
# 添加：requirepass redis_pass

# 启动服务
redis-server --service-install redis.windows.conf --service-name Redis
redis-server --service-start
```

#### 2.2.3 Neo4j 5

1. 下载 Neo4j Community Edition for Windows：
   https://neo4j.com/download-center/
2. 解压到 `C:\neo4j`
3. 配置 `conf\neo4j.conf`：
```ini
dbms.security.auth_enabled=true
server.memory.heap.initial_size=512m
server.memory.heap.max_size=1G
server.memory.pagecache.size=512m
```
4. 安装并启动服务：
```powershell
cd C:\neo4j\bin
.\neo4j.bat install-service
.\neo4j.bat start
```

浏览器访问 `http://localhost:7474`，初始密码 `neo4j`，首次登录要求修改。

#### 2.2.4 MinIO

```powershell
# 下载 MinIO Windows 二进制
Invoke-WebRequest -Uri https://dl.min.io/server/minio/release/windows-amd64/minio.exe -OutFile C:\minio\minio.exe

# 启动（以命令行方式，生产环境建议注册为 Windows 服务）
$env:MINIO_ROOT_USER = "minio"
$env:MINIO_ROOT_PASSWORD = "minio_pass"
C:\minio\minio.exe server C:\minio\data --console-address :9001
```

注册为 Windows 服务（使用 NSSM）：
```powershell
choco install nssm
nssm install MinIO "C:\minio\minio.exe"
nssm set MinIO AppDirectory "C:\minio"
nssm set MinIO AppParameters "server C:\minio\data --console-address :9001"
nssm set MinIO Environment "MINIO_ROOT_USER=minio" "MINIO_ROOT_PASSWORD=minio_pass"
nssm start MinIO
```

#### 2.2.5 Meilisearch

```powershell
Invoke-WebRequest -Uri https://github.com/meilisearch/meilisearch/releases/download/v1.6.0/meilisearch-windows-amd64.exe -OutFile C:\meilisearch\meilisearch.exe

# 启动
C:\meilisearch\meilisearch.exe --master-key meili_master_key_dev --http-addr 127.0.0.1:7700
```

#### 2.2.6 RabbitMQ 3.13

```powershell
# 先安装 Erlang
choco install erlang

# 安装 RabbitMQ
choco install rabbitmq

# 创建用户
& "C:\Program Files\RabbitMQ Server\rabbitmq_server-3.13.0\sbin\rabbitmqctl.bat" add_user tcm tcm_pass
& "C:\Program Files\RabbitMQ Server\rabbitmq_server-3.13.0\sbin\rabbitmqctl.bat" set_user_tags tcm administrator
& "C:\Program Files\RabbitMQ Server\rabbitmq_server-3.13.0\sbin\rabbitmqctl.bat" add_vhost /tcm
& "C:\Program Files\RabbitMQ Server\rabbitmq_server-3.13.0\sbin\rabbitmqctl.bat" set_permissions -p /tcm tcm ".*" ".*" ".*"
```

#### 2.2.7 Milvus（WSL2 或 Docker 辅助）

Milvus 官方不提供 Windows 原生二进制。**强烈建议**：在 WSL2 中单独运行 Milvus Standalone，Windows 后端服务通过 `localhost:19530` 连接。

如果完全不能运行 WSL2/Docker，需改用纯 CPU 向量检索替代方案（如 Faiss 的 Windows 版本），但这需要修改 `knowledge-service` 代码。

### 2.3 编译后端服务

```powershell
cd C:\dev\TCM-History-AI\backend

# 编译全部服务
$services = @("gateway", "user-service", "history-service", "knowledge-service", "graph-service", "ai-service", "learning-service")
New-Item -ItemType Directory -Force -Path .\bin
foreach ($svc in $services) {
    Write-Host "Building $svc..."
    go build -o .\bin\$svc.exe .\$svc\cmd\$svc
}
```

### 2.4 配置文件

创建 `C:\dev\TCM-History-AI\.env`：

```ini
# ===== 数据库 =====
DB_HOST=127.0.0.1
DB_PORT=5432
DB_USER=tcm
DB_PASSWORD=tcm_pass
DB_NAME=tcm_history
DB_SSLMODE=disable

# ===== Redis =====
REDIS_HOST=127.0.0.1
REDIS_PORT=6379
REDIS_PASSWORD=redis_pass

# ===== Neo4j =====
NEO4J_URI=bolt://127.0.0.1:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=neo4j_pass

# ===== MinIO =====
MINIO_ENDPOINT=127.0.0.1:9000
MINIO_ACCESS_KEY=minio
MINIO_SECRET_KEY=minio_pass
MINIO_USE_SSL=false

# ===== Milvus（如果在 WSL2 中运行）=====
MILVUS_HOST=127.0.0.1
MILVUS_PORT=19530

# ===== Meilisearch =====
MEILISEARCH_HOST=http://127.0.0.1:7700
MEILISEARCH_API_KEY=meili_master_key_dev

# ===== RabbitMQ =====
RABBITMQ_HOST=127.0.0.1
RABBITMQ_PORT=5672
RABBITMQ_USER=tcm
RABBITMQ_PASSWORD=tcm_pass
RABBITMQ_VHOST=/tcm

# ===== 应用 =====
APP_ENV=production
LOG_LEVEL=info
JWT_SECRET=CHANGE_ME_TO_A_LONG_RANDOM_STRING

# ===== 服务端口 =====
GATEWAY_HTTP_PORT=8080
```

### 2.5 使用 NSSM 注册为 Windows 服务

NSSM（Non-Sucking Service Manager）是将任意可执行文件注册为 Windows 服务的最佳工具：

```powershell
# 安装 NSSM
choco install nssm

# 注册 Gateway 服务
nssm install TCM-Gateway "C:\dev\TCM-History-AI\backend\bin\gateway.exe"
nssm set TCM-Gateway AppDirectory "C:\dev\TCM-History-AI\backend"
nssm set TCM-Gateway Environment "DB_HOST=127.0.0.1" "DB_PORT=5432" "DB_PASSWORD=tcm_pass" ...
nssm set TCM-Gateway Start SERVICE_AUTO_START
nssm start TCM-Gateway

# 注册其他服务
$services = @("user-service", "history-service", "knowledge-service", "graph-service", "ai-service", "learning-service")
foreach ($svc in $services) {
    nssm install "TCM-$svc" "C:\dev\TCM-History-AI\backend\bin\$svc.exe"
    nssm set "TCM-$svc" AppDirectory "C:\dev\TCM-History-AI\backend"
    nssm start "TCM-$svc"
}
```

查看服务状态：
```powershell
Get-Service TCM-*
Start-Service TCM-*
Stop-Service TCM-*
Restart-Service TCM-Gateway
```

查看日志（NSSM 自动捕获 stdout/stderr）：
```powershell
# 日志文件位置
Get-Content "C:\Windows\System32\LogFiles\TCM-Gateway.log" -Wait
```

### 2.6 前端编译

```powershell
cd C:\dev\TCM-History-AI\frontend
pnpm install
pnpm build
```

### 2.7 IIS / Nginx for Windows 反向代理

#### 方案 A：Nginx for Windows

1. 下载 Nginx for Windows：http://nginx.org/en/download.html
2. 解压到 `C:\nginx`
3. 编辑 `conf\nginx.conf`：

```nginx
server {
    listen       80;
    server_name  localhost;

    location / {
        root   C:/dev/TCM-History-AI/frontend/dist;
        index  index.html;
        try_files $uri $uri/ /index.html;
    }

    location /api/ {
        proxy_pass http://127.0.0.1:8080/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    location /mcp/sse {
        proxy_pass http://127.0.0.1:8080/mcp/sse;
        proxy_buffering off;
        proxy_cache off;
        proxy_read_timeout 86400s;
    }
}
```

4. 启动：`C:\nginx\nginx.exe`

#### 方案 B：IIS + ARR（Application Request Routing）

如果服务器已有 IIS：
1. 安装 ARR 和 URL Rewrite 模块
2. 创建网站指向 `frontend\dist`
3. 添加反向代理规则到 `http://localhost:8080`
4. 配置 `web.config` 的 `rewrite` 规则处理 Vue Router history 模式

### 2.8 Windows 防火墙

```powershell
# 放行 HTTP
New-NetFirewallRule -DisplayName "TCM-HTTP" -Direction Inbound -Protocol TCP -LocalPort 80 -Action Allow
New-NetFirewallRule -DisplayName "TCM-HTTPS" -Direction Inbound -Protocol TCP -LocalPort 443 -Action Allow
New-NetFirewallRule -DisplayName "TCM-Gateway" -Direction Inbound -Protocol TCP -LocalPort 8080 -Action Allow

# 如需局域网访问各组件端口
New-NetFirewallRule -DisplayName "TCM-Postgres" -Direction Inbound -Protocol TCP -LocalPort 5432 -Action Allow
New-NetFirewallRule -DisplayName "TCM-Redis" -Direction Inbound -Protocol TCP -LocalPort 6379 -Action Allow
```

---

## 3 方案对比与建议

| 维度 | WSL2 方案 | 原生 Windows 方案 |
|------|-----------|-------------------|
| 安装复杂度 | 低（复用 Linux 部署流程） | 中（需逐个安装 Windows 版组件） |
| 组件成熟度 | 高（Linux 版官方维护） | 中（部分组件 Windows 版为社区维护） |
| 性能 | 接近原生 Linux（WSL2 内核优化） | Go 服务性能一致，但部分组件（Redis/Neo4j）Windows 版性能略低 |
| 内存占用 | WSL2 VM 内存可限制 | 各服务独立进程，总占用可能更高 |
| 生产适用性 | 适合中小型生产 | 适合必须与 Windows AD/生态集成的场景 |
| 维护难度 | 低（Linux 运维标准化） | 中（Windows 服务管理 + Linux 服务管理混合） |
| Milvus 支持 | 完全支持 | 需借助 WSL2/Docker，或改用替代方案 |

---

## 4 常见问题

### Q1: WSL2 中 `systemctl` 报错 "Failed to connect to bus"

WSL2 默认不使用 systemd，需要启用：
```bash
# 在 WSL2 中
sudo tee /etc/wsl.conf << 'EOF'
[boot]
systemd=true
EOF
# 然后在 PowerShell 中执行
wsl --shutdown
# 重新进入 WSL2
```

### Q2: Windows 版 Redis 没有 `redis-cli` 或版本过旧

使用微软维护的 Redis for Windows（较旧），或改用 Memurai（商业但兼容）。开发测试也可用 WSL2 中的 Redis。

### Q3: Go 编译报错 "exec: gcc: executable file not found"

某些依赖（如 CGO 数据库驱动）需要 gcc：
```powershell
# 安装 MinGW
choco install mingw
# 或
winget install LLVM.LLVM
```

### Q4: 前端构建报错 "ENOENT" 或路径问题

确保使用 PowerShell 或 CMD，不要在旧版 Windows Command Shell 中运行。路径分隔符问题可通过使用 PowerShell 避免。

### Q5: WSL2 磁盘空间膨胀

```powershell
# 压缩 WSL2 虚拟磁盘
wsl --shutdown
diskpart
# diskpart 中执行：
# select vdisk file="C:\Users\<用户名>\AppData\Local\Packages\CanonicalGroupLimited...\LocalState\ext4.vhdx"
# attach vdisk readonly
# compact vdisk
# detach vdisk
# exit
```

---

## 5 附录：PowerShell 一键安装脚本

```powershell
# deploy-windows.ps1 - WSL2 方案一键安装
param(
    [string]$InstallDir = "C:\tcm-history-ai",
    [string]$WslDistro = "Ubuntu-22.04"
)

Write-Host "=== TCM-History-AI Windows 部署脚本（WSL2 方案）===" -ForegroundColor Green

# 检查 WSL2
$wslStatus = wsl --status 2>$null
if ($LASTEXITCODE -ne 0) {
    Write-Host "正在启用 WSL2..." -ForegroundColor Yellow
    wsl --install --no-distribution
    Write-Host "请重启计算机后重新运行此脚本" -ForegroundColor Red
    exit 1
}

# 检查 Ubuntu
$distros = wsl --list --quiet
if ($distros -notcontains $WslDistro) {
    Write-Host "正在安装 $WslDistro..." -ForegroundColor Yellow
    wsl --install -d $WslDistro
}

# 克隆代码
if (-not (Test-Path $InstallDir)) {   New-Item -ItemType Directory -Force -Path $InstallDir
}

# 在 WSL2 中执行部署
$deployScript = @'
#!/bin/bash
set -e
cd /opt
git clone https://github.com/zhangzhenling/TCM-History-AI.git 2>/dev/null || (cd TCM-History-AI && git pull)
# 后续调用 Linux 部署脚本...
'@

Write-Host "部署完成！访问 http://localhost 查看" -ForegroundColor Green
```

---

本章提供了 Windows 环境下的两种部署路径。WSL2 方案因与 Linux 生态完全兼容、组件成熟度高，是绝大多数场景的首选；原生 Windows 方案作为兜底，适用于无法运行 WSL2 的特殊环境。无论哪种方案，生产环境都建议配合 [第十四章 部署方案](./14-部署方案.md) 的安全与监控基线实施。
