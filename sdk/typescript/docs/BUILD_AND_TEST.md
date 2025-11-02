# StreamASR TypeScript SDK - 构建和测试指南

## 概述

这个指南介绍如何构建和测试 StreamASR TypeScript SDK。构建流程会自动生成多个测试文件来验证SDK的功能。

## 快速开始

### 1. 安装依赖

```bash
npm install
```

### 2. 构建SDK和测试文件

```bash
npm run build
```

这个命令会：
- 编译TypeScript代码
- 生成UMD、ESM和React三种格式的JS文件
- 自动创建测试文件到 `test-build/` 目录

### 3. 启动测试服务器

```bash
npm run serve:test
```

这会启动一个HTTP服务器并自动打开浏览器到测试页面 (http://localhost:3003/test-build/basic-usage.html)。

## 构建输出

### 构建文件 (`dist/` 目录)

- `index.js` - UMD格式，适合浏览器直接使用
- `index.esm.js` - ES模块格式，适合现代构建工具
- `index.d.ts` - TypeScript类型定义文件
- `react.js` - React hooks的ES模块

### 测试文件 (`test-build/` 目录)

- `basic-usage.html` - 原生JavaScript测试页面
- `react-test.html` - React hooks测试页面
- `test-node.js` - Node.js CommonJS模块测试
- `README.md` - 测试说明文档
- `build-info.json` - 构建信息

## 测试功能

### 1. 基础功能测试 (`basic-usage.html`)

- SDK加载检测
- WebSocket连接测试
- 音频录制和转录
- 实时状态监控
- 错误处理

**使用方法：**
1. 确保StreamASR服务器运行在 `ws://localhost:8080/v1/realtime`
2. 修改页面中的API密钥（如果需要）
3. 点击"Connect"连接服务器
4. 点击"Start Recording"开始录音测试

### 2. React Hook测试 (`react-test.html`)

- React组件集成测试
- Hook状态管理测试
- 生命周期管理测试

**使用方法：**
在浏览器中打开该文件，使用页面上的控制按钮进行测试。

### 3. Node.js模块测试 (`test-node.js`)

```bash
node test-build/test-node.js
```

测试内容：
- CommonJS模块加载
- 类实例化
- 静态方法调用

## 开发流程

### 开发模式（监听文件变化）

```bash
npm run dev
```

这个命令会启动Rollup的监听模式，自动重新构建修改的文件。

### 类型检查

```bash
npm run type-check
```

只进行TypeScript类型检查，不生成文件。

### 代码检查

```bash
npm run lint        # 检查代码风格
npm run lint:fix    # 自动修复可修复的问题
```

## 集成到项目

### 浏览器环境

```html
<script src="path/to/dist/index.js"></script>
<script>
  const client = new StreamASR.StreamASRClient({
    url: 'ws://localhost:8080/v1/realtime',
    apiKey: 'your-api-key'
  });
  // 使用客户端...
</script>
```

### Node.js环境

```javascript
const StreamASR = require('streamasr-sdk');
const client = new StreamASR.StreamASRClient({
  url: 'ws://localhost:8080/v1/realtime',
  apiKey: 'your-api-key'
});
// 使用客户端...
```

### ES模块环境

```javascript
import { StreamASRClient } from 'streamasr-sdk';
const client = new StreamASRClient({
  url: 'ws://localhost:8080/v1/realtime',
  apiKey: 'your-api-key'
});
// 使用客户端...
```

### React环境

```javascript
import { useStreamASR } from 'streamasr-sdk/react';

function MyComponent() {
  const {
    isConnected,
    isRecording,
    transcript,
    connect,
    disconnect,
    startRecording,
    stopRecording
  } = useStreamASR({
    apiKey: 'your-api-key',
    url: 'ws://localhost:8080/v1/realtime'
  });

  // 使用hooks...
}
```

## 故障排除

### 常见问题

1. **WebSocket连接失败**
   - 确保服务器正在运行
   - 检查URL和端口是否正确
   - 检查防火墙设置

2. **音频权限问题**
   - 确保浏览器有麦克风权限
   - 使用HTTPS或localhost环境

3. **模块加载问题**
   - 检查构建文件是否正确生成
   - 验证文件路径是否正确

4. **React hooks问题**
   - 确保React版本 >= 16.8
   - 检查是否在函数组件中使用

### 调试技巧

1. 开启详细日志：
   ```javascript
   const client = new StreamASR.StreamASRClient({
     enableLogging: true
   });
   ```

2. 使用浏览器开发者工具查看WebSocket消息

3. 检查控制台错误信息

## 发布

构建完成后，发布包到npm：

```bash
npm run prepublishOnly  # 确保构建完成
npm publish
```

## 贡献

1. Fork项目
2. 创建功能分支
3. 提交更改
4. 运行测试确保没有问题
5. 提交Pull Request

---

更多信息请参考项目根目录的README.md文件。