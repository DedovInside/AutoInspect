import './HomePage.css';
import { useEffect, useRef, useState } from "react";
import { Link } from 'react-router-dom';
import Icon from "../../components/Icon/Icon";
import { useAuth } from "../../auth/AuthContext";
import { normalizeApiError } from "../../services/apiFoundation";
import { listAvailableSpecializedMLModels } from "../../services/mlModels";
import { submitServiceRegistration } from "../../services/serviceRegistrations";
import { submitTrainingRequest } from "../../services/trainingRequests";

const initialServiceForm = {
  organization: "",
  city: "",
  address: "",
  phone: "",
  description: "",
};

const initialTrainingForm = {
  brand: "",
  model: "",
  generation: "",
  year_from: "",
  year_to: "",
  description: "",
};

const MIN_CAR_YEAR = 1900;
const MAX_CAR_YEAR = new Date().getFullYear() + 1;
const MIN_PHONE_DIGITS = 7;

function parseYear(value) {
  const trimmed = String(value ?? "").trim();
  if (!/^\d{4}$/.test(trimmed)) return null;
  const year = Number(trimmed);
  return Number.isInteger(year) ? year : null;
}

function yearRangeError(value, label) {
  const year = parseYear(value);
  if (!year) return `${label} должен быть указан в формате 4 цифр`;
  if (year < MIN_CAR_YEAR || year > MAX_CAR_YEAR) {
    return `${label} должен быть от ${MIN_CAR_YEAR} до ${MAX_CAR_YEAR}`;
  }
  return "";
}

function phoneDigits(value) {
  return String(value ?? "").replace(/\D/g, "");
}

