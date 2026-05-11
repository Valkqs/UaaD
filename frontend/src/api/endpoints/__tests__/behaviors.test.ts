import MockAdapter from 'axios-mock-adapter';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import api from '../../axios';
import { flushBehaviorQueue, trackBehavior } from '../behaviors';

describe('behavior tracking endpoints', () => {
  let mock: MockAdapter;

  beforeEach(() => {
    mock = new MockAdapter(api);
    localStorage.setItem(
      'auth_session',
      JSON.stringify({
        token: 'mock-token-user-1001',
        userId: 1001,
        role: 'USER',
        username: 'demo',
      }),
    );
    localStorage.setItem('token', 'mock-token-user-1001');
  });

  afterEach(async () => {
    await flushBehaviorQueue().catch(() => undefined);
    mock.restore();
    localStorage.clear();
  });

  it('posts a batched behavior payload when the queue reaches 10 items', async () => {
    let capturedBody: { behaviors: Array<{ activity_id: number; behavior_type: string }> } | null =
      null;

    mock.onPost('/behaviors/batch').reply((config) => {
      capturedBody = JSON.parse(config.data as string) as typeof capturedBody;
      return [
        200,
        {
          code: 0,
          message: 'ok',
          data: { accepted: true, count: 10 },
        },
      ];
    });

    for (let index = 0; index < 10; index += 1) {
      trackBehavior({
        activityId: index + 1,
        behaviorType: 'CLICK',
        detail: { source: 'test' },
      });
    }

    await new Promise((resolve) => {
      window.setTimeout(resolve, 0);
    });

    expect(capturedBody?.behaviors).toHaveLength(10);
    expect(capturedBody?.behaviors[0]).toMatchObject({
      activity_id: 1,
      behavior_type: 'CLICK',
    });
  });

  it('skips behavior writes without an auth session', async () => {
    localStorage.clear();

    trackBehavior({
      activityId: 99,
      behaviorType: 'VIEW',
      detail: { source: 'test' },
    });

    await Promise.resolve();

    expect(mock.history.post).toHaveLength(0);
  });
});
