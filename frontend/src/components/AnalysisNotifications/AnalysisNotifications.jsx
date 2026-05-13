import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../../auth/AuthContext";
import { getAccessToken, isDevAuthBypassEnabled } from "../../services/authService";
import { resolveApiBaseUrl } from "../../services/apiFoundation";
import Icon from "../Icon/Icon";
import "./AnalysisNotifications.css";

const TOKEN_CHECK_MS = 5000;
const TOAST_TTL_MS = 10000;
const RECONNECT_BASE_DELAY_MS = 1500;
const RECONNECT_MAX_DELAY_MS = 15000;

function wsURLWithToken(token) {
  const base = resolveApiBaseUrl();
  const origin = base || (import.meta.env.DEV ? "http://localhost:8080" : window.location.origin);
  const url = new URL("/v1/analyses/ws", origin);
  url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
  url.searchParams.set("token", token);
  return url.toString();
}

function normalizeEvent(raw) {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) return null;

  const type = String(raw.type || "");
  const jobID = String(raw.job_id || raw.jobId || "");
  const status = String(raw.status || "");
  const payload =
    raw.payload && typeof raw.payload === "object" && !Array.isArray(raw.payload)
      ? raw.payload
      : {};

  if (!jobID || !type) return null;

  if (type === "analysis:completed") {
    const damageCount = Number(payload.damage_count ?? payload.damageCount ?? 0);
    return {
      id: `${jobID}:${Date.now()}`,
      jobID,
      kind: "success",
      title: "Анализ завершён",
      message:
        damageCount > 0
          ? `Обнаружено повреждений: ${damageCount}`
          : "Повреждений не найдено",
    };
  }

  if (type === "analysis:failed" || status === "failed") {
    return {
      id: `${jobID}:${Date.now()}`,
      jobID,
      kind: "danger",
      title: "Анализ не выполнен",
      message: String(payload.error || "Не удалось обработать изображения"),
    };
  }

  return null;
}

function currentTokenSnapshot() {
  if (isDevAuthBypassEnabled()) return "";
  return getAccessToken() || "";
}

function AnalysisNotifications() {
  const { isAuthenticated } = useAuth();
  const navigate = useNavigate();
  const [token, setToken] = useState(() => currentTokenSnapshot());
  const [items, setItems] = useState([]);
  const seenEventsRef = useRef(new Map());

  useEffect(() => {
    if (!isAuthenticated) {
      setToken("");
      return undefined;
    }

    const sync = () => {
      const nextToken = currentTokenSnapshot();
      setToken((prev) => (prev === nextToken ? prev : nextToken));
    };

    sync();
    const intervalID = setInterval(sync, TOKEN_CHECK_MS);
    window.addEventListener("storage", sync);

    return () => {
      clearInterval(intervalID);
      window.removeEventListener("storage", sync);
    };
  }, [isAuthenticated]);

  useEffect(() => {
    if (!isAuthenticated || !token) return undefined;

    let socket = null;
    let reconnectTimer = null;
    let reconnectAttempt = 0;
    let closedByEffect = false;

    const connect = () => {
      if (closedByEffect) return;
      socket = new WebSocket(wsURLWithToken(token));

      socket.onopen = () => {
        reconnectAttempt = 0;
      };

      socket.onmessage = (event) => {
        let parsed = null;
        try {
          parsed = JSON.parse(event.data);
        } catch {
          return;
        }

        const notification = normalizeEvent(parsed);
        if (!notification) return;

        const eventKey = `${notification.jobID}:${notification.kind}`;
        const now = Date.now();
        const previousAt = seenEventsRef.current.get(eventKey);
        if (previousAt && now - previousAt < TOAST_TTL_MS) return;
        seenEventsRef.current.set(eventKey, now);

        for (const [key, seenAt] of seenEventsRef.current.entries()) {
          if (now - seenAt > TOAST_TTL_MS * 2) {
            seenEventsRef.current.delete(key);
          }
        }

        setItems((prev) => [notification, ...prev].slice(0, 3));
        setTimeout(() => {
          setItems((prev) => prev.filter((item) => item.id !== notification.id));
        }, TOAST_TTL_MS);
      };

      socket.onclose = () => {
        if (closedByEffect) return;
        reconnectAttempt += 1;
        const delay = Math.min(
          RECONNECT_BASE_DELAY_MS * reconnectAttempt,
          RECONNECT_MAX_DELAY_MS
        );
        reconnectTimer = setTimeout(connect, delay);
      };

      socket.onerror = () => {
        socket?.close();
      };
    };

    connect();

    return () => {
      closedByEffect = true;
      if (reconnectTimer) clearTimeout(reconnectTimer);
      socket?.close();
    };
  }, [isAuthenticated, token]);

  if (!isAuthenticated || items.length === 0) return null;

  return (
    <div className="analysis-notifications" aria-live="polite" aria-atomic="false">
      {items.map((item) => (
        <div
          key={item.id}
          className={`analysis-notification analysis-notification-${item.kind}`}
        >
          <span className="analysis-notification-icon" aria-hidden="true">
            <Icon name={item.kind === "success" ? "checkCircle" : "alert"} size={18} />
          </span>
          <button
            type="button"
            className="analysis-notification-body"
            onClick={() => navigate(`/result/${item.jobID}`)}
          >
            <span className="analysis-notification-title">{item.title}</span>
            <span className="analysis-notification-text">{item.message}</span>
          </button>
          <button
            type="button"
            className="analysis-notification-close"
            aria-label="Закрыть уведомление"
            onClick={() => {
              setItems((prev) => prev.filter((x) => x.id !== item.id));
            }}
          >
            <Icon name="x" size={16} />
          </button>
        </div>
      ))}
    </div>
  );
}

export default AnalysisNotifications;
