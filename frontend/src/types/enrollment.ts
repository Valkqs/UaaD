export type EnrollmentStatus = 'QUEUING' | 'SUCCESS' | 'FAILED' | 'CANCELLED';

export interface CreateEnrollmentResult {
  code: number;
  message: string;
  status: EnrollmentStatus;
  enrollmentId?: number;
  activityId?: number;
  queuePosition?: number;
  estimatedWaitSeconds?: number;
  stockRemaining?: number;
  orderNo?: string;
}

export interface EnrollmentStatusDetail {
  enrollmentId: number;
  activityId: number;
  status: EnrollmentStatus;
  submittedAt: string;
  activityTitle?: string;
  orderNo?: string;
  finalizedAt?: string;
  queuePosition?: number;
  estimatedWaitSeconds?: number;
}

export interface EnrollmentListItem {
  id: number;
  userId: number;
  activityId: number;
  status: EnrollmentStatus;
  queuePosition?: number | null;
  enrolledAt: string;
  finalizedAt?: string | null;
}

export interface EnrollmentListResult {
  list: EnrollmentListItem[];
  total: number;
  page: number;
  pageSize: number;
}

export interface EnrollmentStatusItem {
  enrollmentId: number;
  activityId: number;
  activityTitle?: string;
  status: EnrollmentStatus;
  queuePosition?: number;
  estimatedWaitSeconds?: number;
  submittedAt?: string;
  finalizedAt?: string;
  orderNo?: string;
}

export interface CancelEnrollmentResult {
  enrollmentId: number;
  status: EnrollmentStatus;
}
