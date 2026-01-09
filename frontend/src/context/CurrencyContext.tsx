import { createContext, useContext, useState, useCallback, ReactNode } from 'react';
import {
  CurrencyConfig,
  getDefaultCurrency,
  saveCurrency,
  formatCurrency as formatCurrencyUtil,
  EXCHANGE_RATES,
} from '../utils/currency';

interface CurrencyContextValue {
  currency: CurrencyConfig;
  setCurrency: (config: CurrencyConfig) => void;
  selectCurrency: (code: string) => void;
  setCustomRate: (code: string, rate: number, symbol?: string) => void;
  formatCost: (amountUSD: number, showOriginal?: boolean) => string;
}

const CurrencyContext = createContext<CurrencyContextValue | null>(null);

export function CurrencyProvider({ children }: { children: ReactNode }) {
  const [currency, setCurrencyState] = useState<CurrencyConfig>(getDefaultCurrency);

  const setCurrency = useCallback((config: CurrencyConfig) => {
    setCurrencyState(config);
    saveCurrency(config);
  }, []);

  const selectCurrency = useCallback((code: string) => {
    const preset = EXCHANGE_RATES[code];
    if (preset) {
      setCurrency({
        code,
        rate: preset.rate,
        symbol: preset.symbol,
        name: preset.name,
        isCustom: false,
      });
    }
  }, [setCurrency]);

  const setCustomRate = useCallback((code: string, rate: number, symbol?: string) => {
    const preset = EXCHANGE_RATES[code];
    setCurrency({
      code,
      rate,
      symbol: symbol || preset?.symbol || code,
      name: preset?.name || code,
      isCustom: true,
    });
  }, [setCurrency]);

  const formatCost = useCallback(
    (amountUSD: number, showOriginal?: boolean) => {
      return formatCurrencyUtil(amountUSD, currency, { showOriginal });
    },
    [currency]
  );

  return (
    <CurrencyContext.Provider
      value={{
        currency,
        setCurrency,
        selectCurrency,
        setCustomRate,
        formatCost,
      }}
    >
      {children}
    </CurrencyContext.Provider>
  );
}

export function useCurrency() {
  const context = useContext(CurrencyContext);
  if (!context) {
    throw new Error('useCurrency must be used within a CurrencyProvider');
  }
  return context;
}

