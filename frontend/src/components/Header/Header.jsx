import './Header.css';
import { Link, useNavigate } from "react-router-dom";
import { getStoredUser, hasAccessToken, logout } from "../../services/authService";

function Header() {
  const navigate = useNavigate();
  const token = hasAccessToken();
  const user = getStoredUser();

  const displayName = user?.username || "Пользователь";
  const email = user?.email || "";
  const avatarURL = user?.avatar_url || user?.avatarUrl || "";

  const initialsSource = (displayName || email || "U").trim();
  const initials = initialsSource.slice(0, 2).toUpperCase();

  const handleLogout = async () => {
    await logout();
    navigate("/auth");
  };

  return (
    <header className='header'>
      <nav className='header-nav'>
        
        {/* Левая часть */}
        <div>
          <Link to="/home" className='header-logo'>
            AutoInspect
          </Link>
        </div>

        {/* Правая часть */}
        <div className='header-links'>
          {token && (
            <>
              <Link to="/upload">Анализ</Link>
              <Link to="/history">История</Link>

              <div className="profile-menu">
                <button type="button" className="profile-trigger" aria-label="Профиль пользователя">
                  {avatarURL ? (
                    <img src={avatarURL} alt="avatar" className="profile-avatar" />
                  ) : (
                    <span className="profile-initials">{initials}</span>
                  )}
                </button>

                <div className="profile-dropdown">
                  <div className="profile-name">{displayName}</div>
                  {email ? <div className="profile-email">{email}</div> : null}
                  <button className='logout-button' onClick={handleLogout}>
                    Выйти
                  </button>
                </div>
              </div>
            </>
          )}

          {!token && (
            <Link to="/auth">Войти</Link>
          )}
        </div>

      </nav>
    </header>
  );
}

export default Header;