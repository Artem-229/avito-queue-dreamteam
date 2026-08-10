export interface SimulationParams {
  /** Товар, на котором проводим нагрузку. */
  itemId: string;
  /** Сколько синтетических пользователей одновременно встают в очередь. */
  count: number;
}

/**
 * Результат реального прогона на бэкенде: N параллельных вызовов настоящего
 * Entry. joined — сколько встали в очередь, failed — сколько получили доменную
 * ошибку (например «уже в очереди»). Никаких выдуманных распределений: цифры
 * приходят от системы, которую и надо проверить.
 */
export interface SimulationResult {
  itemId: string;
  requested: number;
  joined: number;
  failed: number;
}
