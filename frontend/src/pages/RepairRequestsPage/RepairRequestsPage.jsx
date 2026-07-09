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
  cancelRepairRequest,
  getIncomingRepairRequestDetails,
  mergeRepairRequest,
  normalizeRepairRequestList,
} from "../../services/repairRequests";
import { listMyServiceRegistrations } from "../../services/serviceRegistrations";
import { listMyTrainingRequests } from "../../services/trainingRequests";
import DamageOverlayImage from "../../components/DamageOverlayImage/DamageOverlayImage";

function fmtDateTime(iso) {
  if (!iso) return "дата не указана";
  try {
    const d = new Date(iso);
    if (Number.isNaN(d.getTime())) return String(iso);
    return d.toLocaleString("ru-RU", {
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
    canceled: {
      className: "repair-status-pill repair-status-pill-unknown",
      icon: "xCircle",
      label: "Отменена",
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

function formatMoney(value) {
  const n = Number(value);
  if (!Number.isFinite(n)) return "";
  return new Intl.NumberFormat("ru-RU", {
    maximumFractionDigits: 0,
  }).format(n);
}

function formatPriceRange(min, max) {
  const from = formatMoney(min);
  const to = formatMoney(max);
  if (from && to && from !== to) return `${from}-${to} ₽`;
  if (from || to) return `${from || to} ₽`;
  return "";
}

function vehicleTitle(r) {
  const analysis = r.analysis || {};
  const parts = [
    analysis.car_make || r.car_brand,
    analysis.car_model,
    analysis.car_generation,
    analysis.car_year,
  ].filter(Boolean);
  return parts.length > 0 ? parts.join(" ") : "Заявка на ремонт";
}

function serviceContactLine(service) {
  if (!service) return "";
  return [service.city, service.address].filter(Boolean).join(", ");
}

function damageTypeText(damageType) {
  if (!damageType || typeof damageType !== "object") return "";
  const name = damageType.name_ru || damageType.code || "повреждение";
  const count = Number(damageType.count || 0);
  return count > 1 ? `${name} ×${count}` : name;
}

function repairSummaryRows(summary) {
  if (!Array.isArray(summary)) return [];
  return summary
    .map((item) => {
      if (!item || typeof item !== "object") return null;
      const part = item.part_name_ru || item.part_name || item.parent_name_ru || item.parent_name || "Деталь";
      const side = item.side_ru || item.side || "";
      const damages = Array.isArray(item.damage_types)
        ? item.damage_types.map(damageTypeText).filter(Boolean).join(", ")
        : "";
      return {
        key: `${item.part_name || part}:${item.side || side}:${damages}`,
        title: `${part}${side ? `, ${side}` : ""}`,
        damages,
      };
    })
    .filter(Boolean);
}

function IncomingCustomerContact({ request }) {
  const user = request?.user || {};
  const name = user.name || request?.customer_name || "";
  const phone = user.phone || request?.customer_phone || "";
  const email = user.email || request?.customer_email || "";
  const comment = request?.customer_comment || "";

  if (!name && !phone && !email && !comment) return null;

  return (
    <div className="repair-contact-block repair-contact-block-neutral">
      <Icon name="user" size={16} />
      <div className="service-user-contact-lines">
        <div className="repair-contact-intro">Контакты клиента</div>
        {name ? (
          <div className="repair-contact-line">
            Имя: <b>{name}</b>
          </div>
        ) : null}
        {phone ? (
          <div className="repair-contact-line">
            Телефон: <b>{phone}</b>
          </div>
        ) : null}
        {email ? (
          <div className="repair-contact-email">
            Email: <b>{email}</b>
          </div>
        ) : null}
        {comment ? (
          <div className="repair-contact-comment">
            <span>Комментарий клиента</span>
            {comment}
          </div>
        ) : null}
      </div>
    </div>
  );
}

function estimateRows(estimate) {
  if (!Array.isArray(estimate)) return [];
  return estimate
    .map((item) => {
      if (!item || typeof item !== "object") return null;
      const part = item.part_name_ru || item.part_name || item.parent_name_ru || item.parent_name || "Деталь";
      const side = item.side_ru || item.side || "";
      const damage = item.damage_name_ru || item.damage_code || "повреждение";
      const quantity = Number(item.quantity || 0);
      const price = formatPriceRange(item.price_min, item.price_max);
      return {
        key: `${item.part_name || part}:${item.damage_code || damage}:${item.side || side}`,
        title: `${part}${side ? `, ${side}` : ""}`,
        meta: `${damage}${quantity > 1 ? ` ×${quantity}` : ""}`,
        price,
        comment: item.comment || "",
      };
    })
    .filter(Boolean);
}

function estimateItemsFromRepairSummary(summary) {
  if (!Array.isArray(summary)) return [];
  const items = [];

  summary.forEach((part) => {
    if (!part || typeof part !== "object") return;
    const damageTypes = Array.isArray(part.damage_types) ? part.damage_types : [];
    damageTypes.forEach((damage) => {
      if (!damage || typeof damage !== "object") return;
      items.push({
        part_name: part.part_name || "",
        part_name_ru: part.part_name_ru || "",
        parent_name: part.parent_name || "",
        parent_name_ru: part.parent_name_ru || "",
        is_pair: Boolean(part.is_pair),
        side: part.side || "",
        side_ru: part.side_ru || "",
        damage_code: damage.code || "",
        damage_name_ru: damage.name_ru || "",
        quantity: Number(damage.count || part.damage_count || 1) || 1,
        price_min: "",
        price_max: "",
        comment: "",
      });
    });
  });

  return items;
}

function estimateItemTitle(item) {
  const part = item.part_name_ru || item.part_name || item.parent_name_ru || item.parent_name || "Деталь";
  const side = item.side_ru || item.side || "";
  const damage = item.damage_name_ru || item.damage_code || "повреждение";
  const quantity = Number(item.quantity || 0);
  return {
    part: `${part}${side ? `, ${side}` : ""}`,
    damage: `${damage}${quantity > 1 ? ` ×${quantity}` : ""}`,
  };
}

function parseOptionalPrice(value) {
  const s = String(value ?? "").replace(",", ".").trim();
  if (!s) return undefined;
  const n = Number(s);
  return Number.isFinite(n) && n >= 0 ? n : NaN;
}

function parseRequiredPrice(value) {
  const n = parseOptionalPrice(value);
  return typeof n === "number" ? n : NaN;
}

function requestStatusIndicator(status) {
  const map = {
    pending: {
      className: "repair-status-pill repair-status-pill-pending",
      icon: "clock",
      label: "На рассмотрении",
    },
    approved: {
      className: "repair-status-pill repair-status-pill-accepted",
      icon: "checkCircle",
      label: "Одобрена",
    },
    accepted: {
      className: "repair-status-pill repair-status-pill-accepted",
      icon: "checkCircle",
      label: "Одобрена",
    },
    in_progress: {
      className: "repair-status-pill repair-status-pill-pending",
      icon: "clock",
      label: "В работе",
    },
    completed: {
      className: "repair-status-pill repair-status-pill-accepted",
      icon: "checkCircle",
      label: "Завершена",
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
    label: String(status || "Неизвестно"),
  };
  return (
    <span className={s.className} role="status">
      <Icon name={s.icon} size={16} />
      <span className="repair-status-pill-label">{s.label}</span>
    </span>
  );
}

function SystemRequestsPanel({ role, serviceApplications, trainingRequests, loading }) {
  const showServiceApplications = role === "USER";
  const visibleTrainingRequests = role !== "ADMIN" ? trainingRequests : [];
  const hasRows =
    (showServiceApplications && serviceApplications.length > 0) ||
    visibleTrainingRequests.length > 0;

  if (loading) {
    return (
      <section className="system-requests-section">
        <div className="system-requests-header">
          <div>
            <h2>Мои заявки в системе</h2>
            <p>Статусы заявок на роль автосервиса и обучение ML-моделей.</p>
          </div>
        </div>
        <div className="system-request-card system-request-card--muted">
          Загрузка заявок...
        </div>
      </section>
    );
  }

  if (!hasRows) return null;

  return (
    <section className="system-requests-section">
      <div className="system-requests-header">
        <div>
          <h2>Мои заявки в системе</h2>
          <p>Здесь отображаются заявки, которые рассматривает администратор.</p>
        </div>
      </div>

      <div className="system-request-list">
        {showServiceApplications &&
          serviceApplications.map((r) => (
            <article className="system-request-card" key={`service:${r.id}`}>
              <div className="system-request-head">
                <div>
                  <div className="system-request-kicker">Регистрация автосервиса</div>
                  <div className="system-request-title">{r.organization || "Автосервис"}</div>
                  <div className="repair-card-sub">
                    Заявка от {fmtDateTime(r.submitted_at || r.created_at)}
                  </div>
                </div>
                {requestStatusIndicator(r.status)}
              </div>

              <div className="repair-meta">
                <div>
                  <div className="label">Город</div>
                  <div className="val">{r.city || "Не указан"}</div>
                </div>
                <div>
                  <div className="label">Адрес</div>
                  <div className="val">{r.address || "Не указан"}</div>
                </div>
              </div>

              {r.description ? (
                <div className="system-request-description">{r.description}</div>
              ) : null}

              {r.status === "rejected" && r.rejection_reason ? (
                <div className="system-request-reason">
                  <Icon name="alert" size={16} />
                  <div>
                    <b>Причина отклонения:</b> {r.rejection_reason}
                  </div>
                </div>
              ) : null}

              {r.status === "approved" ? (
                <div className="system-request-note">
                  <Icon name="checkCircle" size={16} />
                  <div>
                    Заявка одобрена. Если в шапке ещё отображается роль пользователя,
                    обновите сессию или войдите заново.
                  </div>
                </div>
              ) : null}
            </article>
          ))}

        {visibleTrainingRequests.map((r) => (
          <article className="system-request-card" key={`training:${r.id}`}>
            <div className="system-request-head">
              <div>
                <div className="system-request-kicker">Обучение ML-модели</div>
                <div className="system-request-title">
                  {[r.brand, r.model, r.generation].filter(Boolean).join(" ") || "Новая модель"}
                </div>
                <div className="repair-card-sub">
                  Заявка от {fmtDateTime(r.submitted_at || r.created_at)}
                </div>
              </div>
              {requestStatusIndicator(r.status)}
            </div>

            <div className="repair-meta">
              <div>
                <div className="label">Годы выпуска</div>
                <div className="val">{r.years || r.year || "Не указаны"}</div>
              </div>
              <div>
                <div className="label">Поколение</div>
                <div className="val">{r.generation || "Не указано"}</div>
              </div>
            </div>

            {r.description ? (
              <div className="system-request-description">{r.description}</div>
            ) : null}
          </article>
        ))}
      </div>
    </section>
  );
}

function UserView({ tab, setTab, items, setItems }) {
  const filtered = useMemo(() => {
    if (tab === "all") return items;
    return items.filter((r) => r.status === tab);
  }, [tab, items]);

  const handleCancel = async (id) => {
    const snapshot = items;
    setItems((prev) => prev.filter((r) => r.id !== id));
    try {
      await cancelRepairRequest(id);
    } catch {
      setItems(snapshot);
    }
  };

  return (
    <>
      <section className="repair-requests-section">
        <div className="repair-section-header">
          <div>
            <h2>Заявки на ремонт</h2>
            <p>Заявки, созданные по результатам анализа автомобиля.</p>
          </div>
        </div>

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
          <button className={"tab-btn" + (tab === "canceled" ? " active" : "")} onClick={() => setTab("canceled")}>
            Отменены
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
          {filtered.map((r) => {
            const service = r.service || {};
            const summaryRows = repairSummaryRows(r.repair_summary);
            const serviceAddress = serviceContactLine(service);
            const priceRange = formatPriceRange(r.estimated_price_min, r.estimated_price_max);
            const estimates = estimateRows(r.service_estimate);

            return (
              <article className="repair-card" key={r.id}>
                <div className="repair-card-header">
                  <div>
                    <div className="repair-card-title">{vehicleTitle(r)}</div>
                    <div className="repair-card-sub">Заявка от {fmtDateTime(r.created_at)}</div>
                  </div>
                  {statusIndicator(r.status)}
                </div>

                <div className="repair-meta repair-meta-user">
                  <div>
                    <div className="label">Автосервис</div>
                    <div className="val repair-service-title">
                      {service.name || "Название автосервиса недоступно"}
                    </div>
                    {serviceAddress ? (
                      <div className="repair-meta-hint">{serviceAddress}</div>
                    ) : null}
                  </div>
                  {(service.phone || service.email) ? (
                    <div>
                      <div className="label">Контакты</div>
                      {service.phone ? <div className="val">{service.phone}</div> : null}
                      {service.email ? <div className="repair-meta-hint">{service.email}</div> : null}
                    </div>
                  ) : null}
                </div>

                {summaryRows.length > 0 ? (
                  <div className="repair-summary-block">
                    <div className="repair-block-title">Повреждения из заявки</div>
                    <div className="repair-summary-list">
                      {summaryRows.map((row, index) => (
                        <div className="repair-summary-row" key={`${row.key}:${index}`}>
                          <span>{row.title}</span>
                          {row.damages ? <b>{row.damages}</b> : null}
                        </div>
                      ))}
                    </div>
                  </div>
                ) : null}

                {r.customer_comment ? (
                  <div className="repair-note-block">
                    <span>Ваш комментарий</span>
                    {r.customer_comment}
                  </div>
                ) : null}

                {(r.status === "accepted" || r.status === "rejected") && r.service_comment ? (
                  <div className={r.status === "accepted" ? "repair-contact-block" : "repair-rejected-block"}>
                    <Icon name={r.status === "accepted" ? "checkCircle" : "xCircle"} size={16} />
                    <div>
                      <div className="repair-contact-intro">
                        {r.status === "accepted" ? "Ответ автосервиса" : "Причина отклонения"}
                      </div>
                      {r.service_comment}
                    </div>
                  </div>
                ) : null}

                {r.status === "accepted" && priceRange ? (
                  <div className="repair-price-block">
                    <span>Предварительная стоимость</span>
                    <b>{priceRange}</b>
                  </div>
                ) : null}

                {r.status === "accepted" && estimates.length > 0 ? (
                  <div className="repair-summary-block">
                    <div className="repair-block-title">Смета по повреждениям</div>
                    <div className="repair-summary-list">
                      {estimates.map((row, index) => (
                        <div className="repair-summary-row repair-estimate-row" key={`${row.key}:${index}`}>
                          <span>
                            {row.title}
                            <small>{row.meta}</small>
                            {row.comment ? <em>{row.comment}</em> : null}
                          </span>
                          {row.price ? <b>{row.price}</b> : null}
                        </div>
                      ))}
                    </div>
                  </div>
                ) : null}

                {r.status === "pending" && (
                  <div className="service-request-actions">
                    <button
                      type="button"
                      className="btn btn-secondary btn-sm"
                      onClick={() => handleCancel(r.id)}
                    >
                      Отменить заявку
                    </button>
                  </div>
                )}
              </article>
            );
          })}
        </div>
      )}
      </section>
    </>
  );
}

function IncomingRepairDetails({ details, fallbackRequest }) {
  const request = details?.request || fallbackRequest || {};
  const rows = repairSummaryRows(request.repair_summary || fallbackRequest?.repair_summary);
  const analysisResults = Array.isArray(details?.analysis?.result?.results)
    ? details.analysis.result.results
    : [];
  const images = Array.isArray(details?.images)
    ? details.images
        .map((image, index) => ({
          index: image.index ?? index,
          url: image.url || image.URL || "",
        }))
        .filter((image) => image.url)
    : [];

  return (
    <div className="incoming-details">
      {images.length > 0 ? (
        <div className="incoming-images">
          {images.map((image) => (
            <div className="incoming-image-frame" key={`${image.index}:${image.url}`}>
              <DamageOverlayImage
                src={image.url}
                imageResult={analysisResults[image.index]}
                alt={`Фото автомобиля ${Number(image.index) + 1}`}
                className="incoming-damage-overlay"
              />
            </div>
          ))}
        </div>
      ) : null}

      {rows.length > 0 ? (
        <div className="incoming-damage-table-wrap">
          <table className="incoming-damage-table">
            <thead>
              <tr>
                <th>Деталь</th>
                <th>Повреждения</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row, index) => (
                <tr key={`${row.key}:${index}`}>
                  <td>{row.title}</td>
                  <td>{row.damages || "Не указано"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}
    </div>
  );
}

function RejectRepairRequestModal({ request, onClose, onConfirm }) {
  const [comment, setComment] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const submit = async (e) => {
    e.preventDefault();
    const cleanComment = comment.trim();
    if (!cleanComment) {
      setError("Укажите причину отклонения");
      return;
    }

    setSubmitting(true);
    setError("");
    try {
      await onConfirm({ service_comment: cleanComment });
    } catch (err) {
      const msg =
        err instanceof Error && err.message.trim()
          ? err.message.trim()
          : "Не удалось отклонить заявку";
      setError(msg);
      setSubmitting(false);
    }
  };

  return (
    <div className="repair-modal-overlay" role="dialog" aria-modal="true">
      <div className="repair-modal-card repair-modal-card-sm">
        <form onSubmit={submit} noValidate>
          <div className="repair-modal-header">
            <h3>Отклонение заявки</h3>
            <button type="button" className="btn btn-ghost btn-sm" onClick={onClose} disabled={submitting} aria-label="Закрыть">
              <Icon name="x" size={16} />
            </button>
          </div>

          <div className="repair-modal-body">
            <div className="repair-modal-summary">
              <div className="repair-modal-summary-title">{vehicleTitle(request)}</div>
              <div className="text-xs muted">Заявка от {fmtDateTime(request?.created_at)}</div>
            </div>

            <div className="form-row">
              <label className="form-label" htmlFor="repair-reject-comment">Комментарий для клиента *</label>
              <textarea
                id="repair-reject-comment"
                className={"textarea" + (error ? " input-error" : "")}
                placeholder="Например, сейчас нет свободных слотов или требуется другой тип работ"
                value={comment}
                onChange={(ev) => {
                  setComment(ev.target.value);
                  if (error) setError("");
                }}
                disabled={submitting}
                autoFocus
              />
              {error ? <div className="field-error">{error}</div> : null}
            </div>
          </div>

          <div className="repair-modal-footer">
            <button type="button" className="btn btn-ghost btn-sm" onClick={onClose} disabled={submitting}>
              Отмена
            </button>
            <button type="submit" className="btn btn-danger btn-sm" disabled={submitting}>
              Отклонить заявку
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

function AcceptRepairRequestModal({ request, onClose, onConfirm }) {
  const initialEstimate = useMemo(
    () => estimateItemsFromRepairSummary(request?.repair_summary),
    [request]
  );
  const [comment, setComment] = useState("Готовы принять автомобиль на предварительный осмотр");
  const [priceMin, setPriceMin] = useState("");
  const [priceMax, setPriceMax] = useState("");
  const [estimate, setEstimate] = useState(initialEstimate);
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const updateEstimate = (index, field, value) => {
    setEstimate((prev) =>
      prev.map((item, idx) => (idx === index ? { ...item, [field]: value } : item))
    );
    if (error) setError("");
  };

  const submit = async (e) => {
    e.preventDefault();
    const cleanComment = comment.trim();
    const estimatedPriceMin = parseRequiredPrice(priceMin);
    const estimatedPriceMax = parseRequiredPrice(priceMax);

    if (!cleanComment) {
      setError("Укажите комментарий для клиента");
      return;
    }
    if (!Number.isFinite(estimatedPriceMin) || !Number.isFinite(estimatedPriceMax)) {
      setError("Укажите общую минимальную и максимальную стоимость");
      return;
    }
    if (estimatedPriceMin > estimatedPriceMax) {
      setError("Минимальная стоимость не должна быть больше максимальной");
      return;
    }

    const serviceEstimate = [];
    for (const item of estimate) {
      const itemMin = parseOptionalPrice(item.price_min);
      const itemMax = parseOptionalPrice(item.price_max);
      if (Number.isNaN(itemMin) || Number.isNaN(itemMax)) {
        setError("Стоимость по повреждениям должна быть неотрицательным числом");
        return;
      }
      if (
        Number.isFinite(itemMin) &&
        Number.isFinite(itemMax) &&
        itemMin > itemMax
      ) {
        setError("В строках сметы минимальная стоимость не должна быть больше максимальной");
        return;
      }
      serviceEstimate.push({
        ...item,
        price_min: Number.isFinite(itemMin) ? itemMin : undefined,
        price_max: Number.isFinite(itemMax) ? itemMax : undefined,
        comment: String(item.comment || "").trim() || undefined,
      });
    }

    setSubmitting(true);
    setError("");
    try {
      await onConfirm({
        service_comment: cleanComment,
        estimated_price_min: estimatedPriceMin,
        estimated_price_max: estimatedPriceMax,
        service_estimate: serviceEstimate,
      });
    } catch (err) {
      const msg =
        err instanceof Error && err.message.trim()
          ? err.message.trim()
          : "Не удалось принять заявку";
      setError(msg);
      setSubmitting(false);
    }
  };

  return (
    <div className="repair-modal-overlay" role="dialog" aria-modal="true">
      <div className="repair-modal-card repair-modal-card-wide">
        <form onSubmit={submit} noValidate>
          <div className="repair-modal-header">
            <h3>Принятие заявки</h3>
            <button type="button" className="btn btn-ghost btn-sm" onClick={onClose} disabled={submitting} aria-label="Закрыть">
              <Icon name="x" size={16} />
            </button>
          </div>

          <div className="repair-modal-body">
            <div className="repair-modal-summary">
              <div className="repair-modal-summary-title">{vehicleTitle(request)}</div>
              <div className="text-xs muted">Предварительная оценка будет видна пользователю в его заявке.</div>
            </div>

            <div className="form-row">
              <label className="form-label" htmlFor="repair-accept-comment">Комментарий для клиента *</label>
              <textarea
                id="repair-accept-comment"
                className="textarea"
                value={comment}
                onChange={(ev) => {
                  setComment(ev.target.value);
                  if (error) setError("");
                }}
                disabled={submitting}
              />
            </div>

            <div className="form-row-inline">
              <div className="form-row">
                <label className="form-label" htmlFor="repair-price-min">Минимальная стоимость, ₽ *</label>
                <input
                  id="repair-price-min"
                  className="input"
                  inputMode="decimal"
                  value={priceMin}
                  onChange={(ev) => {
                    setPriceMin(ev.target.value);
                    if (error) setError("");
                  }}
                  disabled={submitting}
                  placeholder="17000"
                />
              </div>
              <div className="form-row">
                <label className="form-label" htmlFor="repair-price-max">Максимальная стоимость, ₽ *</label>
                <input
                  id="repair-price-max"
                  className="input"
                  inputMode="decimal"
                  value={priceMax}
                  onChange={(ev) => {
                    setPriceMax(ev.target.value);
                    if (error) setError("");
                  }}
                  disabled={submitting}
                  placeholder="26000"
                />
              </div>
            </div>

            {estimate.length > 0 ? (
              <div className="repair-estimate-editor">
                <div className="repair-block-title">Смета по повреждениям</div>
                {estimate.map((item, index) => {
                  const title = estimateItemTitle(item);
                  return (
                    <div className="repair-estimate-editor-row" key={`${item.part_name}:${item.damage_code}:${item.side}:${index}`}>
                      <div className="repair-estimate-editor-main">
                        <b>{title.part}</b>
                        <span>{title.damage}</span>
                      </div>
                      <input
                        className="input"
                        inputMode="decimal"
                        value={item.price_min}
                        onChange={(ev) => updateEstimate(index, "price_min", ev.target.value)}
                        disabled={submitting}
                        placeholder="от, ₽"
                        aria-label="Минимальная стоимость по повреждению"
                      />
                      <input
                        className="input"
                        inputMode="decimal"
                        value={item.price_max}
                        onChange={(ev) => updateEstimate(index, "price_max", ev.target.value)}
                        disabled={submitting}
                        placeholder="до, ₽"
                        aria-label="Максимальная стоимость по повреждению"
                      />
                      <input
                        className="input repair-estimate-comment"
                        value={item.comment}
                        onChange={(ev) => updateEstimate(index, "comment", ev.target.value)}
                        disabled={submitting}
                        placeholder="Комментарий"
                        aria-label="Комментарий по повреждению"
                      />
                    </div>
                  );
                })}
              </div>
            ) : null}

            {error ? <div className="field-error">{error}</div> : null}
          </div>

          <div className="repair-modal-footer">
            <button type="button" className="btn btn-ghost btn-sm" onClick={onClose} disabled={submitting}>
              Отмена
            </button>
            <button type="submit" className="btn btn-primary btn-sm" disabled={submitting}>
              Принять заявку
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

function ServiceView({ tab, setTab, items, setItems, refetchIncoming }) {
  const filtered = useMemo(() => {
    if (tab === "all") return items;
    return items.filter((r) => r.status === tab);
  }, [items, tab]);

  const aliveRef = useRef(true);
  const [busyById, setBusyById] = useState({});
  const [detailsById, setDetailsById] = useState({});
  const [acceptTarget, setAcceptTarget] = useState(null);
  const [rejectTarget, setRejectTarget] = useState(null);

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

  const handleAccept = async (id, payload) => {
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
      const row = await acceptRepairRequest(id, {
        service_comment: payload.service_comment,
        estimated_price_min: payload.estimated_price_min,
        estimated_price_max: payload.estimated_price_max,
        service_estimate: payload.service_estimate,
      });
      if (!aliveRef.current) return;
      setItems((prev) =>
        prev.map((r) =>
          r.id === id ? mergeRepairRequest(r, row) : r
        )
      );
      setAcceptTarget(null);
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

  const handleReject = async (id, payload) => {
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
      const row = await rejectRepairRequest(id, { reason: payload.service_comment });
      if (!aliveRef.current) return;
      setItems((prev) =>
        prev.map((r) =>
          r.id === id ? mergeRepairRequest(r, row) : r
        )
      );
      setRejectTarget(null);
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

  const toggleDetails = async (id) => {
    if (detailsById[id]) {
      setDetailsById((prev) => {
        const next = { ...prev };
        delete next[id];
        return next;
      });
      return;
    }
    try {
      const details = await getIncomingRepairRequestDetails(id);
      setDetailsById((prev) => ({ ...prev, [id]: details }));
    } catch {
      setDetailsById((prev) => ({ ...prev, [id]: { error: true } }));
    }
  };

  return (
    <>
      <section className="repair-requests-section">
        <div className="repair-section-header">
          <div>
            <h2>Входящие заявки на ремонт</h2>
            <p>Заявки от пользователей, выбравших ваш автосервис.</p>
          </div>
        </div>

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
                  <div className="title">{vehicleTitle(r)}</div>
                  <div className="text-sm muted">{fmtDateTime(r.created_at)}</div>
                </div>
                {statusIndicator(r.status)}
              </div>

              <div className="body">
                <span className="muted">Повреждения: </span>{r.damage_summary || "сводка доступна в подробностях"}
              </div>

              {detailsById[r.id] && !detailsById[r.id].error && (
                <IncomingRepairDetails details={detailsById[r.id]} fallbackRequest={r} />
              )}

              <IncomingCustomerContact request={r} />

              <div className="service-request-actions">
                {r.status === "pending" ? (
                  <>
                    <button
                      type="button"
                      className="btn btn-primary btn-sm"
                      disabled={!!busyById[r.id]}
                      onClick={() => setAcceptTarget(r)}
                    >
                      Принять
                    </button>
                    <button
                      type="button"
                      className="btn btn-secondary btn-sm"
                      disabled={!!busyById[r.id]}
                      onClick={() => setRejectTarget(r)}
                    >
                      Отклонить
                    </button>
                    <button type="button" className="btn btn-ghost btn-sm" onClick={() => toggleDetails(r.id)}>
                      Подробнее
                    </button>
                  </>
                ) : r.status === "accepted" ? (
                  <button type="button" className="btn btn-ghost btn-sm" onClick={() => toggleDetails(r.id)}>
                    Подробнее
                  </button>
                ) : r.status === "rejected" ? (
                  <p className="service-request-rejected-hint" role="status">
                    Заявка отклонена.
                  </p>
                ) : (
                  <button type="button" className="btn btn-ghost btn-sm" onClick={() => toggleDetails(r.id)}>
                    Подробнее
                  </button>
                )}
              </div>
            </article>
          ))}
        </div>
      )}
      </section>
      {acceptTarget ? (
        <AcceptRepairRequestModal
          request={acceptTarget}
          onClose={() => setAcceptTarget(null)}
          onConfirm={(payload) => handleAccept(acceptTarget.id, payload)}
        />
      ) : null}
      {rejectTarget ? (
        <RejectRepairRequestModal
          request={rejectTarget}
          onClose={() => setRejectTarget(null)}
          onConfirm={(payload) => handleReject(rejectTarget.id, payload)}
        />
      ) : null}
    </>
  );
}

function RepairRequestsPage() {
  const { role } = useAuth();
  const [section, setSection] = useState("repair");
  const [userTab, setUserTab] = useState("all");
  const [serviceTab, setServiceTab] = useState("pending");
  const [userItems, setUserItems] = useState([]);
  const [serviceItems, setServiceItems] = useState([]);
  const [serviceApplications, setServiceApplications] = useState([]);
  const [trainingRequests, setTrainingRequests] = useState([]);
  const [systemRequestsLoading, setSystemRequestsLoading] = useState(false);

  const isService = role === "SERVICE";
  const availableSections = useMemo(() => {
    if (role === "ADMIN") {
      return [{ id: "repair", label: "Заявки на ремонт" }];
    }
    if (isService) {
      return [
        { id: "repair", label: "Входящие заявки" },
        { id: "system", label: "Заявки на обучение" },
      ];
    }
    return [
      { id: "repair", label: "Заявки на ремонт" },
      { id: "system", label: "Заявки в системе" },
    ];
  }, [isService, role]);

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

  useEffect(() => {
    if (!availableSections.some((item) => item.id === section)) {
      setSection(availableSections[0]?.id || "repair");
    }
  }, [availableSections, section]);

  useEffect(() => {
    let cancelled = false;
    const shouldLoadServiceApplications = role === "USER";
    const shouldLoadTrainingRequests = role !== "ADMIN";

    if (!shouldLoadServiceApplications && !shouldLoadTrainingRequests) {
      setServiceApplications([]);
      setTrainingRequests([]);
      setSystemRequestsLoading(false);
      return () => {
        cancelled = true;
      };
    }

    setSystemRequestsLoading(true);

    Promise.allSettled([
      shouldLoadServiceApplications
        ? listMyServiceRegistrations()
        : Promise.resolve([]),
      shouldLoadTrainingRequests
        ? listMyTrainingRequests()
        : Promise.resolve([]),
    ]).then(([serviceResult, trainingResult]) => {
      if (cancelled) return;

      setServiceApplications(
        serviceResult.status === "fulfilled" && Array.isArray(serviceResult.value)
          ? serviceResult.value
          : []
      );
      setTrainingRequests(
        trainingResult.status === "fulfilled" && Array.isArray(trainingResult.value)
          ? trainingResult.value
          : []
      );
      setSystemRequestsLoading(false);
    });

    return () => {
      cancelled = true;
    };
  }, [role]);

  return (
    <div>
      <div className="page-header">
        <div>
          <h1 className="page-title">Заявки</h1>
          <p className="page-subtitle">
            {isService
              ? "Входящие обращения и ваши системные заявки"
              : "Ваши обращения к автосервисам и администратору"}
          </p>
        </div>
      </div>

      {availableSections.length > 1 ? (
        <div className="tabs repair-section-tabs" role="tablist">
          {availableSections.map((item) => (
            <button
              key={item.id}
              type="button"
              className={"tab-btn" + (section === item.id ? " active" : "")}
              onClick={() => setSection(item.id)}
            >
              {item.label}
            </button>
          ))}
        </div>
      ) : null}

      {section === "system" ? (
        <SystemRequestsPanel
          role={role}
          serviceApplications={serviceApplications}
          trainingRequests={trainingRequests}
          loading={systemRequestsLoading}
        />
      ) : null}

      {section === "repair" && (
        isService ? (
          <ServiceView
            tab={serviceTab}
            setTab={setServiceTab}
            items={serviceItems}
            setItems={setServiceItems}
            refetchIncoming={refetchService}
          />
        ) : (
          <UserView
            tab={userTab}
            setTab={setUserTab}
            items={userItems}
            setItems={setUserItems}
          />
        )
      )}
    </div>
  );
}

export default RepairRequestsPage;
