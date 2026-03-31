import { useState } from "react";
import { useNavigate } from "react-router-dom";

function UploadPage() {
  const [file, setFile] = useState(null);
  const navigate = useNavigate();

  const handleFileChange = (e) => {
    const selectedFile = e.target.files[0];
    setFile(selectedFile);
    console.log(selectedFile);
  };

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
  
     <input
        type="file"
        accept="image/*"
        onChange={handleFileChange}
     />

     {file && (
        <div>
            <h3>Предпросмотр:</h3>
            <img 
                src={URL.createObjectURL(file)} 
                alt="preview" 
                style={{ width: "300px", marginTop: "10px" }}
            />
        </div>
    )}

        <button onClick={handleSubmit}>
            Отправить
        </button>
    </div>
  );
}

export default UploadPage;