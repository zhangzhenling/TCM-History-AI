# ============================================================
# TCM-History-AI 一键部署脚本 (Windows PowerShell)
# 支持: 一键打包、启动、停止、重启、查看状态
# 用法:
#   .\deploy-windows.ps1 build            # 仅打包（后端+前端）
#   .\deploy-windows.ps1 start            # 打包并启动所有服务
#   .\deploy-windows.ps1 start-dev        # 以开发模式启动（直接 go run）
#   .\deploy-windows.ps1 stop             # 停止所有服务
#   .\deploy-windows.ps1 restart          # 重启所有服务
#   .\deploy-windows.ps1 status           # 查看服务状态
#   .\deploy-windows.ps1 logs [service]   # 查看日志
#   .\deploy-windows.ps1 clean            # 清理构建产物
# ============================================================

[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [ValidateSet("build", "start", "start-dev", "stop", "restart", "status", "logs", "clean", "help")]
    [string]$Action = "help",

    [Parameter(Position = 1)]
    [string]$Arg1 = "",

    [Parameter(Position = 2)]
    [string]$Arg2 = ""
)

# ---------- 颜色输出 ----------
function Write-ColorOutput {
    param(
        [string]$Text,
        [string]$Color = "White",
        [string]$Prefix = ""
    )
    if ($Prefix) {
        $prefixColor = switch ($Color) {
            "Green" { "DarkGreen" }
            "Red" { "DarkRed" }
            "Yellow" { "DarkYellow" }
            "Cyan" { "DarkCyan" }
            default { "White" }
        }
        Write-Host -ForegroundColor $prefixColor -NoNewline "$Prefix "
    }
    Write-Host -ForegroundColor $Color $Text
}

function Write-Info { param($Text) Write-ColorOutput -Text $Text -Color Cyan -Prefix "[INFO]" }
function Write-Ok { param($Text) Write-ColorOutput -Text $Text -Color Green -Prefix "[OK]" }
function Write-Warn { param($Text) Write-ColorOutput -Text $Text -Color Yellow -Prefix "[WARN]" }
function Write-Error2 { param($Text) Write-ColorOutput -Text $Text -Color Red -Prefix "[ERROR]" }
function Write-Step { param($Text) Write-Host ""; Write-ColorOutput -Text $Text -Color Green }

# ---------- 配置 ----------
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectDir = Resolve-Path (Join-Path $ScriptDir "..\..")
$BackendDir = Join-Path $ProjectDir "backend"
$FrontendDir = Join-Path $ProjectDir "frontend"
$BinDir = Join-Path $BackendDir "bin"
$DeployDir = Join-Path $ProjectDir ".deploy"
$PidDir = Join-Path $DeployDir "pids"
$LogDir = Join-Path $DeployDir "logs"

# 服务列表
$Script:Services = @("gateway", "user-service", "history-service", "knowledge-service", "graph-service", "ai-service", "learning-service")

# 端口配置
$Script:ServicePorts = @{
    "gateway" = 8080
    "user-service" = 9001
    "history-service" = 9002
    "knowledge-service" = 9003
    "graph-service" = 9004
    "ai-service" = 9005
    "learning-service" = 9006
}

# 服务启动顺序（按依赖关系）
$Script:StartupOrder = @("gateway", "user-service", "history-service", "graph-service", "knowledge-service", "ai-service", "learning-service")

