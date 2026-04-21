import { useState } from "react";
import { useNavigate } from "react-router-dom";
import Button from "../../components/Button/Button";
import FileUploader from "../../components/FileUploader/FileUploader";
import BrandSelector from "../../components/BrandSelector/BrandSelector";
import { uploadImages } from "../../services/analysisService";

function UploadPage() {
  const [files, setFiles] = useState([]);
  const [brand, setBrand] = useState("");
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const handleSubmit = async () => {
    setError("");
  
    if (files.length === 0) {
      setError("Выберите изображение/я");
      return;
    }
  
    if (!brand) {
      setError("Выберите марку автомобиля");
      return;
    }
  
    try {
      setLoading(true);
  
      const result = await uploadImages(files, brand);
  
      const analysisId = result.analysis_id;
  
      navigate(`/result/${analysisId}`);
    } catch (err) {
      console.error(err);
      setError("Не удалось отправить изображения");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      <h2>Загрузка изображений</h2>

      <BrandSelector value={brand} onChange={(e) => setBrand(e.target.value)} />

      <FileUploader onFileSelect={setFiles} />

      {error && <p style={{ color: "red"}}>{error}</p>}

      <Button onClick={handleSubmit} disabled={loading}>
          {loading ? "Отправка..." : "Отправить"}
      </Button>

      {loading && <p>Идет загрузка</p>}

      <p>
        Если вашей марки нет в списке, анализ может быть проведён с использованием общей модели при выбори "Другое". Точность может быть немного ниже.
      </p>

    </div>
  );
}

export default UploadPage;