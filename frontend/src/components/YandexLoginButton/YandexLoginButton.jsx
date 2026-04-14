function YandexLoginButton({ onClick, disabled = false }) {
  return (
    <button type="button" onClick={onClick} disabled={disabled}>
      {disabled ? "Переходим в Яндекс..." : "Войти через Яндекс"}
    </button>
  );
}

export default YandexLoginButton;
