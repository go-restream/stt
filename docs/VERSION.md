# 版本管理指南

**🌐 语言:** [English](VERSION_EN.md) | [中文](VERSION.md)

本文档介绍 streamASR 项目的版本管理规范和使用方法。

## 版本格式

项目采用语义化版本控制 (Semantic Versioning) 格式：`v主版本号.次版本号.修订号`

- **主版本号 (Major)**：不兼容的 API 修改
- **次版本号 (Minor)**：向下兼容的功能性新增
- **修订号 (Patch)**：向下兼容的问题修正

当前版本：**v0.1.1**

## 版本管理命令

### 查看版本信息

```bash
# 查看当前版本信息
make version

# 仅显示版本号
make version-show

# 查看所有 Git 标签
make tag-list
```

### 版本升级

```bash
# 修订版本升级 (v0.1.1 -> v0.1.2)
make version-bump-patch

# 次版本升级 (v0.1.1 -> v0.2.0)
make version-bump-minor

# 主版本升级 (v0.1.1 -> v1.0.0)
make version-bump-major

# 手动设置版本
make version-set NEW_VERSION=v0.2.0
```

### Git 标签管理

```bash
# 创建当前版本的 Git 标签
make tag

# 查看所有版本标签
make tag-list
```

## 构建时的版本信息

版本信息会在构建时自动注入到二进制文件中：

- `version`: 版本号 (从 VERSION 文件读取)
- `buildTime`: 构建时间 (ISO 8601 格式)
- `gitCommit`: Git 提交哈希 (短格式)

### 运行时查看版本

```bash
# 构建后查看版本
./build/streamASR -v
./build/streamASR --version

# 启动日志中也会显示版本信息
./build/streamASR -c config.yaml
```

## Docker 版本管理

### 构建带版本的 Docker 镜像

```bash
# 构建生产版本镜像 (会自动打两个标签)
make docker-build
# 生成：streamasr:latest 和 streamasr:v0.1.1

# 构建开发版本镜像
make docker-build-dev
# 生成：streamasr:dev

# 通过 docker-compose 构建版本化镜像
make docker-compose-build
```

### Docker 镜像版本标签

Docker 镜像会使用以下标签策略：

- `streamasr:latest` - 最新版本
- `streamasr:v0.1.1` - 具体版本号
- `streamasr:dev` - 开发版本

## 发布流程

### 开发版本发布

1. 更新版本号：
   ```bash
   make version-bump-patch
   ```

2. 构建和测试：
   ```bash
   make clean
   make build
   make test
   ```

3. 创建 Git 标签：
   ```bash
   make tag
   ```

4. 构建 Docker 镜像：
   ```bash
   make docker-build
   ```

### 生产版本发布

1. 更新版本号（根据修改类型选择）：
   ```bash
   make version-bump-patch    # 修复版本
   make version-bump-minor    # 功能版本
   make version-bump-major    # 破坏性变更版本
   ```

2. 提交代码变更：
   ```bash
   git add VERSION
   git commit -m "Bump version to v0.1.2"
   ```

3. 创建标签：
   ```bash
   make tag
   ```

4. 构建生产镜像：
   ```bash
   make docker-deploy
   ```

## 版本示例

### 修订版本发布 (v0.1.1 -> v0.1.2)

```bash
# 1. 升级修订版本
make version-bump-patch

# 2. 提交变更
git add VERSION
git commit -m "Bump version to v0.1.2"

# 3. 创建标签
make tag

# 4. 构建和部署
make docker-deploy
```

### 次版本发布 (v0.1.1 -> v0.2.0)

```bash
# 1. 升级次版本
make version-bump-minor

# 2. 提交变更
git add VERSION
git commit -m "Add new feature, bump version to v0.2.0"

# 3. 创建标签
make tag

# 4. 构建和部署
make docker-deploy
```

## 注意事项

1. **版本文件**：版本信息存储在 `VERSION` 文件中，不要手动编辑
2. **Git 状态**：确保工作目录是干净的状态再创建标签
3. **标签推送**：`make tag` 会自动推送标签到远程仓库
4. **构建顺序**：先升级版本，再构建镜像，确保版本信息正确
5. **版本回退**：如需回退版本，可以：
   ```bash
   git checkout v0.1.1  # 检出到特定标签
   make version-set NEW_VERSION=v0.1.1  # 重置版本文件
   ```

## 环境变量

在 CI/CD 环境中，可以通过环境变量覆盖版本信息：

```bash
export VERSION="v0.1.2-custom"
export BUILD_TIME="2024-01-01T00:00:00Z"
export GIT_COMMIT="abc123def"

make docker-build
```