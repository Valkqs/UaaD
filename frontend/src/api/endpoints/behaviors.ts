import api, { AUTH_REDIRECT_BYPASS_HEADER } from '../axios';
import { getStoredAuthSession } from '../../utils/auth';

export type BehaviorType = 'VIEW' | 'COLLECT' | 'SHARE' | 'CLICK' | 'SEARCH';

export interface BehaviorEventPayload {
  activityId: number;
  behaviorType: BehaviorType;
  detail?: Record<string, unknown>;
  timestamp?: number;
}

interface BackendPayload<T> {
  code: number;
  message: string;
  data: T;
}

interface BehaviorBatchRequest {
  behaviors: Array<{
    activity_id: number;
    behavior_type: BehaviorType;
    detail?: Record<string, unknown>;
    timestamp?: number;
  }>;
}

interface BehaviorAcceptedResponse {
  accepted: boolean;
  count?: number;
}

const DEFAULT_TIMEOUT_MS = 1200;
const BATCH_SIZE = 10;
const FLUSH_INTERVAL_MS = 60_000;
const BEHAVIOR_REQUEST_HEADERS = {
  [AUTH_REDIRECT_BYPASS_HEADER]: '1',
} as const;

let behaviorQueue: BehaviorEventPayload[] = [];
let flushTimer: number | null = null;

function hasBehaviorSession() {
  return Boolean(getStoredAuthSession()?.token);
}

function toBackendPayload(payload: BehaviorEventPayload) {
  return {
    activity_id: payload.activityId,
    behavior_type: payload.behaviorType,
    detail: payload.detail,
    timestamp: payload.timestamp ?? Date.now(),
  };
}

function clearFlushTimer() {
  if (flushTimer !== null) {
    window.clearTimeout(flushTimer);
    flushTimer = null;
  }
}

async function postSingleBehavior(payload: BehaviorEventPayload, timeoutMs = DEFAULT_TIMEOUT_MS) {
  if (!hasBehaviorSession()) {
    return;
  }

  await api.post<BackendPayload<BehaviorAcceptedResponse>>('/behaviors', toBackendPayload(payload), {
    timeout: timeoutMs,
    headers: BEHAVIOR_REQUEST_HEADERS,
  });
}

async function postBatchBehaviors(payloads: BehaviorEventPayload[], timeoutMs = DEFAULT_TIMEOUT_MS) {
  if (!hasBehaviorSession()) {
    return;
  }

  const request: BehaviorBatchRequest = {
    behaviors: payloads.map(toBackendPayload),
  };

  await api.post<BackendPayload<BehaviorAcceptedResponse>>('/behaviors/batch', request, {
    timeout: timeoutMs,
    headers: BEHAVIOR_REQUEST_HEADERS,
  });
}

async function flushQueueInternal(timeoutMs = DEFAULT_TIMEOUT_MS) {
  clearFlushTimer();
  if (behaviorQueue.length === 0) {
    return;
  }

  const queueSnapshot = [...behaviorQueue];
  behaviorQueue = [];

  try {
    await postBatchBehaviors(queueSnapshot, timeoutMs);
  } catch {
    // Keep behavior reporting fire-and-forget. Failures should not block user flows.
  }
}

function scheduleFlush() {
  if (flushTimer !== null) {
    return;
  }

  flushTimer = window.setTimeout(() => {
    void flushQueueInternal();
  }, FLUSH_INTERVAL_MS);
}

export async function flushBehaviorQueue() {
  await flushQueueInternal();
}

export function trackBehavior(
  payload: BehaviorEventPayload,
  options: { immediate?: boolean; timeoutMs?: number } = {},
) {
  if (!payload.activityId || payload.activityId <= 0 || !hasBehaviorSession()) {
    return;
  }

  const timeoutMs = options.timeoutMs ?? DEFAULT_TIMEOUT_MS;

  if (options.immediate) {
    void postSingleBehavior(payload, timeoutMs).catch(() => {
      // Keep behavior reporting fire-and-forget. Failures should not block user flows.
    });
    return;
  }

  behaviorQueue.push(payload);
  if (behaviorQueue.length >= BATCH_SIZE) {
    void flushQueueInternal(timeoutMs);
    return;
  }

  scheduleFlush();
}
