INSERT INTO fees.app_settings (key, value)
VALUES
  ('reminder_payment_recipient_name', 'Knirpsenstadt e.V.'),
  ('reminder_payment_iban', 'DE33370205000003321400'),
  ('reminder_payment_bic', 'BFSWDE33XXX')
ON CONFLICT (key) DO NOTHING;
