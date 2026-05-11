import { Link } from 'react-router-dom';
import { MapPin } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { trackBehavior } from '../../api/endpoints';
import type { ActivityListItem } from '../../types';
import { formatCurrency, formatDateRange } from '../../utils/formatters';
import { StatusChip } from './StatusChip';

export function ActivityGridCard({ item }: { item: ActivityListItem }) {
  const { t } = useTranslation();

  return (
    <article className="overflow-hidden rounded-[28px] border border-slate-200 bg-white shadow-sm transition hover:-translate-y-1 hover:shadow-xl">
      <Link
        to={`/activity/${item.id}`}
        onClick={() =>
          trackBehavior(
            {
              activityId: item.id,
              behaviorType: 'CLICK',
              detail: {
                source: 'legacy_activity_grid',
                category: item.category,
              },
            },
            { immediate: true, timeoutMs: 1000 },
          )
        }
        className="block"
      >
        <div className="relative aspect-[4/5] overflow-hidden bg-slate-100">
          {item.coverUrl ? (
            <img src={item.coverUrl} alt={item.title} className="h-full w-full object-cover" />
          ) : (
            <div className="h-full w-full bg-gradient-to-br from-rose-100 via-white to-orange-100" />
          )}
          <div className="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/60 to-transparent p-5">
            <StatusChip status={item.status} />
          </div>
        </div>

        <div className="space-y-3 p-5">
          <p className="text-xs font-semibold uppercase tracking-[0.2em] text-slate-400">
            {t(`categories.${item.category}`)}
          </p>
          <h3 className="line-clamp-2 text-lg font-bold leading-7 text-slate-900">
            {item.title}
          </h3>
          <div className="flex items-center gap-2 text-sm text-slate-500">
            <MapPin size={14} />
            <span className="line-clamp-1">{item.location}</span>
          </div>
          <p className="text-sm text-slate-500">
            {formatDateRange(item.enrollOpenAt, item.enrollCloseAt)}
          </p>
          <div className="flex items-center justify-between">
            <p className="text-xl font-black text-rose-600">{formatCurrency(item.price)}</p>
            <span className="text-sm font-medium text-slate-400">
              {item.enrollCount.toLocaleString()} {t('public.enrolled')}
            </span>
          </div>
        </div>
      </Link>
    </article>
  );
}
