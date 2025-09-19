import { Client, Session } from '@heroiclabs/nakama-js';

const NAKAMA_HOST = (process.env.NAKAMA_HOST as string) || '127.0.0.1';
const NAKAMA_PORT = Number(process.env.NAKAMA_PORT || 7350);
const USE_SSL = Boolean(process.env.NAKAMA_SSL || false);
const SERVER_KEY = 'defaultkey';

let client: Client | null = null;
let session: Session | null = null;

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

