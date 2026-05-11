import { Search } from 'lucide-react';
import { useEffect, useMemo, useRef, useState } from 'react';
import { useOutletContext, useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { getHotRecommendations, listActivities, trackBehavior } from '../api/endpoints';
import { EmptyState } from '../components/public/EmptyState';
import { LoadingCards } from '../components/public/LoadingCards';
import { Pagination } from '../components/public/Pagination';
import { RecommendationList } from '../components/public/RecommendationList';
import { SearchResultItem } from '../components/public/SearchResultItem';
import {
  CATEGORY_OPTIONS,
  CITY_OPTIONS,
  DEFAULT_ACTIVITY_SEARCH,
  SORT_OPTIONS,
} from '../constants/public';
import type { ActivityListItem, ActivitySearchParams, RecommendationSectionItem } from '../types';
import type { PublicLayoutContext } from '../layouts/PublicLayout';

function readParams(searchParams: URLSearchParams, preferredCity: string): ActivitySearchParams {
  const region = searchParams.get('region') ?? preferredCity;

  return {
    keyword: searchParams.get('keyword') ?? DEFAULT_ACTIVITY_SEARCH.keyword,
    region: region || DEFAULT_ACTIVITY_SEARCH.region,
    artist: searchParams.get('artist') ?? DEFAULT_ACTIVITY_SEARCH.artist,
    category:
      (searchParams.get('category') as ActivitySearchParams['category']) ??
      DEFAULT_ACTIVITY_SEARCH.category,
    sort:
      (searchParams.get('sort') as ActivitySearchParams['sort']) ??
      DEFAULT_ACTIVITY_SEARCH.sort,
    page: Number(searchParams.get('page') ?? DEFAULT_ACTIVITY_SEARCH.page),
    pageSize: DEFAULT_ACTIVITY_SEARCH.pageSize,
  };
}

export default function PublicActivitiesPage() {
  const { t } = useTranslation();
  const { preferredCity, setPreferredCity } = useOutletContext<PublicLayoutContext>();
  const [searchParams, setSearchParams] = useSearchParams();
  const [draftKeyword, setDraftKeyword] = useState('');
  const [draftArtist, setDraftArtist] = useState('');
  const [items, setItems] = useState<ActivityListItem[]>([]);
  const [recommendations, setRecommendations] = useState<RecommendationSectionItem[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const reportedSearchRef = useRef<string | null>(null);

  const filters = useMemo(
    () => readParams(searchParams, preferredCity),
    [searchParams, preferredCity],
  );

  useEffect(() => {
    setDraftKeyword(filters.keyword);
    setDraftArtist(filters.artist);
  }, [filters.keyword, filters.artist]);

  useEffect(() => {
    window.scrollTo({ top: 0, left: 0, behavior: 'auto' });
  }, [searchParams]);

  useEffect(() => {
    let active = true;

    async function load() {
      try {
        setLoading(true);
        setError('');

        const [result, hotList] = await Promise.all([
          listActivities(filters),
          getHotRecommendations(3).catch(() => []),
        ]);

        if (!active) {
          return;
        }

        setItems(result.list);
        setTotal(result.total);
        setRecommendations(hotList);

        const hasSearchFilters = Boolean(
          filters.keyword.trim() ||
            filters.artist.trim() ||
            filters.region !== 'ALL' ||
            filters.category !== 'ALL' ||
            filters.sort !== DEFAULT_ACTIVITY_SEARCH.sort,
        );
        const searchSignature = hasSearchFilters
          ? JSON.stringify({
              keyword: filters.keyword.trim(),
              artist: filters.artist.trim(),
              region: filters.region,
              category: filters.category,
              sort: filters.sort,
            })
          : '';

        if (!hasSearchFilters) {
          reportedSearchRef.current = null;
        }

        if (
          hasSearchFilters &&
          reportedSearchRef.current !== searchSignature
        ) {
          const behaviorTargetId = result.list[0]?.id ?? hotList[0]?.id ?? null;

          if (behaviorTargetId === null) {
            return;
          }

          reportedSearchRef.current = searchSignature;
          trackBehavior(
            {
              activityId: behaviorTargetId,
              behaviorType: 'SEARCH',
              detail: {
                keyword: filters.keyword.trim(),
                artist: filters.artist.trim(),
                region: filters.region,
                category: filters.category,
                sort: filters.sort,
                result_count: result.total,
              },
            },
            { immediate: true, timeoutMs: 1000 },
          );
        }
      } catch {
        if (!active) {
          return;
        }

        setError(t('public.errorDescription'));
        setItems([]);
        setTotal(0);
        setRecommendations([]);
      } finally {
        if (active) {
          setLoading(false);
        }
      }
    }

    load();

    return () => {
      active = false;
    };
  }, [filters, t]);

  const updateParams = (next: Partial<ActivitySearchParams>) => {
    const merged: ActivitySearchParams = {
      ...filters,
      ...next,
      page: next.page ?? (next.sort || next.category || next.region || next.keyword !== undefined || next.artist !== undefined ? 1 : filters.page),
    };

    const params = new URLSearchParams();

    if (merged.keyword) {
      params.set('keyword', merged.keyword);
    }

    if (merged.region && merged.region !== 'ALL') {
      params.set('region', merged.region);
    }

    if (merged.artist) {
      params.set('artist', merged.artist);
    }

    if (merged.category !== 'ALL') {
      params.set('category', merged.category);
    }

    if (merged.sort !== DEFAULT_ACTIVITY_SEARCH.sort) {
      params.set('sort', merged.sort);
    }

    if (merged.page > 1) {
      params.set('page', String(merged.page));
    }

    setSearchParams(params);
  };

  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    updateParams({
      keyword: draftKeyword.trim(),
      artist: draftArtist.trim(),
    });
  };

  return (
    <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_320px]">
      <div className="space-y-6">
        <section className="rounded-[32px] border border-slate-200 bg-white p-5 shadow-sm lg:p-8">
          <p className="text-sm font-semibold text-slate-400">
            {t('activities.totalProducts', { count: total })}
          </p>

          <form onSubmit={handleSubmit} className="mt-6 space-y-6">
            <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_280px_auto]">
              <label className="flex items-center gap-3 rounded-full border border-slate-200 bg-slate-50 px-4 py-3">
                <Search size={18} className="text-slate-400" />
                <input
                  value={draftKeyword}
                  onChange={(event) => setDraftKeyword(event.target.value)}
                  className="w-full bg-transparent text-sm text-slate-700 outline-none placeholder:text-slate-400"
                  placeholder={t('activities.keywordPlaceholder')}
                />
              </label>
              <input
                value={draftArtist}
                onChange={(event) => setDraftArtist(event.target.value)}
                className="rounded-full border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-700 outline-none placeholder:text-slate-400"
                placeholder={t('activities.artistPlaceholder')}
              />
              <button
                type="submit"
                className="rounded-full bg-rose-500 px-6 py-3 text-sm font-bold text-white transition hover:bg-rose-600"
              >
                {t('public.searchAction')}
              </button>
            </div>

            <div className="space-y-5 border-t border-slate-100 pt-6">
              <div className="grid gap-3 lg:grid-cols-[84px_minmax(0,1fr)]">
                <span className="pt-2 text-base font-bold text-slate-400">{t('activities.region')}</span>
                <div className="flex flex-wrap gap-3">
                  {CITY_OPTIONS.map((city) => (
                    <button
                      key={city.value}
                      type="button"
                      onClick={() => {
                        setPreferredCity(city.value);
                        updateParams({ region: city.value });
                      }}
                      className={`rounded-full px-4 py-2 text-sm font-semibold transition ${
                        filters.region === city.value
                          ? 'bg-rose-500 text-white'
                          : 'bg-slate-50 text-slate-500 hover:bg-rose-50 hover:text-rose-600'
                      }`}
                    >
                      {t(`cities.${city.value}`)}
                    </button>
                  ))}
                </div>
              </div>

              <div className="grid gap-3 lg:grid-cols-[84px_minmax(0,1fr)]">
                <span className="pt-2 text-base font-bold text-slate-400">{t('activities.category')}</span>
                <div className="flex flex-wrap gap-3">
                  {CATEGORY_OPTIONS.map((category) => (
                    <button
                      key={category.value}
                      type="button"
                      onClick={() =>
                        updateParams({
                          category: category.value,
                        })
                      }
                      className={`rounded-full px-4 py-2 text-sm font-semibold transition ${
                        filters.category === category.value
                          ? 'bg-rose-500 text-white'
                          : 'bg-slate-50 text-slate-500 hover:bg-rose-50 hover:text-rose-600'
                      }`}
                    >
                      {t(`categories.${category.value}`)}
                    </button>
                  ))}
                </div>
              </div>
            </div>
          </form>
        </section>

        <section className="overflow-hidden rounded-[32px] border border-slate-200 bg-white shadow-sm">
          <div className="flex flex-wrap border-b border-slate-100">
            {SORT_OPTIONS.map((option) => (
              <button
                key={option.value}
                type="button"
                onClick={() => updateParams({ sort: option.value })}
                className={`border-r border-slate-100 px-6 py-4 text-sm font-bold transition last:border-r-0 ${
                  filters.sort === option.value
                    ? 'bg-rose-50 text-rose-600'
                    : 'bg-white text-slate-400 hover:bg-slate-50 hover:text-slate-700'
                }`}
              >
                {t(`sort.${option.value}`)}
              </button>
            ))}
          </div>

          <div className="p-5 lg:p-8">
            {loading ? (
              <LoadingCards count={6} />
            ) : error ? (
              <EmptyState title={t('public.errorTitle')} description={error} />
            ) : items.length === 0 ? (
              <EmptyState />
            ) : (
              <>
                {items.map((item) => (
                  <SearchResultItem key={item.id} item={item} />
                ))}
                <Pagination
                  currentPage={filters.page}
                  total={total}
                  pageSize={filters.pageSize}
                  onPageChange={(page) => updateParams({ page })}
                />
              </>
            )}
          </div>
        </section>
      </div>

      <div className="space-y-6">
        <RecommendationList
          items={recommendations}
          isLoading={loading}
          title={t('activities.youMayAlsoLike')}
        />
      </div>
    </div>
  );
}
