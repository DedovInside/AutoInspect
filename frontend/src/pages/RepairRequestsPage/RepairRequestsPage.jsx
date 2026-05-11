import "./RepairRequestsPage.css";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Link } from "react-router-dom";
import Icon from "../../components/Icon/Icon";
import { useAuth } from "../../auth/AuthContext";
import { normalizeApiError } from "../../services/apiFoundation";
import { repairDevLog } from "../../services/repairDebug";
import {
  listMyRepairRequests,
  listIncomingRepairRequests,
  acceptRepairRequest,
  rejectRepairRequest,
  mergeRepairRequest,
  normalizeRepairRequestList,
} from "../../services/repairRequests";

function fmtDateTime(iso) {
  try {
    return new Date(iso).toLocaleString("ru-RU", {
      day: "2-digit",
      month: "short",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return iso;
  }
}

function statusIndicator(status) {
  const map = {
    pending: {
      className: "repair-status-pill repair-status-pill-pending",
      icon: "clock",
      label: "На рассмотрении",
    },
    accepted: {
      className: "repair-status-pill repair-status-pill-accepted",
      icon: "checkCircle",
      label: "Принята",
    },
    rejected: {
      className: "repair-status-pill repair-status-pill-rejected",
      icon: "xCircle",
      label: "Отклонена",
    },
  };
  const s = map[status] || {
    className: "repair-status-pill repair-status-pill-unknown",
    icon: "alert",
    label: String(status),
  };
  return (
    <span className={s.className} role="status">
      <Icon name={s.icon} size={16} />
      <span className="repair-status-pill-label">{s.label}</span>
    </span>
  );
}

function UserView({ tab, setTab, items }) {
  const filtered = useMemo(() => {
    if (tab === "all") return items;
    return items.filter((r) => r.status === tab);
  }, [tab, items]);

  return (
    <>
      <div className="repair-toolbar">
        <div className="tabs">
          <button className={"tab-btn" + (tab === "all" ? " active" : "")} onClick={() => setTab("all")}>
            Все
          </button>
          <button className={"tab-btn" + (tab === "pending" ? " active" : "")} onClick={() => setTab("pending")}>
            Ожидают
          </button>
          <button className={"tab-btn" + (tab === "accepted" ? " active" : "")} onClick={() => setTab("accepted")}>
            Приняты
          </button>
          <button className={"tab-btn" + (tab === "rejected" ? " active" : "")} onClick={() => setTab("rejected")}>
            Отклонены
          </button>
        </div>
      </div>

      {filtered.length === 0 ? (
        <div className="empty-state">
          <span className="empty-state-icon"><Icon name="inbox" size={22} /></span>
          <div className="empty-state-title">Заявок пока нет</div>
          <div className="empty-state-text">
            Создайте заявку со страницы результата анализа - она появится здесь.
          </div>
          <Link to="/history" className="btn btn-primary mt-3">
            Перейти к истории
          </Link>
        </div>
      ) : (
        <div className="repair-list">
          {filtered.map((r) => (
            <article className="repair-card" key={r.id}>
              <div className="repair-card-header">
                <div>
                  <div className="repair-card-title">{r.car_brand}</div>
                  <div className="repair-card-sub">Заявка от {fmtDateTime(r.created_at)}</div>
                </div>
                {statusIndicator(r.status)}
              </div>

              <div className="repair-meta">
                <div>
                  <div className="label">Автосервис</div>
                  <div className="val">{r.service.name}</div>
                </div>
                <div>
                  <div className="label">Анализ</div>
                  <div className="val">
                    <Link to={`/result/${r.analysis_id}`} style={{ color: "var(--brand-700)" }}>
                      #{r.analysis_id}
                    </Link>
                  </div>
                </div>
              </div>

              {r.status === "accepted" && (
                <div className="repair-contact-block">
                  <Icon name="checkCircle" size={16} />
                  <div>
                    Контакты автосервиса: <b>{r.service.phone}</b>, {r.service.address}
                  </div>
                </div>
              )}
            </article>
          ))}
        </div>
      )}
    </>
  );
}

function ServiceView({ tab, setTab, items, setItems, refetchIncoming }) {
  const filtered = useMemo(() => {
    if (tab === "all") return items;
    return items.filter((r) => r.status === tab);
  }, [items, tab]);

  const aliveRef = useRef(true);
  const [busyById, setBusyById] = useState({});

  useEffect(() => {
    aliveRef.current = true;
    return () => {
      aliveRef.current = false;
    };
  }, []);

  const markBusy = (rowId, v) => {
    setBusyById((b) => {
      const next = { ...b };
      if (v) next[rowId] = true;
      else delete next[rowId];
      return next;
    });
  };

  const handleAccept = async (id) => {
    if (busyById[id]) return;

    let snapshot = null;
    setItems((prev) => {
      const cur = prev.find((r) => r.id === id);
      if (!cur || cur.status !== "pending") return prev;
      snapshot = cur;
      return prev.map((r) =>
        r.id === id ? { ...r, status: "accepted" } : r
      );
    });

    if (!snapshot) return;

    markBusy(id, true);

    try {
      const row = await acceptRepairRequest(id);
      if (!aliveRef.current) return;
      setItems((prev) =>
        prev.map((r) =>
          r.id === id ? mergeRepairRequest(r, row) : r
        )
      );
    } catch (e) {
      if (normalizeApiError(e).code === "aborted") {
        if (aliveRef.current) {
          setItems((prev) =>
            prev.map((r) => (r.id === id ? snapshot : r))
          );
        }
        return;
      }
      if (import.meta.env.DEV) {
        repairDevLog("ui.optimistic_rollback", { action: "accept" });
      }
      if (aliveRef.current) {
        setItems((prev) =>
          prev.map((r) => (r.id === id ? snapshot : r))
        );
        if (typeof refetchIncoming === "function") {
          try {
            await refetchIncoming();
          } catch {
            /* keep rolled-back local state */
          }
        }
      }
    } finally {
      if (aliveRef.current) markBusy(id, false);
    }
  };

  const handleReject = async (id) => {
    if (busyById[id]) return;

    let snapshot = null;
    setItems((prev) => {
      const cur = prev.find((r) => r.id === id);
      if (!cur || cur.status !== "pending") return prev;
      snapshot = cur;
      return prev.map((r) =>
        r.id === id ? { ...r, status: "rejected" } : r
      );
    });

    if (!snapshot) return;

    markBusy(id, true);

    try {
      const row = await rejectRepairRequest(id, { reason: "" });
      if (!aliveRef.current) return;
      setItems((prev) =>
        prev.map((r) =>
          r.id === id ? mergeRepairRequest(r, row) : r
        )
      );
    } catch (e) {
      if (normalizeApiError(e).code === "aborted") {
        if (aliveRef.current) {
          setItems((prev) =>
            prev.map((r) => (r.id === id ? snapshot : r))
          );
        }
        return;
      }
      if (import.meta.env.DEV) {
        repairDevLog("ui.optimistic_rollback", { action: "reject" });
      }
      if (aliveRef.current) {
        setItems((prev) =>
          prev.map((r) => (r.id === id ? snapshot : r))
        );
        if (typeof refetchIncoming === "function") {
          try {
            await refetchIncoming();
          } catch {
            /* keep rolled-back local state */
          }
        }
      }
    } finally {
      if (aliveRef.current) markBusy(id, false);
    }
  };

  return (
    <>
      <div className="repair-toolbar">
        <div className="tabs">
          <button className={"tab-btn" + (tab === "pending" ? " active" : "")} onClick={() => setTab("pending")}>
            Новые
          </button>
          <button className={"tab-btn" + (tab === "accepted" ? " active" : "")} onClick={() => setTab("accepted")}>
            Принятые
          </button>
          <button className={"tab-btn" + (tab === "rejected" ? " active" : "")} onClick={() => setTab("rejected")}>
            Отклонённые
          </button>
          <button className={"tab-btn" + (tab === "all" ? " active" : "")} onClick={() => setTab("all")}>
            Все
          </button>
        </div>
      </div>

      {filtered.length === 0 ? (
        <div className="empty-state">
          <span className="empty-state-icon"><Icon name="inbox" size={22} /></span>
          <div className="empty-state-title">Нет входящих заявок</div>
          <div className="empty-state-text">
            Когда пользователи отправят заявки на ремонт - они появятся здесь.
          </div>
        </div>
      ) : (
        <div className="repair-list">
          {filtered.map((r) => (
            <article
              className={
                "service-request-card" + (r.status === "rejected" ? " service-request-card--rejected" : "")
              }
              key={r.id}
            >
              <div className="head">
                <div>
                  <div className="title">{r.car_brand}</div>
                  <div className="text-sm muted">{fmtDateTime(r.created_at)}</div>
                </div>
                {statusIndicator(r.status)}
              </div>

              <div className="meta">
                <span className="row">
                  <Icon name="layers" size={13} />
                  Анализ{" "}
                  {r.status === "rejected" ? (
                    <span className="service-request-analysis-id service-request-analysis-id--muted" title="Анализ недоступен по отклонённой заявке">
                      #{r.analysis_id}
                    </span>
                  ) : (
                    <Link to={`/result/${r.analysis_id}`} className="service-request-analysis-link">
                      #{r.analysis_id}
                    </Link>
                  )}
                </span>
                <span className="row"><Icon name="alert" size={13} /> {r.severity}</span>
              </div>

              <div className="body">
                <span className="muted">Повреждения: </span>{r.damage_summary}
              </div>

              {r.status === "accepted" && r.user && (
                <div className="repair-contact-block">
                  <Icon name="checkCircle" size={16} />
                  <div className="service-user-contact-lines">
                    <div className="repair-contact-intro">Контакты для связи с клиентом</div>
                    {r.user?.email ? (
                      <div className="repair-contact-email">
                        Email пользователя: <b>{r.user.email}</b>
                      </div>
                    ) : (
                      <div className="text-sm muted">Email недоступен</div>
                    )}
                  </div>
                </div>
              )}

              <div className="service-request-actions">
                {r.status === "pending" ? (
                  <>
                    <button
                      type="button"
                      className="btn btn-primary btn-sm"
                      disabled={!!busyById[r.id]}
                      onClick={() => handleAccept(r.id)}
                    >
                      Принять
                    </button>
                    <button
                      type="button"
                      className="btn btn-secondary btn-sm"
                      disabled={!!busyById[r.id]}
                      onClick={() => handleReject(r.id)}
                    >
                      Отклонить
                    </button>
                    <Link to={`/result/${r.analysis_id}`} className="btn btn-ghost btn-sm">
                      Перейти к анализу
                    </Link>
                  </>
                ) : r.status === "accepted" ? (
                  <Link to={`/result/${r.analysis_id}`} className="btn btn-ghost btn-sm">
                    Перейти к анализу
                  </Link>
                ) : r.status === "rejected" ? (
                  <p className="service-request-rejected-hint" role="status">
                    Заявка отклонена - просмотр анализа недоступен.
                  </p>
                ) : (
                  <Link to={`/result/${r.analysis_id}`} className="btn btn-ghost btn-sm">
                    Перейти к анализу
                  </Link>
                )}
              </div>
            </article>
          ))}
        </div>
      )}
    </>
  );
}

function RepairRequestsPage() {
  const { role } = useAuth();
  const [userTab, setUserTab] = useState("all");
  const [serviceTab, setServiceTab] = useState("pending");
  const [userItems, setUserItems] = useState([]);
  const [serviceItems, setServiceItems] = useState([]);

  const isService = role === "SERVICE";

  const refetchService = useCallback(() => {
    return listIncomingRepairRequests()
      .then((data) => setServiceItems(Array.isArray(data) ? data : normalizeRepairRequestList(data)))
      .catch(() => setServiceItems([]));
  }, []);

  useEffect(() => {
    let cancelled = false;
    const loader = isService ? listIncomingRepairRequests : listMyRepairRequests;
    loader()
      .then((data) => {
        if (cancelled) return;
        const rows = Array.isArray(data) ? data : normalizeRepairRequestList(data);
        if (isService) setServiceItems(rows);
        else setUserItems(rows);
      })
      .catch(() => {
        if (cancelled) return;
        if (isService) setServiceItems([]);
        else setUserItems([]);
      });
    return () => {
      cancelled = true;
    };
  }, [isService]);

  return (
    <div>
      <div className="page-header">
        <div>
          <h1 className="page-title">Заявки на ремонт</h1>
          <p className="page-subtitle">
            {isService
              ? "Входящие заявки от пользователей"
              : "Заявки, созданные по результатам анализа"}
          </p>
        </div>
      </div>

      {isService ? (
        <ServiceView
          tab={serviceTab}
          setTab={setServiceTab}
          items={serviceItems}
          setItems={setServiceItems}
          refetchIncoming={refetchService}
        />
      ) : (
        <UserView tab={userTab} setTab={setUserTab} items={userItems} />
      )}
    </div>
  );
}

export default RepairRequestsPage;
