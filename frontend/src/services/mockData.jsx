/* eslint-disable react-refresh/only-export-components */
/* Lightweight mock data for diploma screenshots.
   No backend coupling — used only by mocked service helpers. */

const PLACEHOLDER_CARS = [
  // Inline SVG placeholders so screens never depend on external network.
  buildCarPlaceholder("#0f172a", "#1e293b", "#4f46e5"),
  buildCarPlaceholder("#1e1b4b", "#3730a3", "#22d3ee"),
  buildCarPlaceholder("#0f172a", "#312e81", "#f59e0b"),
  buildCarPlaceholder("#1f2937", "#374151", "#10b981"),
];

function buildCarPlaceholder(bg1 = "#1f2937", bg2 = "#0f172a", accent = "#6366f1") {
  const svg = `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1200 750">
  <defs>
    <linearGradient id="g" x1="0" y1="0" x2="1" y2="1">
      <stop offset="0%" stop-color="${bg1}"/>
      <stop offset="100%" stop-color="${bg2}"/>
    </linearGradient>
    <radialGradient id="r" cx="50%" cy="100%" r="80%">
      <stop offset="0%" stop-color="${accent}" stop-opacity="0.45"/>
      <stop offset="100%" stop-color="${accent}" stop-opacity="0"/>
    </radialGradient>
  </defs>
  <rect width="1200" height="750" fill="url(#g)"/>
  <rect width="1200" height="750" fill="url(#r)"/>
  <g transform="translate(110,260)" stroke="#cbd5e1" stroke-width="3" fill="none" stroke-linecap="round" stroke-linejoin="round" opacity="0.95">
    <!-- Stylized side profile of a car -->
    <path d="M40 220 L120 130 Q160 90 220 80 L640 80 Q700 90 740 130 L820 220 L920 220 Q970 220 970 260 L970 300 Q970 320 950 320 L40 320 Q20 320 20 300 L20 260 Q20 220 60 220 Z"
      fill="#0b1220" stroke="#94a3b8" />
    <path d="M170 220 L260 110 L470 110 L470 220 Z" fill="#1e293b" stroke="#475569"/>
    <path d="M480 110 L600 110 Q650 110 700 160 L740 220 L480 220 Z" fill="#1e293b" stroke="#475569"/>
    <circle cx="220" cy="320" r="60" fill="#0f172a" stroke="#94a3b8"/>
    <circle cx="220" cy="320" r="28" fill="#1f2937" stroke="#cbd5e1"/>
    <circle cx="780" cy="320" r="60" fill="#0f172a" stroke="#94a3b8"/>
    <circle cx="780" cy="320" r="28" fill="#1f2937" stroke="#cbd5e1"/>
  </g>
  <g fill="#fff" font-family="-apple-system, Inter, sans-serif" opacity="0.55">
    <text x="60" y="80" font-size="22" font-weight="600">AutoInspect • DEMO PHOTO</text>
  </g>
</svg>`;
  return `data:image/svg+xml;utf8,${encodeURIComponent(svg)}`;
}

export const CAR_BRANDS = [
  { value: "lada", label: "LADA (ВАЗ)" },
  { value: "kia", label: "Kia" },
  { value: "hyundai", label: "Hyundai" },
  { value: "toyota", label: "Toyota" },
  { value: "bmw", label: "BMW" },
  { value: "audi", label: "Audi" },
  { value: "mercedes", label: "Mercedes-Benz" },
  { value: "volkswagen", label: "Volkswagen" },
  { value: "skoda", label: "Škoda" },
  { value: "renault", label: "Renault" },
  { value: "nissan", label: "Nissan" },
  { value: "mazda", label: "Mazda" },
  { value: "other", label: "Другое" },
];

export const SERVICE_OPTIONS = [
  "Кузовной ремонт",
  "Покраска",
  "Удаление вмятин",
  "Замена стекол",
  "Полировка",
  "Замена бамперов",
  "Сварочные работы",
];

export const DAMAGE_LEVELS = ["Лёгкий", "Средний", "Сложный"];

/* ---- Analyses ---- */

