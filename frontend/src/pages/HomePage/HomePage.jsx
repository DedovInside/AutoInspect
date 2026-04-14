import './HomePage.css';
import { Link } from 'react-router-dom';
import Button from "../../components/Button/Button";

function HomePage() {
  return (
    <div className='home-page'>
      <h1 className='home-title'>
        AutoInspect
      </h1>
      <p className='home-subtitle'>
        Сервис автоматического анализа повреждений кузова автомобиля
      </p>

      <Link to="/upload">
        <Button className='start-button'>
          Начать анализ
        </Button>
      </Link>
    </div>
  );
}

export default HomePage;