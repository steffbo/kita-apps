// Shared helpers for fee expectations and bank transactions.
import type { BankTransaction, FeeExpectation, PaymentMatch } from '@/api/types';

export function getFeeTypeName(type?: string): string {
  switch (type) {
    case 'MEMBERSHIP':
      return 'Vereinsbeitrag';
    case 'FOOD':
      return 'Essensgeld';
    case 'CHILDCARE':
      return 'Platzgeld';
    case 'REMINDER':
      return 'Mahngebühr';
    default:
      return type || 'Unbekannt';
  }
}

export function getFeeTypeColor(type?: string): string {
  switch (type) {
    case 'MEMBERSHIP':
      return 'bg-purple-100 text-purple-700';
    case 'FOOD':
      return 'bg-orange-100 text-orange-700';
    case 'CHILDCARE':
      return 'bg-blue-100 text-blue-700';
    case 'REMINDER':
      return 'bg-red-100 text-red-700';
    default:
      return 'bg-gray-100 text-gray-700';
  }
}

/** All payment matches of a fee (split payments first, else the single match). */
export function getFeeMatches(fee: FeeExpectation): PaymentMatch[] {
  if (fee.partialMatches && fee.partialMatches.length > 0) return fee.partialMatches;
  if (fee.matchedBy) return [fee.matchedBy];
  return [];
}

export function getFeeRemainingAmount(fee: FeeExpectation): number {
  const matched = fee.matchedAmount ?? 0;
  const remaining = fee.amount - matched;
  return remaining > 0 ? remaining : 0;
}

export function getTxRemainingAmount(tx: BankTransaction): number {
  const remaining = tx.amount - (tx.matchedAmount ?? 0);
  return remaining > 0.005 ? remaining : 0;
}

export function maskIban(iban?: string): string {
  if (!iban) return '';
  const trimmed = iban.replace(/\s+/g, '');
  if (trimmed.length <= 8) return trimmed;
  return `${trimmed.slice(0, 4)}…${trimmed.slice(-4)}`;
}

export function formatConfidence(confidence: number): string {
  return `${Math.round(confidence * 100)}%`;
}

export function formatMatchedBy(reason?: string): string {
  switch (reason) {
    case 'trusted_iban':
      return 'IBAN (bekannt)';
    case 'member_number':
      return 'Mitgliedsnummer';
    case 'name':
      return 'Name';
    case 'parent_name':
      return 'Elternname';
    case 'combined':
      return 'Sammelzahlung';
    default:
      return 'Unbekannt';
  }
}

export function getConfidenceColor(confidence: number): string {
  if (confidence >= 0.8) return 'text-green-600 bg-green-100';
  if (confidence >= 0.5) return 'text-amber-600 bg-amber-100';
  return 'text-red-600 bg-red-100';
}

export function getConfidenceLabel(confidence: number): string {
  if (confidence >= 0.8) return 'Hoch';
  if (confidence >= 0.5) return 'Mittel';
  return 'Niedrig';
}
