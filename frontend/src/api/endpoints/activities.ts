import api from '../axios';
import type {
  ActivityDetail,
  ActivityCategory,
  ActivityListItem,
  MerchantActivityInput,
  MerchantMutationResult,
  ActivitySearchParams,
  ActivitySearchResult,
  ActivityStockSnapshot,
  ActivitySort,
} from '../../types';

interface BackendListPayload<T> {
  code: number;
  message: string;
  data: {
    list: T[];
    total: number;
    page: number;
    page_size: number;
  };
}

interface BackendActivity {
  id?: number;
  activity_id?: number;
  title: string;
  description?: string;
  cover_url?: string | null;
  location: string;
  category: ActivityCategory;
  tags?: string[] | string | null;
  max_capacity: number;
  price: number;
  enroll_open_at: string;
  enroll_close_at: string;
  activity_at: string;
  status: ActivityListItem['status'];
  enroll_count?: number;
  view_count?: number;
  stock_remaining?: number;
  latitude?: number | null;
  longitude?: number | null;
  created_by?: {
    user_id?: number;
    username?: string;
  };
}

interface BackendSinglePayload<T> {
  code: number;
  message: string;
  data: T;
}

interface BackendMerchantMutationPayload {
  activity_id: number;
  status: ActivityListItem['status'];
  stock_in_cache?: number;
}

function parseTags(value: BackendActivity['tags']): string[] {
  if (Array.isArray(value)) {
    return value.filter((item): item is string => typeof item === 'string');
  }

  if (typeof value === 'string' && value.trim()) {
    try {
      const parsed = JSON.parse(value);
      if (Array.isArray(parsed)) {
        return parsed.filter((item): item is string => typeof item === 'string');
      }
    } catch {
      return value
        .split(',')
        .map((item) => item.trim())
        .filter(Boolean);
    }
  }

  return [];
}

function normalizeActivity(activity: BackendActivity): ActivityListItem {
  const enrollCount = activity.enroll_count ?? 0;
  const maxCapacity = activity.max_capacity ?? 0;

  return {
    id: activity.id ?? activity.activity_id ?? 0,
    title: activity.title,
    description: activity.description ?? '',
    coverUrl: activity.cover_url ?? null,
    location: activity.location,
    category: activity.category,
    tags: parseTags(activity.tags),
    maxCapacity,
    price: activity.price,
    enrollOpenAt: activity.enroll_open_at,
    enrollCloseAt: activity.enroll_close_at,
    activityAt: activity.activity_at,
    status: activity.status,
    enrollCount,
    viewCount: activity.view_count ?? 0,
    stockRemaining:
      activity.stock_remaining ?? Math.max(maxCapacity - enrollCount, 0),
  };
}

function normalizeActivityDetail(activity: BackendActivity): ActivityDetail {
  const base = normalizeActivity(activity);

  return {
    ...base,
    latitude: activity.latitude ?? null,
    longitude: activity.longitude ?? null,
    organizerName: activity.created_by?.username,
  };
}

function toBackendActivityInput(payload: MerchantActivityInput) {
  return {
    title: payload.title,
    description: payload.description,
    cover_url: payload.coverUrl || undefined,
    location: payload.location,
    category: payload.category,
    max_capacity: payload.maxCapacity,
    price: payload.price,
    enroll_open_at: payload.enrollOpenAt,
    enroll_close_at: payload.enrollCloseAt,
    activity_at: payload.activityAt,
  };
}

function containsMatch(source: string, keyword: string) {
  return source.toLowerCase().includes(keyword.toLowerCase());
}

function computeRelevance(item: ActivityListItem, keyword: string, region: string, artist: string) {
  let score = 0;
  const normalizedKeyword = keyword.trim().toLowerCase();
  const normalizedRegion = region.trim().toLowerCase();
  const normalizedArtist = artist.trim().toLowerCase();
  const title = item.title.toLowerCase();
  const description = item.description.toLowerCase();
  const location = item.location.toLowerCase();
  const tags = item.tags.join(' ').toLowerCase();

  if (normalizedKeyword) {
    if (title === normalizedKeyword) {
      score += 120;
    } else if (title.includes(normalizedKeyword)) {
      score += 90;
    } else if (description.includes(normalizedKeyword)) {
      score += 30;
    }
  }

  if (normalizedArtist) {
    if (tags.includes(normalizedArtist) || title.includes(normalizedArtist)) {
      score += 70;
    }
  }

  if (normalizedRegion && location.includes(normalizedRegion)) {
    score += 50;
  }

  score += item.enrollCount / 100;
  score += item.viewCount / 500;

  return score;
}

function sortActivities(
  list: ActivityListItem[],
  sort: ActivitySort,
  keyword: string,
  region: string,
  artist: string,
) {
  return [...list].sort((left, right) => {
    if (sort === 'relevance') {
      return (
        computeRelevance(right, keyword, region, artist) -
        computeRelevance(left, keyword, region, artist)
      );
    }

    if (sort === 'hot') {
      return right.enrollCount - left.enrollCount || right.viewCount - left.viewCount;
    }

    if (sort === 'soon') {
      return (
        new Date(left.enrollOpenAt).getTime() - new Date(right.enrollOpenAt).getTime()
      );
    }

    return new Date(right.activityAt).getTime() - new Date(left.activityAt).getTime();
  });
}

