import './AuthPage.css';
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import YandexLoginButton from "../../components/YandexLoginButton/YandexLoginButton";
import { isDevAuthBypassEnabled, startYandexOAuth } from "../../services/authService";

function AuthPage() {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);

  const handleLogin = async () => {
    if (isDevAuthBypassEnabled()) {
      navigate("/home", { replace: true });
      return;
    }

    setLoading(true);

    try {
      const authURL = await startYandexOAuth();
      window.location.assign(authURL);
    } catch (error) {
      console.error("Auth error:", error);
      alert(error.message || "Ошибка авторизации");
      setLoading(false);
    }
  };

  return (
    <div className='auth-page'>
      <div className='auth-card'>
        <h1 className='auth-title'>AutoInspect</h1>
        <p className='auth-subtitle'>
          Войдите или зарегистрируйтесь через Яндекс ID
        </p>
        <YandexLoginButton onClick={handleLogin} disabled={loading} />
      </div>
    </div>
  );
}

export default AuthPage;