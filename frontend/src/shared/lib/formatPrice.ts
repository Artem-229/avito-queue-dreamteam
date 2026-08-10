const formatter = new Intl.NumberFormat('ru-RU', {
  style: 'currency',
  currency: 'RUB',
  maximumFractionDigits: 0,
});

/**
 * Деньги приходят с бэкенда целым числом копеек и в таком же виде живут в
 * модели: дробные рубли во float дают накопительную погрешность при любой
 * арифметике на клиенте. Деление на 100 происходит только здесь, на границе
 * отображения.
 */
export function formatPrice(kopecks: number): string {
  return formatter.format(kopecks / 100);
}
