// ========================================
// Billing Types - Italian Electronic Invoicing (FatturaPA)
// ========================================

// ========================================
// Enums
// ========================================

export type InvoiceDirection = 'issued' | 'received';

export type InvoiceStatus =
  | 'draft'
  | 'pending'
  | 'sent'
  | 'delivered'
  | 'rejected'
  | 'accepted'
  | 'paid'
  | 'cancelled';

export type SDIStatus = '' | 'RC' | 'NS' | 'MC' | 'NE' | 'DT' | 'AT';

export type DocumentType =
  | 'TD01' // Fattura
  | 'TD02' // Acconto/Anticipo su fattura
  | 'TD03' // Acconto/Anticipo su parcella
  | 'TD04' // Nota di Credito
  | 'TD05' // Nota di Debito
  | 'TD06' // Parcella
  | 'TD07' // Fattura semplificata
  | 'TD08' // Nota di credito semplificata
  | 'TD09' // Nota di debito semplificata
  | 'TD10' // Fattura per acquisto intracomunitario beni
  | 'TD11' // Fattura per acquisto intracomunitario servizi
  | 'TD12' // Documento riepilogo fatture acquisto
  | 'TD16' // Integrazione fattura reverse charge interno
  | 'TD17' // Integrazione/autofattura per acquisto servizi dall'estero
  | 'TD18' // Integrazione per acquisto beni intracomunitari
  | 'TD19' // Integrazione/autofattura per acquisto di beni ex art.17 c.2 DPR 633/72
  | 'TD20' // Autofattura per regolarizzazione e integrazione delle fatture
  | 'TD21' // Autofattura per splafonamento
  | 'TD22' // Estrazione beni da Deposito IVA
  | 'TD23' // Estrazione beni da Deposito IVA con versamento dell'IVA
  | 'TD24' // Fattura differita di cui all'art. 21, comma 4, lett. a)
  | 'TD25' // Fattura differita di cui all'art. 21, comma 4, terzo periodo lett. b)
  | 'TD26' // Cessione di beni ammortizzabili e per passaggi interni
  | 'TD27' // Fattura per autoconsumo o per cessioni gratuite senza rivalsa
  | 'TD28'; // Acquisti da San Marino con IVA

export type RegimeFiscale =
  | 'RF01' // Ordinario
  | 'RF02' // Contribuenti minimi
  | 'RF04' // Agricoltura e attività connesse e pesca
  | 'RF05' // Vendita sali e tabacchi
  | 'RF06' // Commercio fiammiferi
  | 'RF07' // Editoria
  | 'RF08' // Gestione servizi telefonia pubblica
  | 'RF09' // Rivendita documenti di trasporto pubblico e di sosta
  | 'RF10' // Intrattenimenti, giochi e altre attività
  | 'RF11' // Agenzie viaggi e turismo
  | 'RF12' // Agriturismo
  | 'RF13' // Vendite a domicilio
  | 'RF14' // Rivendita beni usati
  | 'RF15' // Agenzie di vendite all'asta
  | 'RF16' // IVA per cassa P.A.
  | 'RF17' // IVA per cassa
  | 'RF18' // Altro
  | 'RF19'; // Regime forfettario

export type PaymentMethod =
  | 'MP01' // Contanti
  | 'MP02' // Assegno
  | 'MP03' // Assegno circolare
  | 'MP04' // Contanti presso Tesoreria
  | 'MP05' // Bonifico
  | 'MP06' // Vaglia cambiario
  | 'MP07' // Bollettino bancario
  | 'MP08' // Carta di pagamento
  | 'MP09' // RID
  | 'MP10' // RID utenze
  | 'MP11' // RID veloce
  | 'MP12' // Riba
  | 'MP13' // MAV
  | 'MP14' // Quietanza erario stato
  | 'MP15' // Giroconto su conti di contabilità speciale
  | 'MP16' // Domiciliazione bancaria
  | 'MP17' // Domiciliazione postale
  | 'MP18' // Bollettino di c/c postale
  | 'MP19' // SEPA Direct Debit
  | 'MP20' // SEPA Direct Debit CORE
  | 'MP21' // SEPA Direct Debit B2B
  | 'MP22' // Trattenuta su somme già riscosse
  | 'MP23'; // PagoPA

