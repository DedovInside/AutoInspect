import "./UploadPage.css";
import { useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { uploadImages } from "../../services/analyses";
import Icon from "../../components/Icon/Icon";

function formatBytes(bytes) {
  if (!bytes && bytes !== 0) return "";
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

const initialVehicle = {
  brand: "",
  model: "",
  generation: "",
  year: "",
};

function UploadPage() {
  const navigate = useNavigate();
  const inputRef = useRef(null);

  const [items, setItems] = useState([]);
  const [vehicle, setVehicle] = useState(initialVehicle);
  const [fieldErrors, setFieldErrors] = useState({});
  const [dragging, setDragging] = useState(false);
  const [loading, setLoading] = useState(false);
  const [submitError, setSubmitError] = useState("");

  const onFiles = (fileList) => {
    const fresh = Array.from(fileList || []).map((f) => ({
      id: `${f.name}-${f.size}-${Math.random().toString(36).slice(2, 7)}`,
      file: f,
      url: URL.createObjectURL(f),
      name: f.name,
      size: f.size,
    }));
    setItems((prev) => [...prev, ...fresh]);
    if (fresh.length) {
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
    if (!vehicle.brand.trim()) {
      e.brand = "Введите марку автомобиля";
    }
    if (!vehicle.model.trim()) {
      e.model = "Введите модель автомобиля";
    }
    if (!vehicle.generation.trim()) {
      e.generation = "Введите поколение";
    }
    if (!vehicle.year.trim()) {
      e.year = "Введите год";
    }
    if (items.length === 0) {
      e.files = "Выберите изображение/я";
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
        brand: vehicle.brand.trim(),
        model: vehicle.model.trim(),
        generation: vehicle.generation.trim(),
        year: vehicle.year.trim(),
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

        <div className="form-row">
          <label className="form-label" htmlFor="up-brand">Марка автомобиля *</label>
          <input
            id="up-brand"
            className={"input" + (fieldErrors.brand ? " input-error" : "")}
            placeholder="Например, Kia"
            value={vehicle.brand}
            onChange={(e) => {
              setVehicle((v) => ({ ...v, brand: e.target.value }));
              clearFieldError("brand");
            }}
          />
          {fieldErrors.brand && <div className="field-error">{fieldErrors.brand}</div>}
        </div>

        <div className="form-row">
          <label className="form-label" htmlFor="up-model">Модель автомобиля *</label>
          <input
            id="up-model"
            className={"input" + (fieldErrors.model ? " input-error" : "")}
            placeholder="Например, Coolray"
            value={vehicle.model}
            onChange={(e) => {
              setVehicle((v) => ({ ...v, model: e.target.value }));
              clearFieldError("model");
            }}
          />
          {fieldErrors.model && <div className="field-error">{fieldErrors.model}</div>}
        </div>

        <div className="form-row-inline">
          <div className="form-row">
            <label className="form-label" htmlFor="up-gen">Поколение *</label>
            <input
              id="up-gen"
              className={"input" + (fieldErrors.generation ? " input-error" : "")}
              placeholder="I, II, FL"
              value={vehicle.generation}
              onChange={(e) => {
                setVehicle((v) => ({ ...v, generation: e.target.value }));
                clearFieldError("generation");
              }}
            />
            {fieldErrors.generation && (
              <div className="field-error">{fieldErrors.generation}</div>
            )}
          </div>
          <div className="form-row">
            <label className="form-label" htmlFor="up-year">Год *</label>
            <input
              id="up-year"
              className={"input" + (fieldErrors.year ? " input-error" : "")}
              placeholder="2018"
              value={vehicle.year}
              onChange={(e) => {
                setVehicle((v) => ({ ...v, year: e.target.value }));
                clearFieldError("year");
              }}
            />
            {fieldErrors.year && <div className="field-error">{fieldErrors.year}</div>}
          </div>
        </div>

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
          {(items.length > 0 || Object.values(vehicle).some((v) => v.trim())) && !loading && (
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
