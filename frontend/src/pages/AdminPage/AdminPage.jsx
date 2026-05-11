import "./AdminPage.css";
import { Fragment, useEffect, useRef, useState } from "react";
import Icon from "../../components/Icon/Icon";
import {
  listMLModels,
  uploadMLModel,
  activateMLModel,
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

/** Год из заявки: API может вернуть `year` или `years` (как у записей ML-моделей). */
function formatTrainingYear(r) {
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
    modelFile: null,
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
    if (!form.modelFile) e.modelFile = "Загрузите файл модели";
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
          modelFile: form.modelFile,
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
  const activate = async (id) => {
    const row = models.find((m) => m.id === id);
    if (!row) return;
    setModels((prev) =>
      prev.map((m) => (m.id === id ? { ...m, status: "active" } : m))
    );
    try {
      const updated = await activateMLModel(id, { snapshot: row });
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
                    ) : (
                      <button className="kbd-action success" onClick={() => activate(m.id)}>
                        Активировать
                      </button>
                    )}
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
  const reject = async (id) => {
    const row = items.find((r) => r.id === id);
    if (!row || row.status !== "pending") return;
    setItems((prev) =>
      prev.map((r) => (r.id === id ? { ...r, status: "rejected" } : r))
    );
    try {
      const merged = await rejectServiceRegistration(id, {
        snapshot: row,
        from: row.status,
        reason: "",
      });
      if (merged)
        setItems((prev) => prev.map((r) => (r.id === id ? merged : r)));
    } catch {
      setItems((prev) => prev.map((r) => (r.id === id ? row : r)));
    }
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
            return (
              <tr key={r.id}>
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
                  {r.status === "pending" ? (
                    <div className="actions-col">
                      <button type="button" className="kbd-action success" onClick={() => approve(r.id)}>
                        Одобрить
                      </button>
                      <button type="button" className="kbd-action danger" onClick={() => reject(r.id)}>
                        Отклонить
                      </button>
                    </div>
                  ) : (
                    <span className="text-xs muted">—</span>
                  )}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

function TrainingRequestsTab() {
  const [items, setItems] = useState([]);

  useEffect(() => {
    let cancelled = false;
    listTrainingRequests()
      .then((data) => { if (!cancelled) setItems(data); })
      .catch(() => { if (!cancelled) setItems([]); });
    return () => { cancelled = true; };
  }, []);

  const setStatus = async (id, status) => {
    const row = items.find((r) => r.id === id);
    if (!row) return;
    setItems((prev) =>
      prev.map((r) => (r.id === id ? { ...r, status } : r))
    );
    try {
      const merged = await updateTrainingRequestStatus(id, status, {
        snapshot: row,
        from: row.status,
      });
      if (merged)
        setItems((prev) => prev.map((r) => (r.id === id ? merged : r)));
    } catch {
      setItems((prev) => prev.map((r) => (r.id === id ? row : r)));
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
                ["Год", formatTrainingYear(r)],
              ].map(([label, value]) => (
                <Fragment key={label}>
                  <span className="training-card-spec-label">{label}</span>
                  <span className="training-card-spec-value">{value}</span>
                </Fragment>
              ))}
            </div>
            <div className="training-card-desc">{r.description}</div>
            <div className="training-card-sub">Заявка от {fmtDate(r.submitted_at)}</div>
            <div>{trainingRequestStatusBadge(r.status)}</div>
          </div>
          <div className="actions-col">
            {r.status === "pending" && (
              <>
                <button type="button" className="kbd-action success" onClick={() => setStatus(r.id, "approved")}>
                  Одобрить
                </button>
                <button type="button" className="kbd-action danger" onClick={() => setStatus(r.id, "rejected")}>
                  Отклонить
                </button>
              </>
            )}
            {(r.status === "approved" || r.status === "in_progress") && (
              <button type="button" className="kbd-action training-completed" onClick={() => setStatus(r.id, "completed")}>
                Выполнено
              </button>
            )}
          </div>
        </div>
      ))}
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
