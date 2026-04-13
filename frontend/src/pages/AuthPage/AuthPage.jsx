import './AuthPage.css';
import { useNavigate } from "react-router-dom";
import YandexLoginButton from "../../components/YandexLoginButton/YandexLoginButton";
import { loginWithYandex } from "../../services/authService";

function AuthPage() {
  const navigate = useNavigate();

  const handleLogin = async () => {
    try {
      const data = await loginWithYandex();

      localStorage.setItem("token", data.accessToken);

      navigate("/");
    } catch (error) {
      console.error("Auth error:", error);
      alert("Ошибка авторизации");
    }
  };

  return (
    <div className='auth-page'>
      <div className='auth-card'>
        <h1 className='auth-title'>AutoInspect</h1>
        <p className='auth-subtitle'>
          Войдите или зарегистрируйтесь через Яндекс ID
        </p>
        <YandexLoginButton onClick={handleLogin} />
      </div>
    </div>
  );
}

export default AuthPage;