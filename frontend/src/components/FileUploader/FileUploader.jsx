import { useState } from "react";

function FileUploader({ onFileSelect }) {
  const [file, setFile] = useState(null);

  const handleChange = (e) => {
    const selectedFile = e.target.files[0];
    setFile(selectedFile);
    onFileSelect(selectedFile);
  };

  return (
    <div>
      <input 
        type="file" 
        accept="image/*"
        onChange={handleChange}
      />

      {file && (
        <div>
          <h3>Превью:</h3>
          <img
            src={URL.createObjectURL(file)}
            alt="preview"
            style={{ width: "300px", marginTop: "10px" }}
          />
        </div>
      )}
    </div>
  );
}

export default FileUploader;