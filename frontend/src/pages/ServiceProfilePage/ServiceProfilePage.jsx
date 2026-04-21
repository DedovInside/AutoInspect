import { useState, useEffect } from "react";
import "./ServiceProfilePage.css";

const SERVICE_OPTIONS = [
  "Кузовной ремонт",
  "Покраска",
  "Удаление вмятин",
  "Замена стекол",
];

const LEVELS = ["Лёгкий", "Средний", "Сложный"];

function ServiceProfilePage() {
  const [form, setForm] = useState({
    name: "",
    phone: "",
    address: "",
    description: "",
    services: [],
  });

  const [isEditing, setIsEditing] = useState(true);
  const [touched, setTouched] = useState({});

  // имитация загрузки с backend
  useEffect(() => {
    const saved = localStorage.getItem("service_profile");
    if (saved) {
      setForm(JSON.parse(saved));
      setIsEditing(false);
    }
  }, []);

  const isValid =
    form.name.trim() &&
    form.phone.trim() &&
    form.address.trim() &&
    form.services.length > 0 &&
    form.services.every((s) => s.levels && s.levels.length > 0);

  const handleChange = (e) => {
    setForm({
      ...form,
      [e.target.name]: e.target.value,
    });
  };

  const handleServiceToggle = (service) => {
    setForm((prev) => {
      const exists = prev.services.find((s) => s.type === service);

      if (exists) {
        return {
          ...prev,
          services: prev.services.filter((s) => s.type !== service),
        };
      }

      return {
        ...prev,
        services: [...prev.services, { type: service, levels: [] }],
      };
    });
  };

  const handleLevelToggle = (service, level) => {
    setForm((prev) => ({
      ...prev,
      services: prev.services.map((s) => {
        if (s.type !== service) return s;

        const exists = s.levels.includes(level);

        return {
          ...s,
          levels: exists
            ? s.levels.filter((l) => l !== level)
            : [...s.levels, level],
        };
      }),
    }));
  };

  const handleSubmit = (e) => {
    e.preventDefault();

    if (!isValid) return;

    // пока сохраняем локально
    localStorage.setItem("service_profile", JSON.stringify(form));

    setIsEditing(false);

    console.log("SEND TO BACKEND:", form);
  };

  const markTouched = (field) => {
    setTouched((prev) => ({ ...prev, [field]: true }));
  };

  // ================= UI =================

  if (!isEditing) {
    return (
      <div className="service-page">
        <h1>{form.name}</h1>

        <p><b>Телефон:</b> {form.phone}</p>
        <p><b>Адрес:</b> {form.address}</p>

        {form.description && (
          <p><b>Описание:</b> {form.description}</p>
        )}

        <div>
          <b>Услуги:</b>
          <ul>
            {form.services.map((s) => (
              <li key={s.type}>
                {s.type} ({s.levels.join(", ") || "без уточнения"})
              </li>
            ))}
          </ul>
        </div>

        <button onClick={() => setIsEditing(true)}>
          Редактировать профиль
        </button>
      </div>
    );
  }

  return (
    <div className="service-page">
      <h1>Заполните профиль автосервиса</h1>

      <form className="service-form" onSubmit={handleSubmit}>
        
        <input
          type="text"
          name="name"
          placeholder="Название сервиса *"
          value={form.name}
          onChange={handleChange}
          onBlur={() => markTouched("name")}
          className={!form.name && touched.name ? "error" : ""}
        />

        <input
            type="tel"
            name="phone"
            placeholder="Телефон *"
            value={form.phone}
            onChange={handleChange}
            onBlur={() => markTouched("phone")}
            className={!form.phone && touched.phone ? "error" : ""}
            pattern="^[0-9+()\\-\\s]*$"
            inputMode="tel"
        />

        <input
          type="text"
          name="address"
          placeholder="Адрес *"
          value={form.address}
          onChange={handleChange}
          onBlur={() => markTouched("address")}
          className={!form.address && touched.address ? "error" : ""}
        />

        <textarea
          name="description"
          placeholder="Описание (необязательно, до 300 символов)"
          maxLength={300}
          value={form.description}
          onChange={handleChange}
        />

        <div className="services-block">
          <p>За ремонт каких частей вы готовы взяться и какой сложности? *</p>

          {SERVICE_OPTIONS.map((service) => {
            const selected = form.services.find(s => s.type === service);

            return (
              <div key={service} className="service-item">
                <label>
                  <input
                    type="checkbox"
                    checked={!!selected}
                    onChange={() => handleServiceToggle(service)}
                  />
                  {service}
                </label>

                {selected && (
                  <div className="levels">
                    {LEVELS.map((level) => (
                      <label key={level}>
                        <input
                          type="checkbox"
                          checked={selected.levels.includes(level)}
                          onChange={() =>
                            handleLevelToggle(service, level)
                          }
                        />
                        {level}
                      </label>
                    ))}
                  </div>
                )}
              </div>
            );
          })}
        </div>

        <button type="submit" disabled={!isValid}>
          Сохранить
        </button>
      </form>
    </div>
  );
}

export default ServiceProfilePage;