function HomePage() {
  const { role } = useAuth();
  const [serviceFormSent, setServiceFormSent] = useState(false);
  const [trainingFormSent, setTrainingFormSent] = useState(false);

  const serviceSubmitGenRef = useRef(0);
  const trainingSubmitGenRef = useRef(0);
  const serviceAcRef = useRef(null);
  const trainingAcRef = useRef(null);

  const [serviceSubmitError, setServiceSubmitError] = useState("");
  const [trainingSubmitError, setTrainingSubmitError] = useState("");
  const [serviceSubmitting, setServiceSubmitting] = useState(false);
  const [trainingSubmitting, setTrainingSubmitting] = useState(false);

  const [serviceForm, setServiceForm] = useState(initialServiceForm);

  const [trainingForm, setTrainingForm] = useState(initialTrainingForm);
  const [availableModels, setAvailableModels] = useState([]);
  const [modelsLoading, setModelsLoading] = useState(false);
  const [modelsLoadError, setModelsLoadError] = useState("");

  const [serviceErrors, setServiceErrors] = useState({});
  const [trainingErrors, setTrainingErrors] = useState({});

  useEffect(() => {
    const modelsAc = new AbortController();
    setModelsLoading(true);
    setModelsLoadError("");

    listAvailableSpecializedMLModels({ signal: modelsAc.signal })
      .then((items) => setAvailableModels(Array.isArray(items) ? items : []))
      .catch((err) => {
        const n = normalizeApiError(err);
        if (n.code !== "aborted") {
          setAvailableModels([]);
          setModelsLoadError("Не удалось загрузить список доступных моделей");
        }
      })
      .finally(() => {
        if (!modelsAc.signal.aborted) {
          setModelsLoading(false);
        }
      });

    return () => {
      modelsAc.abort();
      serviceAcRef.current?.abort();
      trainingAcRef.current?.abort();
    };
  }, []);

  const validateServiceForm = () => {
    const e = {};
    if (!serviceForm.organization.trim()) {
      e.organization = "Введите название автосервиса";
    }
    if (!serviceForm.city.trim()) {
      e.city = "Введите город";
    }
    if (!serviceForm.address.trim()) {
      e.address = "Введите адрес";
    }
    if (!serviceForm.phone.trim()) {
      e.phone = "Введите номер телефона";
    } else if (phoneDigits(serviceForm.phone).length < MIN_PHONE_DIGITS) {
      e.phone = "Введите корректный номер телефона";
    }
    if (!serviceForm.description.trim()) {
      e.description = "Введите описание";
    }
    return e;
  };

  const validateTrainingForm = () => {
    const e = {};
    if (!trainingForm.brand.trim()) {
      e.brand = "Введите марку автомобиля";
    }
    if (!trainingForm.model.trim()) {
      e.model = "Введите модель автомобиля";
    }
    if (!trainingForm.generation.trim()) {
      e.generation = "Введите поколение";
    }
    if (!trainingForm.year_from.trim()) {
      e.year_from = "Введите год начала выпуска";
    } else {
      const err = yearRangeError(trainingForm.year_from, "Год начала выпуска");
      if (err) e.year_from = err;
    }
    if (!trainingForm.year_to.trim()) {
      e.year_to = "Введите год окончания выпуска";
    } else {
      const err = yearRangeError(trainingForm.year_to, "Год окончания выпуска");
      if (err) e.year_to = err;
    }
    const yearFrom = parseYear(trainingForm.year_from);
    const yearTo = parseYear(trainingForm.year_to);
    if (
      yearFrom &&
      yearTo &&
      yearTo < yearFrom
    ) {
      e.year_to = "Год окончания не может быть раньше года начала";
    }
    if (!trainingForm.description.trim()) {
      e.description = "Введите описание потребности";
    }
    return e;
  };

  const clearServiceError = (field) => {
    setServiceErrors((prev) => {
      if (!prev[field]) return prev;
      const next = { ...prev };
      delete next[field];
      return next;
    });
  };

  const clearTrainingError = (field) => {
    setTrainingErrors((prev) => {
      if (!prev[field]) return prev;
      const next = { ...prev };
      delete next[field];
      return next;
    });
  };

  const onServiceSubmit = async (e) => {
    e.preventDefault();
    const errors = validateServiceForm();
    setServiceErrors(errors);
    if (Object.keys(errors).length > 0) return;

    serviceSubmitGenRef.current += 1;
    const seq = serviceSubmitGenRef.current;

    serviceAcRef.current?.abort();
    const ac = new AbortController();
    serviceAcRef.current = ac;

    setServiceSubmitting(true);
    setServiceSubmitError("");

    try {
      await submitServiceRegistration(serviceForm, { signal: ac.signal });
      if (seq !== serviceSubmitGenRef.current) return;
      setServiceFormSent(true);
      setServiceForm(initialServiceForm);
      setServiceErrors({});
    } catch (err) {
      if (seq !== serviceSubmitGenRef.current) return;
      const n = normalizeApiError(err);
      if (n.code === "aborted") return;
      setServiceSubmitError(
        err?.message ?? "Не удалось отправить заявку. Попробуйте позже."
      );
    } finally {
      setServiceSubmitting(false);
    }
  };

  const onTrainingSubmit = async (e) => {
    e.preventDefault();
    const errors = validateTrainingForm();
    setTrainingErrors(errors);
    if (Object.keys(errors).length > 0) return;

    trainingSubmitGenRef.current += 1;
    const seq = trainingSubmitGenRef.current;

    trainingAcRef.current?.abort();
    const ac = new AbortController();
    trainingAcRef.current = ac;

    setTrainingSubmitting(true);
    setTrainingSubmitError("");

    try {
      await submitTrainingRequest(trainingForm, { signal: ac.signal });
      if (seq !== trainingSubmitGenRef.current) return;
      setTrainingFormSent(true);
      setTrainingForm(initialTrainingForm);
      setTrainingErrors({});
    } catch (err) {
      if (seq !== trainingSubmitGenRef.current) return;
      const n = normalizeApiError(err);
      if (n.code === "aborted") return;
      setTrainingSubmitError(
        err?.message ?? "Не удалось отправить заявку. Попробуйте позже."
      );
    } finally {
      setTrainingSubmitting(false);
    }
  };

  return (
    <div>
      <section className="home-intro">
        <h1>AutoInspect</h1>
        <p className="home-intro-description">
          Сервис автоматического анализа повреждений кузова автомобиля
        </p>
        <div className="home-intro-info">
          <h3>Как пользоваться сервисом</h3>
          <p>
          Загрузите фотографию/и и укажите данные автомобиля - система
          автоматически проанализирует изображение и определит возможные
          повреждения кузова. Результаты обработки отображаются сразу после анализа.
          </p>
          <h4>Рекомендации для точного анализа</h4>
          <ul>
            <li>Делайте фотографии при хорошем освещении.</li>
            <li>Избегайте размытых и нечётких снимков.</li>
            <li>Фотографируйте повреждение с нескольких ракурсов.</li>
            <li>Старайтесь, чтобы повреждённая область полностью находилась в кадре.</li>
          </ul>
        </div>
        <Link to="/upload" className="btn btn-primary">
          <Icon name="upload" size={22} /> Начать анализ
        </Link>
      </section>

      <section>
        <div className="home-forms-header">
          <h2>Подача заявок</h2>
          <p>Регистрация автосервиса или запрос на обучение новой ML-модели.</p>
        </div>

        <div className={role === "USER" ? "grid grid-2" : "grid"}>
          {/* Service registration (только для пользователя) */}
          {role === "USER" && (
          <div className="card card-padded">
            <h3>Заявка на регистрацию автосервиса</h3>
            <p className="desc">
              После одобрения администратором аккаунту будет присвоена роль «Автосервис».
            </p>

            {serviceSubmitError ? (
              <div className="alert alert-danger mb-3" role="alert">
                {serviceSubmitError}
              </div>
            ) : null}

            {serviceFormSent && (
              <div className="home-form-success">
                <Icon name="checkCircle" size={16} /> Заявка на регистрацию отправлена
              </div>
            )}

            <form onSubmit={onServiceSubmit} className="home-form-card home-form-card--service" noValidate>
              <div className="form-row">
                <label className="form-label" htmlFor="svc-name">Название автосервиса *</label>
                <input
                  id="svc-name"
                  className={"input" + (serviceErrors.organization ? " input-error" : "")}
                  placeholder="Например, AutoFix"
                  value={serviceForm.organization}
                  onChange={(e) => {
                    setServiceForm({ ...serviceForm, organization: e.target.value });
                    clearServiceError("organization");
                  }}
                />
                {serviceErrors.organization && (
                  <div className="field-error">{serviceErrors.organization}</div>
                )}
              </div>
              <div className="form-row">
                <label className="form-label" htmlFor="svc-city">Город *</label>
                <input
                  id="svc-city"
                  className={"input" + (serviceErrors.city ? " input-error" : "")}
                  placeholder="Москва"
                  value={serviceForm.city}
                  onChange={(e) => {
                    setServiceForm({ ...serviceForm, city: e.target.value });
                    clearServiceError("city");
                  }}
                />
                {serviceErrors.city && (
                  <div className="field-error">{serviceErrors.city}</div>
                )}
              </div>
              <div className="form-row">
                <label className="form-label" htmlFor="svc-address">Адрес *</label>
                <input
                  id="svc-address"
                  className={"input" + (serviceErrors.address ? " input-error" : "")}
                  placeholder="ул. Ленина, 10"
                  value={serviceForm.address}
                  onChange={(e) => {
                    setServiceForm({ ...serviceForm, address: e.target.value });
                    clearServiceError("address");
                  }}
                />
                {serviceErrors.address && (
                  <div className="field-error">{serviceErrors.address}</div>
                )}
              </div>
              <div className="form-row">
                <label className="form-label" htmlFor="svc-phone">Телефон *</label>
                <input
                  id="svc-phone"
                  type="tel"
                  className={"input" + (serviceErrors.phone ? " input-error" : "")}
                  placeholder="+7 (___) ___-__-__"
                  value={serviceForm.phone}
                  onChange={(e) => {
                    setServiceForm({ ...serviceForm, phone: e.target.value });
                    clearServiceError("phone");
                  }}
                />
                {serviceErrors.phone && (
                  <div className="field-error">{serviceErrors.phone}</div>
                )}
              </div>
              <div className="form-row">
                <label className="form-label" htmlFor="svc-desc">Описание *</label>
                <textarea
                  id="svc-desc"
                  className={"textarea" + (serviceErrors.description ? " input-error" : "")}
                  placeholder="Кратко расскажите о сервисе"
                  value={serviceForm.description}
                  onChange={(e) => {
                    setServiceForm({ ...serviceForm, description: e.target.value });
                    clearServiceError("description");
                  }}
                />
                {serviceErrors.description && (
                  <div className="field-error">{serviceErrors.description}</div>
                )}
              </div>
              <button
                type="submit"
                className="btn btn-primary"
                disabled={serviceSubmitting}
              >
                Отправить заявку
              </button>
            </form>
          </div>
          )}

          {/* Training request */}
          <div className="card card-padded">
            <h3>Заявка на обучение ML-модели</h3>
            <p className="desc">
              Если в системе нет вашей марки или модели - оставьте заявку, администратор её рассмотрит.
            </p>

            {trainingSubmitError ? (
              <div className="alert alert-danger mb-3" role="alert">
                {trainingSubmitError}
              </div>
            ) : null}

            {trainingFormSent && (
              <div className="home-form-success">
                <Icon name="checkCircle" size={16} /> Заявка на обучение модели отправлена
              </div>
            )}

            <form onSubmit={onTrainingSubmit} className="home-form-card home-form-card--training" noValidate>
              <div className="form-row">
                <label className="form-label" htmlFor="tr-brand">Марка автомобиля *</label>
                <input
                  id="tr-brand"
                  className={"input" + (trainingErrors.brand ? " input-error" : "")}
                  placeholder="Например, Kia"
                  value={trainingForm.brand}
                  onChange={(e) => {
                    setTrainingForm({ ...trainingForm, brand: e.target.value });
                    clearTrainingError("brand");
                  }}
                />
                {trainingErrors.brand && (
                  <div className="field-error">{trainingErrors.brand}</div>
                )}
              </div>
              <div className="form-row">
                <label className="form-label" htmlFor="tr-model">Модель автомобиля *</label>
                <input
                  id="tr-model"
                  className={"input" + (trainingErrors.model ? " input-error" : "")}
                  placeholder="Например, Coolray"
                  value={trainingForm.model}
                  onChange={(e) => {
                    setTrainingForm({ ...trainingForm, model: e.target.value });
                    clearTrainingError("model");
                  }}
                />
                {trainingErrors.model && (
                  <div className="field-error">{trainingErrors.model}</div>
                )}
              </div>
              <div className="form-row-inline">
                <div className="form-row">
                  <label className="form-label" htmlFor="tr-gen">Поколение *</label>
                  <input
                    id="tr-gen"
                    className={"input" + (trainingErrors.generation ? " input-error" : "")}
                    placeholder="I, II, FL"
                    value={trainingForm.generation}
                    onChange={(e) => {
                      setTrainingForm({ ...trainingForm, generation: e.target.value });
                      clearTrainingError("generation");
                    }}
                  />
                  {trainingErrors.generation && (
                    <div className="field-error">{trainingErrors.generation}</div>
                  )}
                </div>
                <div className="form-row">
                  <label className="form-label" htmlFor="tr-year-from">Год начала выпуска *</label>
                  <input
                    id="tr-year-from"
                    type="text"
                    inputMode="numeric"
                    className={"input" + (trainingErrors.year_from ? " input-error" : "")}
                    placeholder="2018"
                    value={trainingForm.year_from}
                    onChange={(e) => {
                      setTrainingForm({ ...trainingForm, year_from: e.target.value });
                      clearTrainingError("year_from");
                    }}
                  />
                  {trainingErrors.year_from && (
                    <div className="field-error">{trainingErrors.year_from}</div>
                  )}
                </div>
                <div className="form-row">
                  <label className="form-label" htmlFor="tr-year-to">Год окончания выпуска *</label>
                  <input
                    id="tr-year-to"
                    type="text"
                    inputMode="numeric"
                    className={"input" + (trainingErrors.year_to ? " input-error" : "")}
                    placeholder="2024"
                    value={trainingForm.year_to}
                    onChange={(e) => {
                      setTrainingForm({ ...trainingForm, year_to: e.target.value });
                      clearTrainingError("year_to");
                    }}
                  />
                  {trainingErrors.year_to && (
                    <div className="field-error">{trainingErrors.year_to}</div>
                  )}
                </div>
              </div>
              <div className="form-row">
                <label className="form-label" htmlFor="tr-desc">Описание потребности *</label>
                <textarea
                  id="tr-desc"
                  className={"textarea" + (trainingErrors.description ? " input-error" : "")}
                  placeholder="Поясните, почему нужна модель для этой марки"
                  value={trainingForm.description}
                  onChange={(e) => {
                    setTrainingForm({ ...trainingForm, description: e.target.value });
                    clearTrainingError("description");
                  }}
                />
                {trainingErrors.description && (
                  <div className="field-error">{trainingErrors.description}</div>
                )}
              </div>
              <button
                type="submit"
                className="btn btn-primary"
                disabled={trainingSubmitting}
              >
                Отправить заявку
              </button>
            </form>

            <AvailableModelsPanel
              models={availableModels}
              loading={modelsLoading}
              error={modelsLoadError}
            />
          </div>
        </div>
      </section>
    </div>
  );
}

function AvailableModelsPanel({ models, loading, error }) {
  return (
    <div className="available-models-panel">
      <div className="available-models-head">
        <div>
          <h4>Уже доступные специализированные модели</h4>
          <p>
            Если нужного автомобиля нет в списке, при анализе будет использована общая модель.
          </p>
        </div>
      </div>

      {loading ? (
        <div className="available-models-note">Загружаем список моделей...</div>
      ) : error ? (
        <div className="available-models-note available-models-note--danger">{error}</div>
      ) : models.length === 0 ? (
        <div className="available-models-empty">
          Специализированных моделей пока нет. Для всех автомобилей используется общая модель.
        </div>
      ) : (
        <div className="available-models-list">
          {models.map((model) => (
            <div className="available-model-row" key={model.id}>
              <div className="available-model-title">
                {[model.brand, model.model, model.generation].filter(Boolean).join(" ")}
              </div>
              <div className="available-model-meta">
                {formatModelYears(model.years)}
                {model.version ? <span>Версия {model.version}</span> : null}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function formatModelYears(years) {
  const value = String(years ?? "").trim();
  if (!value) return <span>Годы выпуска не указаны</span>;
  return <span>{value}</span>;
}

export default HomePage;
