import "./AdminPage.css";
import { Fragment, useEffect, useRef, useState } from "react";
import Icon from "../../components/Icon/Icon";
import {
  listMLModels,
  uploadMLModel,
  deactivateMLModel,
} from "../../services/mlModels";
import {
  listServiceRegistrations,
  approveServiceRegistration,
  rejectServiceRegistration,
} from "../../services/serviceRegistrations";
import {
  listTrainingRequests,
  updateTrainingRequestStatus,
} from "../../services/trainingRequests";

function statusBadge(status) {
  const map = {
    active: { className: "badge badge-success", label: "Активна" },
    inactive: { className: "badge badge-muted", label: "Деактивирована" },
    deprecated: { className: "badge badge-muted", label: "Деактивирована" },
    pending: { className: "badge badge-warning", label: "На рассмотрении" },
    approved: { className: "badge badge-success", label: "Одобрена" },
    rejected: { className: "badge badge-danger", label: "Отклонена" },
  };
  const s = map[status] || { className: "badge badge-muted", label: status };
  return <span className={s.className}>{s.label}</span>;
}

/** Статусы заявок на обучение ML (отдельные подписи от заявок автосервисов). */
function trainingRequestStatusBadge(status) {
  const map = {
    pending: { className: "badge badge-warning", label: "На рассмотрении" },
    approved: { className: "badge badge-success", label: "Одобрено" },
    rejected: { className: "badge badge-danger", label: "Отклонена" },
    in_progress: { className: "badge badge-warning", label: "Обучение" },
    completed: { className: "badge badge-training-completed", label: "Выполнено" },
  };
  const s = map[status] || { className: "badge badge-muted", label: status };
  return <span className={s.className}>{s.label}</span>;
}

function fmtDate(s) {
  if (!s) return "—";
  try {
    return new Date(s).toLocaleDateString("ru-RU", { day: "2-digit", month: "short", year: "numeric" });
  } catch {
    return s;
  }
}

/** Диапазон годов из заявки: API может вернуть `year_from/year_to` или старые `year/years`. */
function formatTrainingYear(r) {
  const from = r.year_from ?? r.yearFrom;
  const to = r.year_to ?? r.yearTo;
  if (from && to && String(from) !== String(to)) return `${from}-${to}`;
  if (from) return String(from);
  const raw = r.year ?? r.years;
  if (raw == null) return "—";
  const s = String(raw).trim();
  return s || "—";
}

/** Собирает полный адрес для отображения (данные могут иметь раздельно city и address). */
function formatServiceRegistrationAddress(r) {
  const city = (r.city ?? "").trim();
  const addr = (r.address ?? "").trim();
  if (city && addr) return `г. ${city}, ${addr}`;
  if (city) return `г. ${city}`;
  return addr || "—";
}

