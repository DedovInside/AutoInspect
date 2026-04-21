import { useState } from "react";
import { useNavigate } from "react-router-dom";
import Button from "../../components/Button/Button";
import FileUploader from "../../components/FileUploader/FileUploader";
import BrandSelector from "../../components/BrandSelector/BrandSelector";

function UploadPage() {
  const [files, setFiles] = useState([]);
  const [brand, setBrand] = useState("");
  const navigate = useNavigate();

  const handleSubmit = () => {
    if (files.length === 0) {
      alert("Выберите файл/файлы");
      return;
    }

    if (!brand) {
      alert("Выберите марку автомобиля");
      return;
    }
  
    navigate("/result", { state: { files, brand } });
  };

  return (
    <div>
      <h2>Загрузка изображений</h2>

      <BrandSelector value={brand} onChange={(e) => setBrand(e.target.value)} />

      <FileUploader onFileSelect={setFiles} />

      <Button onClick={handleSubmit}>
          Отправить
      </Button>

      <p>
        Если вашей марки нет в списке, анализ может быть проведён с использованием общей модели при выбори "Другое". Точность может быть немного ниже.
      </p>

    </div>
  );
}

export default UploadPage;