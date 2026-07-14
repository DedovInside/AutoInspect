import './Header.css';
import { useEffect, useState } from "react";
import { Link, NavLink, useNavigate } from "react-router-dom";
import { useAuth } from "../../auth/AuthContext";
import { getRoleLabel } from "../../auth/roleLabels";
import Icon from "../Icon/Icon";
import logo from "../../assets/logo.jpeg";

function Header() {
  const navigate = useNavigate();
  const { user, role, isAuthenticated, logout } = useAuth();
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  const displayName =
    user?.display_name ||
    user?.displayName ||
    [user?.first_name || user?.firstName, user?.last_name || user?.lastName]
      .filter(Boolean)
      .join(" ") ||
    user?.username ||
    "Пользователь";
  const email = user?.email || "user@autoinspect.ru";
  const avatarURL = user?.avatar_url || user?.avatarUrl || "";

  const initialsSource = (displayName || email || "U").trim();
  const initials = initialsSource.slice(0, 2).toUpperCase();

  // Reset the "image is broken" flag whenever the avatar URL changes
  // (login → logout → login with a different account, profile photo update, etc.)
  const [avatarBroken, setAvatarBroken] = useState(false);
  useEffect(() => {
    setAvatarBroken(false);
  }, [avatarURL]);

  const showAvatarImage = Boolean(avatarURL) && !avatarBroken;

  const token = isAuthenticated;

  const navItems = token
    ? [
        { to: "/upload", label: "Анализ" },
        { to: "/history", label: "История" },
        { to: "/repair-requests", label: "Заявки" },
        ...(role === "SERVICE"
          ? [{ to: "/service-profile", label: "Профиль сервиса" }]
          : [{ to: "/profile", label: "Профиль" }]),
        ...(role === "ADMIN" ? [{ to: "/admin", label: "Админ панель" }] : []),
      ]
    : [];

  const profilePath = role === "SERVICE" ? "/service-profile" : "/profile";
  const profileLabel = role === "SERVICE" ? "Профиль сервиса" : "Профиль";

  const handleLogout = async () => {
    setMobileMenuOpen(false);
    await logout();
    navigate("/auth");
  };

  return (
    <header className="app-header">
      <div className="app-header-inner">
        <Link to="/home" className="header-brand">
          <img src={logo} alt="AutoInspect logo" className="logo" />
          AutoInspect
        </Link>

        {token && (
          <nav className="header-nav" aria-label="Главная навигация">
            {navItems.map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                className={({ isActive }) => "header-nav-link" + (isActive ? " active" : "")}
              >
                {item.label}
              </NavLink>
            ))}
          </nav>
        )}

        <div className="header-actions">
          {token ? (
            <>
              <div className="mobile-nav-menu">
                <button
                  type="button"
                  className="mobile-nav-trigger"
                  aria-label={mobileMenuOpen ? "Закрыть меню" : "Открыть меню"}
                  aria-expanded={mobileMenuOpen}
                  onClick={() => setMobileMenuOpen((open) => !open)}
                >
                  <Icon name={mobileMenuOpen ? "x" : "menu"} size={18} />
                </button>

                <div className={`mobile-nav-dropdown${mobileMenuOpen ? " open" : ""}`} role="menu">
                  {navItems.map((item) => (
                    <NavLink
                      key={item.to}
                      to={item.to}
                      className={({ isActive }) => "mobile-nav-link" + (isActive ? " active" : "")}
                      onClick={() => setMobileMenuOpen(false)}
                    >
                      {item.label}
                    </NavLink>
                  ))}
                </div>
              </div>

              <div className="profile-menu">
                <button type="button" className="profile-trigger" aria-label="Профиль пользователя">
                  <span className="avatar" aria-hidden="true">
                    {showAvatarImage ? (
                      <img
                        src={avatarURL}
                        alt={displayName}
                        onError={() => setAvatarBroken(true)}
                        referrerPolicy="no-referrer"
                      />
                    ) : (
                      initials
                    )}
                  </span>
                  <span className="name">{displayName}</span>
                  <Icon name="chevronDown" size={14} />
                </button>

                <div className="profile-dropdown" role="menu">
                  <div className="profile-dropdown-header">
                    <div className="profile-dropdown-name">{displayName}</div>
                    <div className="profile-dropdown-email">{email}</div>
                    {role && (
                      <div className="profile-dropdown-role">{getRoleLabel(role)}</div>
                    )}
                  </div>
                  <Link
                    to={profilePath}
                    className="profile-dropdown-item"
                    role="menuitem"
                    onClick={() => setMobileMenuOpen(false)}
                  >
                    <Icon name="user" size={14} /> {profileLabel}
                  </Link>
                  <button type="button" className="profile-dropdown-item danger" onClick={handleLogout}>
                    <Icon name="logout" size={14} /> Выйти
                  </button>
                </div>
              </div>
            </>
          ) : (
            <Link to="/auth" className="btn btn-primary btn-sm">Войти</Link>
          )}
        </div>
      </div>
    </header>
  );
}

export default Header;
