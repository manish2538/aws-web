// Exchange rates relative to USD (1 USD = X units of currency)
// Last updated: Jan 2025 - These are approximate rates
export const EXCHANGE_RATES: Record<string, { rate: number; symbol: string; name: string }> = {
  USD: { rate: 1, symbol: '$', name: 'US Dollar' },
  EUR: { rate: 0.92, symbol: '€', name: 'Euro' },
  GBP: { rate: 0.79, symbol: '£', name: 'British Pound' },
  INR: { rate: 83.5, symbol: '₹', name: 'Indian Rupee' },
  JPY: { rate: 157.5, symbol: '¥', name: 'Japanese Yen' },
  CNY: { rate: 7.25, symbol: '¥', name: 'Chinese Yuan' },
  CAD: { rate: 1.44, symbol: 'C$', name: 'Canadian Dollar' },
  AUD: { rate: 1.58, symbol: 'A$', name: 'Australian Dollar' },
  SGD: { rate: 1.36, symbol: 'S$', name: 'Singapore Dollar' },
  AED: { rate: 3.67, symbol: 'د.إ', name: 'UAE Dirham' },
  CHF: { rate: 0.90, symbol: 'Fr', name: 'Swiss Franc' },
  HKD: { rate: 7.82, symbol: 'HK$', name: 'Hong Kong Dollar' },
  KRW: { rate: 1450, symbol: '₩', name: 'South Korean Won' },
  MXN: { rate: 20.5, symbol: 'MX$', name: 'Mexican Peso' },
  BRL: { rate: 6.1, symbol: 'R$', name: 'Brazilian Real' },
  SEK: { rate: 11.0, symbol: 'kr', name: 'Swedish Krona' },
  NOK: { rate: 11.3, symbol: 'kr', name: 'Norwegian Krone' },
  DKK: { rate: 7.1, symbol: 'kr', name: 'Danish Krone' },
  PLN: { rate: 4.1, symbol: 'zł', name: 'Polish Zloty' },
  THB: { rate: 34.5, symbol: '฿', name: 'Thai Baht' },
  IDR: { rate: 16200, symbol: 'Rp', name: 'Indonesian Rupiah' },
  MYR: { rate: 4.5, symbol: 'RM', name: 'Malaysian Ringgit' },
  PHP: { rate: 58.5, symbol: '₱', name: 'Philippine Peso' },
  VND: { rate: 25400, symbol: '₫', name: 'Vietnamese Dong' },
  ZAR: { rate: 18.8, symbol: 'R', name: 'South African Rand' },
  TRY: { rate: 35.5, symbol: '₺', name: 'Turkish Lira' },
  RUB: { rate: 101, symbol: '₽', name: 'Russian Ruble' },
  NZD: { rate: 1.78, symbol: 'NZ$', name: 'New Zealand Dollar' },
  ILS: { rate: 3.65, symbol: '₪', name: 'Israeli Shekel' },
  SAR: { rate: 3.75, symbol: '﷼', name: 'Saudi Riyal' },
};

export const CURRENCY_CODES = Object.keys(EXCHANGE_RATES);

export interface CurrencyConfig {
  code: string;
  rate: number;
  symbol: string;
  name: string;
  isCustom: boolean;
}

export function getDefaultCurrency(): CurrencyConfig {
  // Try to load from localStorage
  const saved = localStorage.getItem('aws-dashboard-currency');
  if (saved) {
    try {
      return JSON.parse(saved);
    } catch {
      // Ignore parse errors
    }
  }
  return {
    code: 'USD',
    rate: 1,
    symbol: '$',
    name: 'US Dollar',
    isCustom: false,
  };
}

export function saveCurrency(config: CurrencyConfig): void {
  localStorage.setItem('aws-dashboard-currency', JSON.stringify(config));
}

export function convertFromUSD(amountUSD: number, rate: number): number {
  return amountUSD * rate;
}

export function formatCurrency(
  amountUSD: number,
  config: CurrencyConfig,
  options?: { showOriginal?: boolean }
): string {
  const converted = convertFromUSD(amountUSD, config.rate);
  
  // For currencies with very high rates (like VND, IDR), don't show decimals
  const decimals = config.rate > 100 ? 0 : 2;
  
  const formatted = new Intl.NumberFormat(undefined, {
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals,
  }).format(converted);

  const result = `${config.symbol}${formatted}`;
  
  if (options?.showOriginal && config.code !== 'USD') {
    const usdFormatted = new Intl.NumberFormat(undefined, {
      style: 'currency',
      currency: 'USD',
      maximumFractionDigits: 2,
    }).format(amountUSD);
    return `${result} (${usdFormatted})`;
  }
  
  return result;
}