# ---------- 环境检查 ----------
function Check-Environment {
    Write-Step "检查运行环境"

    $missing = 0

    # Go
    $goCmd = Get-Command go -ErrorAction SilentlyContinue
    if (-not $goCmd) {
        Write-Error2 "Go 未安装，请安装 Go 1.22+"
        Write-Info "下载地址: https://go.dev/dl/"
        $missing++
    } else {
        $goVersion = & go version
        if ($goVersion -match 'go(\d+)\.(\d+)') {
            $major = [int]$matches[1]
            $minor = [int]$matches[2]
            if ($major -lt 1 -or ($major -eq 1 -and $minor -lt 22)) {
                Write-Warn "Go 版本过低 (当前: $($goVersion))，建议 Go 1.22+"
            } else {
                Write-Ok "Go $($goVersion)"
            }
        }
    }

    # Node.js
    $nodeCmd = Get-Command node -ErrorAction SilentlyContinue
    if (-not $nodeCmd) {
        Write-Error2 "Node.js 未安装，请安装 Node.js 20+"
        Write-Info "安装: winget install OpenJS.NodeJS.LTS"
        $missing++
    } else {
        $nodeVersion = (& node --version) -replace 'v', ''
        $majorNode = [int]($nodeVersion.Split('.')[0])
        if ($majorNode -lt 20) {
            Write-Warn "Node.js 版本过低 (当前: $nodeVersion)，建议 Node.js 20+"
        } else {
            Write-Ok "Node.js v$nodeVersion"
        }
    }

    # pnpm
    $pnpmCmd = Get-Command pnpm -ErrorAction SilentlyContinue
    if (-not $pnpmCmd) {
        Write-Warn "pnpm 未安装，尝试自动安装..."
        try {
            & npm install -g pnpm@9 2>$null
            Write-Ok "pnpm 安装成功"
        } catch {
            Write-Error2 "pnpm 安装失败，请手动安装: npm install -g pnpm@9"
            $missing++
        }
    } else {
        $pnpmVersion = & pnpm --version
        Write-Ok "pnpm $pnpmVersion"
    }

    # Git
    $gitCmd = Get-Command git -ErrorAction SilentlyContinue
    if (-not $gitCmd) {
        Write-Error2 "Git 未安装"
        Write-Info "安装: winget install Git.Git"
        $missing++
    } else {
        $gitVersion = & git --version
        Write-Ok "Git $gitVersion"
    }

    if ($missing -gt 0) {
        Write-Error2 "缺少 $missing 个必要依赖，请先安装后重试"
        exit 1
    }

    Write-Ok "环境检查完成"
}

# ---------- 目录准备 ----------
function Prepare-Dirs {
    if (-not (Test-Path $PidDir)) { New-Item -ItemType Directory -Path $PidDir -Force | Out-Null }
    if (-not (Test-Path $LogDir)) { New-Item -ItemType Directory -Path $LogDir -Force | Out-Null }
    Write-Info "PID 目录: $PidDir"
    Write-Info "日志目录: $LogDir"
}

# ---------- 后端编译 ----------
function Build-Backend {
    Write-Step "编译后端服务"

    Set-Location $BackendDir

    if (-not (Test-Path "go.mod")) {
        Write-Error2 "在 $BackendDir 未找到 go.mod"
        exit 1
    }

    # 下载依赖
    Write-Info "下载 Go 依赖..."
    & go mod download 2>$null
    if ($LASTEXITCODE -ne 0) {
        & go mod tidy
    }

    # 创建 bin 目录
    if (-not (Test-Path $BinDir)) {
        New-Item -ItemType Directory -Path $BinDir -Force | Out-Null
    }

    # 编译所有服务
    foreach ($svc in $Script:Services) {
        $cmdDir = Join-Path $BackendDir "$svc\cmd\$svc"
        if (-not (Test-Path $cmdDir)) {
            Write-Warn "跳过 $svc (cmd 目录不存在)"
            continue
        }

        Write-Info "编译 $svc ..."
        $output = Join-Path $BinDir "$svc.exe"
        & go build -o $output "./$svc/cmd/$svc"
        if ($LASTEXITCODE -eq 0) {
            Write-Ok "$svc 编译成功"
        } else {
            Write-Error2 "$svc 编译失败"
            exit 1
        }
    }

    Write-Ok "后端编译完成，产物位于 $BinDir\"
    Get-ChildItem $BinDir | Format-Table Name, Length, LastWriteTime
}

# ---------- 前端构建 ----------
function Build-Frontend {
    Write-Step "构建前端"

    Set-Location $FrontendDir

    if (-not (Test-Path "package.json")) {
        Write-Error2 "在 $FrontendDir 未找到 package.json"
        exit 1
    }

    # 安装依赖
    if (-not (Test-Path "node_modules")) {
        Write-Info "安装前端依赖 (pnpm install)..."
        & pnpm install
        if ($LASTEXITCODE -ne 0) {
            Write-Error2 "pnpm install 失败"
            exit 1
        }
    } else {
        Write-Info "前端依赖已安装，跳过 pnpm install"
    }

    # 构建
    Write-Info "构建前端 (pnpm build)..."
    & pnpm build
    if ($LASTEXITCODE -ne 0) {
        Write-Error2 "前端构建失败"
        exit 1
    }

    Write-Ok "前端构建完成"
}

