import { useEffect, useMemo, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { CheckCircle2, Clock3, CreditCard, Loader2, ReceiptText, XCircle } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { cancelEnrollment, getActivityDetail, getOrderDetail, payOrder } from '../api/endpoints';
import type { ActivityDetail, OrderItem } from '../types';
import { formatExactCurrency, formatLongDate } from '../utils/formatters';

const STATUS_STYLES: Record<OrderItem['status'], string> = {
  PENDING: 'border border-amber-200 bg-amber-50 text-amber-700',
  PAID: 'border border-emerald-200 bg-emerald-50 text-emerald-700',
  CLOSED: 'border border-slate-200 bg-slate-100 text-slate-500',
  REFUNDED: 'border border-sky-200 bg-sky-50 text-sky-700',
};

export default function OrderDetailPage() {
  const { t } = useTranslation();
  const { id } = useParams();
  const orderId = Number(id);
  const [order, setOrder] = useState<OrderItem | null>(null);
  const [activity, setActivity] = useState<ActivityDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [feedback, setFeedback] = useState<{
    tone: 'success' | 'error';
    message: string;
  } | null>(null);
  const [paying, setPaying] = useState(false);
  const [cancelling, setCancelling] = useState(false);
  const [now, setNow] = useState(Date.now());

  useEffect(() => {
    if (!Number.isFinite(orderId)) {
      setError(t('orders.invalidOrderId'));
      setLoading(false);
      return;
    }

    let active = true;

    async function load() {
      try {
        setLoading(true);
        setError('');
        const orderDetail = await getOrderDetail(orderId);

        if (!active) {
          return;
        }

        setOrder(orderDetail);

        const activityDetail = await getActivityDetail(orderDetail.activityId).catch(() => null);
        if (!active) {
          return;
        }

        setActivity(activityDetail);
      } catch {
        if (active) {
          setError(t('orders.detailLoadError'));
        }
      } finally {
        if (active) {
          setLoading(false);
        }
      }
    }

    void load();

    return () => {
      active = false;
    };
  }, [orderId, t]);

  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, []);

  const countdown = useMemo(() => {
    if (!order) {
      return { hours: 0, minutes: 0, seconds: 0, expired: false };
    }

    const remainMs = new Date(order.expiredAt).getTime() - now;
    const totalSeconds = Math.max(0, Math.floor(remainMs / 1000));

    return {
      expired: remainMs <= 0,
      hours: Math.floor(totalSeconds / 3600),
      minutes: Math.floor((totalSeconds % 3600) / 60),
      seconds: totalSeconds % 60,
    };
  }, [now, order]);

  const handlePay = async () => {
    if (!order) {
      return;
    }

    setPaying(true);
    setFeedback(null);

    try {
      const result = await payOrder(order.id);
      setOrder((current) =>
        current
          ? {
              ...current,
              status: result.status,
              paidAt: result.paidAt,
              updatedAt: result.paidAt,
            }
          : current,
      );
      setFeedback({
        tone: 'success',
        message: t('orders.paySuccess'),
      });
    } catch (err) {
      const errorWithResponse = err as { response?: { data?: { message?: string } } };
      setFeedback({
        tone: 'error',
        message: errorWithResponse.response?.data?.message || t('orders.payError'),
      });
    } finally {
      setPaying(false);
    }
  };

  const handleCancelPending = async () => {
    if (!order || cancelling) {
      return;
    }

    const confirmed = window.confirm(t('orders.cancelConfirm'));
    if (!confirmed) {
      return;
    }

    setCancelling(true);
    setFeedback(null);

    try {
      await cancelEnrollment(order.enrollmentId);
      const nextOrder = await getOrderDetail(order.id);
      setOrder(nextOrder);
      setFeedback({
        tone: 'success',
        message: t('orders.cancelSuccess'),
      });
    } catch (err) {
      const errorWithResponse = err as { response?: { data?: { message?: string } } };
      setFeedback({
        tone: 'error',
        message: errorWithResponse.response?.data?.message || t('orders.cancelError'),
      });
    } finally {
      setCancelling(false);
    }
  };

  if (loading) {
    return (
      <div className="mx-auto max-w-4xl space-y-5">
        <div className="h-52 animate-pulse rounded-[32px] bg-rose-100" />
        <div className="h-64 animate-pulse rounded-[32px] bg-white" />
      </div>
    );
  }

  if (error || !order) {
    return (
      <div className="mx-auto flex min-h-[55vh] max-w-3xl flex-col items-center justify-center gap-5 px-4 text-center">
        <p className="text-2xl font-bold text-slate-900">{t('public.errorTitle')}</p>
        <p className="text-slate-500">{error || t('orders.detailUnavailable')}</p>
        <Link
          to="/orders"
          className="rounded-full bg-rose-500 px-5 py-2 text-sm font-semibold text-white transition hover:bg-rose-600"
        >
          {t('orders.backToOrders')}
        </Link>
      </div>
    );
  }

  const canPay = order.status === 'PENDING' && !countdown.expired;

  return (
    <div className="mx-auto max-w-4xl space-y-8 pb-12">
      <section className="overflow-hidden rounded-[32px] border border-rose-100 bg-[linear-gradient(135deg,#fff8f3_0%,#fff1eb_60%,#ffe3d8_100%)] shadow-[0_28px_80px_-52px_rgba(244,63,94,0.22)]">
        <div className="grid gap-0 lg:grid-cols-[1.1fr_minmax(0,0.9fr)]">
          <div className="px-6 py-8 lg:px-8">
            <p className="text-sm font-semibold uppercase tracking-[0.24em] text-rose-400">UAAD</p>
            <h2 className="mt-3 text-3xl font-black tracking-tight text-slate-900">
              {t('orders.detailTitle')}
            </h2>
            <p className="mt-3 max-w-2xl text-sm leading-7 text-slate-600 lg:text-base">
              {activity?.title || t('orders.detailSubtitle')}
            </p>
            <div className="mt-6 flex flex-wrap items-center gap-3">
              <span className={`rounded-full border px-3 py-1 text-xs font-semibold uppercase tracking-[0.18em] ${STATUS_STYLES[order.status]}`}>
                {t(`orders.status.${order.status}`)}
              </span>
              <span className="rounded-full border border-slate-200 bg-white px-3 py-1 text-xs font-semibold uppercase tracking-[0.18em] text-slate-400">
                {order.orderNo}
              </span>
            </div>
          </div>

          <div className="min-h-[220px] bg-[#fff1eb]">
            {activity?.coverUrl ? (
              <img src={activity.coverUrl} alt={activity.title} className="h-full w-full object-cover" />
            ) : (
              <div className="flex h-full items-center justify-center bg-[radial-gradient(circle_at_top,_rgba(251,113,133,0.22),_rgba(255,241,235,0.88)_48%,_rgba(255,248,243,1))]">
                <ReceiptText size={52} className="text-rose-300" />
              </div>
            )}
          </div>
        </div>
      </section>

      <section className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_320px]">
        <article className="space-y-6 rounded-[32px] border border-rose-100 bg-white p-6 shadow-sm lg:p-8">
          <div className="grid gap-4 md:grid-cols-2">
            <div className="rounded-[24px] border border-slate-200 bg-[#fffaf7] p-5">
              <p className="text-sm font-semibold text-slate-400">{t('orders.amount')}</p>
              <p className="mt-2 text-3xl font-black text-slate-900">
                {formatExactCurrency(order.amount)}
              </p>
            </div>
            <div className="rounded-[24px] border border-slate-200 bg-[#fffaf7] p-5">
              <p className="text-sm font-semibold text-slate-400">{t('orders.createdLabel')}</p>
              <p className="mt-2 text-lg font-bold text-slate-900">
                {formatLongDate(order.createdAt)}
              </p>
            </div>
            <div className="rounded-[24px] border border-slate-200 bg-[#fffaf7] p-5">
              <p className="text-sm font-semibold text-slate-400">{t('orders.expireLabel')}</p>
              <p className="mt-2 text-lg font-bold text-slate-900">
                {formatLongDate(order.expiredAt)}
              </p>
            </div>
            <div className="rounded-[24px] border border-slate-200 bg-[#fffaf7] p-5">
              <p className="text-sm font-semibold text-slate-400">{t('orders.paymentStatus')}</p>
              <p className="mt-2 text-lg font-bold text-slate-900">
                {order.status === 'PAID' && order.paidAt
                  ? t('orders.paidAt', { time: formatLongDate(order.paidAt) })
                  : t(`orders.statusDescription.${order.status}`)}
              </p>
            </div>
          </div>

          {feedback ? (
            <div
              className={`rounded-[24px] px-5 py-4 text-sm ${
                feedback.tone === 'success'
                  ? 'border border-emerald-200 bg-emerald-50 text-emerald-700'
                  : 'border border-amber-200 bg-amber-50 text-amber-700'
              }`}
            >
              {feedback.message}
            </div>
          ) : null}

          <div className="rounded-[28px] border border-rose-100 bg-[#fff8f3] p-5">
            <p className="text-sm font-semibold text-slate-700">{t('orders.nextStepTitle')}</p>
            <p className="mt-2 text-sm leading-7 text-slate-500">
              {canPay
                ? t('orders.nextStepPending')
                : order.status === 'PAID'
                  ? t('orders.nextStepPaid')
                  : t('orders.nextStepClosed')}
            </p>
          </div>
        </article>

        <aside className="space-y-5 rounded-[32px] border border-rose-100 bg-white p-6 shadow-sm">
          <div className="rounded-[24px] border border-rose-100 bg-[#fff8f3] p-5">
            <p className="text-sm font-semibold text-slate-700">{t('orders.checkoutPanel')}</p>
            {canPay ? (
              <div className="mt-4 grid grid-cols-3 gap-2 text-center">
                <div className="rounded-xl border border-rose-100 bg-white py-3">
                  <p className="text-xl font-black text-slate-900">{countdown.hours}</p>
                  <p className="text-[10px] uppercase tracking-[0.18em] text-slate-400">
                    {t('activityDetail.hour')}
                  </p>
                </div>
                <div className="rounded-xl border border-rose-100 bg-white py-3">
                  <p className="text-xl font-black text-slate-900">{countdown.minutes}</p>
                  <p className="text-[10px] uppercase tracking-[0.18em] text-slate-400">
                    {t('activityDetail.minute')}
                  </p>
                </div>
                <div className="rounded-xl border border-rose-100 bg-white py-3">
                  <p className="text-xl font-black text-slate-900">{countdown.seconds}</p>
                  <p className="text-[10px] uppercase tracking-[0.18em] text-slate-400">
                    {t('activityDetail.second')}
                  </p>
                </div>
              </div>
            ) : (
              <div className="mt-4 rounded-2xl border border-rose-100 bg-white px-4 py-4 text-sm text-slate-500">
                {order.status === 'PAID'
                  ? t('orders.paidSummary')
                  : t('orders.closedSummary')}
              </div>
            )}
          </div>

          {canPay ? (
            <div className="space-y-3">
              <button
                type="button"
                onClick={handlePay}
                disabled={paying || cancelling}
                className="inline-flex w-full items-center justify-center gap-2 rounded-full bg-rose-500 px-6 py-3 text-sm font-bold text-white transition hover:bg-rose-600 disabled:cursor-not-allowed disabled:opacity-60"
              >
                {paying ? <Loader2 size={16} className="animate-spin" /> : <CreditCard size={16} />}
                {paying ? t('orders.paying') : t('orders.payNow')}
              </button>
              <button
                type="button"
                onClick={() => void handleCancelPending()}
                disabled={paying || cancelling}
                className="inline-flex w-full items-center justify-center gap-2 rounded-full border border-rose-200 bg-white px-6 py-3 text-sm font-bold text-rose-600 transition hover:border-rose-300 hover:bg-rose-50 disabled:cursor-not-allowed disabled:opacity-60"
              >
                {cancelling ? <Loader2 size={16} className="animate-spin" /> : <XCircle size={16} />}
                {cancelling ? t('orders.cancelling') : t('orders.cancelPending')}
              </button>
            </div>
          ) : (
            <div className="inline-flex w-full items-center justify-center gap-2 rounded-full border border-emerald-200 bg-emerald-50 px-6 py-3 text-sm font-bold text-emerald-700">
              <CheckCircle2 size={16} />
              {order.status === 'PAID' ? t('orders.payCompleted') : t('orders.unavailableAction')}
            </div>
          )}

          <Link
            to="/orders"
            className="inline-flex w-full items-center justify-center gap-2 rounded-full border border-rose-100 bg-white px-6 py-3 text-sm font-semibold text-slate-600 transition hover:border-rose-200 hover:text-rose-600"
          >
            <Clock3 size={16} />
            {t('orders.backToOrders')}
          </Link>
        </aside>
      </section>
    </div>
  );
}
