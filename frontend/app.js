import { callService } from './lib/api.js';

document.getElementById('ping').onclick = async () => {
  try {
    const msg = await callService('Ping');
    document.getElementById('result').textContent = msg;
  } catch (err) {
    document.getElementById('result').textContent = 'error: ' + err;
  }
};
