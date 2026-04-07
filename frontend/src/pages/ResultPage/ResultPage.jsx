import { useLocation } from "react-router-dom";

function ResultPage() {
    const location = useLocation();
    const file = location.state?.file;
  
    return (
      <div>
        <h2>Результат анализа</h2>
  
        {!file && <p>Нет изображения для отображения.</p>}
  
        {file && (
          <div>
            <h3>Загруженное изображение:</h3>
            <img
              src={URL.createObjectURL(file)}
              alt="uploaded"
              style={{ width: "400px", marginTop: "10px" }}
            />
          </div>
        )}
      </div>
    );
  }
  
  export default ResultPage;
  