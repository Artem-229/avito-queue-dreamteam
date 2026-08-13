import { type PurchaseChance } from '@/shared/lib/chance';

export type { PurchaseChance };

export interface Item {
  id: string;
  title: string;
  /** Цена в копейках, как её отдаёт бэкенд. Форматирование — formatPrice. */
  priceKopecks: number;
  category: string;
  sellerName: string;
  /** Весь тираж товара. */
  totalStock: number;
  /** Свободно прямо сейчас: тираж минус выкупленное и минус удержанные права. */
  available: number;
  /** Выкуплен полностью — в отличие от available это состояние необратимо. */
  soldOut: boolean;
  /** Сколько живёт выданное право на покупку — для шкалы таймера. */
  holdTtlSeconds: number;
  queueEnabled: boolean;
  /** Шанс купить, если встать в очередь сейчас. Есть только на карточке одного
   * товара (GET /catalog/:id); в списке каталога — null. */
  chance: PurchaseChance | null;
  emoji: string;
  accent: string;
}
