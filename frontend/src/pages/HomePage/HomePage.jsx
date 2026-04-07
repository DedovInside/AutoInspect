import { Link } from 'react-router-dom';

function HomePage() {
  return (
    <div>
      <h2>AutoInspect</h2>
      <p>Сервис автоматического анализа повреждений кузова автомобиля</p>

      <nav>
        <ul>
          <li><Link to="/login">Вход</Link></li>
          <li><Link to="/registration">Регистрация</Link></li>
          <li><Link to="/upload">Начать анализ</Link></li>
        </ul>
      </nav>
    </div>
  );
}

export default HomePage;
