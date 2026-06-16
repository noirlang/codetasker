/**
 * NotificationBell — Global notification bell in the top-right of the app.
 *
 * Features:
 * - Bell icon (lucide-react) with red badge for unread count
 * - Click to open/close a dark glass dropdown panel
 * - Each notification shows icon (by type), title, message, and time ago
 * - "Mark all as read" button at the top
 * - Click individual notification to mark it as read
 * - Polls for new notifications every 30 seconds
 * - Pulse animation on bell when there are unread notifications
 */

import { useState, useEffect, useRef, useCallback } from 'react';
import {
  Bell,
  GitMerge,
  MessageSquare,
  UserCheck,
  CheckCheck,
  X,
} from 'lucide-react';
import { notificationsApi } from '../api/client';
import type { Notification } from '../types';

// ── Helpers ──────────────────────────────────────────────────────────────────

/** Format an ISO timestamp as a relative time string */
function timeAgo(isoString: string): string {
  const date = new Date(isoString);
  const now = new Date();
  const diff = now.getTime() - date.getTime();

  const seconds = Math.floor(diff / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);

  if (seconds < 60) return 'just now';
  if (minutes < 60) return `${minutes}m ago`;
  if (hours < 24) return `${hours}h ago`;
  if (days < 7) return `${days}d ago`;
  return `${Math.floor(days / 7)}w ago`;
}

/** Return the appropriate icon component for a notification type */
function NotificationIcon({ type }: { type: Notification['type'] }) {
  const base = 'shrink-0';
  if (type === 'task_assigned') return <UserCheck size={14} className={`${base} text-[#a0a0a0]`} />;
  if (type === 'comment_added') return <MessageSquare size={14} className={`${base} text-[#a0a0a0]`} />;
  if (type === 'pr_merged')     return <GitMerge size={14} className={`${base} text-[#a0a0a0]`} />;
  return <Bell size={14} className={`${base} text-[#a0a0a0]`} />;
}

// ── Component ─────────────────────────────────────────────────────────────────

