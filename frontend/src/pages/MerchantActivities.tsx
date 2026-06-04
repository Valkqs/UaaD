import { useCallback, useEffect, useState } from 'react';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import { Flame, PenLine, PlusCircle, Rocket, RotateCcw } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { listMerchantActivities, preheatMerchantActivity, publishMerchantActivity } from '../api/endpoints';
import type { ActivityListItem } from '../types';
import { MerchantNotice } from '../components/merchant/MerchantNotice';
import { MerchantPageHeader } from '../components/merchant/MerchantPageHeader';
import { MerchantStateCard } from '../components/merchant/MerchantStateCard';
import { StatusChip } from '../components/public/StatusChip';
import { resolveApiErrorMessage } from '../utils/api';
import { formatCurrency } from '../utils/formatters';

export default function MerchantActivitiesPage() {
  const { t } = useTranslation();
  const location = useLocation();
  const navigate = useNavigate();
  const [items, setItems] = useState<ActivityListItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState('');
  const [preheatingId, setPreheatingId] = useState<number | null>(null);
  const [publishingId, setPublishingId] = useState<number | null>(null);
  const [notice, setNotice] = useState<{
    tone: 'success' | 'error' | 'info';
    title?: string;
    message: string;
  } | null>(null);

  useEffect(() => {
    const routeState =
      location.state as
        | {
            message?: string;
            feedback?: { tone: 'success' | 'error' | 'info'; title?: string; message: string };
          }
        | null;

    if (routeState?.feedback) {
      setNotice(routeState.feedback);
      navigate(`${location.pathname}${location.search}`, { replace: true, state: null });
      return;
    }

    if (routeState?.message) {
      setNotice({
        tone: 'success',
        title: t('merchant.successTitle'),
        message: routeState.message,
      });
      navigate(`${location.pathname}${location.search}`, { replace: true, state: null });
    }
  }, [location.pathname, location.search, location.state, navigate, t]);

  const load = useCallback(async () => {
    setLoading(true);
    setLoadError('');

    try {
      const nextItems = await listMerchantActivities();
      setItems(nextItems);
    } catch (error) {
      setItems([]);
      setLoadError(
        resolveApiErrorMessage(error, {
          fallback: t('merchant.listLoadFailed'),
          networkFallback: t('merchant.networkError'),
        }),
      );
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    void load();
  }, [load]);

  const canPublish = (status: ActivityListItem['status']) => status === 'DRAFT' || status === 'PREHEAT';
  const canPreheat = (status: ActivityListItem['status']) => status === 'DRAFT';
  const isPublishedStatus = (status: ActivityListItem['status']) =>
    ['PUBLISHED', 'SELLING_OUT', 'SOLD_OUT'].includes(status);
  const isMutating = preheatingId !== null || publishingId !== null;
  const getPreheatLabel = (item: ActivityListItem) => {
    if (preheatingId === item.id) {
      return t('merchant.preheating');
    }

    if (item.status === 'PREHEAT') {
      return t('merchant.preheatedAction');
    }

    if (canPreheat(item.status)) {
      return t('merchant.preheat');
    }

    return t('merchant.preheatUnavailable');
  };
  const getPublishLabel = (item: ActivityListItem) => {
    if (publishingId === item.id) {
      return t('merchant.publishing');
    }

    if (canPublish(item.status)) {
      return t('merchant.publish');
    }

    if (isPublishedStatus(item.status)) {
      return t('merchant.publishedAction');
    }

    return t('merchant.publishUnavailable');
  };

  const handlePreheat = async (item: ActivityListItem) => {
    if (!canPreheat(item.status) || isMutating) {
      return;
    }

    setPreheatingId(item.id);
    setNotice(null);

    try {
      const result = await preheatMerchantActivity(item.id);
      setItems((current) =>
        current.map((currentItem) =>
          currentItem.id === item.id
            ? {
                ...currentItem,
                status: result.status,
              }
            : currentItem,
        ),
      );
      setNotice({
        tone: 'success',
        title: t('merchant.preheatSuccessTitle'),
        message: result.message || t('merchant.preheatSuccess'),
      });
      void load();
    } catch (error) {
      setNotice({
        tone: 'error',
        title: t('merchant.preheatFailedTitle'),
        message: resolveApiErrorMessage(error, {
          fallback: t('merchant.preheatFailed'),
          networkFallback: t('merchant.networkError'),
        }),
      });
    } finally {
      setPreheatingId(null);
    }
  };

  const handlePublish = async (item: ActivityListItem) => {
    if (!canPublish(item.status) || isMutating) {
      return;
    }

    setPublishingId(item.id);
    setNotice(null);

    try {
      const result = await publishMerchantActivity(item.id);
      setItems((current) =>
        current.map((currentItem) =>
          currentItem.id === item.id
            ? {
                ...currentItem,
                status: result.status,
                stockRemaining: result.stockInCache ?? currentItem.stockRemaining,
              }
            : currentItem,
        ),
      );
      setNotice({
        tone: 'success',
        title: t('merchant.publishSuccessTitle'),
        message: result.message || t('merchant.publishSuccess'),
      });
      void load();
    } catch (error) {
      setNotice({
        tone: 'error',
        title: t('merchant.publishFailedTitle'),
        message: resolveApiErrorMessage(error, {
          fallback: t('merchant.publishFailed'),
          networkFallback: t('merchant.networkError'),
        }),
      });
    } finally {
      setPublishingId(null);
    }
  };

  return (
    <div className="space-y-5">
      <MerchantPageHeader
        eyebrow={t('merchant.panel')}
        title={t('merchant.activityList')}
        description={t('merchant.listSubtitle')}
        actions={
          <Link
            to="/merchant/activities/new"
            className="inline-flex items-center gap-2 rounded-full bg-rose-500 px-5 py-2.5 text-sm font-semibold text-white transition hover:bg-rose-600"
          >
            <PlusCircle size={16} />
            {t('merchant.createActivity')}
          </Link>
        }
      />

      {notice ? (
        <MerchantNotice tone={notice.tone} title={notice.title} message={notice.message} />
      ) : null}

      <section className="overflow-hidden rounded-[32px] border border-rose-100 bg-white shadow-sm">
        <div className="flex flex-wrap items-center justify-between gap-4 border-b border-rose-100 px-6 py-5">
          <div>
            <p className="text-sm font-semibold text-slate-900">{t('merchant.activityList')}</p>
            <p className="mt-1 text-sm text-slate-500">{t('merchant.listDescription')}</p>
          </div>
          <button
            type="button"
            onClick={() => void load()}
            disabled={loading}
            className="inline-flex items-center gap-2 rounded-full border border-rose-100 bg-white px-4 py-2 text-sm font-semibold text-slate-600 transition hover:border-rose-200 hover:text-rose-600 disabled:cursor-not-allowed disabled:opacity-50"
          >
            <RotateCcw size={14} />
            {t('merchant.refresh')}
          </button>
        </div>

        {loading ? (
          <div className="p-6">
            <MerchantStateCard
              compact
              tone="loading"
              title={t('merchant.loadingTitle')}
              description={t('merchant.loadingDescription')}
            />
          </div>
        ) : loadError ? (
          <div className="p-6">
            <MerchantStateCard
              compact
              tone="error"
              title={t('merchant.listLoadFailedTitle')}
              description={loadError}
              action={
                <button
                  type="button"
                  onClick={() => void load()}
                  className="rounded-full bg-rose-500 px-5 py-2.5 text-sm font-semibold text-white transition hover:bg-rose-600"
                >
                  {t('merchant.retry')}
                </button>
              }
            />
          </div>
        ) : items.length === 0 ? (
          <div className="p-6">
            <MerchantStateCard
              compact
              tone="empty"
              title={t('merchant.empty')}
              description={t('merchant.emptyDescription')}
              action={
                <Link
                  to="/merchant/activities/new"
                  className="inline-flex items-center gap-2 rounded-full bg-rose-500 px-5 py-2.5 text-sm font-semibold text-white transition hover:bg-rose-600"
                >
                  <PlusCircle size={15} />
                  {t('merchant.createActivity')}
                </Link>
              }
            />
          </div>
        ) : (
          <>
            <div className="space-y-4 p-4 md:hidden">
            {items.map((item) => (
              <article key={item.id} className="rounded-2xl border border-rose-100 bg-[#fffaf7] p-4">
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <p className="truncate text-sm font-semibold text-slate-900">{item.title}</p>
                    <p className="mt-1 text-xs text-slate-500">{item.location}</p>
                  </div>
                  <StatusChip status={item.status} theme="soft" />
                </div>
                <div className="mt-3 grid grid-cols-2 gap-3 text-xs text-slate-600">
                  <p>{t('merchant.table.price')}: {formatCurrency(item.price)}</p>
                  <p>{t('merchant.table.enroll')}: {item.enrollCount.toLocaleString()}</p>
                </div>
                <div className="mt-4 flex flex-wrap gap-2">
                  <Link
                    to={`/merchant/activities/${item.id}/edit`}
                    className="inline-flex items-center gap-1 rounded-full border border-rose-100 bg-white px-4 py-2 text-xs font-semibold text-slate-600 transition hover:border-rose-200 hover:text-rose-600"
                  >
                    <PenLine size={12} />
                    {t('merchant.edit')}
                  </Link>
                  <button
                    type="button"
                    disabled={!canPreheat(item.status) || isMutating}
                    onClick={() => void handlePreheat(item)}
                    className="inline-flex items-center gap-1 rounded-full border border-amber-100 bg-amber-50 px-4 py-2 text-xs font-semibold text-amber-700 transition hover:border-amber-200 hover:bg-amber-100 disabled:cursor-not-allowed disabled:border-slate-100 disabled:bg-slate-100 disabled:text-slate-400"
                  >
                    <Flame size={12} />
                    {getPreheatLabel(item)}
                  </button>
                  <button
                    type="button"
                    disabled={!canPublish(item.status) || isMutating}
                    onClick={() => void handlePublish(item)}
                    className="inline-flex items-center gap-1 rounded-full bg-rose-500 px-4 py-2 text-xs font-semibold text-white transition hover:bg-rose-600 disabled:cursor-not-allowed disabled:bg-slate-200 disabled:text-slate-400"
                  >
                    <Rocket size={12} />
                    {getPublishLabel(item)}
                  </button>
                </div>
              </article>
            ))}
            </div>
            <div className="hidden overflow-x-auto md:block">
            <table className="min-w-full text-left text-sm">
              <thead className="bg-[#fff8f3] text-slate-400">
                <tr>
                  <th className="px-6 py-4">{t('merchant.table.activity')}</th>
                  <th className="px-6 py-4">{t('merchant.table.status')}</th>
                  <th className="px-6 py-4">{t('merchant.table.price')}</th>
                  <th className="px-6 py-4">{t('merchant.table.enroll')}</th>
                  <th className="px-6 py-4">{t('merchant.table.actions')}</th>
                </tr>
              </thead>
              <tbody>
                {items.map((item) => (
                  <tr key={item.id} className="border-t border-rose-100 text-slate-600">
                    <td className="px-6 py-5 align-top">
                      <p className="font-semibold text-slate-900">{item.title}</p>
                      <p className="mt-1 text-xs text-slate-500">{item.location}</p>
                    </td>
                    <td className="px-6 py-5 align-top">
                      <StatusChip status={item.status} theme="soft" />
                    </td>
                    <td className="px-6 py-5 align-top">{formatCurrency(item.price)}</td>
                    <td className="px-6 py-5 align-top">{item.enrollCount.toLocaleString()}</td>
                    <td className="px-6 py-5 align-top">
                      <div className="flex flex-wrap gap-2">
                        <Link
                          to={`/merchant/activities/${item.id}/edit`}
                          className="inline-flex items-center gap-1 rounded-full border border-rose-100 bg-white px-4 py-2 text-xs font-semibold text-slate-600 transition hover:border-rose-200 hover:text-rose-600"
                        >
                          <PenLine size={12} />
                          {t('merchant.edit')}
                        </Link>
                        <button
                          type="button"
                          disabled={!canPreheat(item.status) || isMutating}
                          onClick={() => void handlePreheat(item)}
                          className="inline-flex items-center gap-1 rounded-full border border-amber-100 bg-amber-50 px-4 py-2 text-xs font-semibold text-amber-700 transition hover:border-amber-200 hover:bg-amber-100 disabled:cursor-not-allowed disabled:border-slate-100 disabled:bg-slate-100 disabled:text-slate-400"
                        >
                          <Flame size={12} />
                          {getPreheatLabel(item)}
                        </button>
                        <button
                          type="button"
                          disabled={!canPublish(item.status) || isMutating}
                          onClick={() => void handlePublish(item)}
                          className="inline-flex items-center gap-1 rounded-full bg-rose-500 px-4 py-2 text-xs font-semibold text-white transition hover:bg-rose-600 disabled:cursor-not-allowed disabled:bg-slate-200 disabled:text-slate-400"
                        >
                          <Rocket size={12} />
                          {getPublishLabel(item)}
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            </div>
          </>
        )}
      </section>
    </div>
  );
}
