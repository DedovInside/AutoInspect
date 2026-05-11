import './HomePage.css';
import { useEffect, useRef, useState } from "react";
import { Link } from 'react-router-dom';
import Icon from "../../components/Icon/Icon";
import { useAuth } from "../../auth/AuthContext";
import { normalizeApiError } from "../../services/apiFoundation";
import { submitServiceRegistration } from "../../services/serviceRegistrations";
import { submitTrainingRequest } from "../../services/trainingRequests";

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

  const [serviceForm, setServiceForm] = useState({
    organization: "",
    address: "",
    phone: "",
    description: "",
  });

  const [trainingForm, setTrainingForm] = useState({
    brand: "",
    model: "",
    generation: "",
    year: "",
    description: "",
  });

  const [serviceErrors, setServiceErrors] = useState({});
  const [trainingErrors, setTrainingErrors] = useState({});

  useEffect(() => {
    return () => {
      serviceAcRef.current?.abort();
      trainingAcRef.current?.abort();
    };
  }, []);

  const validateServiceForm = () => {
    const e = {};
    if (!serviceForm.organization.trim()) {
      e.organization = "Введите название автосервиса";
    }
    if (!serviceForm.address.trim()) {
      e.address = "Введите адрес";
    }
    if (!serviceForm.phone.trim()) {
      e.phone = "Введите номер телефона";
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
    if (!trainingForm.year.trim()) {
      e.year = "Введите год";
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
        <p>
          Сервис автоматического анализа повреждений кузова автомобиля
        </p>
        <Link to="/upload" className="btn btn-primary">
          <Icon name="upload" size={16} /> Начать анализ
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
                <label className="form-label" htmlFor="svc-address">Адрес *</label>
                <input
                  id="svc-address"
                  className={"input" + (serviceErrors.address ? " input-error" : "")}
                  placeholder="г. Москва, ул. Ленина, 10"
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
                <label className="form-label" htmlFor="svc-desc">Описание</label>
                <textarea
                  id="svc-desc"
                  className="textarea"
                  placeholder="Кратко расскажите о сервисе"
                  value={serviceForm.description}
                  onChange={(e) => setServiceForm({ ...serviceForm, description: e.target.value })}
                />
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
                  <label className="form-label" htmlFor="tr-year">Год *</label>
                  <input
                    id="tr-year"
                    className={"input" + (trainingErrors.year ? " input-error" : "")}
                    placeholder="2018"
                    value={trainingForm.year}
                    onChange={(e) => {
                      setTrainingForm({ ...trainingForm, year: e.target.value });
                      clearTrainingError("year");
                    }}
                  />
                  {trainingErrors.year && (
                    <div className="field-error">{trainingErrors.year}</div>
                  )}
                </div>
              </div>
              <div className="form-row">
                <label className="form-label" htmlFor="tr-desc">Описание потребности</label>
                <textarea
                  id="tr-desc"
                  className="textarea"
                  placeholder="Поясните, почему нужна модель для этой марки"
                  value={trainingForm.description}
                  onChange={(e) => setTrainingForm({ ...trainingForm, description: e.target.value })}
                />
              </div>
              <button
                type="submit"
                className="btn btn-primary"
                disabled={trainingSubmitting}
              >
                Отправить заявку
              </button>
            </form>
          </div>
        </div>
      </section>
    </div>
  );
}

export default HomePage;
