import api from '../axios';
import type {
  CancelEnrollmentResult,
  CreateEnrollmentResult,
  EnrollmentListItem,
  EnrollmentListResult,
  EnrollmentStatus,
  EnrollmentStatusItem,
} from '../../types';

interface ApiResponse<T> {
  code: number;
  message: string;
  data: T;
}

interface BackendPaginatedPayload<T> {
  code: number;
  message: string;
  data: {
    list: T[];
    total: number;
    page: number;
    page_size: number;
  };
}

interface BackendCreateEnrollmentData {
  status: EnrollmentStatus;
  queue_position: number;
  enrollment_id?: number;
  activity_id?: number;
  order_no?: string;
  estimated_wait_seconds?: number;
  stock_remaining?: number;
}

interface BackendEnrollmentStatusData {
  enrollment_id: number;
  activity_id: number;
  activity_title?: string;
  status: EnrollmentStatus;
  queue_position?: number;
  estimated_wait_seconds?: number;
  submitted_at?: string;
  finalized_at?: string;
  order_no?: string;
}

interface BackendCancelEnrollmentData {
  enrollment_id: number;
  status: EnrollmentStatus;
}

interface BackendEnrollmentListItem {
  id: number;
  user_id: number;
  activity_id: number;
  status: EnrollmentStatus;
  queue_position?: number | null;
  enrolled_at: string;
  finalized_at?: string | null;
}

function normalizeEnrollmentListItem(item: BackendEnrollmentListItem): EnrollmentListItem {
  return {
    id: item.id,
    userId: item.user_id,
    activityId: item.activity_id,
    status: item.status,
    queuePosition: item.queue_position,
    enrolledAt: item.enrolled_at,
    finalizedAt: item.finalized_at,
  };
}

export async function createEnrollment(activityId: number): Promise<CreateEnrollmentResult> {
  const response = await api.post<ApiResponse<BackendCreateEnrollmentData>>('/enrollments', {
    activity_id: activityId,
  });

  const data = response.data.data;
  return {
    code: response.data.code,
    message: response.data.message,
    status: data.status,
    queuePosition: data.queue_position,
    enrollmentId: data.enrollment_id,
    orderNo: data.order_no,
    estimatedWaitSeconds: data.estimated_wait_seconds,
    stockRemaining: data.stock_remaining,
  };
}

export async function getEnrollmentStatus(enrollmentId: number): Promise<EnrollmentStatusItem> {
  const response = await api.get<ApiResponse<BackendEnrollmentStatusData>>(
    `/enrollments/${enrollmentId}/status`,
  );
  const data = response.data.data;

  return {
    enrollmentId: data.enrollment_id,
    activityId: data.activity_id,
    activityTitle: data.activity_title,
    status: data.status,
    queuePosition: data.queue_position,
    estimatedWaitSeconds: data.estimated_wait_seconds,
    submittedAt: data.submitted_at,
    finalizedAt: data.finalized_at,
    orderNo: data.order_no,
  };
}

export async function cancelEnrollment(enrollmentId: number): Promise<CancelEnrollmentResult> {
  const response = await api.post<ApiResponse<BackendCancelEnrollmentData>>(
    `/enrollments/${enrollmentId}/cancel`,
  );
  const data = response.data.data;

  return {
    enrollmentId: data.enrollment_id,
    status: data.status,
  };
}

export async function listMyEnrollments(page = 1, pageSize = 20): Promise<EnrollmentListResult> {
  const response = await api.get<BackendPaginatedPayload<BackendEnrollmentListItem>>('/enrollments', {
    params: {
      page,
      page_size: pageSize,
    },
  });

  return {
    list: response.data.data.list.map(normalizeEnrollmentListItem),
    total: response.data.data.total,
    page: response.data.data.page,
    pageSize: response.data.data.page_size,
  };
}
