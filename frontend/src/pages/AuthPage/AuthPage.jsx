import './AuthPage.css';
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { isDevAuthBypassEnabled, startYandexOAuth } from "../../services/authService";
import Icon from "../../components/Icon/Icon";
import logo from "../../assets/logo.jpeg";

function AuthPage() {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const handleLogin = async () => {
    setError("");

    if (isDevAuthBypassEnabled()) {
      navigate("/home", { replace: true });
      return;
    }

    setLoading(true);

    try {
      const authURL = await startYandexOAuth();
      window.location.assign(authURL);
    } catch (err) {
      console.error("Auth error:", err);
      setError(err?.message || "Ошибка авторизации");
      setLoading(false);
    }
  };

  return (
    <div className="auth-page">
      <div className="auth-card">
        <img src={logo} alt="AutoInspect logo" className="auth-logo" />
        <h1 className="auth-title">AutoInspect</h1>
        <p className="auth-subtitle">
          Войдите или зарегистрируйтесь через Яндекс ID
        </p>

        {error && (
          <div className="auth-error" role="alert">
            <Icon name="alert" size={16} />
            <div>{error}</div>
          </div>
        )}

        <button
          type="button"
          className="yandex-btn"
          onClick={handleLogin}
          disabled={loading}
          aria-busy={loading}
        >
          {loading ? (
            <>
              <span className="auth-spinner" />
              Переходим в Яндекс...
            </>
          ) : (
            <>
              <span className="y-mark">Я</span>
              Войти через Яндекс
            </>
          )}
        </button>
      </div>
    </div>
  );
}

export default AuthPage;
