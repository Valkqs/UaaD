import { CalendarRange, MapPin } from 'lucide-react';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { trackBehavior } from '../../api/endpoints';
import type { RecommendationSectionItem, HomeSpotlightItem } from '../../types';
import { formatLongDate } from '../../utils/formatters';

type SpotlightItem = RecommendationSectionItem | HomeSpotlightItem;

function getCoverUrl(item: SpotlightItem): string | undefined {
  return 'coverUrl' in item
    ? (item as RecommendationSectionItem).coverUrl ?? undefined
    : (item as HomeSpotlightItem).imageUrl ?? undefined;
}

function getEnrollOpenAt(item: SpotlightItem): string {
  return 'enrollOpenAt' in item ? (item as RecommendationSectionItem).enrollOpenAt : (item as HomeSpotlightItem).openAt;
}

function getDescription(item: SpotlightItem): string {
  if ('recommendReason' in item) {
    const ri = item as RecommendationSectionItem;
    return ri.recommendReason ?? ri.description;
  }
  return (item as HomeSpotlightItem).summary;
}

function getHref(item: SpotlightItem): string {
  return 'href' in item ? item.href : `/activity/${item.id}`;
}

interface SpotlightActivityProps {
  item: SpotlightItem;
  mirrored?: boolean;
}

export function SpotlightActivity({ item, mirrored = false }: SpotlightActivityProps) {
  const { t } = useTranslation();
  const coverUrl = getCoverUrl(item);
  const description = getDescription(item);

  return (
    <section className="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm">
      <div
        className={`flex min-h-[380px] flex-col lg:flex-row ${mirrored ? 'lg:flex-row-reverse' : ''}`}
      >
        {/* Image panel — fixed width so it's the same size in both mirrored states */}
        <div className="shrink-0 overflow-hidden bg-slate-100 lg:w-[380px]">
          {coverUrl ? (
            <img
              src={coverUrl}
              alt={item.title}
              className="h-full min-h-[280px] w-full object-cover lg:min-h-[380px]"
            />
          ) : (
            <div className="h-full min-h-[280px] w-full bg-gradient-to-br from-rose-100 via-white to-orange-100 lg:min-h-[380px]" />
          )}
        </div>

        {/* Content panel */}
        <div
          className={`flex flex-1 flex-col gap-4 p-6 lg:p-8 ${
            mirrored ? 'border-slate-100 lg:border-r' : 'border-slate-100 lg:border-l'
          }`}
        >
          {/* Title box — matches Figma bordered title block */}
          <div className="rounded-xl border border-slate-200 bg-slate-50/60 px-5 py-4">
            <p className="mb-1 text-xs font-semibold uppercase tracking-[0.22em] text-rose-400">
              {t('home.selectedBadge')}
            </p>
            <h2 className="text-2xl font-black tracking-tight text-slate-900 lg:text-[1.65rem]">
              {item.title}
            </h2>
          </div>

          {/* Details box — matches Figma bordered brief-intro block */}
          <div className="flex flex-1 flex-col rounded-xl border border-slate-200 px-5 py-4">
            <p className="mb-3 text-xs font-semibold uppercase tracking-[0.2em] text-slate-400">
              {t('home.spotlightTagline')}
            </p>

            <div className="flex flex-col gap-2 text-sm text-slate-600">
              <div className="flex items-center gap-2">
                <MapPin size={14} className="shrink-0 text-rose-400" />
                <span>{item.location}</span>
              </div>
              <div className="flex items-center gap-2">
                <CalendarRange size={14} className="shrink-0 text-rose-400" />
                <span>{formatLongDate(getEnrollOpenAt(item))}</span>
              </div>
            </div>

            <p className="mt-4 flex-1 text-sm leading-7 text-slate-500">
              {description}
            </p>

            <div className="mt-5 flex flex-wrap gap-3">
              <Link
                to={getHref(item)}
                onClick={() =>
                  trackBehavior(
                    {
                      activityId: item.id,
                      behaviorType: 'CLICK',
                      detail: {
                        source: 'home_spotlight',
                        category: item.category,
                      },
                    },
                    { immediate: true, timeoutMs: 1000 },
                  )
                }
                className="rounded-full bg-rose-600 px-6 py-2.5 text-sm font-bold !text-white shadow-[0_18px_36px_-22px_rgba(244,63,94,0.9)] transition hover:bg-rose-700 hover:!text-white visited:!text-white"
                style={{ color: '#ffffff' }}
              >
                {t('public.viewDetails')}
              </Link>
              <Link
                to={`/activities?category=${item.category}`}
                className="rounded-full border border-slate-200 px-6 py-2.5 text-sm font-bold text-slate-600 transition hover:border-rose-200 hover:text-rose-600"
              >
                {t('home.spotlightCategoryAction', { category: t(`categories.${item.category}`) })}
              </Link>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
