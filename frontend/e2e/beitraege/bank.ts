// Helpers for bank CSV imports in the Beiträge tests.

/** Berlin calendar date, like the backend's util.Today(). */
export function berlinToday() {
  const [year, month, day] = new Intl.DateTimeFormat('en-CA', { timeZone: 'Europe/Berlin' })
    .format(new Date())
    .split('-')
    .map(Number);
  return { year, month, day };
}

/** DD.MM.YYYY as in the bank export. */
export function bankDate({ year, month, day }: { year: number; month: number; day: number }): string {
  return `${String(day).padStart(2, '0')}.${String(month).padStart(2, '0')}.${year}`;
}

/** One-row bank export in the format banking-sync downloads. */
export function bankCsv(row: { date: string; payer: string; iban: string; purpose: string; amount: string }): string {
  return [
    'Bezeichnung Auftragskonto;IBAN Auftragskonto;BIC Auftragskonto;Bankname Auftragskonto;Buchungstag;Valutadatum;Name Zahlungsbeteiligter;IBAN Zahlungsbeteiligter;BIC (SWIFT-Code) Zahlungsbeteiligter;Buchungstext;Verwendungszweck;Betrag;Waehrung;Saldo nach Buchung',
    ['Kita', 'DE1234', 'BIC', 'Bank', row.date, row.date, row.payer, row.iban, 'BIC', 'Gutschrift', row.purpose, row.amount, 'EUR', '1000,00'].join(';'),
    '',
  ].join('\n');
}

/**
 * Unique payer IBAN per test. An auto-matched payment marks its IBAN as trusted
 * for that child, so a shared IBAN would route other tests' payments there.
 */
export function uniqueIban(): string {
  const digits = Array.from({ length: 18 }, () => Math.floor(Math.random() * 10)).join('');
  return `DE00${digits}`;
}
