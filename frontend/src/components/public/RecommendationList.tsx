import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { trackBehavior } from '../../api/endpoints';
import type { RecommendationSectionItem } from '../../types';
import { formatCurrency, formatDateRange } from '../../utils/formatters';

export function RecommendationList({
  items,
  title,
  isLoading = false,
}: {
  items: RecommendationSectionItem[];
  title: string;
  isLoading?: boolean;
}) {
  const { t } = useTranslation();

  return (
    <aside className="rounded-[28px] border border-slate-200 bg-white p-5 shadow-sm lg:p-6">
      <div className="mb-5 flex items-center justify-between">
        <h2 className="text-xl font-black text-slate-900">{title}</h2>
        <Link to="/activities?sort=hot" className="text-sm font-semibold text-slate-400 transition hover:text-rose-600">
          {t('public.viewAll')}
        </Link>
      </div>
      {isLoading ? (
        <div className="space-y-4">
          {Array.from({ length: 3 }).map((_, index) => (
            <div
              key={index}
              className="flex gap-4 rounded-2xl border border-slate-100 p-3"
            >
              <div className="h-24 w-20 shrink-0 animate-pulse rounded-2xl bg-slate-100" />
              <div className="min-w-0 flex-1">
                <div className="h-4 w-4/5 animate-pulse rounded-full bg-slate-100" />
                <div className="mt-3 h-3 w-full animate-pulse rounded-full bg-slate-100" />
                <div className="mt-2 h-3 w-2/3 animate-pulse rounded-full bg-slate-100" />
                <div className="mt-4 h-4 w-24 animate-pulse rounded-full bg-slate-100" />
              </div>
            </div>
          ))}
        </div>
      ) : items.length === 0 ? (
        <div className="rounded-2xl border border-dashed border-slate-200 bg-slate-50 px-4 py-8 text-center text-sm leading-7 text-slate-500">
          {t('public.emptyDescription')}
        </div>
      ) : (
        <div className="space-y-4">
          {items.map((item) => (
            <Link
              key={item.id}
              to={`/activities?keyword=${encodeURIComponent(item.title)}`}
              onClick={() =>
                trackBehavior(
                  {
                    activityId: item.id,
                    behaviorType: 'CLICK',
                    detail: {
                      source: 'recommendation_sidebar',
                      category: item.category,
                    },
                  },
                  { immediate: true, timeoutMs: 1000 },
                )
              }
              className="flex gap-4 rounded-2xl border border-slate-100 p-3 transition hover:border-rose-100 hover:bg-rose-50/50"
            >
              <div className="h-24 w-20 shrink-0 overflow-hidden rounded-2xl bg-slate-100">
                {item.coverUrl ? (
                  <img src={item.coverUrl} alt={item.title} className="h-full w-full object-cover" />
                ) : null}
              </div>
              <div className="min-w-0">
                <h3 className="line-clamp-2 text-sm font-bold leading-6 text-slate-900">
                  {item.title}
                </h3>
                <p className="mt-1 line-clamp-2 text-xs leading-5 text-slate-500">
                  {item.recommendReason ?? item.description}
                </p>
                <p className="mt-2 text-xs font-medium text-slate-400">
                  {formatDateRange(item.enrollOpenAt, item.enrollCloseAt)}
                </p>
                <p className="mt-3 text-sm font-black text-rose-600">
                  {formatCurrency(item.price)}
                </p>
              </div>
            </Link>
          ))}
        </div>
      )}
    </aside>
  );
}
