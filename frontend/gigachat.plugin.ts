import { randomUUID } from 'node:crypto';
import { request as httpsRequest } from 'node:https';
import { type IncomingMessage, type ServerResponse } from 'node:http';

import { type Plugin } from 'vite';

const OAUTH_URL = 'https://ngw.devices.sberbank.ru:9443/api/v2/oauth';
const CHAT_URL = 'https://gigachat.devices.sberbank.ru/api/v1/chat/completions';

interface TokenResponse {
  access_token: string;
  expires_at: number;
}

interface ChatResponse {
  choices?: { message?: { content?: string } }[];
}

interface RecommendationBody {
  soldOutTitle?: string;
  candidates?: { title: string; price: number }[];
}

let cachedToken: { token: string; expiresAt: number } | null = null;

function postJson(
  url: string,
  headers: Record<string, string>,
  body: string,
): Promise<{ status: number; text: string }> {
  return new Promise((resolve, reject) => {
    const parsed = new URL(url);
    const req = httpsRequest(
      {
        hostname: parsed.hostname,
        port: parsed.port ? Number(parsed.port) : 443,
        path: parsed.pathname,
        method: 'POST',
        headers,
        rejectUnauthorized: false,
      },
      (res) => {
        let data = '';
        res.on('data', (chunk) => (data += String(chunk)));
        res.on('end', () => {
          resolve({ status: res.statusCode ?? 0, text: data });
        });
      },
    );
    req.on('error', reject);
    req.write(body);
    req.end();
  });
}

async function getAccessToken(authKey: string): Promise<string> {
  if (cachedToken && cachedToken.expiresAt > Date.now() + 60_000) {
    return cachedToken.token;
  }

  const { text } = await postJson(
    OAUTH_URL,
    {
      Authorization: `Basic ${authKey}`,
      RqUID: randomUUID(),
      'Content-Type': 'application/x-www-form-urlencoded',
      Accept: 'application/json',
    },
    'scope=GIGACHAT_API_PERS',
  );

  const json = JSON.parse(text) as TokenResponse;
  cachedToken = {
    token: json.access_token,
    expiresAt: json.expires_at || Date.now() + 25 * 60_000,
  };
  return cachedToken.token;
}

async function askGigaChat(
  authKey: string,
  body: RecommendationBody,
): Promise<string | null> {
  const token = await getAccessToken(authKey);
  const list = (body.candidates ?? [])
    .map((item) => `- ${item.title} (${String(item.price)} ₽)`)
    .join('\n');

  const payload = JSON.stringify({
    model: 'GigaChat',
    temperature: 0.6,
    max_tokens: 160,
    messages: [
      {
        role: 'system',
        content:
          'Ты — дружелюбный помощник маркетплейса Авито. Отвечай по-русски, 1–2 коротких предложения, без списков и приветствий.',
      },
      {
        role: 'user',
        content: `Пользователь хотел купить «${body.soldOutTitle ?? 'товар'}», но он закончился. Мягко предложи не уходить и посмотреть похожие лоты ниже:\n${list}`,
      },
    ],
  });

  const { text } = await postJson(
    CHAT_URL,
    {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
      Accept: 'application/json',
    },
    payload,
  );

  const json = JSON.parse(text) as ChatResponse;
  return json.choices?.[0]?.message?.content ?? null;
}

function readBody(req: IncomingMessage): Promise<string> {
  return new Promise((resolve) => {
    let raw = '';
    req.on('data', (chunk) => (raw += String(chunk)));
    req.on('end', () => {
      resolve(raw);
    });
  });
}

export function gigachatPlugin(authKey: string | undefined): Plugin {
  return {
    name: 'gigachat-dev-proxy',
    configureServer(server) {
      server.middlewares.use(
        '/ai/recommendation',
        (req: IncomingMessage, res: ServerResponse, next: () => void) => {
          if (req.method !== 'POST') {
            next();
            return;
          }

          res.setHeader('Content-Type', 'application/json');

          void (async () => {
            if (!authKey) {
              res.end(JSON.stringify({ text: null }));
              return;
            }
            try {
              const body = JSON.parse(await readBody(req)) as RecommendationBody;
              const text = await askGigaChat(authKey, body);
              res.end(JSON.stringify({ text }));
            } catch (error) {
              console.error('[gigachat] ошибка запроса:', error);
              res.statusCode = 200;
              res.end(JSON.stringify({ text: null }));
            }
          })();
        },
      );
    },
  };
}
