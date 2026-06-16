/**
 * App — Root component: routing + session hydration.
 *
 * On mount, calls authStore.fetchUser() to restore session from the
 * httpOnly cookie. While loading, renders a full-screen spinner.
 * Once resolved, the Router renders the correct page based on auth state.
 */

import { useEffect } from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import { useAuthStore } from './store/authStore';
import Login from './components/Login';
import Dashboard from './components/Dashboard';
import RepoView from './components/RepoView';
import Spinner from './components/ui/Spinner';

// ── Protected route wrapper ────────────────────────────────────────────────

/**
 * Wraps a route that requires authentication.
 * If the user is not authenticated (after loading), redirect to /login.
 */
function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isLoading } = useAuthStore();

  // Still determining auth state — render nothing (parent handles global spinner)
  if (isLoading) return null;

  // Not authenticated → redirect
  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  return <>{children}</>;
}

// ── App component ──────────────────────────────────────────────────────────

export default function App() {
  const { fetchUser, isLoading } = useAuthStore();

  // Hydrate the session on every page load.
  // fetchUser sets isLoading=false whether or not the user is logged in.
  useEffect(() => {
    fetchUser();
  }, [fetchUser]);



  // ── Global loading state: full-screen spinner ──────────────────────────
  if (isLoading) {
    return (
      <div
        className="flex h-screen items-center justify-center"
        style={{ backgroundColor: '#0a0a0a' }}
        aria-label="Loading application…"
      >
        <div className="flex flex-col items-center gap-4">
          <Spinner size={36} />
          <span
            className="text-sm tracking-[0.2em] text-white/90 select-none font-medium"
            style={{ fontFamily: "'Camiro', serif" }}
          >
            &lt;/ CODETASKER &gt;
          </span>
        </div>
      </div>
    );
  }

  // ── Route tree ─────────────────────────────────────────────────────────

  return (
    <Routes>
      {/* Public routes */}
      <Route path="/login" element={<Login />} />

      {/* Root → dashboard redirect */}
      <Route path="/" element={<Navigate to="/dashboard" replace />} />

      {/* Protected routes */}
      <Route
        path="/dashboard"
        element={
          <ProtectedRoute>
            <Dashboard />
          </ProtectedRoute>
        }
      />
      <Route
        path="/repos/:owner/:repo"
        element={
          <ProtectedRoute>
            <RepoView />
          </ProtectedRoute>
        }
      />

      {/* Catch-all → dashboard */}
      <Route path="*" element={<Navigate to="/dashboard" replace />} />
    </Routes>
  );
}