function filterActivities(list: ActivityListItem[], params: ActivitySearchParams) {
  return list.filter((item) => {
    const matchesCategory =
      params.category === 'ALL' || item.category === params.category;
    const matchesKeyword =
      !params.keyword ||
      containsMatch(item.title, params.keyword) ||
      containsMatch(item.description, params.keyword);
    const matchesRegion =
      params.region === 'ALL' || !params.region || containsMatch(item.location, params.region);
    const matchesArtist =
      !params.artist ||
      containsMatch(item.title, params.artist) ||
      item.tags.some((tag) => containsMatch(tag, params.artist));

    return matchesCategory && matchesKeyword && matchesRegion && matchesArtist;
  });
}

export async function listActivities(params: ActivitySearchParams): Promise<ActivitySearchResult> {
  const needsClientSideRefine =
    params.region !== 'ALL' || Boolean(params.artist.trim()) || params.sort === 'relevance';

  const pageSize = needsClientSideRefine ? 80 : params.pageSize;
  const page = needsClientSideRefine ? 1 : params.page;

  const response = await api.get<BackendListPayload<BackendActivity>>('/activities', {
    params: {
      category: params.category === 'ALL' ? undefined : params.category,
      keyword: params.keyword || undefined,
      region: params.region === 'ALL' ? undefined : params.region,
      artist: params.artist || undefined,
      sort: params.sort,
      page,
      page_size: pageSize,
    },
  });

  const normalized = response.data.data.list.map(normalizeActivity);

  if (!needsClientSideRefine) {
    return {
      list: normalized,
      total: response.data.data.total,
      page: response.data.data.page,
      pageSize: response.data.data.page_size,
    };
  }

  const filtered = sortActivities(
    filterActivities(normalized, params),
    params.sort,
    params.keyword,
    params.region === 'ALL' ? '' : params.region,
    params.artist,
  );

  const start = (params.page - 1) * params.pageSize;
  const paginated = filtered.slice(start, start + params.pageSize);

  return {
    list: paginated,
    total: filtered.length,
    page: params.page,
    pageSize: params.pageSize,
  };
}

export async function getActivityDetail(id: number): Promise<ActivityDetail> {
  const response = await api.get<BackendSinglePayload<BackendActivity>>(`/activities/${id}`);
  return normalizeActivityDetail(response.data.data);
}

export async function getActivityStock(id: number): Promise<ActivityStockSnapshot> {
  const response = await api.get<
    BackendSinglePayload<{
      activity_id: number;
      stock_remaining: number;
      max_capacity: number;
      last_updated?: string;
    }>
  >(`/activities/${id}/stock`);

  return {
    activityId: response.data.data.activity_id,
    stockRemaining: response.data.data.stock_remaining,
    maxCapacity: response.data.data.max_capacity,
    lastUpdated: response.data.data.last_updated,
  };
}

export async function listMerchantActivities(): Promise<ActivityListItem[]> {
  const response = await api.get<BackendSinglePayload<BackendActivity[]>>('/activities/merchant');
  return response.data.data.map(normalizeActivity);
}

function normalizeMerchantMutationResult(
  payload: BackendMerchantMutationPayload,
  message: string,
): MerchantMutationResult {
  return {
    activityId: payload.activity_id,
    status: payload.status,
    stockInCache: payload.stock_in_cache,
    message,
  };
}

export async function createMerchantActivity(
  payload: MerchantActivityInput,
): Promise<MerchantMutationResult> {
  const response = await api.post<BackendSinglePayload<BackendMerchantMutationPayload>>(
    '/activities',
    toBackendActivityInput(payload),
  );
  return normalizeMerchantMutationResult(response.data.data, response.data.message);
}

export async function updateMerchantActivity(
  id: number,
  payload: MerchantActivityInput,
): Promise<MerchantMutationResult> {
  const response = await api.put<BackendSinglePayload<BackendMerchantMutationPayload>>(
    `/activities/${id}`,
    toBackendActivityInput(payload),
  );
  return normalizeMerchantMutationResult(response.data.data, response.data.message);
}

export async function preheatMerchantActivity(id: number): Promise<MerchantMutationResult> {
  const response = await api.put<BackendSinglePayload<BackendMerchantMutationPayload>>(
    `/activities/${id}/preheat`,
  );
  return normalizeMerchantMutationResult(response.data.data, response.data.message);
}

export async function publishMerchantActivity(id: number): Promise<MerchantMutationResult> {
  const response = await api.put<BackendSinglePayload<BackendMerchantMutationPayload>>(
    `/activities/${id}/publish`,
  );
  return normalizeMerchantMutationResult(response.data.data, response.data.message);
}
