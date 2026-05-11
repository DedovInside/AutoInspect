import "./HistoryPage.css";
import { useEffect, useState } from "react";
import { useNavigate, Link } from "react-router-dom";
import { getAnalysisHistory } from "../../services/analyses";
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
      month: "short",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return iso;
  }
}

function HistoryPage() {
  const navigate = useNavigate();
  const [history, setHistory] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let mounted = true;
    (async () => {
      try {
        const data = await getAnalysisHistory();
        if (mounted) setHistory(data);
      } catch (e) {
        console.error(e);
      } finally {
        if (mounted) setLoading(false);
      }
    })();
    return () => { mounted = false; };
  }, []);

  return (
    <div>
      <div className="page-header">
        <div>
          <h1 className="page-title">История анализов</h1>
          <p className="page-subtitle">Список ранее выполненных анализов</p>
        </div>
        <Link to="/upload" className="btn btn-primary btn-sm">
          <Icon name="plus" size={14} /> Новый анализ
        </Link>
      </div>

      {loading ? (
        <div className="history-list">
          {[0, 1, 2].map((i) => (
            <div key={i} className="history-loading">
              <div className="skeleton" style={{ width: 80, height: 56, borderRadius: 6 }} />
              <div style={{ flex: 1 }}>
                <div className="skeleton" style={{ height: 14, width: "30%", marginBottom: 6 }} />
                <div className="skeleton" style={{ height: 12, width: "40%" }} />
              </div>
            </div>
          ))}
        </div>
      ) : history.length === 0 ? (
        <div className="empty-state">
          <span className="empty-state-icon"><Icon name="history" size={22} /></span>
          <div className="empty-state-title">История анализов пуста</div>
          <div className="empty-state-text">
            Запустите первый анализ - все проверки будут сохраняться здесь.
          </div>
          <Link to="/upload" className="btn btn-primary mt-3">
            Начать анализ
          </Link>
        </div>
      ) : (
        <div className="history-list">
          {history.map((it) => (
            <div
              key={it.id}
              className="history-row"
              onClick={() => navigate(`/result/${it.id}`)}
              role="button"
              tabIndex={0}
            >
              <div className="history-thumb" aria-hidden="true">
                <Icon name="fileImage" size={18} />
              </div>
              <div className="history-info">
                <div className="brand">{it.brand}</div>
                <div className="date">{formatDate(it.created_at)}</div>
              </div>
              <span className={severityBadgeClass(it.overall_severity)}>
                {it.overall_severity}
              </span>
              <span className="arrow"><Icon name="arrowRight" size={16} /></span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export default HistoryPage;
