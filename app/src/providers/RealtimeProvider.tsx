import React, { createContext, useContext, useEffect, useMemo, useRef, useState } from 'react';
import { Socket } from '@heroiclabs/nakama-js';
import { connectSocketWithRetry, getSocket, joinQueue, leaveQueue } from '@api/nakama';
import { useAuth } from './AuthProvider';

interface RealtimeContextValue {
  socket: Socket | null;
  isConnected: boolean;
  enqueue: (mode: 'casual' | 'ranked') => Promise<string>;
  dequeue: (ticket: string) => Promise<void>;
}

const RealtimeContext = createContext<RealtimeContextValue | undefined>(undefined);

export const RealtimeProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { token } = useAuth();
  const [isConnected, setConnected] = useState(false);
  const socketRef = useRef<Socket | null>(null);

  useEffect(() => {
    let canceled = false;
    async function run() {
      if (!token) { setConnected(false); return; }
      const s = await connectSocketWithRetry(token, () => setConnected(false));
      if (canceled) return;
      socketRef.current = s;
      setConnected(true);
    }
    run();
    return () => { canceled = true; };
  }, [token]);

  const enqueue = async (mode: 'casual' | 'ranked') => {
    return joinQueue(mode).then(t => t.ticket);
  };
  const dequeue = async (ticket: string) => {
    await leaveQueue(ticket);
  };

  const value = useMemo(() => ({ socket: socketRef.current, isConnected, enqueue, dequeue }), [isConnected]);
  return <RealtimeContext.Provider value={value}>{children}</RealtimeContext.Provider>;
};

export function useRealtime() {
  const ctx = useContext(RealtimeContext);
  if (!ctx) throw new Error('useRealtime must be used within RealtimeProvider');
  return ctx;
}