# ---------- 打包 ----------
function Do-Build {
    Write-Step "========== 开始打包 =========="
    Build-Backend
    Build-Frontend
    Write-Step "========== 打包完成 =========="
    Write-Ok "所有产物已生成！"
    Write-Info "后端二进制: $BinDir\"
    Write-Info "前端产物: $FrontendDir\apps\learner\dist\"
    Write-Info "管理端产物: $FrontendDir\apps\admin\dist\"
}

# ---------- 启动单个服务 ----------
function Start-ServiceProcess {
    param(
        [string]$Svc,
        [string]$Mode = "prod"
    )

    $pidFile = Join-Path $PidDir "$Svc.pid"
    $logFile = Join-Path $LogDir "$Svc.log"
    $port = $Script:ServicePorts[$Svc]

    # 检查是否已在运行
    if (Test-Path $pidFile) {
        $oldPid = Get-Content $pidFile -ErrorAction SilentlyContinue
        if ($oldPid) {
            $oldProcess = Get-Process -Id $oldPid -ErrorAction SilentlyContinue
            if ($oldProcess) {
                Write-Warn "$Svc 已在运行 (PID: $oldPid)"
                return
            }
        }
        Remove-Item $pidFile -ErrorAction SilentlyContinue
    }

    if ($Mode -eq "dev") {
        # 开发模式：使用 go run
        Write-Info "以开发模式启动 $Svc (go run)..."
        Set-Location $BackendDir

        $psi = New-Object System.Diagnostics.ProcessStartInfo
        $psi.FileName = "go"
        $psi.Arguments = "run ./$Svc/cmd/$Svc"
        $psi.WorkingDirectory = $BackendDir
        $psi.UseShellExecute = $false
        $psi.RedirectStandardOutput = $true
        $psi.RedirectStandardError = $true
        $psi.CreateNoWindow = $true

        $process = New-Object System.Diagnostics.Process
        $process.StartInfo = $psi
        $process.EnableRaisingEvents = $false
        [void]$process.Start()

        $process.Id | Out-File $pidFile

        Start-Sleep -Seconds 3

        if (-not $process.HasExited) {
            Write-Ok "$Svc 启动成功 (PID: $($process.Id), 端口: $port)"
        } else {
            Write-Error2 "$Svc 启动失败，查看日志: $logFile"
            Get-Content $logFile -Tail 20
            Remove-Item $pidFile -ErrorAction SilentlyContinue
            return
        }
    } else {
        # 生产模式：使用编译好的二进制
        $binary = Join-Path $BinDir "$Svc.exe"
        if (-not (Test-Path $binary)) {
            Write-Error2 "$binary 不存在，请先执行 build"
            return
        }

        Write-Info "启动 $Svc ..."
        Set-Location $BackendDir

        $psi = New-Object System.Diagnostics.ProcessStartInfo
        $psi.FileName = $binary
        $psi.WorkingDirectory = $BackendDir
        $psi.UseShellExecute = $false
        $psi.RedirectStandardOutput = $true
        $psi.RedirectStandardError = $true
        $psi.CreateNoWindow = $true

        $process = New-Object System.Diagnostics.Process
        $process.StartInfo = $psi
        $process.EnableRaisingEvents = $false
        [void]$process.Start()

        $process.Id | Out-File $pidFile

        Start-Sleep -Seconds 2

        if (-not $process.HasExited) {
            Write-Ok "$Svc 启动成功 (PID: $($process.Id), 端口: $port)"
        } else {
            Write-Error2 "$Svc 启动失败，查看日志: $logFile"
            Remove-Item $pidFile -ErrorAction SilentlyContinue
            return
        }
    }
}

