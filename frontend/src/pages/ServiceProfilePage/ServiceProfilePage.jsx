import "./ServiceProfilePage.css";
import { useEffect, useMemo, useRef, useState } from "react";
import Icon from "../../components/Icon/Icon";
import { normalizeApiError } from "../../services/apiFoundation";
import {
  readMyServiceProfileFromCache,
  getMyServiceProfile,
  saveMyServiceProfile,
  listSpecializationOptions,
  listSpecializations,
  replaceSpecializations,
  listServiceProfileImages,
  uploadServiceProfileImage,
  setPrimaryServiceProfileImage,
  deleteServiceProfileImage,
} from "../../services/serviceProfile";

const ANY_PART_CATEGORY_CODE = "*";

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
  const [specializationOptions, setSpecializationOptions] = useState({
    damage_types: [],
    part_categories: [],
  });
  const [specializations, setSpecializations] = useState([]);
  const [images, setImages] = useState([]);

  const isFresh = !form.name;

  const isValid = useMemo(() => {
    return Boolean(
      form.name.trim() &&
      form.city.trim() &&
      form.address.trim()
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

    Promise.all([
      listSpecializationOptions({ signal: ac.signal }),
      listSpecializations({ signal: ac.signal }),
      listServiceProfileImages({ signal: ac.signal }),
    ])
      .then(([options, items, imageItems]) => {
        if (cancelled) return;
        setSpecializationOptions({
          damage_types: Array.isArray(options?.damage_types) ? options.damage_types : [],
          part_categories: Array.isArray(options?.part_categories) ? options.part_categories : [],
        });
        setSpecializations(Array.isArray(items) ? items : []);
        setImages(Array.isArray(imageItems) ? imageItems : []);
      })
      .catch(() => {
        if (!cancelled) {
          setSpecializationOptions({ damage_types: [], part_categories: [] });
          setSpecializations([]);
        }
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
    if (!form.city.trim()) e.city = "Введите город";
    if (!form.address.trim()) e.address = "Введите адрес";
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

  const hasSpecialization = (damageTypeCode, partCategoryCode) =>
    specializations.some(
      (item) =>
        item.damage_type_code === damageTypeCode &&
        item.part_category_code === partCategoryCode
    );

  const hasAnyPartSpecialization = (damageTypeCode) =>
    hasSpecialization(damageTypeCode, ANY_PART_CATEGORY_CODE);

  const toggleSpecialization = (damageTypeCode, partCategoryCode) => {
    setSpecializations((prev) => {
      const exists = prev.some(
        (item) =>
          item.damage_type_code === damageTypeCode &&
          item.part_category_code === partCategoryCode
      );
      if (exists) {
        return prev.filter(
          (item) =>
            !(
              item.damage_type_code === damageTypeCode &&
              item.part_category_code === partCategoryCode
            )
        );
      }

      if (partCategoryCode === ANY_PART_CATEGORY_CODE) {
        return [
          ...prev.filter((item) => item.damage_type_code !== damageTypeCode),
          {
            id: `${damageTypeCode}:${partCategoryCode}`,
            damage_type_code: damageTypeCode,
            part_category_code: partCategoryCode,
          },
        ];
      }

      return [
        ...prev.filter(
          (item) =>
            !(
              item.damage_type_code === damageTypeCode &&
              item.part_category_code === ANY_PART_CATEGORY_CODE
            )
        ),
        {
          id: `${damageTypeCode}:${partCategoryCode}`,
          damage_type_code: damageTypeCode,
          part_category_code: partCategoryCode,
        },
      ];
    });
  };

  const handleImageUpload = async (event) => {
    const file = event.target.files?.[0];
    if (!file) return;
    try {
      const raw = await uploadServiceProfileImage(file, {
        isPrimary: images.length === 0,
      });
      const image = raw?.image;
      if (image) setImages((prev) => [image, ...prev]);
    } finally {
      event.target.value = "";
    }
  };

  const handleSetPrimaryImage = async (id) => {
    const previous = images;
    setImages((prev) => prev.map((img) => ({ ...img, is_primary: img.id === id })));
    try {
      const raw = await setPrimaryServiceProfileImage(id);
      if (raw?.image) {
        setImages((prev) => prev.map((img) => (img.id === id ? raw.image : { ...img, is_primary: false })));
      }
    } catch {
      setImages(previous);
    }
  };

  const handleDeleteImage = async (id) => {
    const previous = images;
    setImages((prev) => prev.filter((img) => img.id !== id));
    try {
      await deleteServiceProfileImage(id);
    } catch {
      setImages(previous);
    }
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
      await replaceSpecializations(
        specializations.map((item) => ({
          damage_type_code: item.damage_type_code,
          part_category_code: item.part_category_code,
        })),
        { signal: ac.signal }
      );

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
              <div className="label">Город</div>
              <div className="value"><Icon name="mapPin" size={14} /> {form.city || "—"}</div>
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

          <div className="section-divider">Специализация</div>

          {specializations.length === 0 ? (
            <p className="muted">Специализация не указана</p>
          ) : (
            <div className="profile-services-list">
              {specializations.map((s) => {
                const damage = specializationOptions.damage_types.find((d) => d.code === s.damage_type_code);
                const part = specializationOptions.part_categories.find((p) => p.code === s.part_category_code);
                return (
                <div className="profile-service-row" key={`${s.damage_type_code}:${s.part_category_code}`}>
                  <span className="name">{part?.name_ru || s.part_category_code}</span>
                  <span className="badge badge-muted">{damage?.name_ru || s.damage_type_code}</span>
                </div>
              );})}
            </div>
          )}

          <div className="section-divider">Фотографии</div>
          {images.length === 0 ? (
            <p className="muted">Фотографии не загружены</p>
          ) : (
            <div className="service-profile-images">
              {images.map((image) => (
                <div className="service-profile-image" key={image.id}>
                  <img src={image.url} alt={image.original_filename || "Фото автосервиса"} />
                  <div className="meta">
                    <span>{image.is_primary ? "Главное" : image.original_filename}</span>
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
            <label className="form-label">Город *</label>
            <input
              className={"input" + (errors.city ? " input-error" : "")}
              type="text"
              name="city"
              placeholder="Москва"
              value={form.city}
              onChange={handleChange}
            />
            {errors.city && <div className="field-error">{errors.city}</div>}
          </div>
          <div className="form-row">
            <label className="form-label">Телефон</label>
            <input
              className={"input" + (errors.phone ? " input-error" : "")}
              type="tel"
              placeholder="+7 (___) ___-__-__"
              value={form.phone}
              onChange={handlePhone}
            />
            {errors.phone && <div className="field-error">{errors.phone}</div>}
          </div>
        </div>

        <div className="form-row-inline">
          <div className="form-row">
            <label className="form-label">Email</label>
            <input
              className="input"
              type="email"
              name="email"
              placeholder="service@example.ru"
              value={form.email || ""}
              onChange={handleChange}
            />
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

        <div className="section-divider">Специализация по повреждениям и деталям</div>

        <div className="service-services-block">
          {specializationOptions.damage_types.length === 0 ||
          specializationOptions.part_categories.length === 0 ? (
            <p className="muted">Справочники специализаций пока недоступны</p>
          ) : (
            specializationOptions.damage_types.map((damage) => (
              <div key={damage.code} className="service-row selected">
                <div className="profile-service-row specialization-damage-row">
                  <span className="name">{damage.name_ru}</span>
                </div>
                <div className="level-pills">
                  {specializationOptions.part_categories.map((part) => {
                    const checked = hasSpecialization(damage.code, part.code);
                    const disabled =
                      part.code !== ANY_PART_CATEGORY_CODE &&
                      hasAnyPartSpecialization(damage.code);
                    return (
                      <button
                        key={part.code}
                        type="button"
                        className={
                          "level-pill" +
                          (checked ? " active" : "") +
                          (disabled ? " disabled" : "")
                        }
                        disabled={disabled}
                        onClick={() => toggleSpecialization(damage.code, part.code)}
                      >
                        {part.name_ru}
                      </button>
                    );
                  })}
                </div>
              </div>
            ))
          )}
        </div>

        <div className="section-divider">Фотографии автосервиса</div>
        <label className="service-profile-uploader">
          <Icon name="upload" size={16} />
          <strong>Загрузить фото</strong>
          <span className="form-hint">Первое фото станет главным</span>
          <input
            type="file"
            accept="image/*"
            onChange={handleImageUpload}
          />
        </label>

        {images.length > 0 && (
          <div className="service-profile-images" style={{ marginTop: 12 }}>
            {images.map((image) => (
              <div className="service-profile-image" key={image.id}>
                <img src={image.url} alt={image.original_filename || "Фото автосервиса"} />
                <div className="meta">{image.is_primary ? "Главное" : image.original_filename}</div>
                <div className="service-profile-image-actions">
                  {!image.is_primary && (
                    <button type="button" onClick={() => handleSetPrimaryImage(image.id)}>
                      Главное
                    </button>
                  )}
                  <button type="button" onClick={() => handleDeleteImage(image.id)}>
                    Удалить
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}

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
