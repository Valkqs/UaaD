import { MapPin, Sparkles } from 'lucide-react';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { trackBehavior } from '../../api/endpoints';
import type { ActivityListItem } from '../../types';
import { formatCurrency, formatDateRange } from '../../utils/formatters';
import { StatusChip } from './StatusChip';

export function SearchResultItem({ item }: { item: ActivityListItem }) {
  const { t } = useTranslation();
  const handleTrackClick = () =>
    trackBehavior(
      {
        activityId: item.id,
        behaviorType: 'CLICK',
        detail: {
          source: 'public_search_results',
          category: item.category,
        },
      },
      { immediate: true, timeoutMs: 1000 },
    );

  return (
    <article className="grid gap-5 border-b border-slate-100 py-6 last:border-b-0 lg:grid-cols-[220px_minmax(0,1fr)_180px]">
      <Link
        to={`/activity/${item.id}`}
        onClick={handleTrackClick}
        className="block overflow-hidden rounded-[24px] bg-slate-100"
      >
        {item.coverUrl ? (
          <img src={item.coverUrl} alt={item.title} className="h-full w-full object-cover" />
        ) : (
          <div className="aspect-[4/5] bg-gradient-to-br from-rose-100 via-white to-orange-100" />
        )}
      </Link>

      <div className="min-w-0">
        <div className="mb-3 flex flex-wrap items-center gap-3">
          <StatusChip status={item.status} />
          <span className="rounded-full bg-slate-100 px-3 py-1 text-xs font-semibold text-slate-500">
            {t(`categories.${item.category}`)}
          </span>
        </div>
        <Link to={`/activity/${item.id}`} onClick={handleTrackClick}>
          <h3 className="text-2xl font-black leading-9 text-slate-900 transition hover:text-rose-600">
            {item.title}
          </h3>
        </Link>
        <p className="mt-3 line-clamp-2 text-sm leading-7 text-slate-500">
          {item.description}
        </p>

        <div className="mt-5 flex flex-wrap items-center gap-x-6 gap-y-3 text-sm text-slate-500">
          <span className="flex items-center gap-2">
            <MapPin size={16} className="text-rose-400" />
            {item.location}
          </span>
          <span>{formatDateRange(item.enrollOpenAt, item.enrollCloseAt)}</span>
          <span>{item.enrollCount.toLocaleString()} {t('public.enrolled')}</span>
        </div>

        {item.tags.length > 0 ? (
          <div className="mt-4 flex flex-wrap gap-2">
            {item.tags.slice(0, 4).map((tag) => (
              <span
                key={tag}
                className="rounded-full bg-rose-50 px-3 py-1 text-xs font-medium text-rose-500"
              >
                {tag}
              </span>
            ))}
          </div>
        ) : null}
      </div>

      <div className="flex flex-col justify-between rounded-[24px] bg-slate-50 p-5">
        <div className="space-y-3">
          <p className="text-sm font-semibold text-slate-400">{t('public.priceStarts')}</p>
          <p className="text-3xl font-black text-rose-600">{formatCurrency(item.price)}</p>
          <div className="flex items-center gap-2 text-xs font-semibold text-slate-500">
            <Sparkles size={14} className="text-amber-500" />
            {t('public.stockRemaining', { count: item.stockRemaining })}
          </div>
        </div>

        <Link
          to={`/activity/${item.id}`}
          onClick={handleTrackClick}
          className="mt-6 inline-flex items-center justify-center rounded-full bg-white px-4 py-3 text-sm font-bold text-slate-900 shadow-sm transition hover:bg-rose-500 hover:text-white"
        >
          {t('public.viewDetails')}
        </Link>
      </div>
    </article>
  );
}
