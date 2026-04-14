import './Header.css';
import { Link, useNavigate } from "react-router-dom";

function Header() {
  const navigate = useNavigate();
  const token = localStorage.getItem("token");

  const handleLogout = () => {
    localStorage.removeItem("token");
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
              <button className='logout-button' onClick={handleLogout}>
                Выйти
              </button>
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