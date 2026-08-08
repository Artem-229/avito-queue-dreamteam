import { http, HttpResponse } from 'msw';

import { type SimulationParams } from '@/features/simulator/types';

import { ApiFault, queueEngine } from './queue-engine';
import { runSimulation } from './simulator';

const TERMINAL = ['PURCHASED', 'EXPIRED', 'SOLD_OUT', 'LEFT'];

function userIdFrom(request: Request): string {
  return request.headers.get('X-User-Id') ?? 'user-1';
}

function faultResponse(error: unknown) {
  if (error instanceof ApiFault) {
    return HttpResponse.json(
      { code: error.code, message: error.message },
      { status: error.status },
    );
  }
  return HttpResponse.json(
    { code: 'INTERNAL', message: 'Внутренняя ошибка мока' },
    { status: 500 },
  );
}

export const handlers = [
  http.get('/api/items', () => HttpResponse.json(queueEngine.listItems())),

  http.get('/api/metrics', () => HttpResponse.json(queueEngine.getMetrics())),

  http.get('/api/items/:id/similar', ({ params }) => {
    try {
      return HttpResponse.json(queueEngine.getSimilarItems(String(params.id)));
    } catch (error) {
      return faultResponse(error);
    }
  }),

  http.get('/api/items/:id', ({ params }) => {
    try {
      return HttpResponse.json(queueEngine.getItem(String(params.id)));
    } catch (error) {
      return faultResponse(error);
    }
  }),

  http.post('/api/queue/join', async ({ request }) => {
    try {
      const body = (await request.json()) as { itemId?: string };
      const entry = queueEngine.join(userIdFrom(request), String(body.itemId));
      return HttpResponse.json(entry);
    } catch (error) {
      return faultResponse(error);
    }
  }),

  http.get('/api/queue/entry', ({ request }) => {
    try {
      const itemId = new URL(request.url).searchParams.get('itemId') ?? '';
      const entry = queueEngine.getEntryByItem(userIdFrom(request), itemId);
      if (!entry) {
        return HttpResponse.json(
          { code: 'NOT_FOUND', message: 'Запись не найдена' },
          { status: 404 },
        );
      }
      return HttpResponse.json(entry);
    } catch (error) {
      return faultResponse(error);
    }
  }),

  http.get('/api/queue/:entryId/events', ({ params }) => {
    const entryId = String(params.entryId);
    const encoder = new TextEncoder();
    let timer = 0;

    const stream = new ReadableStream({
      start(controller) {
        const push = () => {
          try {
            const entry = queueEngine.getEntry(entryId);
            controller.enqueue(
              encoder.encode(`data: ${JSON.stringify(entry)}\n\n`),
            );
            if (TERMINAL.includes(entry.status)) {
              window.clearInterval(timer);
              controller.close();
            }
          } catch {
            window.clearInterval(timer);
            controller.close();
          }
        };

        push();
        timer = window.setInterval(push, 1000);
      },
      cancel() {
        window.clearInterval(timer);
      },
    });

    return new HttpResponse(stream, {
      headers: {
        'Content-Type': 'text/event-stream',
        'Cache-Control': 'no-cache',
        Connection: 'keep-alive',
      },
    });
  }),

  http.get('/api/queue/:entryId/eta', ({ params }) => {
    try {
      return HttpResponse.json(queueEngine.getEta(String(params.entryId)));
    } catch (error) {
      return faultResponse(error);
    }
  }),

  http.get('/api/queue/:entryId', ({ params }) => {
    try {
      return HttpResponse.json(queueEngine.getEntry(String(params.entryId)));
    } catch (error) {
      return faultResponse(error);
    }
  }),

  http.post('/api/queue/:entryId/leave', ({ params }) => {
    try {
      queueEngine.leave(String(params.entryId));
      return new HttpResponse(null, { status: 204 });
    } catch (error) {
      return faultResponse(error);
    }
  }),

  http.post('/api/sim/run', async ({ request }) => {
    try {
      const body = (await request.json()) as SimulationParams;
      return HttpResponse.json(runSimulation(body));
    } catch (error) {
      return faultResponse(error);
    }
  }),

  http.post('/api/checkout', async ({ request }) => {
    try {
      const body = (await request.json()) as { entryId?: string };
      const order = queueEngine.checkout(
        userIdFrom(request),
        String(body.entryId),
      );
      return HttpResponse.json(order);
    } catch (error) {
      return faultResponse(error);
    }
  }),
];
