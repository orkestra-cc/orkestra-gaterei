// MFA — multi-factor authentication types mirroring the backend contracts
// at `/v1/auth/mfa/*` and `/v1/auth/me/mfa`. Kept here (not colocated with
// mfaApi) so UI components can import types without pulling the slice.

export type MfaStatus = 'none' | 'pending' | 'active';
export type MfaFactorType = 'totp' | 'webauthn' | '';

export interface MfaStatusResponse {
  status: MfaStatus;
  type?: MfaFactorType;
  backupCodesRemaining: number;
  // True when the user's role obligates MFA enrollment. Computed server-side
  // from the system role + org memberships via RoleRequiresMFA.
  requiresMfa: boolean;
  // Deadline by which a user whose role requires MFA must enroll. Absent
  // before the grace clock starts (first privileged login).
  graceExpiresAt?: string | null;
}

export interface MfaEnrollBeginResponse {
  challengeId: string;
  secret: string;
  provisioningUri: string;
}

export interface MfaEnrollConfirmInput {
  challengeId: string;
  code: string;
}

export interface MfaEnrollConfirmResponse {
  success: boolean;
  backupCodes: string[];
}

export interface MfaVerifyInput {
  code: string;
  useBackup?: boolean;
}

export interface MfaVerifyResponse {
  success: boolean;
  accessToken: string;
  tokenType: string;
  expiresIn: number;
}

export interface MfaLoginVerifyInput {
  challengeId: string;
  code: string;
  useBackup?: boolean;
}

import type { BackendUser } from 'store/api/authApi';

export interface MfaLoginVerifyResponse {
  success: boolean;
  accessToken: string;
  tokenType: string;
  expiresIn: number;
  sessionId: string;
  deviceId?: string;
  user: BackendUser;
}
