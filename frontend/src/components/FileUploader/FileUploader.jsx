import { useState } from "react";

function FileUploader({ onFileSelect }) {
  const [files, setFiles] = useState([]);

  const handleChange = (e) => {
    const selectedFiles = Array.from(e.target.files);
    setFiles(selectedFiles);
    onFileSelect(selectedFiles);
  };

  return (
    <div>
      <input
        type="file"
        accept="image/*"
        multiple
        onChange={handleChange}
      />

      {files.length > 0 && (
        <div>
          <h3>Превью:</h3>
          {files.map((file, index) => (
            <img
            key={index}
            src={URL.createObjectURL(file)}
            alt="preview"
            style={{ width: "300px", marginTop: "10px" }}
          />
          ))}
        </div>
      )}
    </div>
  );
}

export default FileUploader;
