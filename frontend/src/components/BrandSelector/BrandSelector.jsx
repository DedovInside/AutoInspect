function BrandSelector({ value, onChange }) {
  return (
    <div className="brand-selector">
      <label htmlFor="brand-select">Марка автомобиля:</label>
      <select id="brand-select" value={value} onChange={onChange}>
        <option value="">-- Выберите марку --</option>
        <option value="lada">LADA (ВАЗ)</option>
        <option value="other">Другое</option>
      </select>
    </div>
  );
}

export default BrandSelector;
