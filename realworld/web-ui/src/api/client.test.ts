import { beforeAll, afterAll, afterEach, describe, expect, it } from 'vitest';
import { http, HttpResponse } from 'msw';
import { server } from '../test/server';
import { createApiClient } from './client';
import { documentFixture } from '../test/fixtures';

const api = createApiClient({ basePath: '/api/v1' });

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

describe('document API client', () => {
  it('sends requests without credentials and reads document records', async () => {
    let authorization: string | null = 'unexpected';
    server.use(http.get('/api/v1/documents', ({ request }) => {
      authorization = request.headers.get('Authorization');
      return HttpResponse.json({ items: [documentFixture()], total: 1, offset: 0, limit: 20 });
    }));
    const result = await api.list({ limit: 20, offset: 0 });
    expect(authorization).toBeNull();
    expect(result.ok).toBe(true);
  });

  it('sends multipart file uploads without overriding the boundary', async () => {
    let receivedTitle = '';
    let contentType = '';
    server.use(http.post('/api/v1/documents/upload', async ({ request }) => {
      receivedTitle = request.method;
      contentType = request.headers.get('Content-Type') ?? '';
      return HttpResponse.json(documentFixture(), { status: 201 });
    }));
    const result = await api.upload({ title: 'Vendor contract', file: new File(['%PDF-1.4'], 'contract.pdf', { type: 'application/pdf' }) });
    expect(receivedTitle).toBe('POST');
    expect(contentType).toContain('multipart/form-data; boundary=');
    expect(result.ok).toBe(true);
  });

  it('rejects unknown statuses and returns safe HTTP errors with request IDs', async () => {
    server.use(http.get('/api/v1/documents/:id', () => HttpResponse.json({ ...documentFixture(), status: 'pretend_signed' })));
    const malformed = await api.get({ id: documentFixture().id });
    expect(malformed).toMatchObject({ ok: false, error: { kind: 'invalid-response' } });
    server.use(http.get('/api/v1/documents/:id', () => HttpResponse.json({ error: 'unauthorized', request_id: 'req-1' }, { status: 401 })));
    const unauthorized = await api.get({ id: documentFixture().id });
    expect(unauthorized).toMatchObject({ ok: false, error: { kind: 'http', status: 401, requestId: 'req-1' } });
  });

  it('normalizes proxy and abort failures', async () => {
    server.use(http.get('/api/v1/documents/:id', () => HttpResponse.text('<html>proxy</html>', { status: 502 })));
    expect(await api.get({ id: 'a' })).toMatchObject({ ok: false, error: { kind: 'http', status: 502 } });
    const controller = new AbortController();
    controller.abort();
    expect(await api.get({ id: 'a', signal: controller.signal })).toEqual({ ok: false, error: { kind: 'aborted' } });
  });
});
