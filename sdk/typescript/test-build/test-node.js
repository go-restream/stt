#!/usr/bin/env node

// Node.js测试脚本 - 测试构建后的模块
try {
    // 测试CommonJS模块
    console.log('Testing CommonJS module...');
    const StreamASR = require('../dist/index.js');

    if (StreamASR && StreamASR.StreamASRClient) {
        console.log('✅ StreamASRClient class loaded successfully');

        // 测试静态方法
        if (typeof StreamASR.StreamASRClient.isSupported === 'function') {
            console.log('✅ isSupported method available');
        } else {
            console.log('❌ isSupported method not found');
        }

        // 尝试创建实例
        try {
            const client = new StreamASR.StreamASRClient({
                url: 'ws://localhost:8080/v1/realtime',
                apiKey: 'test-key',
                enableLogging: true
            });
            console.log('✅ Client instance created successfully');
        } catch (error) {
            console.log('❌ Failed to create client instance:', error.message);
        }
    } else {
        console.log('❌ StreamASRClient class not found');
    }

} catch (error) {
    console.log('❌ Failed to load CommonJS module:', error.message);
}

console.log('\nNode.js test completed.');
