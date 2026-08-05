import { type Item } from '@/entities/item';

export interface ItemSeed extends Item {
  botsAhead: number;
  botServiceMs: number;
}

export const ITEM_SEEDS: ItemSeed[] = [
  {
    id: 'sneakers-limited',
    title: 'Кроссовки Nike Dunk «Limited»',
    price: 24990,
    category: 'Обувь',
    sellerName: 'SneakerHub',
    stock: 5,
    queueEnabled: true,
    emoji: '👟',
    accent: '#965EEB',
    botsAhead: 3,
    botServiceMs: 4000,
  },
  {
    id: 'console-retro',
    title: 'Ретро-приставка «Neo Geo» (раритет)',
    price: 39990,
    category: 'Игры',
    sellerName: 'RetroWorld',
    stock: 2,
    queueEnabled: true,
    emoji: '🕹️',
    accent: '#00AAFF',
    botsAhead: 3,
    botServiceMs: 3500,
  },
  {
    id: 'vinyl-rare',
    title: 'Винил The Beatles, первое издание',
    price: 15990,
    category: 'Музыка',
    sellerName: 'VinylStore',
    stock: 8,
    queueEnabled: true,
    emoji: '💿',
    accent: '#04E061',
    botsAhead: 1,
    botServiceMs: 3000,
  },
  {
    id: 'sneakers-common',
    title: 'Кроссовки Adidas повседневные',
    price: 6990,
    category: 'Обувь',
    sellerName: 'SportLine',
    stock: 40,
    queueEnabled: false,
    emoji: '👟',
    accent: '#9E9EAE',
    botsAhead: 0,
    botServiceMs: 0,
  },
  {
    id: 'console-common',
    title: 'Игровая консоль, стандартная комплектация',
    price: 34990,
    category: 'Игры',
    sellerName: 'TechMarket',
    stock: 25,
    queueEnabled: false,
    emoji: '🎮',
    accent: '#9E9EAE',
    botsAhead: 0,
    botServiceMs: 0,
  },
  {
    id: 'vinyl-common',
    title: 'Винил, переиздание',
    price: 2990,
    category: 'Музыка',
    sellerName: 'MusicBox',
    stock: 60,
    queueEnabled: false,
    emoji: '🎵',
    accent: '#9E9EAE',
    botsAhead: 0,
    botServiceMs: 0,
  },
];

export function toItem(seed: ItemSeed): Item {
  return {
    id: seed.id,
    title: seed.title,
    price: seed.price,
    category: seed.category,
    sellerName: seed.sellerName,
    stock: seed.stock,
    queueEnabled: seed.queueEnabled,
    emoji: seed.emoji,
    accent: seed.accent,
  };
}