function UploadModelModal({ onClose, onUploaded }) {
  const uploadAbortRef = useRef(null);
  const initialForm = {
    brand: "",
    model: "",
    generation: "",
    years: "",
    version: "v1",
    modelFile: null,
    partsConfigFile: null,
    partsCatalogFile: null,
  };
  const [form, setForm] = useState(initialForm);
  const [errors, setErrors] = useState({});
  const [submitError, setSubmitError] = useState("");

  useEffect(() => {
    uploadAbortRef.current = new AbortController();
    return () => uploadAbortRef.current?.abort();
  }, []);

  const clearError = (field) => {
    setErrors((prev) => {
      if (!prev[field]) return prev;
      const next = { ...prev };
      delete next[field];
      return next;
    });
  };

  const validate = () => {
    const e = {};
    if (!form.brand.trim()) e.brand = "Введите марку";
    if (!form.model.trim()) e.model = "Введите модель";
    if (!form.generation.trim()) e.generation = "Введите поколение";
    if (!form.years.trim()) e.years = "Введите год";
    if (!form.version.trim()) e.version = "Введите версию";
    if (!form.modelFile) e.modelFile = "Загрузите файл модели";
    if (!form.partsConfigFile) e.partsConfigFile = "Загрузите конфигурацию инференса";
    if (!form.partsCatalogFile) e.partsCatalogFile = "Загрузите parts catalog";
    return e;
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setSubmitError("");
    const nextErrors = validate();
    setErrors(nextErrors);
    if (Object.keys(nextErrors).length > 0) return;

    try {
      const created = await uploadMLModel(
        {
          brand: form.brand.trim(),
          model: form.model.trim(),
          generation: form.generation.trim(),
          years: form.years.trim(),
          version: form.version.trim(),
          modelFile: form.modelFile,
          partsConfigFile: form.partsConfigFile,
          partsCatalogFile: form.partsCatalogFile,
        },
        {
          signal: uploadAbortRef.current?.signal,
        }
      );
      onUploaded?.(created);
      onClose();
    } catch (err) {
      const msg =
        err instanceof Error && err.message.trim() ? err.message.trim() : "";
      setSubmitError(msg || "Не удалось загрузить модель");
    }
  };

  return (
    <div className="modal-overlay" role="dialog" aria-modal="true">
      <div className="modal-card">
        <form onSubmit={handleSubmit} noValidate>
          <div className="modal-header">
            <h3>Загрузка ML-модели</h3>
            <button type="button" className="btn btn-ghost btn-sm" onClick={onClose} aria-label="Закрыть">
              <Icon name="x" size={16} />
            </button>
          </div>

          <div className="modal-body">
            <div className="form-row-inline">
              <div className="form-row">
                <label className="form-label" htmlFor="um-brand">Марка *</label>
                <input
                  id="um-brand"
                  className={"input" + (errors.brand ? " input-error" : "")}
                  placeholder="Например, BMW"
                  value={form.brand}
                  onChange={(ev) => {
                    setForm((f) => ({ ...f, brand: ev.target.value }));
                    clearError("brand");
                  }}
                />
                {errors.brand && <div className="field-error">{errors.brand}</div>}
              </div>
              <div className="form-row">
                <label className="form-label" htmlFor="um-model">Модель *</label>
                <input
                  id="um-model"
                  className={"input" + (errors.model ? " input-error" : "")}
                  placeholder="Например, X5"
                  value={form.model}
                  onChange={(ev) => {
                    setForm((f) => ({ ...f, model: ev.target.value }));
                    clearError("model");
                  }}
                />
                {errors.model && <div className="field-error">{errors.model}</div>}
              </div>
            </div>

            <div className="form-row">
              <label className="form-label" htmlFor="um-version">Версия модели *</label>
              <input
                id="um-version"
                className={"input" + (errors.version ? " input-error" : "")}
                placeholder="v1"
                value={form.version}
                onChange={(ev) => {
                  setForm((f) => ({ ...f, version: ev.target.value }));
                  clearError("version");
                }}
              />
              {errors.version && <div className="field-error">{errors.version}</div>}
            </div>

            <div className="form-row-inline">
              <div className="form-row">
                <label className="form-label" htmlFor="um-gen">Поколение *</label>
                <input
                  id="um-gen"
                  className={"input" + (errors.generation ? " input-error" : "")}
                  placeholder="G05"
                  value={form.generation}
                  onChange={(ev) => {
                    setForm((f) => ({ ...f, generation: ev.target.value }));
                    clearError("generation");
                  }}
                />
                {errors.generation && <div className="field-error">{errors.generation}</div>}
              </div>
              <div className="form-row">
                <label className="form-label" htmlFor="um-years">Год *</label>
                <input
                  id="um-years"
                  className={"input" + (errors.years ? " input-error" : "")}
                  placeholder="2018"
                  value={form.years}
                  onChange={(ev) => {
                    setForm((f) => ({ ...f, years: ev.target.value }));
                    clearError("years");
                  }}
                />
                {errors.years && <div className="field-error">{errors.years}</div>}
              </div>
            </div>

            <div className="form-row-inline">
              <div className="form-row">
                <span className="form-label">Файл модели (.pt) *</span>
                <label className={"uploader-tile" + (errors.modelFile ? " uploader-tile-error" : "")}>
                  <Icon name="upload" size={16} />
                  <strong>{form.modelFile ? form.modelFile.name : "model.pt"}</strong>
                  <span className="uploader-tile-hint">Нажмите, чтобы выбрать</span>
                  <input
                    type="file"
                    accept=".pt"
                    className="uploader-tile-input"
                    onChange={(ev) => {
                      const file = ev.target.files?.[0] ?? null;
                      setForm((f) => ({ ...f, modelFile: file }));
                      clearError("modelFile");
                    }}
                  />
                </label>
                {errors.modelFile && <div className="field-error">{errors.modelFile}</div>}
              </div>
              <div className="form-row">
                <span className="form-label">Конфигурация инференса (.json) *</span>
                <label className={"uploader-tile" + (errors.partsConfigFile ? " uploader-tile-error" : "")}>
                  <Icon name="upload" size={16} />
                  <strong>{form.partsConfigFile ? form.partsConfigFile.name : "parts_inference_config.json"}</strong>
                  <span className="uploader-tile-hint">Нажмите, чтобы выбрать</span>
                  <input
                    type="file"
                    accept=".json,application/json"
                    className="uploader-tile-input"
                    onChange={(ev) => {
                      const file = ev.target.files?.[0] ?? null;
                      setForm((f) => ({ ...f, partsConfigFile: file }));
                      clearError("partsConfigFile");
                    }}
                  />
                </label>
                {errors.partsConfigFile && (
                  <div className="field-error">{errors.partsConfigFile}</div>
                )}
              </div>
            </div>

            <div className="form-row-inline">
              <div className="form-row">
                <span className="form-label">Каталог деталей (.json) *</span>
                <label className={"uploader-tile" + (errors.partsCatalogFile ? " uploader-tile-error" : "")}>
                  <Icon name="upload" size={16} />
                  <strong>{form.partsCatalogFile ? form.partsCatalogFile.name : "parts_catalog.json"}</strong>
                  <span className="uploader-tile-hint">Нажмите, чтобы выбрать</span>
                  <input
                    type="file"
                    accept=".json,application/json"
                    className="uploader-tile-input"
                    onChange={(ev) => {
                      const file = ev.target.files?.[0] ?? null;
                      setForm((f) => ({ ...f, partsCatalogFile: file }));
                      clearError("partsCatalogFile");
                    }}
                  />
                </label>
                {errors.partsCatalogFile && (
                  <div className="field-error">{errors.partsCatalogFile}</div>
                )}
              </div>
            </div>

            {submitError && (
              <div className="field-error" style={{ marginTop: 4 }}>{submitError}</div>
            )}
          </div>

          <div className="modal-footer">
            <button type="button" className="btn btn-ghost btn-sm" onClick={onClose}>
              Отмена
            </button>
            <button type="submit" className="btn btn-primary btn-sm">
              Загрузить
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

function RejectServiceRegistrationModal({ request, onClose, onConfirm }) {
  const [reason, setReason] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const submit = async (e) => {
    e.preventDefault();
    const cleanReason = reason.trim();
    if (!cleanReason) {
      setError("Укажите причину отклонения");
      return;
    }

    setSubmitting(true);
    setError("");
    try {
      await onConfirm(cleanReason);
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
    <div className="modal-overlay" role="dialog" aria-modal="true">
      <div className="modal-card modal-card-sm">
        <form onSubmit={submit} noValidate>
          <div className="modal-header">
            <h3>Отклонение заявки</h3>
            <button
              type="button"
              className="btn btn-ghost btn-sm"
              onClick={onClose}
              aria-label="Закрыть"
              disabled={submitting}
            >
              <Icon name="x" size={16} />
            </button>
          </div>

          <div className="modal-body">
            <div className="reject-service-summary">
              <div className="reject-service-summary-title">
                {request?.organization || "Заявка автосервиса"}
              </div>
              <div className="text-xs muted">
                {request ? formatServiceRegistrationAddress(request) : ""}
              </div>
            </div>

            <div className="form-row">
              <label className="form-label" htmlFor="reject-service-reason">
                Причина отклонения *
              </label>
              <textarea
                id="reject-service-reason"
                className={"textarea" + (error ? " input-error" : "")}
                placeholder="Например, недостаточно контактных данных или описание организации требует уточнения"
                value={reason}
                onChange={(ev) => {
                  setReason(ev.target.value);
                  if (error) setError("");
                }}
                disabled={submitting}
                autoFocus
              />
              {error ? <div className="field-error">{error}</div> : null}
            </div>
          </div>

          <div className="modal-footer">
            <button
              type="button"
              className="btn btn-ghost btn-sm"
              onClick={onClose}
              disabled={submitting}
            >
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

const TRAINING_STATUS_ACTIONS = {
  approved: {
    title: "Одобрение заявки",
    button: "Одобрить заявку",
    tone: "success",
    label: "Комментарий администратора",
    placeholder: "Например, заявка принята в работу после проверки доступности данных",
  },
  rejected: {
    title: "Отклонение заявки",
    button: "Отклонить заявку",
    tone: "danger",
    label: "Причина отклонения",
    placeholder: "Например, модель уже поддерживается или недостаточно данных для обучения",
  },
  in_progress: {
    title: "Начало обучения",
    button: "Перевести в обучение",
    tone: "training",
    label: "Комментарий администратора",
    placeholder: "Например, датасет передан ML-разработчику, обучение начато",
  },
  completed: {
    title: "Завершение заявки",
    button: "Отметить выполненной",
    tone: "training",
    label: "Комментарий администратора",
    placeholder: "Например, модель обучена и загружена в систему",
  },
};

function TrainingRequestStatusModal({ request, status, onClose, onConfirm }) {
  const config = TRAINING_STATUS_ACTIONS[status] || TRAINING_STATUS_ACTIONS.approved;
  const [comment, setComment] = useState(request?.admin_comment || "");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const isRequired = true;

  const submit = async (e) => {
    e.preventDefault();
    const cleanComment = comment.trim();
    if (isRequired && !cleanComment) {
      setError("Укажите комментарий администратора");
      return;
    }

    setSubmitting(true);
    setError("");
    try {
      await onConfirm(cleanComment);
    } catch (err) {
      const msg =
        err instanceof Error && err.message.trim()
          ? err.message.trim()
          : "Не удалось обновить заявку";
      setError(msg);
      setSubmitting(false);
    }
  };

  return (
    <div className="modal-overlay" role="dialog" aria-modal="true">
      <div className="modal-card modal-card-sm">
        <form onSubmit={submit} noValidate>
          <div className="modal-header">
            <h3>{config.title}</h3>
            <button
              type="button"
              className="btn btn-ghost btn-sm"
              onClick={onClose}
              aria-label="Закрыть"
              disabled={submitting}
            >
              <Icon name="x" size={16} />
            </button>
          </div>

          <div className="modal-body">
            <div className="reject-service-summary">
              <div className="reject-service-summary-title">
                {[request?.brand, request?.model, request?.generation].filter(Boolean).join(" ") || "Заявка на обучение"}
              </div>
              <div className="text-xs muted">Годы: {request ? formatTrainingYear(request) : "—"}</div>
            </div>

            <div className="form-row">
              <label className="form-label" htmlFor="training-admin-comment">
                {config.label}{isRequired ? " *" : ""}
              </label>
              <textarea
                id="training-admin-comment"
                className={"textarea" + (error ? " input-error" : "")}
                placeholder={config.placeholder}
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

          <div className="modal-footer">
            <button
              type="button"
              className="btn btn-ghost btn-sm"
              onClick={onClose}
              disabled={submitting}
            >
              Отмена
            </button>
            <button type="submit" className={(config.tone === "danger" ? "btn btn-danger" : "btn btn-primary") + " btn-sm"} disabled={submitting}>
              {config.button}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

function ModelsTab() {
  const [models, setModels] = useState([]);
  const [showUpload, setShowUpload] = useState(false);

  useEffect(() => {
    let cancelled = false;
    listMLModels()
      .then((data) => { if (!cancelled) setModels(data); })
      .catch(() => { if (!cancelled) setModels([]); });
    return () => { cancelled = true; };
  }, []);

  const deactivate = async (id) => {
    const row = models.find((m) => m.id === id);
    if (!row) return;
    setModels((prev) =>
      prev.map((m) => (m.id === id ? { ...m, status: "deprecated" } : m))
    );
    try {
      const updated = await deactivateMLModel(id, { snapshot: row });
      if (updated)
        setModels((prev) => prev.map((m) => (m.id === id ? updated : m)));
    } catch {
      setModels((prev) => prev.map((m) => (m.id === id ? row : m)));
    }
  };
  return (
    <>
      <div className="admin-toolbar">
        <span className="text-sm muted">Всего моделей: {models.length}</span>
        <div className="right">
          <button className="btn btn-primary btn-sm" onClick={() => setShowUpload(true)}>
            <Icon name="plus" size={14} /> Загрузить модель
          </button>
        </div>
      </div>

      <div className="table-wrap">
        <table className="table">
          <thead>
            <tr>
              <th>Марка</th>
              <th>Модель</th>
              <th>Поколение</th>
              <th>Год</th>
              <th>Дата</th>
              <th>Статус</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {models.map((m) => (
              <tr key={m.id}>
                <td><b style={{ color: "var(--text)" }}>{m.brand}</b></td>
                <td>{m.model}</td>
                <td>{m.generation}</td>
                <td>{m.years}</td>
                <td>{fmtDate(m.created_at)}</td>
                <td>{statusBadge(m.status)}</td>
                <td>
                  <div className="actions-col">
                    {m.status === "active" ? (
                      <button className="kbd-action danger" onClick={() => deactivate(m.id)}>
                        Деактивировать
                      </button>
                    ) : null}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {showUpload && (
        <UploadModelModal
          onClose={() => setShowUpload(false)}
          onUploaded={(created) => setModels((prev) => [created, ...prev])}
        />
      )}
    </>
  );
}

function ServiceRequestsTab() {
  const [items, setItems] = useState([]);
  const [expanded, setExpanded] = useState({});
  const [rejectTarget, setRejectTarget] = useState(null);

  useEffect(() => {
    let cancelled = false;
    listServiceRegistrations()
      .then((data) => { if (!cancelled) setItems(data); })
      .catch(() => { if (!cancelled) setItems([]); });
    return () => { cancelled = true; };
  }, []);

  const approve = async (id) => {
    const row = items.find((r) => r.id === id);
    if (!row || row.status !== "pending") return;
    setItems((prev) =>
      prev.map((r) => (r.id === id ? { ...r, status: "approved" } : r))
    );
    try {
      const merged = await approveServiceRegistration(id, {
        snapshot: row,
        from: row.status,
      });
      if (merged)
        setItems((prev) => prev.map((r) => (r.id === id ? merged : r)));
    } catch {
      setItems((prev) => prev.map((r) => (r.id === id ? row : r)));
    }
  };
  const reject = async (id, reason) => {
    const row = items.find((r) => r.id === id);
    if (!row || row.status !== "pending") return;
    setItems((prev) =>
      prev.map((r) => (r.id === id ? { ...r, status: "rejected" } : r))
    );
    try {
      const merged = await rejectServiceRegistration(id, {
        snapshot: row,
        from: row.status,
        reason,
      });
      if (merged)
        setItems((prev) => prev.map((r) => (r.id === id ? merged : r)));
      setRejectTarget(null);
    } catch {
      setItems((prev) => prev.map((r) => (r.id === id ? row : r)));
      throw new Error("Не удалось отклонить заявку");
    }
  };

  const toggleExpanded = (id) => {
    setExpanded((prev) => ({ ...prev, [id]: !prev[id] }));
  };

  return (
    <div className="table-wrap">
      <table className="table">
        <thead>
          <tr>
            <th>Организация</th>
            <th>Контакт</th>
            <th>Дата</th>
            <th>Статус</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {items.map((r) => {
            const email = r.contact_email || r.email;
            const isExpanded = Boolean(expanded[r.id]);
            const hasDetails =
              Boolean(r.description) ||
              Boolean(r.contact_info) ||
              Boolean(r.rejection_reason);
            return (
              <Fragment key={r.id}>
                <tr>
                  <td>
                    <b style={{ color: "var(--text)" }}>{r.organization}</b>
                    <div className="text-xs muted">{formatServiceRegistrationAddress(r)}</div>
                  </td>
                  <td>
                    {r.contact_phone ? (
                      <div>{r.contact_phone}</div>
                    ) : (
                      <span className="text-xs muted">—</span>
                    )}
                    {email ? <div className="text-xs muted">{email}</div> : null}
                  </td>
                  <td>{fmtDate(r.submitted_at)}</td>
                  <td>{statusBadge(r.status)}</td>
                  <td>
                    <div className="actions-col">
                      {hasDetails ? (
                        <button
                          type="button"
                          className="kbd-action"
                          onClick={() => toggleExpanded(r.id)}
                        >
                          {isExpanded ? "Скрыть" : "Подробнее"}
                        </button>
                      ) : null}
                      {r.status === "pending" ? (
                        <>
                          <button type="button" className="kbd-action success" onClick={() => approve(r.id)}>
                            Одобрить
                          </button>
                          <button type="button" className="kbd-action danger" onClick={() => setRejectTarget(r)}>
                            Отклонить
                          </button>
                        </>
                      ) : null}
                      {r.status !== "pending" && !hasDetails ? (
                        <span className="text-xs muted">—</span>
                      ) : null}
                    </div>
                  </td>
                </tr>
                {isExpanded ? (
                  <tr className="service-registration-details-row">
                    <td colSpan={5}>
                      <div className="service-registration-details">
                        {r.description ? (
                          <div>
                            <div className="service-registration-details-label">Описание</div>
                            <div className="service-registration-details-text">{r.description}</div>
                          </div>
                        ) : null}
                        {r.contact_info ? (
                          <div>
                            <div className="service-registration-details-label">Дополнительные контакты</div>
                            <div className="service-registration-details-text">{r.contact_info}</div>
                          </div>
                        ) : null}
                        {r.rejection_reason ? (
                          <div className="service-registration-details-reason">
                            <div className="service-registration-details-label">Причина отклонения</div>
                            <div className="service-registration-details-text">{r.rejection_reason}</div>
                          </div>
                        ) : null}
                      </div>
                    </td>
                  </tr>
                ) : null}
              </Fragment>
            );
          })}
        </tbody>
      </table>
      {rejectTarget ? (
        <RejectServiceRegistrationModal
          request={rejectTarget}
          onClose={() => setRejectTarget(null)}
          onConfirm={(reason) => reject(rejectTarget.id, reason)}
        />
      ) : null}
    </div>
  );
}

function TrainingRequestsTab() {
  const [items, setItems] = useState([]);
  const [statusAction, setStatusAction] = useState(null);

  useEffect(() => {
    let cancelled = false;
    listTrainingRequests()
      .then((data) => { if (!cancelled) setItems(data); })
      .catch(() => { if (!cancelled) setItems([]); });
    return () => { cancelled = true; };
  }, []);

  const openStatusAction = (request, status) => {
    if (!request) return;
    setStatusAction({ request, status });
  };

  const setStatus = async (id, status, adminComment) => {
    const row = items.find((r) => r.id === id);
    if (!row) return;
    setItems((prev) =>
      prev.map((r) =>
        r.id === id ? { ...r, status, admin_comment: adminComment } : r
      )
    );
    try {
      const merged = await updateTrainingRequestStatus(id, status, {
        snapshot: row,
        from: row.status,
        admin_comment: adminComment,
      });
      if (merged)
        setItems((prev) => prev.map((r) => (r.id === id ? merged : r)));
      setStatusAction(null);
    } catch {
      setItems((prev) => prev.map((r) => (r.id === id ? row : r)));
      throw new Error("Не удалось обновить заявку");
    }
  };

  return (
    <div>
      {items.map((r) => (
        <div key={r.id} className="training-card">
          <div className="training-card-meta">
            <div className="training-card-spec" aria-label="Параметры заявки на обучение">
              {[
                ["Марка", r.brand || "—"],
                ["Модель", r.model || "—"],
                ["Поколение", r.generation || "—"],
                ["Годы", formatTrainingYear(r)],
              ].map(([label, value]) => (
                <Fragment key={label}>
                  <span className="training-card-spec-label">{label}</span>
                  <span className="training-card-spec-value">{value}</span>
                </Fragment>
              ))}
            </div>
            <div className="training-card-desc">{r.description}</div>
            <div className="training-card-sub">Заявка от {fmtDate(r.submitted_at)}</div>
            {r.admin_comment ? (
              <div className="training-card-admin-comment">
                <span>Комментарий администратора</span>
                {r.admin_comment}
              </div>
            ) : null}
            <div>{trainingRequestStatusBadge(r.status)}</div>
          </div>
          <div className="actions-col">
            {r.status === "pending" && (
              <>
                <button type="button" className="kbd-action success" onClick={() => openStatusAction(r, "approved")}>
                  Одобрить
                </button>
                <button type="button" className="kbd-action danger" onClick={() => openStatusAction(r, "rejected")}>
                  Отклонить
                </button>
              </>
            )}
            {r.status === "approved" && (
              <button type="button" className="kbd-action" onClick={() => openStatusAction(r, "in_progress")}>
                В работу
              </button>
            )}
            {(r.status === "approved" || r.status === "in_progress") && (
              <>
                <button type="button" className="kbd-action training-completed" onClick={() => openStatusAction(r, "completed")}>
                  Выполнено
                </button>
                <button type="button" className="kbd-action danger" onClick={() => openStatusAction(r, "rejected")}>
                  Отклонить
                </button>
              </>
            )}
          </div>
        </div>
      ))}
      {statusAction ? (
        <TrainingRequestStatusModal
          request={statusAction.request}
          status={statusAction.status}
          onClose={() => setStatusAction(null)}
          onConfirm={(adminComment) =>
            setStatus(statusAction.request.id, statusAction.status, adminComment)
          }
        />
      ) : null}
    </div>
  );
}

function AdminPage() {
  const [tab, setTab] = useState("models");

  return (
    <div>
      <div className="page-header">
        <div>
          <h1 className="page-title">Панель администратора</h1>
          <p className="page-subtitle">Управление моделями и заявками</p>
        </div>
      </div>

      <div className="tabs mb-6" role="tablist">
        <button className={"tab-btn" + (tab === "models" ? " active" : "")} onClick={() => setTab("models")}>
          ML-модели
        </button>
        <button className={"tab-btn" + (tab === "services" ? " active" : "")} onClick={() => setTab("services")}>
          Заявки автосервисов
        </button>
        <button className={"tab-btn" + (tab === "training" ? " active" : "")} onClick={() => setTab("training")}>
          Заявки на обучение
        </button>
      </div>

      {tab === "models" && <ModelsTab />}
      {tab === "services" && <ServiceRequestsTab />}
      {tab === "training" && <TrainingRequestsTab />}
    </div>
  );
}

export default AdminPage;
