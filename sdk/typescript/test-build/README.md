# StreamASR SDK Build Test

这个目录包含了构建后的SDK文件和测试页面。

## 文件说明

- `basic-usage.html` - 使用构建后JS文件的测试页面
- `../dist/` - 构建后的JavaScript文件目录

## 使用方法

1. 确保已运行 `npm run build` 构建SDK
2. 运行 `npm run serve:test` 启动测试服务器
3. 在浏览器中访问 http://localhost:3001/basic-usage.html
4. 或者直接在浏览器中打开 basic-usage.html 文件

## 测试功能

- SDK加载测试
- WebSocket连接测试
- 音频录制和转录测试
- 实时状态监控
- 错误处理测试

## 服务器要求

确保StreamASR服务器正在运行并监听 `ws://localhost:8080/v1/realtime`

如果使用其他地址，请在页面中修改WebSocket URL。
