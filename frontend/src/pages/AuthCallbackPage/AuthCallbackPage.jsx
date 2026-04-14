import { useEffect, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { exchangeYandexCode } from "../../services/authService";
import Loader from "../../components/Loader/Loader";
import "./AuthCallbackPage.css";

const exchangeInFlight = new Map();

function getExchangePromise(key, code, state) {
  const existing = exchangeInFlight.get(key);
  if (existing) {
    return existing;
  }

  const created = exchangeYandexCode(code, state).finally(() => {
    exchangeInFlight.delete(key);
  });
  exchangeInFlight.set(key, created);
  return created;
}

function AuthCallbackPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const [errorMessage, setErrorMessage] = useState("");

  useEffect(() => {
    let active = true;

    const run = async () => {
      const code = searchParams.get("code");
      const state = searchParams.get("state");
      const oauthError = searchParams.get("error");

      if (oauthError) {
        if (active) {
          setErrorMessage("Не удалось войти через Яндекс. Попробуйте снова.");
        }
        return;
      }

      if (!code || !state) {
        if (active) {
          setErrorMessage("Некорректный ответ авторизации. Попробуйте снова.");
        }
        return;
      }

      try {
        const requestKey = `${code}:${state}`;
        await getExchangePromise(requestKey, code, state);
        navigate("/home", { replace: true });
      } catch (error) {
        if (active) {
          setErrorMessage(error.message || "Не удалось завершить вход. Попробуйте снова.");
        }
      }
    };

    run();

    return () => {
      active = false;
    };
  }, [navigate, searchParams]);

  return (
    <div className="auth-callback-page">
      <div className="auth-callback-card">
        <h2 className="auth-callback-title">Выполняем вход</h2>
        {errorMessage ? (
          <>
            <p className="auth-callback-error">{errorMessage}</p>
            <button className="auth-callback-button" onClick={() => navigate("/auth", { replace: true })}>
              Вернуться ко входу
            </button>
          </>
        ) : (
          <Loader label="Проверяем данные и завершаем авторизацию..." />
        )}
      </div>
    </div>
  );
}

export default AuthCallbackPage;