# ---------- 启动所有服务 ----------
function Do-Start {
    param([string]$Mode = "prod")

    Write-Step "========== 启动所有服务 (模式: $Mode) =========="

    # 检查是否需要打包
    if ($Mode -eq "prod") {
        $needBuild = $false
        foreach ($svc in $Script:Services) {
            $binary = Join-Path $BinDir "$svc.exe"
            if (-not (Test-Path $binary)) {
                Write-Warn "缺少 $svc 二进制，需要重新打包"
                $needBuild = $true
                break
            }
        }
        if ($needBuild) {
            Do-Build
        }
    }

    # 按依赖顺序启动
    foreach ($svc in $Script:StartupOrder) {
        Start-ServiceProcess -Svc $svc -Mode $Mode
        Start-Sleep -Seconds 1
    }

    Write-Step "========== 所有服务已启动 =========="
    Write-Host ""
    Write-Info "服务访问地址:"
    Write-Host "  前端:     http://localhost (需配合 Nginx)"
    Write-Host "  Gateway:  http://localhost:$($Script:ServicePorts['gateway'])"
    Write-Host "  API:      http://localhost:$($Script:ServicePorts['gateway'])/api/v1"
    Write-Host ""
    Write-Info "查看状态: .\deploy-windows.ps1 status"
    Write-Info "查看日志: .\deploy-windows.ps1 logs <service>"
    Write-Info "停止服务: .\deploy-windows.ps1 stop"
}

# ---------- 停止单个服务 ----------
function Stop-ServiceProcess {
    param([string]$Svc)

    $pidFile = Join-Path $PidDir "$Svc.pid"

    if (Test-Path $pidFile) {
        $pid = Get-Content $pidFile -ErrorAction SilentlyContinue
        if ($pid) {
            $process = Get-Process -Id $pid -ErrorAction SilentlyContinue
            if ($process) {
                Write-Info "停止 $Svc (PID: $pid)..."
                Stop-Process -Id $pid -Force -ErrorAction SilentlyContinue
                Start-Sleep -Milliseconds 500
                Remove-Item $pidFile -ErrorAction SilentlyContinue
                Write-Ok "$Svc 已停止"
            } else {
                Write-Warn "$Svc 未在运行 (旧 PID: $pid)"
                Remove-Item $pidFile -ErrorAction SilentlyContinue
            }
        }
    } else {
        # 尝试通过进程名查找
        $processName = "$Svc.exe"
        $processes = Get-Process -Name $processName -ErrorAction SilentlyContinue
        if ($processes) {
            Write-Info "停止 $Svc (PID: $($processes.Id -join ', '))..."
            $processes | Stop-Process -Force -ErrorAction SilentlyContinue
            Write-Ok "$Svc 已停止"
        } else {
            Write-Warn "$Svc 未运行"
        }
    }
}

# ---------- 停止所有服务 ----------
function Do-Stop {
    Write-Step "========== 停止所有服务 =========="
    foreach ($svc in $Script:Services) {
        Stop-ServiceProcess -Svc $svc
    }
    Write-Ok "所有服务已停止"
}

# ---------- 重启所有服务 ----------
function Do-Restart {
    param([string]$Mode = "prod")
    Do-Stop
    Start-Sleep -Seconds 2
    Do-Start -Mode $Mode
}

# ---------- 查看状态 ----------
function Do-Status {
    Write-Step "========== 服务状态 =========="

    $allRunning = $true
    $tableData = @()

    foreach ($svc in $Script:Services) {
        $pidFile = Join-Path $PidDir "$svc.pid"
        $status = "已停止"
        $pid = "-"

        if (Test-Path $pidFile) {
            $pid = Get-Content $pidFile -ErrorAction SilentlyContinue
            if ($pid) {
                $process = Get-Process -Id $pid -ErrorAction SilentlyContinue
                if ($process) {
                    $status = "运行中"
                } else {
                    $status = "已停止"
                    $pid = "-"
                    Remove-Item $pidFile -ErrorAction SilentlyContinue
                    $allRunning = $false
                }
            } else {
                $pid = "-"
                $allRunning = $false
            }
        } else {
            # 尝试通过进程名查找
            $process = Get-Process -Name "$svc.exe" -ErrorAction SilentlyContinue
            if ($process) {
                $status = "运行中"
                $pid = $process.Id
                $allRunning = $true
            } else {
                $allRunning = $false
            }
        }

        $tableData += [PSCustomObject]@{
            "服务"   = $svc
            "状态"   = $status
            "PID"    = $pid
            "端口"   = $Script:ServicePorts[$svc]
        }
    }

    $tableData | Format-Table -AutoSize

    if ($allRunning) {
        Write-Ok "所有服务正常运行"
    } else {
        Write-Warn "部分服务未运行，可使用 '.\deploy-windows.ps1 start' 启动"
    }
}

