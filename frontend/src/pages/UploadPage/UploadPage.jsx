import "./UploadPage.css";
import { useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { uploadImages } from "../../services/analyses";
import {
  getVehicleGenerations,
  getVehicleMakes,
  getVehicleModels,
} from "../../services/vehicleCatalog";
import Icon from "../../components/Icon/Icon";

const MAX_IMAGE_COUNT = 10;
const MAX_IMAGE_SIZE = 100 * 1024 * 1024;
const ALLOWED_IMAGE_TYPES = new Set(["image/jpeg", "image/png", "image/webp"]);
const ALLOWED_IMAGE_LABEL = "JPG, PNG или WEBP";

function formatBytes(bytes) {
  if (!bytes && bytes !== 0) return "";
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

const initialVehicle = {
  makeID: "",
  modelID: "",
  generationID: "",
  year: "",
};

function validateImageFile(file) {
  if (!ALLOWED_IMAGE_TYPES.has(file.type)) {
    return `Файл «${file.name}» должен быть в формате ${ALLOWED_IMAGE_LABEL}`;
  }
  if (file.size > MAX_IMAGE_SIZE) {
    return `Файл «${file.name}» больше ${formatBytes(MAX_IMAGE_SIZE)}`;
  }
  return "";
}

function UploadPage() {
  const navigate = useNavigate();
  const inputRef = useRef(null);

  const [items, setItems] = useState([]);
  const [vehicle, setVehicle] = useState(initialVehicle);
  const [makes, setMakes] = useState([]);
  const [models, setModels] = useState([]);
  const [generations, setGenerations] = useState([]);
  const [catalogLoading, setCatalogLoading] = useState({
    makes: true,
    models: false,
    generations: false,
  });
  const [catalogError, setCatalogError] = useState("");
  const [fieldErrors, setFieldErrors] = useState({});
  const [dragging, setDragging] = useState(false);
  const [loading, setLoading] = useState(false);
  const [submitError, setSubmitError] = useState("");

  const selectedMake = useMemo(
    () => makes.find((make) => String(make.id) === String(vehicle.makeID)) || null,
    [makes, vehicle.makeID]
  );
  const selectedModel = useMemo(
    () => models.find((model) => String(model.id) === String(vehicle.modelID)) || null,
    [models, vehicle.modelID]
  );
  const selectedGeneration = useMemo(
    () =>
      generations.find((generation) => String(generation.id) === String(vehicle.generationID)) ||
      null,
    [generations, vehicle.generationID]
  );
  const yearOptions = selectedGeneration?.year_options ?? [];

  useEffect(() => {
    const controller = new AbortController();

    async function loadMakes() {
      try {
        setCatalogLoading((prev) => ({ ...prev, makes: true }));
        setCatalogError("");
        const items = await getVehicleMakes({ signal: controller.signal });
        setMakes(items);
      } catch (err) {
        if (err?.code !== "aborted") {
          console.error(err);
          setCatalogError("Не удалось загрузить справочник автомобилей");
        }
      } finally {
        setCatalogLoading((prev) => ({ ...prev, makes: false }));
      }
    }

    loadMakes();
    return () => controller.abort();
  }, []);

  useEffect(() => {
    if (!vehicle.makeID) {
      setModels([]);
      return;
    }

    const controller = new AbortController();

    async function loadModels() {
      try {
        setCatalogLoading((prev) => ({ ...prev, models: true }));
        setCatalogError("");
        const items = await getVehicleModels(vehicle.makeID, { signal: controller.signal });
        setModels(items);
      } catch (err) {
        if (err?.code !== "aborted") {
          console.error(err);
          setCatalogError("Не удалось загрузить модели выбранной марки");
        }
      } finally {
        setCatalogLoading((prev) => ({ ...prev, models: false }));
      }
    }

    loadModels();
    return () => controller.abort();
  }, [vehicle.makeID]);

  useEffect(() => {
    if (!vehicle.modelID) {
      setGenerations([]);
      return;
    }

    const controller = new AbortController();

    async function loadGenerations() {
      try {
        setCatalogLoading((prev) => ({ ...prev, generations: true }));
        setCatalogError("");
        const items = await getVehicleGenerations(vehicle.modelID, {
          signal: controller.signal,
        });
        setGenerations(items);
      } catch (err) {
        if (err?.code !== "aborted") {
          console.error(err);
          setCatalogError("Не удалось загрузить поколения выбранной модели");
        }
      } finally {
        setCatalogLoading((prev) => ({ ...prev, generations: false }));
      }
    }

    loadGenerations();
    return () => controller.abort();
  }, [vehicle.modelID]);

  const onFiles = (fileList) => {
    const incoming = Array.from(fileList || []);
    const nextErrors = [];
    const freeSlots = Math.max(0, MAX_IMAGE_COUNT - items.length);
    const accepted = incoming.slice(0, freeSlots);

    if (incoming.length > freeSlots) {
      nextErrors.push(`Можно загрузить не более ${MAX_IMAGE_COUNT} изображений`);
    }

    const validFiles = [];
    for (const file of accepted) {
      const error = validateImageFile(file);
      if (error) {
        nextErrors.push(error);
      } else {
        validFiles.push(file);
      }
    }

    if (nextErrors.length > 0) {
      setFieldErrors((prev) => ({ ...prev, files: nextErrors[0] }));
    }

    const fresh = validFiles.map((f) => ({
      id: `${f.name}-${f.size}-${Math.random().toString(36).slice(2, 7)}`,
      file: f,
      url: URL.createObjectURL(f),
      name: f.name,
      size: f.size,
    }));
    setItems((prev) => [...prev, ...fresh]);
    if (fresh.length && nextErrors.length === 0) {
      clearFieldError("files");
    }
  };

  const removeItem = (id) => {
    setItems((prev) => {
      const next = prev.filter((it) => it.id !== id);
      const removed = prev.find((it) => it.id === id);
      if (removed) URL.revokeObjectURL(removed.url);
      return next;
    });
  };

  const onDrop = (e) => {
    e.preventDefault();
    setDragging(false);
    if (e.dataTransfer?.files?.length) {
      onFiles(e.dataTransfer.files);
    }
  };

  const validate = () => {
    const e = {};
    if (!selectedMake) {
      e.makeID = "Выберите марку автомобиля";
    }
    if (!selectedModel) {
      e.modelID = "Выберите модель автомобиля";
    }
    if (!selectedGeneration) {
      e.generationID = "Выберите поколение";
    }
    if (!vehicle.year) {
      e.year = "Выберите год выпуска";
    } else if (!yearOptions.map(String).includes(String(vehicle.year))) {
      e.year = "Выберите год из доступного диапазона";
    }
    if (items.length === 0) {
      e.files = "Выберите изображение/я";
    } else if (items.length > MAX_IMAGE_COUNT) {
      e.files = `Можно загрузить не более ${MAX_IMAGE_COUNT} изображений`;
    } else {
      const invalid = items.map((it) => validateImageFile(it.file)).find(Boolean);
      if (invalid) e.files = invalid;
    }
    return e;
  };

  const clearFieldError = (field) => {
    setFieldErrors((prev) => {
      if (!prev[field]) return prev;
      const next = { ...prev };
      delete next[field];
      return next;
    });
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setSubmitError("");

    const errors = validate();
    setFieldErrors(errors);
    if (Object.keys(errors).length > 0) {
      return;
    }

    try {
      setLoading(true);
      const result = await uploadImages({
        brand: selectedMake.name,
        model: selectedModel.name,
        generation: selectedGeneration.name,
        year: String(vehicle.year),
        files: items.map((it) => it.file),
      });
      navigate(`/result/${result.analysis_id}`);
    } catch (err) {
      console.error(err);
      setSubmitError("Не удалось отправить изображения");
    } finally {
      setLoading(false);
    }
  };

  const clearAll = () => {
    setItems((prev) => {
      prev.forEach((it) => URL.revokeObjectURL(it.url));
      return [];
    });
    setVehicle(initialVehicle);
    setFieldErrors({});
    setSubmitError("");
  };

  const updateVehicle = (patch) => {
    setVehicle((prev) => ({ ...prev, ...patch }));
  };

  const clearDependentErrors = (...fields) => {
    fields.forEach(clearFieldError);
  };

  return (
    <div>
      <div className="page-header">
        <div>
          <h1 className="page-title">Загрузка изображений</h1>
          <p className="page-subtitle">
            Укажите данные об автомобиле и загрузите фотографии для анализа
          </p>
        </div>
      </div>

      <form className="upload-section" onSubmit={handleSubmit} noValidate>
        <div className="upload-ml-info" role="note">
          <span className="upload-ml-info__icon" aria-hidden>
            <Icon name="shield" size={16} />
          </span>
          <p>
            Для некоторых марок используются отдельные ML-модели. Если для вашей марки отдельная
            модель отсутствует - будет использована общая модель анализа.
          </p>
        </div>

        {catalogError && (
          <div className="alert alert-warning upload-catalog-alert" role="alert">
            <span className="alert-icon">
              <Icon name="alert" size={16} />
            </span>
            <div>{catalogError}</div>
          </div>
        )}

        <div className="form-row">
          <label className="form-label" htmlFor="up-make">Марка автомобиля *</label>
          <select
            id="up-make"
            className={"select" + (fieldErrors.makeID ? " select-error" : "")}
            value={vehicle.makeID}
            disabled={catalogLoading.makes}
            onChange={(e) => {
              updateVehicle({
                makeID: e.target.value,
                modelID: "",
                generationID: "",
                year: "",
              });
              setModels([]);
              setGenerations([]);
              clearDependentErrors("makeID", "modelID", "generationID", "year");
            }}
          >
            <option value="">
              {catalogLoading.makes ? "Загрузка марок..." : "Выберите марку"}
            </option>
            {makes.map((make) => (
              <option key={make.id} value={make.id}>
                {make.name}
              </option>
            ))}
          </select>
          {fieldErrors.makeID && <div className="field-error">{fieldErrors.makeID}</div>}
        </div>

        <div className="form-row">
          <label className="form-label" htmlFor="up-model">Модель автомобиля *</label>
          <select
            id="up-model"
            className={"select" + (fieldErrors.modelID ? " select-error" : "")}
            value={vehicle.modelID}
            disabled={!vehicle.makeID || catalogLoading.models}
            onChange={(e) => {
              updateVehicle({
                modelID: e.target.value,
                generationID: "",
                year: "",
              });
              setGenerations([]);
              clearDependentErrors("modelID", "generationID", "year");
            }}
          >
            <option value="">
              {!vehicle.makeID
                ? "Сначала выберите марку"
                : catalogLoading.models
                  ? "Загрузка моделей..."
                  : "Выберите модель"}
            </option>
            {models.map((model) => (
              <option key={model.id} value={model.id}>
                {model.name}
              </option>
            ))}
          </select>
          {fieldErrors.modelID && <div className="field-error">{fieldErrors.modelID}</div>}
        </div>

        <div className="form-row-inline">
          <div className="form-row">
            <label className="form-label" htmlFor="up-gen">Поколение *</label>
            <select
              id="up-gen"
              className={"select" + (fieldErrors.generationID ? " select-error" : "")}
              value={vehicle.generationID}
              disabled={!vehicle.modelID || catalogLoading.generations}
              onChange={(e) => {
                updateVehicle({
                  generationID: e.target.value,
                  year: "",
                });
                clearDependentErrors("generationID", "year");
              }}
            >
              <option value="">
                {!vehicle.modelID
                  ? "Сначала выберите модель"
                  : catalogLoading.generations
                    ? "Загрузка поколений..."
                    : "Выберите поколение"}
              </option>
              {generations.map((generation) => (
                <option key={generation.id} value={generation.id}>
                  {generation.name}
                  {generation.year_from ? ` (${generation.year_from}-${generation.year_to || "н.в."})` : ""}
                </option>
              ))}
            </select>
            {fieldErrors.generationID && (
              <div className="field-error">{fieldErrors.generationID}</div>
            )}
          </div>
          <div className="form-row">
            <label className="form-label" htmlFor="up-year">Год *</label>
            <select
              id="up-year"
              className={"select" + (fieldErrors.year ? " select-error" : "")}
              value={vehicle.year}
              disabled={!selectedGeneration}
              onChange={(e) => {
                updateVehicle({ year: e.target.value });
                clearFieldError("year");
              }}
            >
              <option value="">
                {selectedGeneration ? "Выберите год" : "Сначала выберите поколение"}
              </option>
              {yearOptions.map((year) => (
                <option key={year} value={year}>
                  {year}
                </option>
              ))}
            </select>
            {fieldErrors.year && <div className="field-error">{fieldErrors.year}</div>}
          </div>
        </div>

        {selectedMake && selectedModel && selectedGeneration && vehicle.year && (
          <div className="upload-vehicle-summary">
            <span className="upload-vehicle-summary__icon" aria-hidden>
              <Icon name="car" size={15} />
            </span>
            <span>
              Выбран автомобиль:{" "}
              <strong>
                {[selectedMake.name, selectedModel.name, selectedGeneration.name, vehicle.year]
                  .filter(Boolean)
                  .join(" ")}
              </strong>
            </span>
          </div>
        )}

        <div className="form-row">
          <label className="form-label">Изображения автомобиля *</label>
          <label
            className={
              "dropzone" +
              (dragging ? " dragging" : "") +
              (fieldErrors.files ? " dropzone-error" : "")
            }
            onDragOver={(e) => {
              e.preventDefault();
              setDragging(true);
            }}
            onDragLeave={() => setDragging(false)}
            onDrop={onDrop}
          >
            <span className="ic">
              <Icon name="upload" size={22} />
            </span>
            <div>
              <strong>Перетащите фотографии сюда</strong> или нажмите, чтобы выбрать файлы
            </div>
            <div className="hint">Поддерживаются JPG, PNG, WEBP</div>
            <input
              ref={inputRef}
              type="file"
              accept="image/*"
              multiple
              onChange={(e) => onFiles(e.target.files)}
            />
          </label>
          {fieldErrors.files && <div className="field-error">{fieldErrors.files}</div>}
        </div>

        {items.length > 0 && (
          <div className="preview-grid">
            {items.map((it) => (
              <div key={it.id} className="preview-tile">
                <img src={it.url} alt={it.name} />
                <div className="meta">
                  <span
                    style={{
                      overflow: "hidden",
                      textOverflow: "ellipsis",
                      whiteSpace: "nowrap",
                    }}
                  >
                    {it.name}
                  </span>
                  <span>{formatBytes(it.size)}</span>
                </div>
                <button
                  type="button"
                  className="remove-btn"
                  onClick={() => removeItem(it.id)}
                  aria-label="Удалить"
                >
                  <Icon name="x" size={12} />
                </button>
              </div>
            ))}
          </div>
        )}

        {submitError && (
          <div className="alert alert-danger mt-4" role="alert">
            <span className="alert-icon">
              <Icon name="alert" size={16} />
            </span>
            <div>{submitError}</div>
          </div>
        )}

        <div className="upload-actions">
          <button type="submit" className="btn btn-primary" disabled={loading}>
            {loading ? "Отправка..." : "Отправить"}
          </button>
          {(items.length > 0 || Object.values(vehicle).some(Boolean)) && !loading && (
            <button type="button" className="btn btn-secondary" onClick={clearAll}>
              Очистить
            </button>
          )}
        </div>

        {loading && <p className="form-hint mt-3">Идёт загрузка</p>}
      </form>
    </div>
  );
}

export default UploadPage;
