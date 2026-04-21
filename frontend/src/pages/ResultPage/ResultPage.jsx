import { useParams } from "react-router-dom";
import { useEffect, useState } from "react";
import { getAnalysis } from "../../services/analysisService";

function ResultPage() {
  const { id } = useParams();

  const [loading, setLoading] = useState(true);
  const [analysis, setAnalysis] = useState(null);

  useEffect(() => {
    let interval;

    const fetchData = async () => {
      const data = await getAnalysis(id);

      if (data.status === "done") {
        setAnalysis(data);
        setLoading(false);
        clearInterval(interval);
      }
    };

    fetchData();

    interval = setInterval(fetchData, 3000);

    return () => clearInterval(interval);
  }, [id]);

  if (loading) {
    return (
      <div>
        <h2>Анализируем изображения...</h2>
        <p>Это может занять несколько секунд</p>
      </div>
    );
  }

  return (
    <div>
      <h2>Результат анализа</h2>

      <h3>{analysis.brand}</h3>
      <p>{new Date(analysis.created_at).toLocaleString()}</p>

      {/* Изображение */}
      {analysis.image_url && (
        <img
          src={analysis.image_url}
          alt="car"
          style={{ maxWidth: "400px" }}
        />
      )}

      {/* Повреждения */}
      <h3>Обнаруженные повреждения</h3>

      {analysis.damages.length === 0 ? (
        <p>Повреждений не найдено</p>
      ) : (
        analysis.damages.map((d, i) => (
          <p key={i}>
            {d.part} — {d.severity}
          </p>
        ))
      )}

      {/* Сервисы */}
      <h3>Подходящие автосервисы</h3>

      {analysis.services.length === 0 ? (
        <p>Подходящие сервисы не найдены</p>
      ) : (
        analysis.services.map((s, i) => (
          <div key={i} style={{ marginBottom: "12px" }}>
            <b>{s.name}</b>
            <p>{s.phone}</p>
            <p>{s.address}</p>
            {s.description && <p>{s.description}</p>}
          </div>
        ))
      )}
    </div>
  );
}

export default ResultPage;