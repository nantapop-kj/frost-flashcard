const wails = await import('/wails/runtime.js');
const module = 'github.com/nantapop-kj/frost-flashcard';

const service = (pkg, type) => {
  const prefix = `${module}/${pkg}.${type}`;

  return (method, ...args) => wails.Call.ByName(`${prefix}.${method}`, ...args);
};

export const callService = service('internal/services', 'Service');

export const errorCode = (err) => err?.cause?.code ?? 'unknown';