export const MOCK_ANALYSES = [
  {
    id: "a-1042",
    status: "done",
    brand: "BMW X5",
    brand_value: "bmw",
    created_at: "2026-05-06T14:32:00",
    overall_severity: "Средний",
    damages: [
      { part: "Передний бампер", severity: "Средний" },
      { part: "Левое переднее крыло", severity: "Лёгкий" },
      { part: "Левая фара", severity: "Лёгкий" },
    ],
    services: [
      { id: "s1", name: "AutoFix Премиум", phone: "+7 (495) 555-12-34", address: "Москва, ул. Ленина 24", description: "Кузовной ремонт любой сложности, работа с премиум-марками", rating: 4.9, distance_km: 2.1, levels: ["Лёгкий", "Средний", "Сложный"] },
      { id: "s2", name: "BodyShop Garage", phone: "+7 (499) 333-44-55", address: "Москва, Профсоюзная 42", description: "Покраска и кузовные работы. Оригинальные краски.", rating: 4.7, distance_km: 4.3, levels: ["Лёгкий", "Средний"] },
      { id: "s3", name: "Гараж 24/7", phone: "+7 (985) 222-11-00", address: "Москва, Каширское ш. 12", description: "Срочный ремонт без записи", rating: 4.5, distance_km: 6.8, levels: ["Лёгкий"] },
    ],
  },
  {
    id: "a-1037",
    status: "done",
    brand: "Toyota Camry",
    brand_value: "toyota",
    created_at: "2026-05-04T09:15:00",
    overall_severity: "Лёгкий",
    damages: [
      { part: "Задний бампер", severity: "Лёгкий" },
    ],
    services: [
      { id: "s4", name: "Toyota Service Center", phone: "+7 (495) 777-44-21", address: "Москва, Дмитровское ш. 99", description: "Авторизованный сервис Toyota", rating: 4.8, distance_km: 5.2, levels: ["Лёгкий", "Средний"] },
      { id: "s2", name: "BodyShop Garage", phone: "+7 (499) 333-44-55", address: "Москва, Профсоюзная 42", description: "Покраска и кузовные работы", rating: 4.7, distance_km: 4.3, levels: ["Лёгкий", "Средний"] },
    ],
  },
  {
    id: "a-1029",
    status: "done",
    brand: "LADA Vesta",
    brand_value: "lada",
    created_at: "2026-04-30T18:42:00",
    overall_severity: "Сложный",
    damages: [
      { part: "Передний бампер", severity: "Сложный" },
      { part: "Капот", severity: "Средний" },
      { part: "Правая фара", severity: "Сложный" },
      { part: "Решётка радиатора", severity: "Средний" },
    ],
    services: [
      { id: "s5", name: "АвтоМастер", phone: "+7 (495) 111-22-33", address: "Москва, Варшавское ш. 7", description: "Полный цикл кузовных работ для российских марок", rating: 4.6, distance_km: 3.4, levels: ["Лёгкий", "Средний", "Сложный"] },
      { id: "s1", name: "AutoFix Премиум", phone: "+7 (495) 555-12-34", address: "Москва, ул. Ленина 24", description: "Кузовной ремонт любой сложности", rating: 4.9, distance_km: 2.1, levels: ["Лёгкий", "Средний", "Сложный"] },
    ],
  },
  {
    id: "a-1018",
    status: "done",
    brand: "Hyundai Solaris",
    brand_value: "hyundai",
    created_at: "2026-04-22T11:08:00",
    overall_severity: "Без повреждений",
    damages: [],
    services: [],
  },
  {
    id: "a-1011",
    status: "done",
    brand: "Audi Q5",
    brand_value: "audi",
    created_at: "2026-04-14T16:55:00",
    overall_severity: "Лёгкий",
    damages: [
      { part: "Левая дверь", severity: "Лёгкий" },
    ],
    services: [
      { id: "s6", name: "Audi Centre", phone: "+7 (495) 999-00-77", address: "Москва, Ленинградский пр-т 39", description: "Авторизованный сервис Audi", rating: 4.9, distance_km: 7.1, levels: ["Лёгкий", "Средний"] },
    ],
  },
];

export function getMockAnalysis(id) {
  return MOCK_ANALYSES.find((a) => a.id === id) || MOCK_ANALYSES[0];
}

/* ---- Repair requests ---- */

