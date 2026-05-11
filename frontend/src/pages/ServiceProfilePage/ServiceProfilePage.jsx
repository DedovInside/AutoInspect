import "./ServiceProfilePage.css";
import { useEffect, useMemo, useRef, useState } from "react";
import Icon from "../../components/Icon/Icon";
import { normalizeApiError } from "../../services/apiFoundation";
import { SERVICE_OPTIONS, DAMAGE_LEVELS } from "../../services/constants";
import {
  readMyServiceProfileFromCache,
  getMyServiceProfile,
  saveMyServiceProfile,
} from "../../services/serviceProfile";

function ServiceProfilePage() {
  const saveGenRef = useRef(0);
  const submitAcRef = useRef(null);

  const [form, setForm] = useState(readMyServiceProfileFromCache);
  const [errors, setErrors] = useState({});
  const [isEditing, setIsEditing] = useState(false);
  const [savedToast, setSavedToast] = useState(false);
  const [loadError, setLoadError] = useState("");
  const [saveError, setSaveError] = useState("");
  const [isSaving, setIsSaving] = useState(false);

  const isFresh = !form.name && form.services.length === 0;

  const isValid = useMemo(() => {
    return Boolean(
      form.name.trim() &&
      form.phone.trim() &&
      form.address.trim() &&
      form.services.length > 0 &&
      form.services.every(
        (s) => Array.isArray(s.levels) && s.levels.length > 0
      )
    );
  }, [form]);

  useEffect(() => {
    let cancelled = false;
    const ac = new AbortController();

    getMyServiceProfile({ signal: ac.signal })
      .then((profile) => {
        if (cancelled) return;
        setForm(profile);
        setLoadError("");
      })
      .catch((err) => {
        if (cancelled) return;
        const n = normalizeApiError(err);
        if (n.code === "aborted") return;
        setLoadError(err?.message ?? "Не удалось загрузить профиль.");
      });

    return () => {
      cancelled = true;
      ac.abort();
    };
  }, []);

  useEffect(() => {
    return () => {
      submitAcRef.current?.abort();
    };
  }, []);

  const validate = () => {
    const e = {};
    if (!form.name.trim()) e.name = "Введите название сервиса";
    if (!form.phone.trim()) e.phone = "Введите номер телефона";
    if (!form.address.trim()) e.address = "Введите адрес";
    if (form.services.length === 0) e.services = "Выберите хотя бы одну услугу";
    if (form.services.some((s) => !Array.isArray(s.levels) || s.levels.length === 0)) e.levels = "Выберите сложность для каждой услуги";
    return e;
  };

  const handleChange = (e) => {
    const { name, value } = e.target;
    setForm((p) => ({ ...p, [name]: value }));
    setErrors((p) => ({ ...p, [name]: "" }));
  };

  const handlePhone = (e) => {
    const value = e.target.value.replace(/[^\d+\-()\s]/g, "");
    setForm((p) => ({ ...p, phone: value }));
    setErrors((p) => ({ ...p, phone: "" }));
  };

  const toggleService = (service) => {
    setForm((prev) => {
      const exists = prev.services.find((s) => s.type === service);
      if (exists) {
        return { ...prev, services: prev.services.filter((s) => s.type !== service) };
      }
      return { ...prev, services: [...prev.services, { type: service, levels: [] }] };
    });
    setErrors((p) => ({ ...p, services: "", levels: "" }));
  };

  const toggleLevel = (service, level) => {
    setForm((prev) => ({
      ...prev,
      services: prev.services.map((s) => {
        if (s.type !== service) return s;
        const has = s.levels.includes(level);
        return { ...s, levels: has ? s.levels.filter((l) => l !== level) : [...s.levels, level] };
      }),
    }));
    setErrors((p) => ({ ...p, levels: "" }));
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    const v = validate();
    setErrors(v);
    if (Object.keys(v).length > 0) return;

    saveGenRef.current += 1;
    const seq = saveGenRef.current;

    submitAcRef.current?.abort();
    const ac = new AbortController();
    submitAcRef.current = ac;

    setIsSaving(true);
    setSaveError("");

    try {
      const saved = await saveMyServiceProfile(form, { signal: ac.signal });

      if (seq !== saveGenRef.current) {
        return;
      }

      setForm(saved);
      setIsEditing(false);
      setSavedToast(true);
      setTimeout(() => setSavedToast(false), 2200);
    } catch (err) {
      if (seq !== saveGenRef.current) return;
      const n = normalizeApiError(err);
      if (n.code === "aborted") return;
      setSaveError(err?.message ?? "Не удалось сохранить профиль.");
    } finally {
      setIsSaving(false);
    }
  };

  if (!isEditing) {
    return (
      <div>
        <div className="page-header">
          <div>
            <h1 className="page-title">{form.name || "Профиль автосервиса"}</h1>
            <p className="page-subtitle">Информация об автосервисе</p>
          </div>
          <button className="btn btn-primary btn-sm" onClick={() => setIsEditing(true)}>
            <Icon name="edit" size={14} /> Редактировать профиль
          </button>
        </div>

        {loadError ? (
          <div className="alert alert-danger mb-4" role="alert">{loadError}</div>
        ) : null}

        {savedToast && (
          <div className="alert alert-success mb-4">
            <span className="alert-icon"><Icon name="checkCircle" size={16} /></span>
            <div>Профиль сохранён</div>
          </div>
        )}

        <div className="profile-card">
          <div className="profile-info-grid">
            <div className="profile-info-item">
              <div className="label">Телефон</div>
              <div className="value"><Icon name="phone" size={14} /> {form.phone || "—"}</div>
            </div>
            <div className="profile-info-item">
              <div className="label">Адрес</div>
              <div className="value"><Icon name="mapPin" size={14} /> {form.address || "—"}</div>
            </div>
            <div className="profile-info-item" style={{ gridColumn: "span 2" }}>
              <div className="label">Описание</div>
              <div className="value" style={{ alignItems: "flex-start" }}>
                {form.description || <span className="muted">Описание не заполнено</span>}
              </div>
            </div>
          </div>

          <div className="section-divider">Услуги</div>

          {form.services.length === 0 ? (
            <p className="muted">Услуги не указаны</p>
          ) : (
            <div className="profile-services-list">
              {form.services.map((s) => (
                <div className="profile-service-row" key={s.type}>
                  <span className="name">{s.type}</span>
                  <div className="level-pills">
                    {(Array.isArray(s.levels) ? s.levels : []).length === 0 ? (
                      <span className="muted text-sm">Без уровня</span>
                    ) : (Array.isArray(s.levels) ? s.levels : []).map((lvl) => (
                      <span key={lvl} className="badge badge-muted">{lvl}</span>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    );
  }

  return (
    <div>
      <div className="page-header">
        <div>
          <h1 className="page-title">
            {isFresh ? "Заполните профиль автосервиса" : "Редактирование профиля"}
          </h1>
          <p className="page-subtitle">Заполните обязательные поля и выберите услуги с уровнями сложности</p>
        </div>
        {!isFresh && (
          <button className="btn btn-ghost btn-sm" onClick={() => setIsEditing(false)}>
            Отмена
          </button>
        )}
      </div>

      {loadError ? (
        <div className="alert alert-danger mb-4" role="alert">{loadError}</div>
      ) : null}

      <form onSubmit={handleSubmit} className="profile-card">
        {saveError ? (
          <div className="alert alert-danger mb-4" role="alert">{saveError}</div>
        ) : null}
        <div className="form-row">
          <label className="form-label" htmlFor="sp-name">Название сервиса *</label>
          <input
            id="sp-name"
            className={"input" + (errors.name ? " input-error" : "")}
            type="text"
            name="name"
            placeholder="Например, AutoFix"
            value={form.name}
            onChange={handleChange}
          />
          {errors.name && <div className="field-error">{errors.name}</div>}
        </div>

        <div className="form-row-inline">
          <div className="form-row">
            <label className="form-label">Телефон *</label>
            <input
              className={"input" + (errors.phone ? " input-error" : "")}
              type="tel"
              placeholder="+7 (___) ___-__-__"
              value={form.phone}
              onChange={handlePhone}
            />
            {errors.phone && <div className="field-error">{errors.phone}</div>}
          </div>
          <div className="form-row">
            <label className="form-label">Адрес *</label>
            <input
              className={"input" + (errors.address ? " input-error" : "")}
              type="text"
              name="address"
              placeholder="Город, улица, дом"
              value={form.address}
              onChange={handleChange}
            />
            {errors.address && <div className="field-error">{errors.address}</div>}
          </div>
        </div>

        <div className="form-row">
          <label className="form-label">Описание</label>
          <textarea
            className="textarea"
            name="description"
            placeholder="Расскажите о сервисе (до 300 символов)"
            maxLength={300}
            value={form.description}
            onChange={handleChange}
          />
          <div className="form-hint">{form.description.length}/300 символов</div>
        </div>

        <div className="section-divider">Услуги и уровни сложности</div>

        <div className="service-services-block">
          {SERVICE_OPTIONS.map((service) => {
            const selected = form.services.find((s) => s.type === service);
            return (
              <div key={service} className={"service-row" + (selected ? " selected" : "")}>
                <label className="checkbox-row">
                  <input
                    type="checkbox"
                    checked={!!selected}
                    onChange={() => toggleService(service)}
                  />
                  <span>{service}</span>
                </label>
                {selected && (
                  <div className="level-pills">
                    {DAMAGE_LEVELS.map((level) => (
                      <button
                        key={level}
                        type="button"
                        className={
                          "level-pill" +
                          ((Array.isArray(selected.levels)
                            ? selected.levels
                            : []
                          ).includes(level)
                            ? " active"
                            : "")
                        }
                        onClick={() => toggleLevel(service, level)}
                      >
                        {level}
                      </button>
                    ))}
                  </div>
                )}
              </div>
            );
          })}
        </div>

        {errors.services && <div className="field-error mt-2">{errors.services}</div>}
        {errors.levels && <div className="field-error mt-2">{errors.levels}</div>}

        <div className="flex gap-3 mt-6">
          <button type="submit" className="btn btn-primary" disabled={!isValid || isSaving}>
            Сохранить
          </button>
          {!isFresh && (
            <button type="button" className="btn btn-secondary" onClick={() => setIsEditing(false)}>
              Отмена
            </button>
          )}
        </div>
      </form>
    </div>
  );
}

export default ServiceProfilePage;
