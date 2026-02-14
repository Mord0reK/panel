import { WebSocketServer } from 'ws';

const PORT = process.env.PORT || 8080;

const wss = new WebSocketServer({ port: PORT });

console.log(`WebSocket server running on ws://localhost:${PORT}`);

wss.on('connection', (ws) => {
  console.log('Client connected');

  ws.on('message', (message) => {
    try {
      const data = JSON.parse(message);
      console.log('Received:', JSON.stringify(data, null, 2));

      if (data.type === 'metrics') {
        console.log('--- METRICS ---');
        console.log(JSON.stringify(data.data, null, 2));
      } else if (data.type === 'result') {
        console.log('--- RESULT ---');
        console.log(JSON.stringify(data.data, null, 2));
      }
    } catch (e) {
      console.log('Raw message:', message.toString());
    }
  });

  ws.on('close', () => {
    console.log('Client disconnected');
  });

  ws.send(JSON.stringify({ type: 'welcome', message: 'Connected to WebSocket server' }));

  setInterval(() => {
    if (ws.readyState === 1) {
      ws.send(JSON.stringify({
        type: 'command',
        action: 'info'
      }));
    }
  }, 5000);
});

console.log('\nAvailable commands to send manually:');
console.log('  {"type":"command","action":"stats"}');
console.log('  {"type":"command","action":"info"}');
console.log('  {"type":"command","action":"stop","target":"container-name"}');
console.log('  {"type":"command","action":"start","target":"container-name"}');
console.log('  {"type":"command","action":"restart","target":"container-name"}');
console.log('  {"type":"command","action":"check-updates"}');
console.log('  {"type":"command","action":"update","target":"project-or-container"}');
