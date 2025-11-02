# Docker 部署指南

**🌐 语言:** [English](DOCKER_EN.md) | [中文](DOCKER.md)

本文档介绍如何使用 Docker 来构建和运行 streamASR 项目。

## 快速开始

### 使用 docker-compose（推荐）

```bash
# 构建并启动服务
make docker-deploy

# 查看日志
make docker-compose-logs

# 停止服务
make docker-compose-down
```

### 使用原生 Docker 命令

```bash
# 构建镜像
make docker-build

# 运行容器
make docker-run

# 查看日志
make docker-logs

# 停止容器
make docker-stop
```

## 可用的 Makefile 命令

### 基础构建和运行

- `make docker-build` - 构建 Docker 镜像（生产版本）
- `make docker-build-dev` - 构建开发版本镜像
- `make docker-run` - 运行容器
- `make docker-stop` - 停止并删除容器

### Docker Compose 操作

- `make docker-compose-up` - 使用 docker-compose 启动服务
- `make docker-compose-down` - 停止 docker-compose 服务
- `make docker-compose-logs` - 查看服务日志
- `make docker-compose-build` - 使用 docker-compose 构建

### 开发和调试

- `make docker-dev` - 开发环境（构建开发版本并运行）
- `make docker-exec` - 进入运行中的容器
- `make docker-debug` - 以调试模式启动容器（进入交互式 shell）
- `make docker-ps` - 查看容器状态

### 维护操作

- `make docker-clean` - 清理所有 Docker 相关资源
- `make docker-deploy` - 完整部署（构建并启动服务）

## 手动 Docker 命令

### 构建镜像

```bash
# 生产版本
docker build -t streamasr:latest .

# 开发版本
docker build -t streamasr:dev .
```

### 运行容器

```bash
# 运行生产版本
docker run -d --name streamasr-container \
  -p 8088:8088 \
  -v $(pwd)/config/config.yaml:/app/config/config.yaml:ro \
  -v $(pwd)/vad/model:/app/vad/model:ro \
  -v $(pwd)/audio:/app/audio \
  -v $(pwd)/logs:/app/logs \
  streamasr:latest
```

### 使用 docker-compose

```bash
# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f streamASR

# 停止服务
docker-compose down

# 重新构建并启动
docker-compose up -d --build
```

## 配置说明

### 环境变量

可以在 `docker-compose.yml` 中设置以下环境变量：

- `VERSION` - 应用版本
- `BUILD_TIME` - 构建时间
- `GIT_COMMIT` - Git 提交哈希
- `CONFIG_PATH` - 配置文件路径

### 挂载的目录

- `./config/config.yaml` - 应用配置文件（只读）
- `./vad/model` - VAD 模型文件（只读）
- `./audio` - 音频文件存储目录
- `./logs` - 日志文件目录
- `./static` - 静态文件目录（可选）

## 健康检查

容器内置了健康检查功能：

```bash
# 检查容器健康状态
docker ps

# 查看健康检查日志
docker inspect streamasr-container | grep Health -A 10
```

## 故障排除

### 常见问题

1. **端口冲突**
   ```bash
   # 检查端口占用
   lsof -i :8088

   # 使用不同端口
   docker run -p 9088:8088 streamasr:latest
   ```

2. **VAD 模型文件缺失**
   ```bash
   # 确保 VAD 模型文件存在
   ls -la vad/model/

   # 如果模型文件缺失，需要下载相应的模型文件
   ```

3. **权限问题**
   ```bash
   # 确保音频和日志目录有正确的权限
   chmod 755 audio logs
   ```

### 查看日志

```bash
# 查看容器日志
docker logs streamasr-container

# 实时查看日志
docker logs -f streamasr-container

# 查看最近的日志
docker logs --tail 100 streamasr-container
```

## 生产部署建议

1. **使用 docker-compose** 推荐在生产环境中使用 docker-compose 进行服务编排
2. **配置持久化** 确保音频文件和日志目录正确挂载
3. **资源限制** 在生产环境中设置适当的资源限制
4. **日志管理** 配置日志轮转和监控
5. **健康检查** 启用健康检查并配置适当的监控

```yaml
# 示例生产配置
services:
  streamASR:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 1G
        reservations:
          cpus: '1'
          memory: 512M
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```