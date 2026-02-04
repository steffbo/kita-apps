# Testdaten (Beiträge) – Seed-Setup

Stand: 2026-02-04

## Datenbank
- Test-DB Container: `kita-db-e2e` (Postgres 16)
- DB: `kita_e2e`
- Schema: `fees`
- Ports: `localhost:5433`

## Backend
- Backend-fees läuft i.d.R. auf `http://localhost:8081`

## CSV-Datei für Import
- Pfad: `example-import.csv`
- Enthält 6 Essensgeld-Transaktionen (45,40 EUR), jeweils mit trusted IBAN und Mitgliedsnummer

## Beispielhaushalte & Kinder (Einkommensfälle)

### 1) Höchstsatz akzeptiert (MAX_ACCEPTED)
- Haushalt: **Familie Keller**
- Eltern: Lisa Keller (primary), Jan Keller
- Kind: **Noah Keller** (Mitgliedsnummer `13001`)
  - Geburtsdatum: 2024-02-10 (U3)
  - Betreuungsstunden: 45
- IBAN trusted: `DE44500105175432193001`

### 2) Entlastungspaket (< 55.000, PROVIDED)
- Haushalt: **Familie Sommer**, Einkommen 48.000
- Eltern: Nina Sommer (primary), Paul Sommer
- Kind: **Mila Sommer** (Mitgliedsnummer `13002`)
  - Geburtsdatum: 2024-07-22 (U3)
  - Betreuungsstunden: 35
- IBAN trusted: `DE66500105175432193002`

### 3) Geschwisterrabatt (2 Kinder, PROVIDED)
- Haushalt: **Familie Brandt**, Einkommen 65.000, `children_count_for_fees = 2`
- Eltern: Mara Brandt (primary), Sven Brandt
- Kinder:
  - **Lukas Brandt** (`13003`), 2024-01-15 (U3), 45 Std
  - **Ella Brandt** (`13004`), 2023-10-05 (U3), 30 Std
- IBANs trusted:
  - Lukas: `DE88500105175432193003`
  - Ella:  `DE44500105175432193004`

### 4) Pflegefamilie (FOSTER_FAMILY)
- Haushalt: **Familie Fischer**
- Eltern: Lea Fischer (primary), Ben Fischer
- Kind: **Jonas Fischer** (`13005`)
  - Geburtsdatum: 2024-09-30 (U3)
  - Betreuungsstunden: 30
- IBAN trusted: `DE99500105175432193005`

### 5) Ü3-Kind (PROVIDED)
- Haushalt: **Familie Schröder**, Einkommen 62.000
- Eltern: Katrin Schroeder (primary), Oliver Schroeder
- Kind: **Lea Schroeder** (`13006`)
  - Geburtsdatum: 2020-01-20 (Ü3)
  - Betreuungsstunden: 35
- IBAN trusted: `DE77500105175432193006`

## Hinweise
- Essensgeld (FOOD) ist für alle Kinder als offener Beitrag für den текущigen Monat angelegt.
- Platzgeld (CHILDCARE) wird erst durch "Beiträge generieren" erzeugt und ist abhängig von:
  - U3/Ü3
  - income_status + annual_household_income
  - Betreuungsstunden
  - Geschwisteranzahl (`children_count_for_fees`)
- Trusted IBAN hat höchste Matching-Konfidenz ("trusted_iban").

