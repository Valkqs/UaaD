import { useEffect, useState } from 'react';
import { Link, useLocation, useParams } from 'react-router-dom';
import { CheckCircle2, Loader2, TicketX, XCircle } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { cancelEnrollment, findOrderByOrderNo, getEnrollmentStatus } from '../api/endpoints';
import type { EnrollmentStatusItem, OrderItem } from '../types';
import { formatLongDate } from '../utils/formatters';

export default function EnrollStatusPage() {
  const { t } = useTranslation();
  const location = useLocation();
  const { id } = useParams();
  const enrollmentId = Number(id);
  const [status, setStatus] = useState<EnrollmentStatusItem | null>(null);
  const [order, setOrder] = useState<OrderItem | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [actionMessage, setActionMessage] = useState<{
    tone: 'success' | 'error';
    text: string;
  } | null>(null);
  const [cancelling, setCancelling] = useState(false);

  useEffect(() => {
    if (!Number.isFinite(enrollmentId)) {
      setError(t('enrollStatus.invalidId'));
      setLoading(false);
      return;
    }

    let active = true;
    let timer: number | undefined;

    const load = async () => {
      try {
        const nextStatus = await getEnrollmentStatus(enrollmentId);
        if (!active) {
          return;
        }

        setStatus(nextStatus);
        setError('');

        if (nextStatus.orderNo) {
          const matchedOrder = await findOrderByOrderNo(nextStatus.orderNo).catch(() => null);
          if (active) {
            setOrder(matchedOrder);
          }
        }

        if (nextStatus.status === 'QUEUING') {
          timer = window.setTimeout(() => {
            void load();
          }, 4000);
        }
      } catch {
        if (active) {
          setError(t('enrollStatus.loadError'));
        }
      } finally {
        if (active) {
          setLoading(false);
        }
      }
    };

    void load();

    return () => {
      active = false;
      if (timer) {
        window.clearTimeout(timer);
      }
    };
  }, [enrollmentId, t]);

  const handleCancel = async () => {
    if (!status || cancelling) {
      return;
    }

    const confirmed = window.confirm(t('enrollStatus.cancelConfirm'));
    if (!confirmed) {
      return;
    }

    setCancelling(true);
    setActionMessage(null);

    try {
      await cancelEnrollment(status.enrollmentId);
      const nextStatus = await getEnrollmentStatus(status.enrollmentId);
      setStatus(nextStatus);
      setOrder(null);
      setActionMessage({
        tone: 'success',
        text: t('enrollStatus.cancelSuccess'),
      });
    } catch (err) {
      const errorWithResponse = err as { response?: { data?: { message?: string } } };
      setActionMessage({
        tone: 'error',
        text: errorWithResponse.response?.data?.message || t('enrollStatus.cancelError'),
      });
    } finally {
      setCancelling(false);
    }
  };

  if (loading) {
    return (
      <div className="mx-auto max-w-3xl rounded-[32px] border border-rose-100 bg-white px-6 py-14 text-center shadow-sm">
        <Loader2 className="mx-auto animate-spin text-rose-500" size={28} />
        <p className="mt-4 text-sm text-slate-500">{t('enrollStatus.loading')}</p>
      </div>
    );
  }

  if (error || !status) {
    return (
      <div className="mx-auto flex min-h-[55vh] max-w-3xl flex-col items-center justify-center gap-5 px-4 text-center">
        <p className="text-2xl font-bold text-slate-900">{t('public.errorTitle')}</p>
        <p className="text-slate-500">{error || t('enrollStatus.unavailable')}</p>
        <Link
          to="/activities"
          className="rounded-full bg-rose-500 px-5 py-2 text-sm font-semibold text-white transition hover:bg-rose-600"
        >
          {t('activityDetail.backToList')}
        </Link>
      </div>
    );
  }

  const activityTitle =
    status.activityTitle ||
    (location.state as { activityTitle?: string } | null)?.activityTitle ||
    t('enrollStatus.defaultActivityTitle');

  return (
    <div className="mx-auto max-w-3xl space-y-6 pb-12">
      <section className="overflow-hidden rounded-[32px] border border-rose-100 bg-[linear-gradient(135deg,#fff8f3_0%,#fff1eb_60%,#ffe3d8_100%)] px-6 py-8 shadow-[0_28px_80px_-52px_rgba(244,63,94,0.28)]">
        <p className="text-sm font-semibold uppercase tracking-[0.24em] text-rose-400">UAAD</p>
        <h2 className="mt-3 text-3xl font-black tracking-tight text-slate-900">
          {t('enrollStatus.title')}
        </h2>
        <p className="mt-3 text-sm leading-7 text-slate-500 lg:text-base">{activityTitle}</p>
      </section>

      <section className="rounded-[32px] border border-rose-100 bg-white p-6 shadow-sm lg:p-8">
        {status.status === 'QUEUING' ? (
          <div className="text-center">
            <Loader2 className="mx-auto animate-spin text-rose-500" size={28} />
            <h3 className="mt-4 text-2xl font-black text-slate-900">{t('enrollStatus.queueTitle')}</h3>
            <p className="mt-3 text-sm leading-7 text-slate-500">{t('enrollStatus.queueDescription')}</p>
            {actionMessage ? (
              <div
                className={`mt-5 rounded-2xl px-4 py-3 text-sm ${
                  actionMessage.tone === 'success'
                    ? 'border border-emerald-200 bg-emerald-50 text-emerald-700'
                    : 'border border-amber-200 bg-amber-50 text-amber-700'
                }`}
              >
                {actionMessage.text}
              </div>
            ) : null}
            <div className="mt-6 grid gap-4 sm:grid-cols-2">
              <div className="rounded-[24px] border border-rose-100 bg-[#fff8f3] p-5">
                <p className="text-sm font-semibold text-slate-400">{t('enrollStatus.queuePosition')}</p>
                <p className="mt-2 text-3xl font-black text-slate-900">
                  #{status.queuePosition ?? '--'}
                </p>
              </div>
              <div className="rounded-[24px] border border-rose-100 bg-[#fff8f3] p-5">
                <p className="text-sm font-semibold text-slate-400">{t('enrollStatus.estimatedWait')}</p>
                <p className="mt-2 text-3xl font-black text-slate-900">
                  {status.estimatedWaitSeconds ?? '--'}s
                </p>
              </div>
            </div>
            <button
              type="button"
              onClick={() => void handleCancel()}
              disabled={cancelling}
              className="mt-6 inline-flex items-center justify-center gap-2 rounded-full border border-rose-200 bg-white px-5 py-3 text-sm font-bold text-rose-600 transition hover:border-rose-300 hover:bg-rose-50 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {cancelling ? <Loader2 size={16} className="animate-spin" /> : <XCircle size={16} />}
              {cancelling ? t('enrollStatus.cancelling') : t('enrollStatus.cancelQueue')}
            </button>
          </div>
        ) : status.status === 'SUCCESS' ? (
          <div className="text-center">
            <CheckCircle2 className="mx-auto text-emerald-400" size={34} />
            <h3 className="mt-4 text-2xl font-black text-slate-900">{t('enrollStatus.successTitle')}</h3>
            <p className="mt-3 text-sm leading-7 text-slate-500">{t('enrollStatus.successDescription')}</p>
            <div className="mt-6 space-y-3 rounded-[28px] border border-rose-100 bg-[#fff8f3] p-5 text-left">
              <p className="text-sm text-slate-600">
                {t('enrollStatus.enrollmentCode')}: <span className="font-semibold text-slate-900">{status.enrollmentId}</span>
              </p>
              {status.orderNo ? (
                <p className="text-sm text-slate-600">
                  {t('enrollStatus.orderCode')}: <span className="font-semibold text-slate-900">{status.orderNo}</span>
                </p>
              ) : null}
              {status.finalizedAt ? (
                <p className="text-sm text-slate-600">
                  {t('enrollStatus.finalizedAt')}: <span className="font-semibold text-slate-900">{formatLongDate(status.finalizedAt)}</span>
                </p>
              ) : null}
            </div>
            <div className="mt-6 flex flex-wrap justify-center gap-3">
              {order ? (
                <Link
                  to={`/orders/${order.id}`}
                  className="rounded-full bg-rose-500 px-5 py-3 text-sm font-bold text-white transition hover:bg-rose-600"
                >
                  {t('enrollStatus.goToPay')}
                </Link>
              ) : (
                <Link
                  to="/orders"
                  className="rounded-full bg-rose-500 px-5 py-3 text-sm font-bold text-white transition hover:bg-rose-600"
                >
                  {t('enrollStatus.checkOrders')}
                </Link>
              )}
              <Link
                to="/activities"
                className="rounded-full border border-rose-100 bg-white px-5 py-3 text-sm font-semibold text-slate-600 transition hover:border-rose-200 hover:text-rose-600"
              >
                {t('orders.browseActivities')}
              </Link>
            </div>
          </div>
        ) : (
          <div className="text-center">
            <TicketX className="mx-auto text-rose-300" size={32} />
            <h3 className="mt-4 text-2xl font-black text-slate-900">{t('enrollStatus.failedTitle')}</h3>
            <p className="mt-3 text-sm leading-7 text-slate-500">{t('enrollStatus.failedDescription')}</p>
            <Link
              to="/activities"
              className="mt-6 inline-flex rounded-full bg-rose-500 px-5 py-3 text-sm font-bold text-white transition hover:bg-rose-600"
            >
              {t('orders.browseActivities')}
            </Link>
          </div>
        )}
      </section>
    </div>
  );
}
