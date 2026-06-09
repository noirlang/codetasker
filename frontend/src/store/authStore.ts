/**
 * CodeTasker — Auth Zustand Store
 *
 * Manages the authenticated user state.
 * Session is maintained via an httpOnly cookie set by the Go backend;
 * we simply call /api/auth/me to hydrate the user on each page load.
 */

import { create } from 'zustand';
import { authApi } from '../api/client';
import type { User } from '../types';

// ── State shape ────────────────────────────────────────────────────────────

interface AuthState {
  /** The currently authenticated user, or null if not logged in. */
  user: User | null;

  /** True while the initial /api/auth/me fetch is in flight. */
  isLoading: boolean;

  /** Derived flag: true when user is non-null. */
  isAuthenticated: boolean;

  /**
   * Fetch the current user from the backend using the session cookie.
   * Call this once on application startup.
   */
  fetchUser: () => Promise<void>;

  /**
   * Log out the current user: clear server session and local state,
   * then redirect to /login.
   */
  logout: () => Promise<void>;
}

// ── Store implementation ───────────────────────────────────────────────────

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  isLoading: true,         // start as true so App.tsx shows a spinner
  isAuthenticated: false,

  fetchUser: async () => {
    set({ isLoading: true });
    try {
      const user = await authApi.getMe();
      set({ user, isAuthenticated: true, isLoading: false });
    } catch {
      // Not authenticated or network error — clear state and let the router redirect
      set({ user: null, isAuthenticated: false, isLoading: false });
    }
  },

  logout: async () => {
    try {
      await authApi.logout();
    } catch {
      // Best-effort: even if the server call fails, clear local state
    } finally {
      set({ user: null, isAuthenticated: false });
      window.location.href = '/login';
    }
  },
}));