# ---------- 查看日志 ----------
function Do-Logs {
    param(
        [string]$Svc = "",
        [int]$Lines = 50
    )

    if ([string]::IsNullOrEmpty($Svc)) {
        # 查看所有服务的最新日志
        foreach ($svc in $Script:Services) {
            $logFile = Join-Path $LogDir "$svc.log"
            if ((Test-Path $logFile) -and ((Get-Item $logFile).Length -gt 0)) {
                Write-Host ""
                Write-Step "========== $svc 日志 =========="
                Get-Content $logFile -Tail 10
            }
        }
    } else {
        $logFile = Join-Path $LogDir "$Svc.log"
        if (Test-Path $logFile) {
            Write-Step "========== $Svc 日志 (最近 $Lines 行) =========="
            Get-Content $logFile -Tail $Lines
        } else {
            Write-Error2 "日志文件不存在: $logFile"
        }
    }
}

# ---------- 清理构建产物 ----------
function Do-Clean {
    Write-Step "========== 清理构建产物 =========="

    # 停止所有服务
    Do-Stop

    # 清理后端产物
    if (Test-Path $BinDir) {
        Remove-Item $BinDir -Recurse -Force
        Write-Info "已删除 $BinDir\"
    }

    # 清理 PID 和日志
    if (Test-Path $DeployDir) {
        Remove-Item $DeployDir -Recurse -Force
        Write-Info "已删除 $DeployDir\"
    }

    # 清理前端产物
    $learnerDist = Join-Path $FrontendDir "apps\learner\dist"
    if (Test-Path $learnerDist) {
        Remove-Item $learnerDist -Recurse -Force
        Write-Info "已删除前端构建产物"
    }
    $adminDist = Join-Path $FrontendDir "apps\admin\dist"
    if (Test-Path $adminDist) {
        Remove-Item $adminDist -Recurse -Force
    }

    Write-Ok "清理完成"
}

# ---------- 显示帮助 ----------
function Show-Help {
    Write-Host "TCM-History-AI 一键部署脚本 (Windows PowerShell)"
    Write-Host ""
    Write-Host "用法: .\deploy-windows.ps1 <命令> [参数]"
    Write-Host ""
    Write-Host "命令:"
    Write-Host "  build            仅打包（编译后端 + 构建前端）"
    Write-Host "  start            打包并以生产模式启动所有服务"
    Write-Host "  start-dev        以开发模式启动所有服务（go run）"
    Write-Host "  stop             停止所有服务"
    Write-Host "  restart [mode]   重启所有服务 (mode: prod/dev, 默认 prod)"
    Write-Host "  status           查看服务运行状态"
    Write-Host "  logs [service]   查看日志 (可选服务名)"
    Write-Host "  clean            清理构建产物和日志"
    Write-Host "  help             显示帮助信息"
    Write-Host ""
    Write-Host "服务列表:"
    Write-Host "  $($Script:Services -join ', ')"
    Write-Host ""
    Write-Host "示例:"
    Write-Host "  .\deploy-windows.ps1 build              # 打包"
    Write-Host "  .\deploy-windows.ps1 start              # 打包并启动"
    Write-Host "  .\deploy-windows.ps1 logs gateway       # 查看 gateway 日志"
    Write-Host "  .\deploy-windows.ps1 restart dev        # 以开发模式重启"
}

# ---------- 主入口 ----------
switch ($Action) {
    "build" {
        Check-Environment
        Prepare-Dirs
        Do-Build
    }
    "start" {
        Check-Environment
        Prepare-Dirs
        Do-Start -Mode "prod"
    }
    "start-dev" {
        Check-Environment
        Prepare-Dirs
        Do-Start -Mode "dev"
    }
    "stop" {
        Prepare-Dirs
        Do-Stop
    }
    "restart" {
        Check-Environment
        Prepare-Dirs
        $mode = if ($Arg1) { $Arg1 } else { "prod" }
        Do-Restart -Mode $mode
    }
    "status" {
        Prepare-Dirs
        Do-Status
    }
    "logs" {
        Prepare-Dirs
        $lines = if ($Arg2) { [int]$Arg2 } else { 50 }
        Do-Logs -Svc $Arg1 -Lines $lines
    }
    "clean" {
        Do-Clean
    }
    "help" {
        Show-Help
    }
    default {
        Show-Help
    }
}