export type PaymentCondition =
  | 'TP01' // Pagamento a rate
  | 'TP02' // Pagamento completo
  | 'TP03'; // Anticipo

export type VATNature =
  | 'N1' // Escluse ex art.15
  | 'N2' // Non soggette (obsoleto)
  | 'N2.1' // Non soggette ad IVA - artt. da 7 a 7-septies DPR 633/72
  | 'N2.2' // Non soggette - altri casi
  | 'N3' // Non imponibili (obsoleto)
  | 'N3.1' // Non imponibili - esportazioni
  | 'N3.2' // Non imponibili - cessioni intracomunitarie
  | 'N3.3' // Non imponibili - cessioni verso San Marino
  | 'N3.4' // Non imponibili - operazioni assimilate alle cessioni all'esportazione
  | 'N3.5' // Non imponibili - dichiarazioni d'intento
  | 'N3.6' // Non imponibili - altre operazioni non imponibili
  | 'N4' // Esenti
  | 'N5' // Regime del margine / IVA non esposta
  | 'N6' // Inversione contabile (obsoleto)
  | 'N6.1' // Inversione contabile - cessione rottami
  | 'N6.2' // Inversione contabile - cessione oro e argento
  | 'N6.3' // Inversione contabile - subappalto edilizia
  | 'N6.4' // Inversione contabile - cessione fabbricati
  | 'N6.5' // Inversione contabile - cessione telefoni cellulari
  | 'N6.6' // Inversione contabile - cessione prodotti elettronici
  | 'N6.7' // Inversione contabile - prestazioni comparto edile e settori connessi
  | 'N6.8' // Inversione contabile - operazioni settore energetico
  | 'N6.9' // Inversione contabile - altri casi
  | 'N7'; // IVA assolta in altro stato UE

export type NotificationType = 'RC' | 'NS' | 'MC' | 'NE' | 'DT' | 'AT';

export type NEOutcome = 'EC01' | 'EC02'; // EC01 = Accepted, EC02 = Rejected

export type UnitOfMeasure =
  | 'PZ' // Pezzo
  | 'KG' // Chilogrammo
  | 'LT' // Litro
  | 'MT' // Metro
  | 'MQ' // Metro quadrato
  | 'MC' // Metro cubo
  | 'H' // Ora
  | 'GG' // Giorno
  | 'MESE' // Mese
  | 'ANNO' // Anno
  | '%'; // Percentuale

// ========================================
// Customer Types
// ========================================

export interface Customer {
  id: string;
  fiscalIdCountry: string;
  fiscalIdCode: string;
  codiceFiscale?: string;
  isCompany: boolean;
  denomination?: string;
  name?: string;
  surname?: string;
  address: string;
  city: string;
  province?: string;
  postalCode: string;
  country: string;
  email?: string;
  pec?: string;
  phone?: string;
  codiceDestinatario?: string;
  pecDestinatario?: string;
  isPA: boolean;
  codiceUfficio?: string;
  riferimentoAmm?: string;
  convenzioneNumero?: string;
  notes?: string;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
  deletedAt?: string;
  createdBy?: string;
}

export interface CreateCustomerInput {
  fiscalIdCountry: string;
  fiscalIdCode: string;
  codiceFiscale?: string;
  isCompany: boolean;
  denomination?: string;
  name?: string;
  surname?: string;
  address: string;
  city: string;
  province?: string;
  postalCode: string;
  country: string;
  email?: string;
  pec?: string;
  phone?: string;
  codiceDestinatario?: string;
  pecDestinatario?: string;
  isPA?: boolean;
  codiceUfficio?: string;
  notes?: string;
}

export interface UpdateCustomerInput {
  denomination?: string;
  name?: string;
  surname?: string;
  address?: string;
  city?: string;
  province?: string;
  postalCode?: string;
  email?: string;
  pec?: string;
  phone?: string;
  codiceDestinatario?: string;
  pecDestinatario?: string;
  notes?: string;
}