export const MOCK_REPAIR_REQUESTS_USER = [
  {
    id: "r-501",
    created_at: "2026-05-06T15:00:00",
    car_brand: "BMW X5",
    analysis_id: "a-1042",
    service: { name: "AutoFix Премиум", phone: "+7 (495) 555-12-34", address: "Москва, ул. Ленина 24" },
    status: "accepted",
    note: "Записаны на 12 мая, 10:00. Ожидаем подтверждения по запчастям.",
    damage_summary: "Передний бампер, левое крыло, левая фара",
  },
  {
    id: "r-498",
    created_at: "2026-05-04T10:42:00",
    car_brand: "Toyota Camry",
    analysis_id: "a-1037",
    service: { name: "Toyota Service Center", phone: "+7 (495) 777-44-21", address: "Москва, Дмитровское ш. 99" },
    status: "pending",
    note: "Заявка отправлена, ожидаем ответа автосервиса.",
    damage_summary: "Задний бампер",
  },
  {
    id: "r-471",
    created_at: "2026-04-30T19:30:00",
    car_brand: "LADA Vesta",
    analysis_id: "a-1029",
    service: { name: "АвтоМастер", phone: "+7 (495) 111-22-33", address: "Москва, Варшавское ш. 7" },
    status: "rejected",
    note: "Загружены текущей очередью на ближайшие 3 недели. Рекомендуем обратиться в AutoFix Премиум.",
    damage_summary: "Передний бампер, капот, правая фара, решётка",
  },
];

export const MOCK_REPAIR_REQUESTS_SERVICE = [
  {
    id: "r-512",
    created_at: "2026-05-07T09:14:00",
    car_brand: "Mercedes-Benz E200",
    analysis_id: "a-1051",
    user: { email: "mikhail.p@yandex.ru" },
    status: "pending",
    damage_summary: "Левая передняя дверь, переднее крыло",
    severity: "Средний",
    image_url: PLACEHOLDER_CARS[0],
  },
  {
    id: "r-511",
    created_at: "2026-05-07T08:42:00",
    car_brand: "Volkswagen Tiguan",
    analysis_id: "a-1050",
    user: { email: "olga.s@yandex.ru" },
    status: "pending",
    damage_summary: "Задний бампер",
    severity: "Лёгкий",
    image_url: PLACEHOLDER_CARS[2],
  },
  {
    id: "r-509",
    created_at: "2026-05-06T16:08:00",
    car_brand: "BMW 3 series",
    analysis_id: "a-1049",
    user: { email: "d.alex@yandex.ru" },
    status: "accepted",
    damage_summary: "Передний бампер, капот",
    severity: "Средний",
    image_url: PLACEHOLDER_CARS[1],
  },
  {
    id: "r-505",
    created_at: "2026-05-06T11:21:00",
    car_brand: "Kia Rio",
    analysis_id: "a-1048",
    user: { email: "yk@yandex.ru" },
    status: "rejected",
    damage_summary: "Все стороны кузова",
    severity: "Сложный",
    image_url: PLACEHOLDER_CARS[3],
  },
];

/* ---- Admin: ML models ---- */

export const MOCK_ML_MODELS = [
  {
    id: "m-001",
    brand: "BMW",
    model: "X5",
    generation: "G05",
    years: "2018",
    version: "v2.4.1",
    file: "bmw_x5_g05_v241.pt",
    parts_catalog: "parts_catalog.json",
    accuracy: 0.946,
    created_at: "2026-04-12",
    status: "active",
  },
  {
    id: "m-002",
    brand: "Toyota",
    model: "Camry",
    generation: "XV70",
    years: "2017",
    version: "v3.0.0",
    file: "toyota_camry_xv70_v300.pt",
    parts_catalog: "parts_catalog.json",
    accuracy: 0.962,
    created_at: "2026-04-22",
    status: "active",
  },
  {
    id: "m-003",
    brand: "LADA",
    model: "Vesta",
    generation: "I",
    years: "2015",
    version: "v1.8.2",
    file: "lada_vesta_v182.pt",
    parts_catalog: "parts_catalog.json",
    accuracy: 0.918,
    created_at: "2026-03-08",
    status: "active",
  },
  {
    id: "m-004",
    brand: "Hyundai",
    model: "Solaris",
    generation: "II",
    years: "2017",
    version: "v1.4.0",
    file: "hyundai_solaris_v140.pt",
    parts_catalog: "parts_catalog.json",
    accuracy: 0.901,
    created_at: "2026-02-19",
    status: "deprecated",
  },
  {
    id: "m-005",
    brand: "Audi",
    model: "Q5",
    generation: "II",
    years: "2017",
    version: "v2.1.0",
    file: "audi_q5_v210.pt",
    parts_catalog: "parts_catalog.json",
    accuracy: 0.939,
    created_at: "2026-01-30",
    status: "active",
  },
];

