import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { ChevronRight, Clock3, CreditCard, Loader2, ReceiptText, XCircle } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { cancelEnrollment, getOrderDetail, listOrders } from '../api/endpoints';
import type { OrderItem } from '../types';
import { formatExactCurrency, formatLongDate } from '../utils/formatters';

const STATUS_STYLES: Record<OrderItem['status'], string> = {
  PENDING: 'border border-amber-200 bg-amber-50 text-amber-700',
  PAID: 'border border-emerald-200 bg-emerald-50 text-emerald-700',
  CLOSED: 'border border-slate-200 bg-slate-100 text-slate-500',
  REFUNDED: 'border border-sky-200 bg-sky-50 text-sky-700',
};

export default function OrdersPage() {
  const { t } = useTranslation();
  const [orders, setOrders] = useState<OrderItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [busyOrderId, setBusyOrderId] = useState<number | null>(null);
  const [actionMessage, setActionMessage] = useState<{
    tone: 'success' | 'error';
    text: string;
  } | null>(null);

  useEffect(() => {
    let active = true;

    listOrders(1, 50)
      .then((result) => {
        if (!active) {
          return;
        }

        setOrders(result.list);
        setError('');
      })
      .catch(() => {
        if (active) {
          setOrders([]);
          setError(t('orders.loadError'));
        }
      })
      .finally(() => {
        if (active) {
          setLoading(false);
        }
      });

    return () => {
      active = false;
    };
  }, [t]);

  const handleCancelOrder = async (order: OrderItem) => {
    const confirmed = window.confirm(t('orders.cancelConfirm'));
    if (!confirmed) {
      return;
    }

    setBusyOrderId(order.id);
    setActionMessage(null);

    try {
      await cancelEnrollment(order.enrollmentId);
      const nextOrder = await getOrderDetail(order.id);
      setOrders((current) =>
        current.map((item) => (item.id === order.id ? nextOrder : item)),
      );
      setActionMessage({
        tone: 'success',
        text: t('orders.cancelSuccess'),
      });
    } catch (err) {
      const errorWithResponse = err as { response?: { data?: { message?: string } } };
      setActionMessage({
        tone: 'error',
        text: errorWithResponse.response?.data?.message || t('orders.cancelError'),
      });
    } finally {
      setBusyOrderId(null);
    }
  };

  return (
    <div className="mx-auto max-w-5xl space-y-8 pb-12">
      <section className="overflow-hidden rounded-[32px] border border-rose-100 bg-[linear-gradient(135deg,#fff8f3_0%,#fff1eb_60%,#ffe3d8_100%)] px-6 py-8 shadow-[0_28px_80px_-52px_rgba(244,63,94,0.28)] lg:px-8">
        <p className="text-sm font-semibold uppercase tracking-[0.24em] text-rose-400">UAAD</p>
        <h2 className="mt-3 text-3xl font-black tracking-tight text-slate-900">
          {t('orders.title')}
        </h2>
        <p className="mt-3 max-w-2xl text-sm leading-7 text-slate-500 lg:text-base">
          {t('orders.subtitle')}
        </p>
      </section>

      {loading ? (
        <div className="space-y-4">
          {Array.from({ length: 3 }).map((_, index) => (
            <div
              key={index}
              className="rounded-[28px] border border-rose-100 bg-white p-6 shadow-sm"
            >
              <div className="h-5 w-44 animate-pulse rounded-full bg-rose-100" />
              <div className="mt-4 h-4 w-full animate-pulse rounded-full bg-rose-100" />
              <div className="mt-2 h-4 w-2/3 animate-pulse rounded-full bg-rose-100" />
            </div>
          ))}
        </div>
      ) : error ? (
        <div className="rounded-[28px] border border-amber-200 bg-amber-50 px-6 py-5 text-sm text-amber-700 shadow-sm">
          {error}
        </div>
      ) : orders.length === 0 ? (
        <div className="rounded-[32px] border border-dashed border-rose-200 bg-white px-6 py-12 text-center shadow-sm">
          <ReceiptText className="mx-auto text-rose-300" size={28} />
          <p className="mt-4 text-lg font-bold text-slate-900">{t('orders.emptyTitle')}</p>
          <p className="mt-2 text-sm leading-7 text-slate-500">{t('orders.emptyDescription')}</p>
          <Link
            to="/activities"
            className="mt-6 inline-flex items-center gap-2 rounded-full bg-rose-500 px-5 py-3 text-sm font-bold text-white transition hover:bg-rose-600"
          >
            {t('orders.browseActivities')}
            <ChevronRight size={16} />
          </Link>
        </div>
      ) : (
        <div className="space-y-4">
          {actionMessage ? (
            <div
              className={`rounded-[24px] px-5 py-4 text-sm ${
                actionMessage.tone === 'success'
                  ? 'border border-emerald-200 bg-emerald-50 text-emerald-700'
                  : 'border border-amber-200 bg-amber-50 text-amber-700'
              }`}
            >
              {actionMessage.text}
            </div>
          ) : null}
          {orders.map((order) => (
            <article
              key={order.id}
              className="rounded-[28px] border border-rose-100 bg-white p-6 shadow-sm"
            >
              <div className="flex flex-wrap items-start justify-between gap-4">
                <div className="space-y-3">
                  <div className="flex flex-wrap items-center gap-3">
                    <span
                      className={`rounded-full border px-3 py-1 text-xs font-semibold uppercase tracking-[0.18em] ${STATUS_STYLES[order.status]}`}
                    >
                      {t(`orders.status.${order.status}`)}
                    </span>
                    <p className="text-xs font-semibold uppercase tracking-[0.2em] text-slate-400">
                      {order.orderNo}
                    </p>
                  </div>
                  <div className="space-y-1">
                    <p className="text-2xl font-black text-slate-900">
                      {formatExactCurrency(order.amount)}
                    </p>
                    <p className="text-sm text-slate-500">
                      {t('orders.createdAt', { time: formatLongDate(order.createdAt) })}
                    </p>
                  </div>
                </div>

                <div className="rounded-2xl border border-slate-200 bg-[#fff8f3] px-4 py-3 text-sm text-slate-500">
                  {order.status === 'PAID' ? (
                    <p>{t('orders.paidAt', { time: formatLongDate(order.paidAt || order.updatedAt) })}</p>
                  ) : (
                    <p>{t('orders.expireAt', { time: formatLongDate(order.expiredAt) })}</p>
                  )}
                </div>
              </div>

              <div className="mt-5 flex flex-wrap justify-end gap-3">
                {order.status === 'PENDING' ? (
                  <>
                    <button
                      type="button"
                      onClick={() => void handleCancelOrder(order)}
                      disabled={busyOrderId === order.id}
                      className="inline-flex items-center gap-2 rounded-full border border-rose-200 bg-white px-4 py-2 text-sm font-bold text-rose-600 transition hover:border-rose-300 hover:bg-rose-50 disabled:cursor-not-allowed disabled:opacity-60"
                    >
                      {busyOrderId === order.id ? (
                        <Loader2 size={16} className="animate-spin" />
                      ) : (
                        <XCircle size={16} />
                      )}
                      {busyOrderId === order.id ? t('orders.cancelling') : t('orders.cancelPending')}
                    </button>
                    <Link
                      to={`/orders/${order.id}`}
                      className="inline-flex items-center gap-2 rounded-full bg-rose-500 px-4 py-2 text-sm font-bold text-white transition hover:bg-rose-600"
                    >
                      <CreditCard size={16} />
                      {t('orders.payNow')}
                    </Link>
                  </>
                ) : null}
                <Link
                  to={`/orders/${order.id}`}
                  className="inline-flex items-center gap-2 rounded-full border border-rose-100 bg-white px-4 py-2 text-sm font-semibold text-slate-600 transition hover:border-rose-200 hover:text-rose-600"
                >
                  <Clock3 size={16} />
                  {t('orders.viewDetail')}
                </Link>
              </div>
            </article>
          ))}
        </div>
      )}
    </div>
  );
}
