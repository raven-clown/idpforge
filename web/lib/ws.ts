"use client";

import { useEffect, useRef } from "react";
import { Announcement } from "./api";

export interface WSAuditEvent {
  type: "audit_log";
  actor_id?: string;
  action: string;
  target_resource?: string;
  status: string;
  timestamp: string;
}

export interface WSAnnouncementEvent {
  type: "announcement";
  announcement: Announcement;
}

export type WSEvent = WSAuditEvent | WSAnnouncementEvent;

// Subscribes to the realtime feed (audit events + announcements) for the
// lifetime of the calling component. Reconnects with backoff on drop --
// the connection is a nice-to-have live layer, never the source of truth,
// so silently retrying is the right failure mode.
export function useRealtime(onEvent: (e: WSEvent) => void) {
  const onEventRef = useRef(onEvent);
  useEffect(() => {
    onEventRef.current = onEvent;
  }, [onEvent]);

  useEffect(() => {
    let ws: WebSocket | null = null;
    let closed = false;
    let retryDelay = 1000;
    let retryTimer: ReturnType<typeof setTimeout> | undefined;

    function connect() {
      if (closed) return;
      const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
      ws = new WebSocket(`${proto}//${window.location.host}/api/v1/ws`);

      ws.onmessage = (ev) => {
        try {
          const data = JSON.parse(ev.data) as WSEvent;
          onEventRef.current(data);
        } catch {
          // ignore malformed frame
        }
      };
      ws.onopen = () => {
        retryDelay = 1000;
      };
      ws.onclose = () => {
        if (closed) return;
        retryTimer = setTimeout(connect, retryDelay);
        retryDelay = Math.min(retryDelay * 2, 30000);
      };
    }
    connect();

    return () => {
      closed = true;
      if (retryTimer) clearTimeout(retryTimer);
      ws?.close();
    };
  }, []);
}