/* ---- Admin: Service registration requests ---- */

export const MOCK_SERVICE_REQUESTS = [
  {
    id: "sr-014",
    organization: "AutoFix Премиум",
    city: "Москва",
    address: "ул. Ленина, 24",
    contact_name: "Иван Михайлов",
    contact_phone: "+7 (495) 555-12-34",
    contact_email: "premium@autofix.example",
    description: "Кузовной ремонт и покраска любой сложности. 12 лет на рынке.",
    submitted_at: "2026-05-05",
    status: "pending",
  },
  {
    id: "sr-013",
    organization: "BodyShop Garage",
    city: "Москва",
    address: "Профсоюзная, 42",
    contact_name: "Анастасия Кравченко",
    contact_phone: "+7 (499) 333-44-55",
    contact_email: "office@bodyshop-garage.example",
    description: "Локальная покраска, кузовные работы.",
    submitted_at: "2026-05-03",
    status: "approved",
  },
  {
    id: "sr-012",
    organization: "Гараж 24/7",
    city: "Подольск",
    address: "Каширское ш., 12",
    contact_name: "Сергей Орлов",
    contact_phone: "+7 (985) 222-11-00",
    contact_email: "info@garage24.example",
    description: "Круглосуточный ремонт без записи.",
    submitted_at: "2026-05-02",
    status: "pending",
  },
  {
    id: "sr-011",
    organization: "Авто-Эксперт",
    city: "Санкт-Петербург",
    address: "Невский пр-т, 88",
    contact_name: "Алексей Сидоров",
    contact_phone: "+7 (812) 555-77-99",
    contact_email: "zakaz@auto-expert.example",
    description: "",
    submitted_at: "2026-04-29",
    status: "rejected",
  },
];

/* ---- Admin: Training requests ---- */

export const MOCK_TRAINING_REQUESTS = [
  {
    id: "tr-027",
    brand: "Geely",
    model: "Coolray",
    generation: "I",
    year: "2023",
    description: "Растущая популярность модели, частые обращения от пользователей.",
    submitted_by: "user_872",
    submitted_at: "2026-05-06",
    status: "pending",
  },
  {
    id: "tr-026",
    brand: "Chery",
    model: "Tiggo 7 Pro",
    generation: "I",
    year: "2022",
    description: "Запросы от автосервисов, нет обученной модели в системе.",
    submitted_by: "user_645",
    submitted_at: "2026-05-04",
    status: "approved",
  },
  {
    id: "tr-025",
    brand: "Haval",
    model: "Jolion",
    generation: "I",
    year: "2021",
    description: "Популярная модель, требуется отдельная модель распознавания.",
    submitted_by: "user_512",
    submitted_at: "2026-05-01",
    status: "completed",
  },
  {
    id: "tr-024",
    brand: "Exeed",
    model: "TXL",
    generation: "I",
    year: "2020",
    description: "Уточнить параметры модели по фото от автосервисов.",
    submitted_by: "user_388",
    submitted_at: "2026-04-22",
    status: "rejected",
  },
];

/* ---- Service profile (default mocked) ---- */

export const MOCK_SERVICE_PROFILE = {
  name: "AutoFix Премиум",
  phone: "+7 (495) 555-12-34",
  address: "Москва, ул. Ленина 24",
  description: "Кузовной ремонт и покраска автомобилей премиум-сегмента. 12 лет на рынке. Гарантия на работы 24 месяца.",
  services: [
    { type: "Кузовной ремонт", levels: ["Лёгкий", "Средний", "Сложный"] },
    { type: "Покраска", levels: ["Лёгкий", "Средний"] },
    { type: "Полировка", levels: ["Лёгкий"] },
  ],
};

/* ---- Stats for admin dashboard ---- */

export const MOCK_ADMIN_STATS = {
  totalAnalyses: 12480,
  analysesGrowth: 12.4,
  modelsActive: 4,
  servicesActive: 28,
  pendingServiceRequests: 2,
  pendingTrainingRequests: 3,
};

export { PLACEHOLDER_CARS };
