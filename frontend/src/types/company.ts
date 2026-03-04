// ========================================
// Company Lookup Types
// ========================================

export interface CompanyAddress {
  street: string;
  streetNumber: string;
  town: string;
  province: string;
  zipCode: string;
  region: string;
}

export interface CompanyLookup {
  uuid: string;
  taxCode: string;
  companyName: string;
  vatCode: string;
  activityStatus: string;
  sdiCode: string;
  registrationDate: string;
  address: CompanyAddress;
  sourceId: string;
  createdAt: string;
  updatedAt: string;
}

export interface CompanyLookupListResponse {
  lookups: CompanyLookup[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

export interface CompanyLookupListParams {
  page?: number;
  pageSize?: number;
  [key: string]: string | number | boolean | undefined;
}

export interface CompanyLookupSearchParams {
  q: string;
  page?: number;
  pageSize?: number;
  [key: string]: string | number | boolean | undefined;
}

// ========================================
// Constants for UI
// ========================================

export const ACTIVITY_STATUS_COLORS: Record<string, string> = {
  ATTIVA: 'success',
  CESSATA: 'danger',
  SOSPESA: 'warning',
};

export const ACTIVITY_STATUS_LABELS: Record<string, string> = {
  ATTIVA: 'Attiva',
  CESSATA: 'Cessata',
  SOSPESA: 'Sospesa',
};
