import "./Loader.css";

function Loader({ label = "Загрузка..." }) {
  return (
    <div className="loader" role="status" aria-live="polite">
      <span className="loader-spinner" aria-hidden="true" />
      <span className="loader-label">{label}</span>
    </div>
  );
}

export default Loader;
