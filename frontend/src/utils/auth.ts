import type { AuthRole, AuthSession } from '../types/auth';
import { normalizeUserRole } from '../types/user';

const AUTH_STORAGE_KEY = 'auth_session';
const LEGACY_TOKEN_KEY = 'token';

export type LoginRedirectReason = 'session_expired';

interface BuildLoginPathOptions {
  redirectTo?: string | null;
  reason?: LoginRedirectReason | null;
}

interface StoredAuthSession {
  token?: string;
  expiresAt?: string | null;
  expires_at?: string | null;
  userId?: number | null;
  user_id?: number | null;
  role?: AuthRole | null;
  username?: string | null;
}

function readStoredNumber(...values: Array<number | null | undefined>) {
  const matched = values.find((value) => typeof value === 'number' && Number.isFinite(value));
  return typeof matched === 'number' ? matched : null;
}

function getLocalStorage(): Storage | null {
  if (typeof window === 'undefined' || !window.localStorage) {
    return null;
  }

  return window.localStorage;
}

export function getStoredAuthSession(): AuthSession | null {
  const storage = getLocalStorage();
  if (!storage) {
    return null;
  }

  try {
    const rawSession = storage.getItem(AUTH_STORAGE_KEY);
    if (rawSession) {
      const parsedSession = JSON.parse(rawSession) as StoredAuthSession;
      if (typeof parsedSession.token === 'string' && parsedSession.token) {
        const normalizedRole =
          typeof parsedSession.role === 'string' && parsedSession.role
            ? normalizeUserRole(parsedSession.role)
            : null;

        return {
          token: parsedSession.token,
          expiresAt: parsedSession.expiresAt ?? parsedSession.expires_at ?? null,
          userId: readStoredNumber(parsedSession.userId, parsedSession.user_id),
          role: normalizedRole,
          username: typeof parsedSession.username === 'string' ? parsedSession.username : null,
        };
      }
    }
  } catch {
    storage.removeItem(AUTH_STORAGE_KEY);
  }

  const legacyToken = storage.getItem(LEGACY_TOKEN_KEY);
  if (!legacyToken) {
    return null;
  }

  return {
    token: legacyToken,
    expiresAt: null,
    userId: null,
    role: null,
    username: null,
  };
}

export function setStoredAuthSession(session: AuthSession) {
  const storage = getLocalStorage();
  if (!storage) {
    return;
  }

  storage.setItem(AUTH_STORAGE_KEY, JSON.stringify(session));
  storage.setItem(LEGACY_TOKEN_KEY, session.token);
}

export function clearStoredAuthSession() {
  const storage = getLocalStorage();
  if (!storage) {
    return;
  }

  storage.removeItem(AUTH_STORAGE_KEY);
  storage.removeItem(LEGACY_TOKEN_KEY);
}

export function getDefaultAuthenticatedPath(role?: AuthRole | null) {
  return role === 'MERCHANT' ? '/merchant/dashboard' : '/';
}

export function normalizeRedirectPath(path?: string | null) {
  if (!path || !path.startsWith('/') || path.startsWith('//') || path.startsWith('/login')) {
    return null;
  }

  return path;
}

export function buildLoginPath(options: BuildLoginPathOptions = {}) {
  const searchParams = new URLSearchParams();
  const redirectPath = normalizeRedirectPath(options.redirectTo);

  if (redirectPath) {
    searchParams.set('redirect', redirectPath);
  }

  if (options.reason) {
    searchParams.set('reason', options.reason);
  }

  const queryString = searchParams.toString();
  return queryString ? `/login?${queryString}` : '/login';
}

export function getPostLoginPath(role: AuthRole | null, requestedPath?: string | null) {
  const normalizedRequestedPath = normalizeRedirectPath(requestedPath);

  if (!normalizedRequestedPath) {
    return getDefaultAuthenticatedPath(role);
  }

  if (normalizedRequestedPath.startsWith('/merchant') && role !== 'MERCHANT') {
    return getDefaultAuthenticatedPath(role);
  }

  return normalizedRequestedPath;
}
