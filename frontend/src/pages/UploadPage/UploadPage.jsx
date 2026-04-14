import { useState } from "react";
import { useNavigate } from "react-router-dom";
import Button from "../../components/Button/Button";
import FileUploader from "../../components/FileUploader/FileUploader";

function UploadPage() {
  const [file, setFile] = useState(null);
  const navigate = useNavigate();

  const handleSubmit = () => {
    if (!file) {
      alert("Выберите файл");
      return;
    }
  
    navigate("/result", { state: { file } });
  };

  return (
    <div>
      <h2>Загрузка изображений</h2>
  
      <FileUploader onFileSelect={setFile} />

        <Button onClick={handleSubmit}>
            Отправить
        </Button>
    </div>
  );
}

export default UploadPage;