export default function NotificationBell() {
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const panelRef = useRef<HTMLDivElement>(null);
  const buttonRef = useRef<HTMLButtonElement>(null);

  const unread = notifications.filter((n) => !n.read).length;

  // ── Fetch notifications ────────────────────────────────────────────────────

  const fetchNotifications = useCallback(async () => {
    setLoading(true);
    try {
      const data = await notificationsApi.list();
      setNotifications(data);
    } catch {
      // Silently fail — bell is non-critical
    } finally {
      setLoading(false);
    }
  }, []);

  // Poll every 30 seconds
  useEffect(() => {
    fetchNotifications();
    const interval = setInterval(fetchNotifications, 30_000);
    return () => clearInterval(interval);
  }, [fetchNotifications]);

  // ── Close on outside click ─────────────────────────────────────────────────

  useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => {
      if (
        panelRef.current &&
        !panelRef.current.contains(e.target as Node) &&
        buttonRef.current &&
        !buttonRef.current.contains(e.target as Node)
      ) {
        setOpen(false);
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, [open]);

  // ── Actions ────────────────────────────────────────────────────────────────

  const handleMarkRead = async (id: string) => {
    // Optimistically mark as read locally
    setNotifications((prev) =>
      prev.map((n) => (n.id === id ? { ...n, read: true } : n))
    );
    try {
      await notificationsApi.markRead(id);
    } catch {
      // Revert if fails
      setNotifications((prev) =>
        prev.map((n) => (n.id === id ? { ...n, read: false } : n))
      );
    }
  };

  const handleMarkAllRead = async () => {
    // Optimistically mark all read
    setNotifications((prev) => prev.map((n) => ({ ...n, read: true })));
    try {
      await notificationsApi.markAllRead();
    } catch {
      // Refetch on failure
      fetchNotifications();
    }
  };

  // ── Render ────────────────────────────────────────────────────────────────

  return (
    <div className="relative">
      {/* Bell button */}
      <button
        ref={buttonRef}
        onClick={() => setOpen((v) => !v)}
        className={[
          'relative flex h-8 w-8 items-center justify-center rounded',
          'border border-[#2a2a2a] bg-transparent',
          'text-[#666666] hover:text-white hover:border-[#3a3a3a]',
          'transition-all duration-150 cursor-pointer',
        ].join(' ')}
        title="Notifications"
        aria-label={`Notifications${unread > 0 ? ` (${unread} unread)` : ''}`}
      >
        {/* Pulse ring when there are unread notifications */}
        {unread > 0 && (
          <span className="absolute inset-0 rounded animate-ping opacity-20 bg-white pointer-events-none" />
        )}

        <Bell size={14} className={unread > 0 ? 'text-white' : ''} />

        {/* Unread badge */}
        {unread > 0 && (
          <span
            className={[
              'absolute -top-1 -right-1 flex h-4 min-w-4 items-center justify-center',
              'rounded-full bg-red-500 px-1',
              'font-mono text-[9px] font-bold text-white',
              'animate__animated animate__bounceIn',
            ].join(' ')}
            style={{ animationDuration: '0.3s' }}
          >
            {unread > 99 ? '99+' : unread}
          </span>
        )}
      </button>

      {/* Dropdown panel */}
      {open && (
        <div
          ref={panelRef}
          className={[
            'absolute right-0 top-full z-50 mt-2',
            'w-80 rounded border border-[#222222] bg-[#111111]',
            'shadow-2xl shadow-black/60',
            'animate__animated animate__fadeInDown',
          ].join(' ')}
          style={{ animationDuration: '0.18s' }}
        >
          {/* Panel header */}
          <div className="flex items-center justify-between border-b border-[#1a1a1a] px-4 py-2.5">
            <div className="flex items-center gap-2">
              <Bell size={13} className="text-[#a0a0a0]" />
              <span className="text-[11px] font-semibold text-white">Notifications</span>
              {unread > 0 && (
                <span className="rounded border border-red-500/30 bg-red-500/10 px-1.5 py-0.5 font-mono text-[9px] text-red-400">
                  {unread} unread
                </span>
              )}
            </div>
            <div className="flex items-center gap-1">
              {unread > 0 && (
                <button
                  onClick={handleMarkAllRead}
                  className="flex items-center gap-1 rounded border border-[#2a2a2a] px-2 py-0.5 text-[9px] text-[#666666] hover:text-white hover:border-[#3a3a3a] transition-all cursor-pointer font-mono"
                  title="Mark all as read"
                >
                  <CheckCheck size={10} />
                  All read
                </button>
              )}
              <button
                onClick={() => setOpen(false)}
                className="rounded p-1 text-[#666666] hover:text-white transition-colors cursor-pointer"
              >
                <X size={12} />
              </button>
            </div>
          </div>

          {/* Notifications list */}
          <div className="max-h-[360px] overflow-y-auto">
            {loading && notifications.length === 0 ? (
              <div className="flex items-center justify-center py-8">
                <span className="text-[11px] text-[#666666]">Loading…</span>
              </div>
            ) : notifications.length === 0 ? (
              <div className="flex flex-col items-center justify-center gap-2 py-10">
                <Bell size={24} className="text-[#333333]" />
                <span className="text-[11px] text-[#666666]">No notifications</span>
              </div>
            ) : (
              notifications.map((notification) => (
                <button
                  key={notification.id}
                  onClick={() => {
                    if (!notification.read) handleMarkRead(notification.id);
                    if (notification.link) {
                      window.open(notification.link, '_blank', 'noopener,noreferrer');
                    }
                  }}
                  className={[
                    'flex w-full items-start gap-3 px-4 py-3 text-left',
                    'border-b border-[#1a1a1a] last:border-b-0',
                    'hover:bg-[#161616] transition-colors duration-100 cursor-pointer',
                    !notification.read ? 'bg-white/[0.02]' : '',
                  ].join(' ')}
                >
                  {/* Icon */}
                  <div className="mt-0.5 shrink-0">
                    <NotificationIcon type={notification.type} />
                  </div>

                  {/* Content */}
                  <div className="min-w-0 flex-1">
                    <div className="flex items-start justify-between gap-2">
                      <p
                        className={[
                          'text-[11px] font-medium leading-4',
                          notification.read ? 'text-[#a0a0a0]' : 'text-white',
                        ].join(' ')}
                      >
                        {notification.title}
                      </p>
                      <span className="shrink-0 font-mono text-[9px] text-[#666666]">
                        {timeAgo(notification.created_at)}
                      </span>
                    </div>
                    <p className="mt-0.5 line-clamp-2 text-[10px] text-[#666666] leading-4">
                      {notification.message}
                    </p>
                  </div>

                  {/* Unread dot */}
                  {!notification.read && (
                    <span className="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full bg-white" />
                  )}
                </button>
              ))
            )}
          </div>
        </div>
      )}
    </div>
  );
}
