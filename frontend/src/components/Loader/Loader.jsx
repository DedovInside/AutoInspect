import "./Loader.css";

function Loader({ label = "Загрузка...", size = "md", variant = "default" }) {
  const classes = ["loader"];
  if (size === "lg") classes.push("loader-lg");
  if (variant === "onDark") classes.push("loader-on-dark");

  return (
    <div className={classes.join(" ")} role="status" aria-live="polite">
      <span className="loader-spinner" aria-hidden="true" />
      <span className="loader-label">{label}</span>
    </div>
  );
}

export default Loader;
