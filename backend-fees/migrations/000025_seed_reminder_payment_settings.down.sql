DELETE FROM fees.app_settings
WHERE key IN (
  'reminder_payment_recipient_name',
  'reminder_payment_iban',
  'reminder_payment_bic'
);
