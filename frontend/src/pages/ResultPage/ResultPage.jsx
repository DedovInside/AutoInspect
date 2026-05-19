import "./ResultPage.css";
import { useEffect, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import {
  getAnalysis,
  getAnalysisImageURL,
  getMatchingCarServices,
  isFailedTerminalAnalysisStatus,
  isSuccessTerminalAnalysisStatus,
} from "../../services/analyses";
import { normalizeApiError } from "../../services/apiFoundation";
import { createRepairRequest } from "../../services/repairRequests";
import Icon from "../../components/Icon/Icon";
import { useAuth } from "../../auth/AuthContext";
import DamageOverlayImage from "../../components/DamageOverlayImage/DamageOverlayImage";

function severityBadgeClass(s) {
  if (s === "Сложный") return "badge badge-severity-heavy";
  if (s === "Средний") return "badge badge-severity-medium";
  if (s === "Лёгкий") return "badge badge-severity-light";
  if (s === "Без повреждений") return "badge badge-severity-clear";
  return "badge badge-muted";
}

function analysisStatusLabel(status) {
  const map = {
    pending: "Ожидает обработки",
    queued: "В очереди",
    processing: "В обработке",
    done: "Завершён",
    failed: "Ошибка анализа",
  };
  return map[status] || "Статус неизвестен";
}

function formatDate(iso) {
  try {
    return new Date(iso).toLocaleString("ru-RU", {
      day: "2-digit",
      month: "long",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return iso;
  }
}

function ResultLoading() {
  return (
    <div>
      <div className="page-header">
        <div>
          <h1 className="page-title">Анализируем изображения...</h1>
          <p className="page-subtitle">Это может занять несколько секунд</p>
        </div>
      </div>
      <div className="result-loading">
        <div className="result-loading-spinner" aria-hidden="true" />
        <h3>Идёт обработка</h3>
        <p>Сервер выполняет распознавание повреждений...</p>
      </div>
    </div>
  );
}

const ANALYSIS_POLL_MS = 3000;
const ANALYSIS_MAX_WAIT_MS = 15 * 60 * 1000;
const MIN_CONTACT_PHONE_DIGITS = 7;

const initialRepairContactForm = {
  name: "",
  phone: "",
  email: "",
  comment: "",
};

function contactPhoneDigits(value) {
  return String(value ?? "").replace(/\D/g, "");
}

function isValidContactEmail(value) {
  const email = String(value ?? "").trim();
  if (!email) return true;
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
}

function validateRepairContactForm(form) {
  const errors = {};
  const hasPhone = contactPhoneDigits(form.phone).length > 0;
  const hasEmail = String(form.email ?? "").trim() !== "";

  if (!hasPhone && !hasEmail) {
    errors.contact = "Укажите телефон или email для связи";
  }
  if (hasPhone && contactPhoneDigits(form.phone).length < MIN_CONTACT_PHONE_DIGITS) {
    errors.phone = "Введите корректный номер телефона";
  }
  if (hasEmail && !isValidContactEmail(form.email)) {
    errors.email = "Введите корректный email";
  }
  if (String(form.comment ?? "").length > 2000) {
    errors.comment = "Комментарий не должен превышать 2000 символов";
  }

  return errors;
}

function ResultErrorBanner({ message, onBack }) {
  return (
    <div>
      <div className="page-header">
        <div>
          <button
            type="button"
            className="btn btn-ghost btn-sm"
            onClick={onBack}
            style={{ marginBottom: 6, marginLeft: -10 }}
          >
            <Icon name="arrowLeft" size={14} /> К истории
          </button>
          <h1 className="page-title">Результат анализа</h1>
        </div>
      </div>
      <div className="alert alert-danger mt-4" role="alert">
        {message}
      </div>
    </div>
  );
}

function resultImages(result) {
  return Array.isArray(result?.results) ? result.results : [];
}

function ServiceGalleryModal({ service, index, setIndex, onClose }) {
  const images = Array.isArray(service?.images) ? service.images : [];
  const current = images[index] || images[0] || null;
  const hasMany = images.length > 1;
  const hasContacts =
    Boolean(service?.city) ||
    Boolean(service?.address && service.address !== "—") ||
    Boolean(service?.phone && service.phone !== "—") ||
    Boolean(service?.email);

  const shift = (delta) => {
    if (!hasMany) return;
    setIndex((prev) => (prev + delta + images.length) % images.length);
  };

  return (
    <div className="service-gallery-overlay" role="dialog" aria-modal="true">
      <div className="service-gallery-modal">
        <div className="service-gallery-header">
          <div>
            <div className="service-gallery-title">{service.name}</div>
            <div className="service-gallery-subtitle">
              Фото {index + 1} из {images.length}
            </div>
          </div>
          <button type="button" className="btn btn-ghost btn-sm" onClick={onClose} aria-label="Закрыть">
            <Icon name="x" size={16} />
          </button>
        </div>

        <div className="service-gallery-body">
          {current && hasMany ? (
            <button
              type="button"
              className="service-gallery-arrow service-gallery-arrow-left"
              onClick={() => shift(-1)}
              aria-label="Предыдущее фото"
            >
              <Icon name="arrowLeft" size={18} />
            </button>
          ) : null}

          {current ? (
            <img src={current.url} alt={current.original_filename || service.name} />
          ) : (
            <div className="service-gallery-placeholder">
              <Icon name="building" size={26} />
              <span>Фотографии автосервиса не загружены</span>
            </div>
          )}

          {current && hasMany ? (
            <button
              type="button"
              className="service-gallery-arrow service-gallery-arrow-right"
              onClick={() => shift(1)}
              aria-label="Следующее фото"
            >
              <Icon name="arrowRight" size={18} />
            </button>
          ) : null}
        </div>

        <div className="service-gallery-details">
          {service?.description ? (
            <div className="service-gallery-info-block">
              <div className="service-gallery-info-label">Описание</div>
              <div className="service-gallery-description">{service.description}</div>
            </div>
          ) : (
            <div className="service-gallery-empty-note">Описание автосервиса не заполнено</div>
          )}

          {hasContacts ? (
            <div className="service-gallery-info-grid">
              {service.city ? (
                <div>
                  <div className="service-gallery-info-label">Город</div>
                  <div className="service-gallery-info-value">{service.city}</div>
                </div>
              ) : null}
              {service.address && service.address !== "—" ? (
                <div>
                  <div className="service-gallery-info-label">Адрес</div>
                  <div className="service-gallery-info-value">{service.address}</div>
                </div>
              ) : null}
              {service.phone && service.phone !== "—" ? (
                <div>
                  <div className="service-gallery-info-label">Телефон</div>
                  <div className="service-gallery-info-value">{service.phone}</div>
                </div>
              ) : null}
              {service.email ? (
                <div>
                  <div className="service-gallery-info-label">Email</div>
                  <div className="service-gallery-info-value">{service.email}</div>
                </div>
              ) : null}
            </div>
          ) : null}
        </div>
      </div>
    </div>
  );
}

function RepairContactModal({
  service,
  form,
  errors,
  submitting,
  onChange,
  onClose,
  onSubmit,
}) {
  return (
    <div className="repair-contact-overlay" role="dialog" aria-modal="true">
      <div className="repair-contact-modal">
        <div className="repair-contact-modal-header">
          <div>
            <h3>Заявка на ремонт</h3>
            <p>
              {service?.name
                ? `Автосервис: ${service.name}`
                : "Укажите контакты, чтобы автосервис мог связаться с вами."}
            </p>
          </div>
          <button type="button" className="btn btn-ghost btn-sm" onClick={onClose} aria-label="Закрыть">
            <Icon name="x" size={16} />
          </button>
        </div>

        <form className="repair-contact-form" onSubmit={onSubmit} noValidate>
          {errors.contact ? (
            <div className="alert alert-danger" role="alert">
              {errors.contact}
            </div>
          ) : null}

          <div className="form-row">
            <label className="form-label" htmlFor="repair-customer-name">Имя</label>
            <input
              id="repair-customer-name"
              className="input"
              value={form.name}
              onChange={(event) => onChange("name", event.target.value)}
              placeholder="Как к вам обращаться"
            />
          </div>

          <div className="repair-contact-grid">
            <div className="form-row">
              <label className="form-label" htmlFor="repair-customer-phone">Телефон</label>
              <input
                id="repair-customer-phone"
                type="tel"
                className={"input" + (errors.phone ? " input-error" : "")}
                value={form.phone}
                onChange={(event) => onChange("phone", event.target.value)}
                placeholder="+7 (___) ___-__-__"
              />
              {errors.phone ? <div className="field-error">{errors.phone}</div> : null}
            </div>

            <div className="form-row">
              <label className="form-label" htmlFor="repair-customer-email">Email</label>
              <input
                id="repair-customer-email"
                type="email"
                className={"input" + (errors.email ? " input-error" : "")}
                value={form.email}
                onChange={(event) => onChange("email", event.target.value)}
                placeholder="name@example.com"
              />
              {errors.email ? <div className="field-error">{errors.email}</div> : null}
            </div>
          </div>

          <div className="form-row">
            <label className="form-label" htmlFor="repair-customer-comment">Комментарий</label>
            <textarea
              id="repair-customer-comment"
              className={"textarea" + (errors.comment ? " input-error" : "")}
              value={form.comment}
              onChange={(event) => onChange("comment", event.target.value)}
              placeholder="Например, когда удобно связаться или приехать на осмотр"
            />
            {errors.comment ? <div className="field-error">{errors.comment}</div> : null}
          </div>

          <div className="repair-contact-actions">
            <button type="button" className="btn btn-secondary" onClick={onClose} disabled={submitting}>
              Отмена
            </button>
            <button type="submit" className="btn btn-primary" disabled={submitting}>
              Отправить заявку
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

function ResultPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const { role } = useAuth();

  const [loading, setLoading] = useState(true);
  const [analysis, setAnalysis] = useState(null);
  const [loadError, setLoadError] = useState("");
  const [selectedServiceId, setSelectedServiceId] = useState(null);
  const [matchedServices, setMatchedServices] = useState([]);
  const [servicesLoading, setServicesLoading] = useState(false);
  const [imageURLs, setImageURLs] = useState([]);
  const [toast, setToast] = useState("");
  const [repairSubmitting, setRepairSubmitting] = useState(false);
  const [repairError, setRepairError] = useState("");
  const [repairContactOpen, setRepairContactOpen] = useState(false);
  const [repairContactForm, setRepairContactForm] = useState(initialRepairContactForm);
  const [repairContactErrors, setRepairContactErrors] = useState({});
  const [galleryService, setGalleryService] = useState(null);
  const [galleryIndex, setGalleryIndex] = useState(0);
  const repairCreateLockRef = useRef(false);
  const repairCreateMountedRef = useRef(true);

  useEffect(() => {
    if (!id) {
      setLoading(false);
      setLoadError("Некорректный идентификатор анализа");
      return undefined;
    }

    let cancelled = false;
    /** @type {ReturnType<typeof setInterval> | null} */
    let intervalId = null;
    let requestInFlight = false;
    const ac = new AbortController();
    const deadline = Date.now() + ANALYSIS_MAX_WAIT_MS;

    const stopPolling = () => {
      if (intervalId != null) {
        clearInterval(intervalId);
        intervalId = null;
      }
      ac.abort();
    };

    const fail = (message) => {
      if (cancelled) return;
      stopPolling();
      setLoadError(message);
      setLoading(false);
    };

    const succeed = (data) => {
      if (cancelled) return;
      stopPolling();
      setAnalysis(data);
      setLoadError("");
      setLoading(false);
    };

    const tick = async () => {
      if (cancelled || requestInFlight) return;
      if (Date.now() > deadline) {
        fail("Превышено время ожидания результата анализа.");
        return;
      }
      requestInFlight = true;
      try {
        const data = await getAnalysis(id, { signal: ac.signal });
        if (cancelled) return;

        if (isSuccessTerminalAnalysisStatus(data.status)) {
          succeed(data);
          return;
        }
        if (isFailedTerminalAnalysisStatus(data.status)) {
          fail(
            "Не удалось выполнить анализ. Попробуйте загрузить изображения ещё раз."
          );
          return;
        }
      } catch (e) {
        if (cancelled) return;
        if (e?.name === "AbortError" || e?.code === "aborted") return;
        fail(e?.message || "Ошибка при получении результата анализа");
      } finally {
        requestInFlight = false;
      }
    };

    setLoading(true);
    setLoadError("");
    setAnalysis(null);

    tick();
    intervalId = setInterval(tick, ANALYSIS_POLL_MS);

    return () => {
      cancelled = true;
      stopPolling();
    };
  }, [id]);

  useEffect(() => {
    if (!analysis || !isSuccessTerminalAnalysisStatus(analysis.status)) return undefined;

    let cancelled = false;
    const ac = new AbortController();
    setServicesLoading(true);

    const imageResults = resultImages(analysis.result);
    const imageCount = Math.max(
      Number(analysis.image_count) || 0,
      imageResults.length
    );

    Promise.allSettled([
      getMatchingCarServices(analysis.id || id, { signal: ac.signal }),
      Promise.allSettled(
        Array.from({ length: imageCount }, (_, index) =>
          getAnalysisImageURL(analysis.id || id, index, { signal: ac.signal })
        )
      ),
    ])
      .then(([servicesResult, imagesResult]) => {
        if (cancelled) return;
        if (servicesResult.status === "fulfilled") {
          setMatchedServices(servicesResult.value);
        }
        if (imagesResult.status === "fulfilled") {
          setImageURLs(
            imagesResult.value.map((item) =>
              item.status === "fulfilled" && item.value && typeof item.value === "object"
                ? item.value.url || ""
                : ""
            )
          );
        }
      })
      .catch(() => {
        if (!cancelled) setMatchedServices([]);
      })
      .finally(() => {
        if (!cancelled) setServicesLoading(false);
      });

    return () => {
      cancelled = true;
      ac.abort();
    };
  }, [analysis, id]);

  useEffect(() => {
    repairCreateMountedRef.current = true;
    return () => {
      repairCreateMountedRef.current = false;
    };
  }, [id]);

  if (loading) return <ResultLoading />;
  if (loadError) {
    return (
      <ResultErrorBanner
        message={loadError}
        onBack={() => navigate("/history")}
      />
    );
  }
  if (!analysis) return null;

  const damages = Array.isArray(analysis.damages) ? analysis.damages : [];
  const services = matchedServices.length > 0
    ? matchedServices
    : Array.isArray(analysis.services)
      ? analysis.services
      : [];

  const damageSummaryText =
    damages.length === 0
      ? "—"
      : damages.map((d) => d.part).filter(Boolean).join(", ");

  const selectedService =
    services.find((s) => s.id === selectedServiceId) ||
    services[0] ||
    null;

  const openRepairContactModal = () => {
    if (role === "SERVICE") return;
    if (services.length === 0) return;

    if (!selectedServiceId && services[0]?.id) {
      setSelectedServiceId(services[0].id);
    }
    setRepairError("");
    setRepairContactErrors({});
    setRepairContactOpen(true);
  };

  const updateRepairContactField = (field, value) => {
    setRepairContactForm((prev) => ({ ...prev, [field]: value }));
    setRepairContactErrors((prev) => {
      if (!prev[field] && !prev.contact) return prev;
      const next = { ...prev };
      delete next[field];
      delete next.contact;
      return next;
    });
  };

  const handleCreateRepair = async (event) => {
    event?.preventDefault?.();
    if (role === "SERVICE") return;
    if (repairCreateLockRef.current || repairSubmitting) return;

    let effectiveServiceId = selectedServiceId;
    if (!effectiveServiceId) {
      effectiveServiceId = services[0]?.id ?? "";
      if (effectiveServiceId) setSelectedServiceId(effectiveServiceId);
    }
    if (!effectiveServiceId || services.length === 0) return;

    const contactErrors = validateRepairContactForm(repairContactForm);
    setRepairContactErrors(contactErrors);
    if (Object.keys(contactErrors).length > 0) return;

    repairCreateLockRef.current = true;
    setRepairSubmitting(true);
    setRepairError("");

    const effectiveService =
      services.find((s) => s.id === effectiveServiceId) || null;

    try {
      await createRepairRequest(
        {
          analysis_id: id,
          service_id: effectiveServiceId,
          car_brand: analysis.brand,
          damage_summary: damageSummaryText,
          customer_name: repairContactForm.name,
          customer_phone: repairContactForm.phone,
          customer_email: repairContactForm.email,
          customer_comment: repairContactForm.comment,
          service: effectiveService
            ? {
                id: effectiveService.id,
                name: effectiveService.name,
                organization_name: effectiveService.name,
                city: effectiveService.city,
                phone: effectiveService.phone,
                email: effectiveService.email,
                address: effectiveService.address,
              }
            : undefined,
        },
        {}
      );

      if (!repairCreateMountedRef.current) return;

      setRepairContactOpen(false);
      setRepairContactForm(initialRepairContactForm);
      setRepairContactErrors({});
      setToast("Заявка на ремонт отправлена");
      setTimeout(() => {
        setToast("");
        navigate("/repair-requests");
      }, 1400);
    } catch (e) {
      if (normalizeApiError(e).code === "aborted") return;

      const msg =
        e instanceof Error
          ? e.message
          : "Не удалось отправить заявку. Попробуйте ещё раз.";
      setRepairError(msg);
    } finally {
      repairCreateLockRef.current = false;
      setRepairSubmitting(false);
    }
  };

  const openServiceGallery = (service, event) => {
    event?.stopPropagation?.();
    if (!service) return;
    const images = Array.isArray(service.images) ? service.images : [];
    const primaryIndex = images.findIndex((image) => image.is_primary);
    setGalleryIndex(primaryIndex >= 0 ? primaryIndex : 0);
    setGalleryService(service);
  };

  return (
    <div>
      <div className="page-header">
        <div>
          <button
            type="button"
            className="btn btn-ghost btn-sm"
            onClick={() => navigate("/history")}
            style={{ marginBottom: 6, marginLeft: -10 }}
          >
            <Icon name="arrowLeft" size={14} /> К истории
          </button>
          <h1 className="page-title">Результат анализа</h1>
        </div>
      </div>

      <div className="result-meta">
        <span className="row"><Icon name="car" size={14} /> {analysis.vehicle_label || analysis.brand}</span>
        <span className="sep">·</span>
        <span className="row"><Icon name="calendar" size={14} /> {formatDate(analysis.created_at)}</span>
        <span className="sep">·</span>
        <span className={severityBadgeClass(
          analysis.status === "done"
            ? "Без повреждений"
            : analysis.status === "failed"
              ? "Сложный"
              : ""
        )}>
          {analysisStatusLabel(analysis.status)}
        </span>
      </div>

      {/* Визуальный результат: в демо изображение не показываем */}
      <section className="result-section">
        <div className="result-images-list">
          {imageURLs.length > 0 ? (
            imageURLs.map((url, index) => (
              <div className="result-image-frame" key={`${index}:${url || "empty"}`}>
                {url ? (
                  <DamageOverlayImage
                    src={url}
                    imageResult={resultImages(analysis.result)[index]}
                    alt={`Загруженное изображение автомобиля ${index + 1}`}
                    className="result-damage-overlay"
                  />
                ) : (
                  <div className="result-image-placeholder">
                    <span className="ph-icon" aria-hidden="true">
                      <Icon name="fileImage" size={20} />
                    </span>
                    <span>Изображение результата недоступно</span>
                  </div>
                )}
              </div>
            ))
          ) : (
            <div className="result-image-frame">
              <div className="result-image-placeholder">
                <span className="ph-icon" aria-hidden="true">
                  <Icon name="fileImage" size={20} />
                </span>
                <span>Изображение результата недоступно</span>
              </div>
            </div>
          )}
        </div>
      </section>

      {/* Damages */}
      <section className="result-section">
        <header className="result-section-header">
          <h3>Обнаруженные повреждения</h3>
          <span className="text-sm muted">{damages.length}</span>
        </header>

        {damages.length === 0 ? (
          <div className="result-section-body">
            <p className="muted">Повреждений не найдено</p>
          </div>
        ) : (
          <div>
            {damages.map((d, i) => (
              <div className="damage-row" key={i}>
                <div className="name">{d.part}</div>
                <span className={severityBadgeClass(d.severity)}>{d.severity}</span>
              </div>
            ))}
          </div>
        )}
      </section>

      {/* Services */}
      <section className="result-section">
        <header className="result-section-header">
          <h3>Подходящие автосервисы</h3>
          <span className="text-sm muted">{services.length}</span>
        </header>

        <div className="result-section-body">
          {services.length === 0 ? (
            <p className="muted">
              {servicesLoading ? "Подбираем автосервисы..." : "Подходящие сервисы не найдены"}
            </p>
          ) : (
            <>
              <div className="services-list">
                {services.map((s) => (
                  <div
                    key={s.id}
                    className={"service-card" + (selectedServiceId === s.id ? " selected" : "")}
                    onClick={() => setSelectedServiceId(s.id)}
                    role="button"
                    tabIndex={0}
                  >
                    <div className="service-card-photo" aria-label="Фото автосервиса">
                      {Array.isArray(s.images) && s.images.length > 0 ? (
                        <img src={s.images[0].url} alt={s.name} />
                      ) : (
                        <span className="service-card-photo-placeholder">
                          <Icon name="building" size={20} />
                          <span>Фото нет</span>
                        </span>
                      )}
                    </div>
                    <div className="service-card-head">
                      <h4>{s.name}</h4>
                    </div>
                    {s.description && <div className="service-card-desc">{s.description}</div>}
                    <div className="service-card-meta">
                      {s.city ? (
                        <div className="row"><Icon name="mapPin" size={13} /> {s.city}</div>
                      ) : null}
                      <div className="row"><Icon name="mapPin" size={13} /> {s.address || "Адрес не указан"}</div>
                      {s.phone && s.phone !== "—" ? (
                        <div className="row"><Icon name="phone" size={13} /> {s.phone}</div>
                      ) : null}
                      {s.email ? (
                        <div className="row"><Icon name="mail" size={13} /> {s.email}</div>
                      ) : null}
                    </div>
                    <div className="service-card-actions">
                      <button
                        type="button"
                        className="btn btn-secondary btn-sm"
                        onClick={(event) => openServiceGallery(s, event)}
                      >
                        Подробнее
                      </button>
                    </div>
                  </div>
                ))}
              </div>

              <div className="result-actions">
                {role !== "SERVICE" ? (
                  <button
                    type="button"
                    className="btn btn-primary"
                    onClick={openRepairContactModal}
                    disabled={services.length === 0 || repairSubmitting}
                  >
                    Создать заявку на ремонт
                  </button>
                ) : null}
                {repairError ? (
                  <div className="alert alert-danger mt-2" role="alert">
                    {repairError}
                  </div>
                ) : null}
                {selectedServiceId && (
                  <span className="form-hint">
                    Выбран: {services.find((s) => s.id === selectedServiceId)?.name}
                  </span>
                )}
              </div>
            </>
          )}
        </div>
      </section>

      {toast && (
        <div className="toast" role="status">
          <Icon name="checkCircle" size={14} /> {toast}
        </div>
      )}

      {galleryService ? (
        <ServiceGalleryModal
          service={galleryService}
          index={galleryIndex}
          setIndex={setGalleryIndex}
          onClose={() => setGalleryService(null)}
        />
      ) : null}

      {repairContactOpen ? (
        <RepairContactModal
          service={selectedService}
          form={repairContactForm}
          errors={repairContactErrors}
          submitting={repairSubmitting}
          onChange={updateRepairContactField}
          onClose={() => {
            if (repairSubmitting) return;
            setRepairContactOpen(false);
          }}
          onSubmit={handleCreateRepair}
        />
      ) : null}
    </div>
  );
}

export default ResultPage;
