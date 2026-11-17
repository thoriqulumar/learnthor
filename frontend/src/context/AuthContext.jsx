import { createContext, useContext, useState, useEffect } from 'react';
import api from '../lib/api';

const AuthContext = createContext(null);

// Decode the payload of a JWT (no signature verification — just for expiry check)
function jwtExpiry(token) {
  try {
    const payload = JSON.parse(atob(token.split('.')[1]));
    return payload.exp ? payload.exp * 1000 : null; // ms since epoch
  } catch {
    return null;
  }
}

function isTokenValid(token) {
  if (!token) return false;
  const exp = jwtExpiry(token);
  if (exp === null) return true; // no expiry field → treat as valid
  return Date.now() < exp;
}

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const token = localStorage.getItem('token');
    const stored = localStorage.getItem('user');
    if (token && stored && isTokenValid(token)) {
      setUser(JSON.parse(stored)); // eslint-disable-line react-hooks/set-state-in-effect -- restore session on mount
    } else if (token) {
      // Token present but expired — clear everything
      localStorage.removeItem('token');
      localStorage.removeItem('user');
    }
    setLoading(false);
  }, []);

  async function register(username, email, password) {
    const { data } = await api.post('/auth/register', {
      username,
      email,
      password,
    });
    localStorage.setItem('token', data.token);
    localStorage.setItem('user', JSON.stringify(data.user));
    setUser(data.user);
    return data;
  }

  async function login(email, password) {
    const { data } = await api.post('/auth/login', { email, password });
    localStorage.setItem('token', data.token);
    localStorage.setItem('user', JSON.stringify(data.user));
    setUser(data.user);
    return data;
  }

  // refreshUser re-reads the stored user and patches a partial update into
  // state + localStorage without re-issuing a JWT. Used after preference
  // updates (e.g. preferred_gloss) so the new value is visible app-wide.
  function refreshUser(patch) {
    if (!patch) return;
    const next = { ...(user ?? {}), ...patch };
    setUser(next);
    localStorage.setItem('user', JSON.stringify(next));
  }

  function logout() {
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    setUser(null);
  }

  return (
    <AuthContext.Provider
      value={{ user, loading, register, login, logout, refreshUser }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  return useContext(AuthContext);
}
