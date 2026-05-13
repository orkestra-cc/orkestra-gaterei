import { describe, it, expect, vi, beforeEach } from 'vitest';
import {
  requestPasswordConfirm,
  completePasswordConfirm,
  subscribePasswordConfirm
} from './passwordConfirm';

// Mirror of stepUp.test.ts. Each test must start with no outstanding
// pending promise — there's no public reset, drain any leftover state
// by cancelling.
beforeEach(() => {
  completePasswordConfirm(false);
});

describe('passwordConfirm coordination', () => {
  it('opens a pending promise and notifies subscribers on request', () => {
    const events: boolean[] = [];
    const unsub = subscribePasswordConfirm(open => events.push(open));

    const p = requestPasswordConfirm();
    expect(events).toEqual([true]);

    completePasswordConfirm(true);
    unsub();
    return expect(p).resolves.toBe(true);
  });

  it('returns false when the user cancels', async () => {
    const p = requestPasswordConfirm();
    completePasswordConfirm(false);
    await expect(p).resolves.toBe(false);
  });

  it('joins concurrent waiters to the same outcome', async () => {
    const a = requestPasswordConfirm();
    const b = requestPasswordConfirm();
    const c = requestPasswordConfirm();

    completePasswordConfirm(true);

    const [ra, rb, rc] = await Promise.all([a, b, c]);
    expect([ra, rb, rc]).toEqual([true, true, true]);
  });

  it('notifies close event to subscribers on completion', () => {
    const listener = vi.fn();
    const unsub = subscribePasswordConfirm(listener);

    requestPasswordConfirm();
    completePasswordConfirm(true);

    expect(listener.mock.calls).toEqual([[true], [false]]);
    unsub();
  });

  it('unsubscribe stops receiving events', () => {
    const listener = vi.fn();
    const unsub = subscribePasswordConfirm(listener);
    unsub();

    requestPasswordConfirm();
    completePasswordConfirm(false);
    expect(listener).not.toHaveBeenCalled();
  });

  it('permits a fresh cycle after one completes', async () => {
    const first = requestPasswordConfirm();
    completePasswordConfirm(true);
    await expect(first).resolves.toBe(true);

    const second = requestPasswordConfirm();
    completePasswordConfirm(false);
    await expect(second).resolves.toBe(false);
  });
});
