import "../ServiceProfilePage/ServiceProfilePage.css";
import "./UserProfilePage.css";
import { useEffect, useMemo, useRef, useState } from "react";
import Icon from "../../components/Icon/Icon";
import { useAuth } from "../../auth/AuthContext";
import { getRoleLabel } from "../../auth/roleLabels";
import { normalizeApiError } from "../../services/apiFoundation";
import {
  getMyUserProfile,
  saveMyUserProfile,
  resolveProfileDisplayName,
} from "../../services/userProfile";

const emptyProfileForm = {
  contact_name: "",
  contact_phone: "",
  contact_email: "",
  email: "",
  avatar_url: "",
  username: "",
  display_name: "",
  first_name: "",
  last_name: "",
};

function profileFormFromApi(profile) {
  return {
    contact_name: profile.contact_name || "",
    contact_phone: profile.contact_phone || "",
    contact_email: profile.contact_email || "",
    email: profile.email || "",
    avatar_url: profile.avatar_url || "",
    username: profile.username || "",
    display_name: profile.display_name || "",
    first_name: profile.first_name || "",
    last_name: profile.last_name || "",
  };
}

function accountDisplayName(form) {
  return (
    form.display_name ||
    [form.first_name, form.last_name].filter(Boolean).join(" ") ||
    form.username ||
    form.email ||
    "Пользователь"
  );
}

function isValidEmail(value) {
  const email = String(value ?? "").trim();
  if (!email) return true;
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
}

function phoneDigits(value) {
  return String(value ?? "").replace(/\D/g, "");
}

