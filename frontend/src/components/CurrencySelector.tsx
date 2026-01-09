import { useState, useRef, useEffect } from 'react';
import { useCurrency } from '../context/CurrencyContext';
import { EXCHANGE_RATES } from '../utils/currency';

// Flatten currencies for search
const ALL_CURRENCIES = Object.entries(EXCHANGE_RATES).map(([code, data]) => ({
  code,
  ...data,
}));

function CurrencySelector() {
  const { currency, setCustomRate } = useCurrency();
  const [isOpen, setIsOpen] = useState(false);
  const [search, setSearch] = useState('');
  const [editRate, setEditRate] = useState(currency.rate.toString());
  const [isEditingRate, setIsEditingRate] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  // Filter currencies based on search
  const filteredCurrencies = ALL_CURRENCIES.filter(
    (c) =>
      c.code.toLowerCase().includes(search.toLowerCase()) ||
      c.name.toLowerCase().includes(search.toLowerCase())
  );

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setIsOpen(false);
        setSearch('');
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // Focus search input when dropdown opens
  useEffect(() => {
    if (isOpen && inputRef.current) {
      inputRef.current.focus();
    }
  }, [isOpen]);

  // Update editRate when currency changes
  useEffect(() => {
    setEditRate(currency.rate.toString());
  }, [currency.rate]);

  const handleSelectCurrency = (code: string) => {
    const preset = EXCHANGE_RATES[code];
    if (preset) {
      setCustomRate(code, preset.rate, preset.symbol);
      setEditRate(preset.rate.toString());
    }
    setIsOpen(false);
    setSearch('');
  };

  const handleRateChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setEditRate(e.target.value);
  };

  const handleRateBlur = () => {
    const newRate = parseFloat(editRate);
    if (!isNaN(newRate) && newRate > 0 && newRate !== currency.rate) {
      setCustomRate(currency.code, newRate, currency.symbol);
    } else {
      setEditRate(currency.rate.toString());
    }
    setIsEditingRate(false);
  };

  const handleRateKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleRateBlur();
    } else if (e.key === 'Escape') {
      setEditRate(currency.rate.toString());
      setIsEditingRate(false);
    }
  };

  return (
    <div className="currency-selector" ref={dropdownRef}>
      {/* Main Button */}
      <button
        type="button"
        onClick={() => setIsOpen(!isOpen)}
        className="currency-button"
      >
        <span className="currency-symbol">{currency.symbol}</span>
        <span className="currency-code">{currency.code}</span>
        <span className="currency-arrow">{isOpen ? '▲' : '▼'}</span>
      </button>

      {/* Rate Display/Edit */}
      <div className="currency-rate">
        <span className="currency-rate-label">1 USD =</span>
        {isEditingRate ? (
          <input
            type="number"
            value={editRate}
            onChange={handleRateChange}
            onBlur={handleRateBlur}
            onKeyDown={handleRateKeyDown}
            className="currency-rate-input"
            step="0.01"
            min="0.0001"
            autoFocus
          />
        ) : (
          <button
            type="button"
            onClick={() => setIsEditingRate(true)}
            className="currency-rate-value"
            title="Click to edit rate"
          >
            {currency.rate} {currency.code}
          </button>
        )}
      </div>

      {/* Dropdown */}
      {isOpen && (
        <div className="currency-dropdown">
          {/* Search Input */}
          <div className="currency-search">
            <input
              ref={inputRef}
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search currency (e.g. INR, Euro)"
              className="currency-search-input"
            />
          </div>

          {/* Currency List */}
          <div className="currency-list">
            {filteredCurrencies.length === 0 ? (
              <div className="currency-empty">No currencies found</div>
            ) : (
              filteredCurrencies.map((c) => (
                <button
                  key={c.code}
                  type="button"
                  onClick={() => handleSelectCurrency(c.code)}
                  className={`currency-item ${c.code === currency.code ? 'active' : ''}`}
                >
                  <span className="currency-item-symbol">{c.symbol}</span>
                  <span className="currency-item-details">
                    <span className="currency-item-code">{c.code}</span>
                    <span className="currency-item-name">{c.name}</span>
                  </span>
                  <span className="currency-item-rate">1 USD = {c.rate}</span>
                </button>
              ))
            )}
          </div>
        </div>
      )}
    </div>
  );
}

export default CurrencySelector;
