import React, { createContext, useCallback, useContext, useMemo, useState } from 'react';
import { rpcDeviceAuth, restoreSessionFromToken } from '@api/nakama';
import { generateUUID } from '@utils/uuid';

interface AuthContextValue {
  isAuthed: boolean;
  token: string | null;
  username: string | null;
  login: () => Promise<void>;
  logout: () => void;
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [token, setToken] = useState<string | null>(null);
  const [username, setUsername] = useState<string | null>(null);

  const login = useCallback(async () => {
    const deviceIdKey = 'device_id';
    let deviceId = typeof localStorage !== 'undefined' ? localStorage.getItem(deviceIdKey) : null;
    if (!deviceId) {
      deviceId = generateUUID();
      if (typeof localStorage !== 'undefined') localStorage.setItem(deviceIdKey, deviceId);
    }
    const nickname = 'Player-' + String(deviceId).slice(0, 6);
    const { token: tkn, username: uname } = await rpcDeviceAuth({ device_id: deviceId!, nickname });
    setToken(tkn);
    setUsername(uname);
    restoreSessionFromToken(tkn);
  }, []);

  const logout = useCallback(() => {
    setToken(null);
    setUsername(null);
  }, []);

  const value = useMemo(() => ({ isAuthed: !!token, token, username, login, logout }), [login, logout, token, username]);

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}

