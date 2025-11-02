#!/usr/bin/env node

const { spawn } = require('child_process');
const path = require('path');

console.log('🚀 Starting StreamASR SDK test server...\n');

const httpServer = spawn('npx', ['http-server', '.', '-p', '3003', '-c-1', '-o', '--cors'], {
    stdio: 'inherit',
    cwd: path.join(__dirname, '..'),
    shell: true
});

httpServer.on('close', (code) => {
    console.log(`\nTest server exited with code ${code}`);
});

process.on('SIGINT', () => {
    console.log('\n🛑 Stopping test server...');
    httpServer.kill('SIGINT');
    process.exit(0);
});

console.log('📁 Serving files from project root');
console.log('🌐 Server will open at: http://localhost:3003');
console.log('📄 Test page: http://localhost:3003/test-build/basic-usage.html');
console.log('⏹️  Press Ctrl+C to stop the server\n');