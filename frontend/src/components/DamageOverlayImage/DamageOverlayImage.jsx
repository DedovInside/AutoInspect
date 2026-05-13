import "./DamageOverlayImage.css";

const DAMAGE_COLORS = [
  "#f97316",
  "#22c55e",
  "#3b82f6",
  "#a855f7",
  "#eab308",
  "#06b6d4",
  "#f43f5e",
  "#84cc16",
];

function toNumber(value) {
  const n = Number(value);
  return Number.isFinite(n) ? n : 0;
}

function normalizePoint(point) {
  if (!Array.isArray(point) || point.length < 2) return null;
  return [toNumber(point[0]), toNumber(point[1])];
}

function polygonPoints(polygon) {
  if (!Array.isArray(polygon)) return "";
  return polygon
    .map(normalizePoint)
    .filter(Boolean)
    .map(([x, y]) => `${x},${y}`)
    .join(" ");
}

function safeBBox(damage) {
  const bbox = Array.isArray(damage?.bbox) ? damage.bbox : [];
  if (bbox.length === 4) {
    const [x1, y1, x2, y2] = bbox.map(toNumber);
    return { x: x1, y: y1, width: Math.max(0, x2 - x1), height: Math.max(0, y2 - y1) };
  }

  const points = Array.isArray(damage?.polygon)
    ? damage.polygon.map(normalizePoint).filter(Boolean)
    : [];
  if (points.length < 1) return null;

  const xs = points.map(([x]) => x);
  const ys = points.map(([, y]) => y);
  const x1 = Math.min(...xs);
  const y1 = Math.min(...ys);
  const x2 = Math.max(...xs);
  const y2 = Math.max(...ys);
  return { x: x1, y: y1, width: Math.max(0, x2 - x1), height: Math.max(0, y2 - y1) };
}

function damageLabel(damage, index) {
  return damage?.damage_name_ru || damage?.damage_type || `Повреждение ${index + 1}`;
}

function imageSize(imageResult) {
  const info = imageResult?.image || imageResult?.image_info || {};
  const width = Number(info.width);
  const height = Number(info.height);
  if (Number.isFinite(width) && width > 0 && Number.isFinite(height) && height > 0) {
    return { width, height };
  }
  return { width: 1, height: 1 };
}

function DamageOverlayImage({ src, imageResult, alt = "Изображение автомобиля", className = "" }) {
  const damages = Array.isArray(imageResult?.damage_instances)
    ? imageResult.damage_instances
    : [];
  const { width, height } = imageSize(imageResult);
  const hasOverlay = damages.length > 0 && width > 1 && height > 1;

  return (
    <div className={`damage-overlay ${className}`.trim()}>
      {src ? (
        <span className="damage-overlay-stage">
          <img src={src} alt={alt} />
          {hasOverlay ? (
            <svg
              className="damage-overlay-svg"
              viewBox={`0 0 ${width} ${height}`}
              preserveAspectRatio="none"
              aria-hidden="true"
            >
              {damages.map((damage, index) => {
                const color = DAMAGE_COLORS[index % DAMAGE_COLORS.length];
                const points = polygonPoints(damage?.polygon);
                const bbox = safeBBox(damage);
                const labelX = bbox ? bbox.x : 8;
                const labelY = bbox ? Math.max(16, bbox.y - 8) : 18;
                return (
                  <g key={damage?.id || `${index}:${points}`}>
                    {points ? (
                      <polygon
                        points={points}
                        fill={color}
                        fillOpacity="0.28"
                        stroke={color}
                        strokeWidth="2"
                        vectorEffect="non-scaling-stroke"
                      />
                    ) : bbox ? (
                      <rect
                        x={bbox.x}
                        y={bbox.y}
                        width={bbox.width}
                        height={bbox.height}
                        fill={color}
                        fillOpacity="0.18"
                        stroke={color}
                        strokeWidth="2"
                        vectorEffect="non-scaling-stroke"
                      />
                    ) : null}
                    {bbox ? (
                      <rect
                        x={bbox.x}
                        y={bbox.y}
                        width={bbox.width}
                        height={bbox.height}
                        fill="none"
                        stroke={color}
                        strokeWidth="2"
                        strokeDasharray="5 4"
                        vectorEffect="non-scaling-stroke"
                      />
                    ) : null}
                    <text
                      x={labelX}
                      y={labelY}
                      fill="#ffffff"
                      stroke="rgba(15, 23, 42, 0.9)"
                      strokeWidth="4"
                      paintOrder="stroke"
                      fontSize="14"
                      fontWeight="700"
                    >
                      {damageLabel(damage, index)}
                    </text>
                  </g>
                );
              })}
            </svg>
          ) : null}
        </span>
      ) : (
        <div className="damage-overlay-placeholder">Изображение недоступно</div>
      )}
    </div>
  );
}

export default DamageOverlayImage;