function UserProfilePage() {
  const { role, syncFromStorage } = useAuth();
  const saveGenRef = useRef(0);
  const submitAcRef = useRef(null);

  const [form, setForm] = useState(emptyProfileForm);
  const [errors, setErrors] = useState({});
  const [isEditing, setIsEditing] = useState(false);
  const [savedToast, setSavedToast] = useState(false);
  const [loadError, setLoadError] = useState("");
  const [saveError, setSaveError] = useState("");
  const [isSaving, setIsSaving] = useState(false);
  const [avatarBroken, setAvatarBroken] = useState(false);

  const displayName = useMemo(() => resolveProfileDisplayName(form), [form]);
  const oauthName = useMemo(() => accountDisplayName(form), [form]);
  const roleLabel = getRoleLabel(role) || "Пользователь";
  const initialsSource = (displayName || form.email || "U").trim();
  const initials = initialsSource.slice(0, 2).toUpperCase();
  const showAvatarImage = Boolean(form.avatar_url) && !avatarBroken;

  useEffect(() => {
    setAvatarBroken(false);
  }, [form.avatar_url]);

  useEffect(() => {
    let cancelled = false;
    const ac = new AbortController();

    getMyUserProfile({ signal: ac.signal })
      .then((profile) => {
        if (cancelled) return;
        setForm(profileFormFromApi(profile));
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
    if (form.contact_phone.trim() && phoneDigits(form.contact_phone).length < 7) {
      e.contact_phone = "Введите корректный номер телефона";
    }
    if (!isValidEmail(form.contact_email)) {
      e.contact_email = "Введите корректный email";
    }
    return e;
  };

  const updateField = (field, value) => {
    setForm((prev) => ({ ...prev, [field]: value }));
    setErrors((prev) => ({ ...prev, [field]: "" }));
  };

  const handlePhone = (event) => {
    const value = event.target.value.replace(/[^\d+\-()\s]/g, "");
    updateField("contact_phone", value);
  };

  const handleSubmit = async (event) => {
    event.preventDefault();
    const validationErrors = validate();
    setErrors(validationErrors);
    if (Object.keys(validationErrors).length > 0) return;

    saveGenRef.current += 1;
    const seq = saveGenRef.current;

    submitAcRef.current?.abort();
    const ac = new AbortController();
    submitAcRef.current = ac;

    setIsSaving(true);
    setSaveError("");

    try {
      const saved = await saveMyUserProfile(
        {
          contact_name: form.contact_name,
          contact_phone: form.contact_phone,
          contact_email: form.contact_email,
        },
        { signal: ac.signal }
      );

      if (seq !== saveGenRef.current) return;

      setForm(profileFormFromApi(saved));
      syncFromStorage();
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

  const handleCancelEdit = () => {
    setIsEditing(false);
    setSaveError("");
    setErrors({});
  };

  if (!isEditing) {
    return (
      <div>
        <div className="user-profile-header">
          <div className="user-profile-identity">
            <div className="user-profile-avatar" aria-hidden="true">
              {showAvatarImage ? (
                <img
                  src={form.avatar_url}
                  alt={oauthName}
                  onError={() => setAvatarBroken(true)}
                  referrerPolicy="no-referrer"
                />
              ) : (
                initials
              )}
            </div>
            <div className="user-profile-title-block">
              <h1 className="page-title">Контактный профиль</h1>
              <div className="user-profile-email">{form.email || "—"}</div>
              <div className="user-profile-role">
                <span className="badge badge-muted">{roleLabel}</span>
              </div>
            </div>
          </div>
        </div>

        {loadError ? (
          <div className="alert alert-danger mb-4" role="alert">
            {loadError}
          </div>
        ) : null}

        {savedToast ? (
          <div className="alert alert-success mb-4">
            <span className="alert-icon">
              <Icon name="checkCircle" size={16} />
            </span>
            <div>Профиль сохранён</div>
          </div>
        ) : null}

        <div className="profile-card user-profile-card">
          <div className="user-profile-card-header">
            <div>
              <div className="user-profile-card-title">Контакты для заявок</div>
              <div className="user-profile-card-subtitle">Эти данные подставляются при создании заявки на ремонт</div>
            </div>
            <button className="btn btn-primary btn-sm" onClick={() => setIsEditing(true)}>
              <Icon name="edit" size={14} /> Редактировать
            </button>
          </div>

          <div className="profile-info-grid">
            <div className="profile-info-item">
              <div className="label">Как к вам обращаться</div>
              <div className="value">
                <Icon name="user" size={14} /> {form.contact_name || "—"}
              </div>
            </div>
            <div className="profile-info-item">
              <div className="label">Телефон</div>
              <div className="value">
                <Icon name="phone" size={14} /> {form.contact_phone || "—"}
              </div>
            </div>
            <div className="profile-info-item">
              <div className="label">Email для связи</div>
              <div className="value">
                <Icon name="mail" size={14} /> {form.contact_email || "—"}
              </div>
            </div>
          </div>

          <div className="section-divider">Аккаунт Яндекс</div>
          <div className="profile-info-grid">
            <div className="profile-info-item">
              <div className="label">Имя в аккаунте</div>
              <div className="value">{oauthName}</div>
            </div>
            <div className="profile-info-item">
              <div className="label">Email аккаунта</div>
              <div className="value">{form.email || "—"}</div>
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div>
      <div className="page-header">
        <div>
          <h1 className="page-title">Редактирование контактов</h1>
          <p className="page-subtitle">Эти данные будут подставляться в заявки на ремонт</p>
        </div>
        <button className="btn btn-ghost btn-sm" onClick={handleCancelEdit}>
          Отмена
        </button>
      </div>

      {loadError ? (
        <div className="alert alert-danger mb-4" role="alert">
          {loadError}
        </div>
      ) : null}

      <form onSubmit={handleSubmit} className="profile-card user-profile-card">
        {saveError ? (
          <div className="alert alert-danger mb-4" role="alert">
            {saveError}
          </div>
        ) : null}

        <div className="section-divider" style={{ marginTop: 0, paddingTop: 0, borderTop: "none" }}>
          Контакты для автосервиса
        </div>

        <div className="form-row">
          <label className="form-label" htmlFor="up-contact-name">
            Как к вам обращаться
          </label>
          <input
            id="up-contact-name"
            className="input"
            type="text"
            placeholder="Иван"
            value={form.contact_name}
            onChange={(event) => updateField("contact_name", event.target.value)}
          />
        </div>

        <div className="form-row-inline">
          <div className="form-row">
            <label className="form-label" htmlFor="up-contact-phone">
              Телефон
            </label>
            <input
              id="up-contact-phone"
              className={"input" + (errors.contact_phone ? " input-error" : "")}
              type="tel"
              placeholder="+7 (___) ___-__-__"
              value={form.contact_phone}
              onChange={handlePhone}
            />
            {errors.contact_phone ? <div className="field-error">{errors.contact_phone}</div> : null}
          </div>
          <div className="form-row">
            <label className="form-label" htmlFor="up-contact-email">
              Email для связи
            </label>
            <input
              id="up-contact-email"
              className={"input" + (errors.contact_email ? " input-error" : "")}
              type="email"
              placeholder="name@example.com"
              value={form.contact_email}
              onChange={(event) => updateField("contact_email", event.target.value)}
            />
            {errors.contact_email ? <div className="field-error">{errors.contact_email}</div> : null}
          </div>
        </div>

        <div className="section-divider">Аккаунт Яндекс</div>
        <div className="profile-info-grid user-profile-account-grid">
          <div className="profile-info-item">
            <div className="label">Имя в аккаунте</div>
            <div className="value">{oauthName}</div>
          </div>
          <div className="profile-info-item">
            <div className="label">Email аккаунта</div>
            <div className="value">{form.email || "—"}</div>
          </div>
        </div>

        <div className="flex gap-3 mt-6">
          <button type="submit" className="btn btn-primary" disabled={isSaving}>
            Сохранить
          </button>
          <button type="button" className="btn btn-secondary" onClick={handleCancelEdit} disabled={isSaving}>
            Отмена
          </button>
        </div>
      </form>
    </div>
  );
}

export default UserProfilePage;
