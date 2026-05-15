import "./HistoryPage.css";
import { useEffect, useState } from "react";
import { useNavigate, Link } from "react-router-dom";
import { getAnalysisHistory, getAnalysisImageURL } from "../../services/analyses";
import Icon from "../../components/Icon/Icon";

function statusBadgeClass(status) {
  if (status === "done") return "badge badge-severity-clear";
  if (status === "failed") return "badge badge-severity-heavy";
  return "badge badge-muted";
}

function analysisStatusLabel(status) {
  const map = {
    pending: "Ожидает обработки",
    queued: "В очереди",
    processing: "В обработке",
    done: "Завершён",
    failed: "Ошибка",
  };
  return map[status] || "Статус неизвестен";
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
  const [thumbnails, setThumbnails] = useState({});
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let mounted = true;
    const ac = new AbortController();

    (async () => {
      try {
        const data = await getAnalysisHistory();
        if (!mounted) return;
        setHistory(data);

        const completedWithImages = data.filter(
          (item) => item.id && item.status === "done" && item.image_count > 0
        );

        const thumbnailResults = await Promise.allSettled(
          completedWithImages.map(async (item) => {
            const payload = await getAnalysisImageURL(item.id, 0, {
              signal: ac.signal,
            });
            return [item.id, payload?.url || ""];
          })
        );

        if (!mounted) return;

        const nextThumbnails = {};
        thumbnailResults.forEach((result) => {
          if (result.status !== "fulfilled") return;
          const [itemID, url] = result.value;
          if (itemID && url) nextThumbnails[itemID] = url;
        });
        setThumbnails(nextThumbnails);
      } catch (e) {
        console.error(e);
      } finally {
        if (mounted) setLoading(false);
      }
    })();

    return () => {
      mounted = false;
      ac.abort();
    };
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
                {thumbnails[it.id] ? (
                  <img src={thumbnails[it.id]} alt="" />
                ) : (
                  <Icon name="fileImage" size={18} />
                )}
              </div>
              <div className="history-info">
                <div className="brand">{it.vehicle_label || it.brand}</div>
                <div className="date">{formatDate(it.created_at)}</div>
              </div>
              <span className={statusBadgeClass(it.status)}>
                {analysisStatusLabel(it.status)}
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
