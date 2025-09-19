import { Client, Session, Socket, WebSocketAdapterText } from '@heroiclabs/nakama-js';

const NAKAMA_HOST = (process.env.NAKAMA_HOST as string) || '127.0.0.1';
const NAKAMA_PORT = Number(process.env.NAKAMA_PORT || 7350);
const USE_SSL = Boolean(process.env.NAKAMA_SSL || false);
const SERVER_KEY = 'defaultkey';

let client: Client | null = null;
let session: Session | null = null;
let socket: Socket | null = null;

export function getClient() {
  if (!client) client = new Client(SERVER_KEY, NAKAMA_HOST, NAKAMA_PORT, USE_SSL);
  return client;
}

export function restoreSessionFromToken(token: string) {
  session = Session.restore(token);
  return session;
}

export async function rpcDeviceAuth(payload: { device_id: string; nickname?: string }) {
  const c = getClient();
  const res = await c.rpc(null, 'auth_device', payload);
  const body = typeof res.payload === 'string' ? JSON.parse(res.payload as string) : (res.payload as any);
  return { token: body.token as string, username: body.username as string };
}

export function getSocket() {
  if (!socket) {
    socket = Socket.from(getClient(), new WebSocketAdapterText(), 30, true);
  }
  return socket;
}

export async function connectSocketWithRetry(tok: string, onClose?: () => void) {
  const s = getSocket();
  const sess = restoreSessionFromToken(tok);
  let attempt = 0;
  const maxDelay = 10000;
  async function connect() {
    try {
      await s.connect(sess, false);
    } catch (e) {
      attempt += 1;
      const delay = Math.min(1000 * Math.pow(2, attempt), maxDelay);
      setTimeout(connect, delay);
    }
  }
  s.onclose = () => {
    onClose && onClose();
    attempt = 0;
    connect();
  };
  await connect();
  return s;
}

