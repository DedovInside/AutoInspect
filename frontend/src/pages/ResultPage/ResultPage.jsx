import "./ResultPage.css";
import { useEffect, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import {
  getAnalysis,
  isFailedTerminalAnalysisStatus,
  isSuccessTerminalAnalysisStatus,
} from "../../services/analyses";
import { normalizeApiError } from "../../services/apiFoundation";
import { createRepairRequest } from "../../services/repairRequests";
import Icon from "../../components/Icon/Icon";

function severityBadgeClass(s) {
  if (s === "Сложный") return "badge badge-severity-heavy";
  if (s === "Средний") return "badge badge-severity-medium";
  if (s === "Лёгкий") return "badge badge-severity-light";
  if (s === "Без повреждений") return "badge badge-severity-clear";
  return "badge badge-muted";
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

function ResultPage() {
  const { id } = useParams();
  const navigate = useNavigate();

  const [loading, setLoading] = useState(true);
  const [analysis, setAnalysis] = useState(null);
  const [loadError, setLoadError] = useState("");
  const [selectedServiceId, setSelectedServiceId] = useState(null);
  const [toast, setToast] = useState("");
  const [repairSubmitting, setRepairSubmitting] = useState(false);
  const [repairError, setRepairError] = useState("");
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
  const services = Array.isArray(analysis.services) ? analysis.services : [];

  const damageSummaryText =
    damages.length === 0
      ? "—"
      : damages.map((d) => d.part).filter(Boolean).join(", ");

  const handleCreateRepair = async () => {
    if (repairCreateLockRef.current || repairSubmitting) return;

    let effectiveServiceId = selectedServiceId;
    if (!effectiveServiceId) {
      effectiveServiceId = services[0]?.id ?? "";
      if (effectiveServiceId) setSelectedServiceId(effectiveServiceId);
    }
    if (!effectiveServiceId || services.length === 0) return;

    repairCreateLockRef.current = true;
    setRepairSubmitting(true);
    setRepairError("");

    const selectedService =
      services.find((s) => s.id === effectiveServiceId) || null;

    try {
      await createRepairRequest(
        {
          analysis_id: id,
          service_id: effectiveServiceId,
          car_brand: analysis.brand,
          damage_summary: damageSummaryText,
          service: selectedService
            ? {
                name: selectedService.name,
                phone: selectedService.phone,
                address: selectedService.address,
              }
            : undefined,
        },
        {}
      );

      if (!repairCreateMountedRef.current) return;

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
        <span className="row"><Icon name="car" size={14} /> {analysis.brand}</span>
        <span className="sep">·</span>
        <span className="row"><Icon name="calendar" size={14} /> {formatDate(analysis.created_at)}</span>
        <span className="sep">·</span>
        <span className={severityBadgeClass(analysis.overall_severity)}>
          {analysis.overall_severity}
        </span>
      </div>

      {/* Визуальный результат: в демо изображение не показываем */}
      <section className="result-section">
        <div className="result-image-frame">
          <div className="result-image-placeholder">
            <span className="ph-icon" aria-hidden="true">
              <Icon name="fileImage" size={20} />
            </span>
            <span>Изображение результата в демо не отображается</span>
          </div>
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
            <p className="muted">Подходящие сервисы не найдены</p>
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
                    <div className="service-card-head">
                      <h4>{s.name}</h4>
                    </div>
                    {s.description && <div className="service-card-desc">{s.description}</div>}
                    <div className="service-card-meta">
                      <div className="row"><Icon name="phone" size={13} /> {s.phone}</div>
                      <div className="row"><Icon name="mapPin" size={13} /> {s.address}</div>
                    </div>
                  </div>
                ))}
              </div>

              <div className="result-actions">
                <button
                  type="button"
                  className="btn btn-primary"
                  onClick={handleCreateRepair}
                  disabled={services.length === 0 || repairSubmitting}
                >
                  Создать заявку на ремонт
                </button>
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
    </div>
  );
}

export default ResultPage;