export interface CustomerListResponse {
  customers: Customer[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

export interface CustomerListParams {
  search?: string;
  isActive?: boolean;
  isPA?: boolean;
  page?: number;
  pageSize?: number;
  [key: string]: string | number | boolean | undefined;
}

// ========================================
// Supplier Types
// ========================================

export interface Supplier {
  id: string;
  fiscalIdCountry: string;
  fiscalIdCode: string;
  codiceFiscale?: string;
  isCompany: boolean;
  denomination?: string;
  name?: string;
  surname?: string;
  regimeFiscale?: RegimeFiscale;
  address: string;
  city: string;
  province?: string;
  postalCode: string;
  country: string;
  email?: string;
  pec?: string;
  phone?: string;
  iban?: string;
  bic?: string;
  notes?: string;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
  deletedAt?: string;
  createdBy?: string;
}

export interface CreateSupplierInput {
  fiscalIdCountry: string;
  fiscalIdCode: string;
  codiceFiscale?: string;
  isCompany: boolean;
  denomination?: string;
  name?: string;
  surname?: string;
  regimeFiscale?: RegimeFiscale;
  address: string;
  city: string;
  province?: string;
  postalCode: string;
  country: string;
  email?: string;
  pec?: string;
  phone?: string;
  iban?: string;
  bic?: string;
  notes?: string;
}

export interface UpdateSupplierInput {
  denomination?: string;
  name?: string;
  surname?: string;
  regimeFiscale?: RegimeFiscale;
  address?: string;
  city?: string;
  province?: string;
  postalCode?: string;
  email?: string;
  pec?: string;
  phone?: string;
  iban?: string;
  bic?: string;
  notes?: string;
}

export interface SupplierListResponse {
  suppliers: Supplier[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

export interface SupplierListParams {
  search?: string;
  isActive?: boolean;
  page?: number;
  pageSize?: number;
  [key: string]: string | number | boolean | undefined;
}

// ========================================
// Party Data (Embedded snapshot in Invoice)
// ========================================

export interface PartyData {
  fiscalIdCountry: string;
  fiscalIdCode: string;
  codiceFiscale?: string;
  isCompany: boolean;
  denomination?: string;
  name?: string;
  surname?: string;
  regimeFiscale?: RegimeFiscale;
  address: string;
  city: string;
  province?: string;
  postalCode: string;
  country: string;
  email?: string;
  pec?: string;
  phone?: string;
  codiceDestinatario?: string;
  pecDestinatario?: string;
}

// ========================================
// Invoice Types
// ========================================

export interface LineDiscount {
  type: 'SC' | 'MG'; // SC=sconto (discount), MG=maggiorazione (surcharge)
  percentage?: number;
  amount?: number;
}

export interface InvoiceLine {
  lineNumber: number;
  description: string;
  quantity: number;
  unitOfMeasure?: UnitOfMeasure;
  unitPrice: number;
  discounts?: LineDiscount[];
  totalPrice: number;
  vatRate: number;
  vatNature?: VATNature;
  vatAmount: number;
  administrativeRef?: string;
  productCode?: string;
  startDate?: string;
  endDate?: string;
}

export interface VATSummaryLine {
  vatRate: number;
  vatNature?: VATNature;
  taxableAmount: number;
  vatAmount: number;
  vatExigibility?: 'I' | 'D' | 'S'; // I=immediata, D=differita, S=split payment
  normativeRef?: string;
}

export interface PaymentInstallment {
  dueDate: string;
  amount: number;
  paid: boolean;
  paidAt?: string;
}

export interface PaymentTerms {
  condition: PaymentCondition;
  paymentMethod: PaymentMethod;
  iban?: string;
  bic?: string;
  dueDate?: string;
  installments?: PaymentInstallment[];
}

export interface RelatedDocument {
  type: string; // ordine, contratto, convenzione, ricezione, fattura collegata
  id?: string;
  date?: string;
  number?: string;
  cig?: string; // Codice Identificativo Gara
  cup?: string; // Codice Unico Progetto
  lineRef?: string;
}

export interface InvoiceAttachment {
  name: string;
  description?: string;
  format?: string; // MIME type
}

export interface Invoice {
  id: string;
  direction: InvoiceDirection;
  documentType: DocumentType;
  sdiIdentifier?: string;
  openApiUuid?: string;
  progressivoInvio?: string;
  number: string;
  date: string;
  currency: string;
  supplierId?: string;
  customerId?: string;
  cedentePrestatore?: PartyData;
  cessionarioCommittente?: PartyData;
  lines: InvoiceLine[];
  vatSummary: VATSummaryLine[];
  totalTaxableAmount: number;
  totalVatAmount: number;
  totalAmount: number;
  rounding?: number;
  paymentTerms?: PaymentTerms;
  status: InvoiceStatus;
  sdiStatus?: SDIStatus;
  legalStorageEnabled: boolean;
  signatureEnabled: boolean;
  preservedDocumentId?: string;
  relatedDocuments?: RelatedDocument[];
  causale?: string[];
  attachments?: InvoiceAttachment[];
  internalNotes?: string;
  createdAt: string;
  updatedAt: string;
  deletedAt?: string;
  createdBy: string;
  sentAt?: string;
  sentBy?: string;
}

export interface InvoiceSummary {
  id: string;
  direction: InvoiceDirection;
  documentType: DocumentType;
  number: string;
  date: string;
  partyName: string;
  totalAmount: number;
  status: InvoiceStatus;
  sdiStatus?: SDIStatus;
  sdiIdentifier?: string;
  createdAt: string;
}

export interface CreateInvoiceLineInput {
  description: string;
  quantity: number;
  unitOfMeasure?: UnitOfMeasure;
  unitPrice: number;
  vatRate: number;
  vatNature?: VATNature;
  discounts?: LineDiscount[];
  productCode?: string;
  startDate?: string;
  endDate?: string;
}

export interface CreatePaymentTermsInput {
  condition: PaymentCondition;
  paymentMethod: PaymentMethod;
  iban?: string;
  bic?: string;
  dueDate?: string;
}

export interface CreateInvoiceInput {
  documentType: DocumentType;
  number: string;
  date: string;
  currency?: string;
  customerId?: string;
  lines: CreateInvoiceLineInput[];
  paymentTerms?: CreatePaymentTermsInput;
  relatedDocuments?: RelatedDocument[];
  causale?: string[];
  internalNotes?: string;
  legalStorageEnabled?: boolean;
  signatureEnabled?: boolean;
}

export interface UpdateInvoiceInput {
  number?: string;
  date?: string;
  lines?: CreateInvoiceLineInput[];
  paymentTerms?: CreatePaymentTermsInput;
  relatedDocuments?: RelatedDocument[];
  causale?: string[];
  internalNotes?: string;
}

export interface InvoiceListResponse {
  invoices: InvoiceSummary[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

export interface InvoiceListParams {
  direction?: InvoiceDirection;
  status?: InvoiceStatus;
  sdiStatus?: SDIStatus;
  customerId?: string;
  supplierId?: string;
  fromDate?: string;
  toDate?: string;
  search?: string;
  documentType?: DocumentType;
  page?: number;
  pageSize?: number;
  limit?: number;
  [key: string]: string | number | boolean | undefined;
}

export interface SendInvoiceResponse {
  invoiceId: string;
  openApiUuid: string;
  sdiIdentifier?: string;
  status: InvoiceStatus;
  message: string;
}

// ========================================
// Notification Types
// ========================================

export interface SDINotification {
  id: string;
  invoiceUuid: string;
  openApiUuid?: string;
  notificationType: NotificationType;
  notificationDate: string;
  sdiIdentifier?: string;
  progressivoInvio?: string;
  description?: string;
  errorCode?: string;
  errorDescription?: string;
  errorSuggestion?: string;
  outcome?: NEOutcome;
  outcomeReason?: string;
  mcDescription?: string;
  nextAttemptDate?: string;
  preservedDocumentId?: string;
  processed: boolean;
  processedAt?: string;
  processedBy?: string;
  createdAt: string;
}

export interface NotificationSummary {
  totalCount: number;
  unprocessedCount: number;
  positiveCount: number;
  negativeCount: number;
  pendingAction: number;
  // Per-type counts used by dashboard components
  total: number;
  unprocessed: number;
  RC: number;
  NS: number;
  MC: number;
  NE: number;
  DT: number;
  AT: number;
}

export interface NotificationListResponse {
  notifications: SDINotification[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

export interface NotificationListParams {
  invoiceUuid?: string;
  notificationType?: NotificationType;
  processed?: boolean;
  fromDate?: string;
  toDate?: string;
  page?: number;
  pageSize?: number;
  limit?: number;
  [key: string]: string | number | boolean | undefined;
}

// ========================================
// Statistics Types
// ========================================

export interface BillingStats {
  issuedTotal: number;
  issuedDraft: number;
  issuedSent: number;
  issuedDelivered: number;
  issuedRejected: number;
  issuedAmount: number;
  receivedTotal: number;
  receivedPending: number;
  receivedAccepted: number;
  receivedRejected: number;
  receivedAmount: number;
  unprocessedNotifications: number;
  pendingActions: number;
  periodStart: string;
  periodEnd: string;
}

export interface BillingStatsParams {
  fromDate?: string;
  toDate?: string;
  [key: string]: string | undefined;
}

// ========================================
// Preserved Documents
// ========================================

export interface PreservedDocument {
  uuid: string;
  status: 'to_be_stored' | 'sent' | 'stored' | 'error';
  receiptTimestamp?: string;
  weight?: number;
  objectId?: string;
  objectType?: string;
}

// ========================================
// Import Invoice
// ========================================

export interface ImportInvoiceInput {
  invoice: string; // Base64-encoded FatturaPA XML content
  invoice_file_name?: string;
  sdi_id?: string;
  metadata?: Record<string, unknown>;
}

export interface ImportInvoiceResponse {
  uuids: string[];
  count: number;
  message: string;
}

// ========================================
// Helper Types for UI
// ========================================

export interface DocumentTypeOption {
  value: DocumentType;
  label: string;
  description: string;
}

export interface RegimeFiscaleOption {
  value: RegimeFiscale;
  label: string;
}

export interface PaymentMethodOption {
  value: PaymentMethod;
  label: string;
}

export interface VATNatureOption {
  value: VATNature;
  label: string;
}

// ========================================
// Constants for UI Labels
// ========================================

export const DOCUMENT_TYPE_LABELS: Record<DocumentType, string> = {
  TD01: 'Fattura',
  TD02: 'Acconto/Anticipo su fattura',
  TD03: 'Acconto/Anticipo su parcella',
  TD04: 'Nota di Credito',
  TD05: 'Nota di Debito',
  TD06: 'Parcella',
  TD07: 'Fattura semplificata',
  TD08: 'Nota di credito semplificata',
  TD09: 'Nota di debito semplificata',
  TD10: 'Fattura acquisto intracomunitario beni',
  TD11: 'Fattura acquisto intracomunitario servizi',
  TD12: 'Documento riepilogo fatture acquisto',
  TD16: 'Integrazione fattura reverse charge',
  TD17: 'Autofattura acquisto servizi estero',
  TD18: 'Integrazione acquisto beni intracomunitari',
  TD19: 'Integrazione acquisto beni art.17',
  TD20: 'Autofattura regolarizzazione',
  TD21: 'Autofattura splafonamento',
  TD22: 'Estrazione beni da Deposito IVA',
  TD23: 'Estrazione beni da Deposito IVA con IVA',
  TD24: 'Fattura differita (art.21 c.4 lett.a)',
  TD25: 'Fattura differita (art.21 c.4 lett.b)',
  TD26: 'Cessione beni ammortizzabili',
  TD27: 'Autoconsumo/cessioni gratuite',
  TD28: 'Acquisti da San Marino con IVA',
};

export const INVOICE_STATUS_LABELS: Record<InvoiceStatus, string> = {
  draft: 'Bozza',
  pending: 'In attesa',
  sent: 'Inviata',
  delivered: 'Consegnata',
  rejected: 'Scartata',
  accepted: 'Accettata',
  paid: 'Pagata',
  cancelled: 'Annullata',
};

export const SDI_STATUS_LABELS: Record<SDIStatus, string> = {
  '': 'Nessuna',
  RC: 'Ricevuta di Consegna',
  NS: 'Notifica di Scarto',
  MC: 'Mancata Consegna',
  NE: 'Notifica Esito',
  DT: 'Decorrenza Termini',
  AT: 'Attestazione',
};

export const REGIME_FISCALE_LABELS: Record<RegimeFiscale, string> = {
  RF01: 'Ordinario',
  RF02: 'Contribuenti minimi',
  RF04: 'Agricoltura e pesca',
  RF05: 'Vendita sali e tabacchi',
  RF06: 'Commercio fiammiferi',
  RF07: 'Editoria',
  RF08: 'Telefonia pubblica',
  RF09: 'Rivendita trasporto pubblico',
  RF10: 'Intrattenimenti e giochi',
  RF11: 'Agenzie viaggi',
  RF12: 'Agriturismo',
  RF13: 'Vendite a domicilio',
  RF14: 'Rivendita beni usati',
  RF15: 'Agenzie vendite aste',
  RF16: 'IVA per cassa P.A.',
  RF17: 'IVA per cassa',
  RF18: 'Altro',
  RF19: 'Regime forfettario',
};

export const PAYMENT_METHOD_LABELS: Record<PaymentMethod, string> = {
  MP01: 'Contanti',
  MP02: 'Assegno',
  MP03: 'Assegno circolare',
  MP04: 'Contanti presso Tesoreria',
  MP05: 'Bonifico',
  MP06: 'Vaglia cambiario',
  MP07: 'Bollettino bancario',
  MP08: 'Carta di pagamento',
  MP09: 'RID',
  MP10: 'RID utenze',
  MP11: 'RID veloce',
  MP12: 'RIBA',
  MP13: 'MAV',
  MP14: 'Quietanza erario',
  MP15: 'Giroconto contabilità speciale',
  MP16: 'Domiciliazione bancaria',
  MP17: 'Domiciliazione postale',
  MP18: 'Bollettino c/c postale',
  MP19: 'SEPA Direct Debit',
  MP20: 'SEPA Direct Debit CORE',
  MP21: 'SEPA Direct Debit B2B',
  MP22: 'Trattenuta su somme riscosse',
  MP23: 'PagoPA',
};

export const PAYMENT_CONDITION_LABELS: Record<PaymentCondition, string> = {
  TP01: 'Pagamento a rate',
  TP02: 'Pagamento completo',
  TP03: 'Anticipo',
};

export const UNIT_OF_MEASURE_LABELS: Record<UnitOfMeasure, string> = {
  PZ: 'Pezzo',
  KG: 'Chilogrammo',
  LT: 'Litro',
  MT: 'Metro',
  MQ: 'Metro quadrato',
  MC: 'Metro cubo',
  H: 'Ora',
  GG: 'Giorno',
  MESE: 'Mese',
  ANNO: 'Anno',
  '%': 'Percentuale',
};

export const NOTIFICATION_TYPE_LABELS: Record<NotificationType, string> = {
  RC: 'Ricevuta di Consegna',
  NS: 'Notifica di Scarto',
  MC: 'Mancata Consegna',
  NE: 'Notifica Esito',
  DT: 'Decorrenza Termini',
  AT: 'Attestazione',
};

// ========================================
// Helper Functions
// ========================================

export function getPartyDisplayName(party: PartyData | Customer | Supplier): string {
  if (party.isCompany && party.denomination) {
    return party.denomination;
  }
  if (party.name && party.surname) {
    return `${party.name} ${party.surname}`;
  }
  if (party.name) {
    return party.name;
  }
  return party.fiscalIdCode;
}

export function formatCurrency(amount: number, currency = 'EUR'): string {
  return new Intl.NumberFormat('it-IT', {
    style: 'currency',
    currency,
  }).format(amount);
}

export function formatItalianDate(dateString: string): string {
  const date = new Date(dateString);
  return new Intl.DateTimeFormat('it-IT', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
  }).format(date);
}
