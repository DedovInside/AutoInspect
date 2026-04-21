import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { getAnalysisHistory } from "../../services/analysisService";

function HistoryPage() {
  const [history, setHistory] = useState([]);
  const [loading, setLoading] = useState(true);

  const navigate = useNavigate();

  useEffect(() => {
    const fetchHistory = async () => {
      try {
        const data = await getAnalysisHistory();
        setHistory(data);
      } catch (err) {
        console.error(err);
      } finally {
        setLoading(false);
      }
    };

    fetchHistory();
  }, []);

  if (loading) {
    return <p>Загрузка истории...</p>;
  }

  if (history.length === 0) {
    return <p>История анализов пуста</p>;
  }

  return (
    <div>
      <h2>История анализов</h2>

      {history.map((item) => (
        <div
          key={item.id}
          onClick={() => navigate(`/result/${item.id}`)}
          style={{
            border: "1px solid #ccc",
            padding: "12px",
            marginBottom: "10px",
            cursor: "pointer",
          }}
        >
          <p><b>Марка:</b> {item.brand}</p>
          <p><b>Дата:</b> {item.created_at}</p>
        </div>
      ))}
    </div>
  );
}

export default HistoryPage;