# 本地CI测试指南

本文档介绍如何在本地环境中运行和测试GitHub Actions工作流。

## 概述

本项目提供了多种方式在本地进行CI/CD测试，包括：

1. **本地脚本测试** - 使用 `scripts/test-local.sh`
2. **Makefile目标** - 使用 `make` 命令
3. **Act工作流** - 使用 `act` 工具运行实际GitHub Actions

## 快速开始

### 1. 安装依赖

确保系统已安装以下工具：

```bash
# macOS
brew install act go docker

# 验证安装
go version
docker --version
act --version
```

### 2. 运行快速测试

```bash
# 运行所有本地测试
make ci-local

# 或者使用脚本
./scripts/test-local.sh all
```

## 详细用法

### 本地脚本 (`scripts/test-local.sh`)

功能完整的测试脚本，支持以下命令：

```bash
# 运行Go测试
./scripts/test-local.sh test

# 构建项目
./scripts/test-local.sh build

# 运行安全扫描
./scripts/test-local.sh security

# 构建Docker镜像
./scripts/test-local.sh docker

# 运行完整CI流程
./scripts/test-local.sh all

# 运行act工作流
./scripts/test-local.sh act [job_name]
```

### Makefile目标

```bash
# 本地测试目标
make test-local        # 运行Go测试
make build-local       # 构建项目
make security-local    # 安全扫描
make docker-local      # Docker构建
make ci-local          # 完整CI流程

# Act工作流目标
make act-test          # 运行act测试工作流
make act-build         # 运行act构建工作流
```

### Act工作流测试

使用act工具运行实际的GitHub Actions：

```bash
# 列出可用工作流
act --list

# 运行特定作业
act -j test
act -j build
act -j security

# 使用Apple Silicon优化
act -j test --container-architecture linux/amd64
```

## 工作流说明

### 原始CI工作流 (`.github/workflows/ci.yml`)

包含以下作业：

- **test**: Go单元测试和覆盖率报告
- **build**: 跨平台二进制构建 (Linux, Windows, macOS)
- **docker**: Docker镜像构建和推送
- **security**: Gosec安全扫描
- **release**: GitHub发布创建

### 本地CI工作流 (`.github/workflows/ci-local.yml`)

为本地测试优化的版本：

- 跳过远程操作 (Docker推送、发布)
- 优化构建矩阵
- 添加错误容忍

## 配置文件

### 环境变量 (`.vars`)

```bash
GO_VERSION=1.23.2
SKIP_DOCKER_PUSH=true
SKIP_RELEASE=true
```

### 密钥配置 (`.secrets`)

```bash
GITHUB_TOKEN=your-github-token
CODECOV_TOKEN=your-codecov-token
```

### Act配置 (`.actrc`)

```bash
-P ubuntu-latest=nektos/act-environments-ubuntu:18.04
-s .secrets
-e .vars
-b
```

## 常见用法场景

### 1. 开发时快速验证

```bash
# 运行测试和构建
make test-local build-local
```

### 2. 提交前完整检查

```bash
# 运行完整CI流程
make ci-local
```

### 3. 调试工作流问题

```bash
# 使用act运行特定作业
act -j test --container-architecture linux/amd64 -v
```

### 4. 安全扫描

```bash
# 运行安全扫描
make security-local

# 查看详细报告
./scripts/test-local.sh security
```

## 故障排除

### Act相关问题

**问题**: `act` 无法下载GitHub Actions
```
解决方案:
1. 检查网络连接
2. 使用代理或VPN
3. 使用本地替代镜像
```

**问题**: Apple Silicon架构兼容性
```bash
# 解决方案：指定容器架构
act -j test --container-architecture linux/amd64
```

### Docker相关问题

**问题**: Docker构建失败
```bash
# 检查Docker状态
docker ps
docker info

# 清理Docker缓存
docker system prune -f
```

**问题**: 权限错误
```bash
# 添加用户到docker组
sudo usermod -aG docker $USER
newgrp docker
```

### Go相关问题

**问题**: 模块下载失败
```bash
# 设置代理
export GOPROXY=https://goproxy.cn,direct

# 清理模块缓存
go clean -modcache
go mod download
```

### 构建约束问题

**问题**: 跨平台构建失败
```bash
# 解决方案：使用Makefile构建
make build

# 或者设置CGO_ENABLED=0
CGO_ENABLED=0 go build -o bin/streamASR .
```

## 性能优化

### 1. 并行测试

```bash
# 并行运行测试
go test -parallel 4 ./...
```

### 2. 缓存利用

```bash
# 使用Go模块缓存
go mod download

# Docker构建缓存
docker build --cache-from streamasr:latest .
```

### 3. 选择性测试

```bash
# 只运行特定包的测试
go test ./pkg/...

# 运行特定测试
go test -run TestWAV ./pkg/wav/
```

## 集成到IDE

### VS Code

在 `.vscode/tasks.json` 中添加任务：

```json
{
    "version": "2.0.0",
    "tasks": [
        {
            "label": "Run Local CI",
            "type": "shell",
            "command": "make ci-local",
            "group": "test",
            "presentation": {
                "echo": true,
                "reveal": "always",
                "focus": false,
                "panel": "shared"
            }
        }
    ]
}
```

### IntelliJ/GoLand

创建运行配置：
- **Go Test**: 运行 `make test-local`
- **Docker**: 运行 `make docker-local`
- **Shell Script**: 运行 `make ci-local`

## 最佳实践

1. **提交前运行**: 始终在提交前运行 `make ci-local`
2. **定期更新**: 定期更新act工具和Docker镜像
3. **监控输出**: 注意测试和安全扫描的输出
4. **版本一致**: 确保本地Go版本与CI一致
5. **缓存管理**: 合理使用构建缓存提高效率

## 贡献指南

如果你为项目添加了新的CI功能：

1. 更新本地测试脚本
2. 添加相应的Makefile目标
3. 更新本文档
4. 测试本地和远程CI的一致性

## 相关链接

- [Act官方文档](https://github.com/nektos/act)
- [GitHub Actions文档](https://docs.github.com/en/actions)
- [Docker官方文档](https://docs.docker.com/)
- [Go测试文档](https://golang.org/pkg/testing/)