# Kita Portal Spec 2026

## Status

| Attribut | Wert |
| --- | --- |
| Stand | 2026-05-25 |
| Ziel-Launch | neues Kita-/Schuljahr ab August 2026 |
| Produktiv-Domain | `portal.knirpsenstadt.de` |
| Staging/Test | bestehendes Homelab vor Vorstandspräsentation |
| Zielbetrieb | neuer Hetzner Cloud VPS |
| Erstes Modul | Elternstunden |
| Review-Status | Phase 0 reviewed und als Launch-Schnitt eingefroren am 2026-05-25 |
| August-MVP-Schnitt | freigegeben: Portal-Shell, Identity/Onboarding, Elternstunden, read-only Stammdaten-Sync, Mail, Audit, Staging/VPS/Backup |

## Phase 0 Review Decision

Stand 2026-05-25 ist diese Spec fuer die Umsetzung reviewed und als verbindlicher August-MVP-Schnitt bestaetigt. Aenderungen vor Launch sind nur Scope, wenn sie fuer die unten genannten Acceptance Criteria zwingend erforderlich sind; alles andere wandert in Phase 5 oder in eine spaetere Portal-Ausbaustufe.

Phase-0-Ergebnis:

- Spec ist reviewed und dient als Handoff-Vertrag fuer Architektur, Produktumfang und Launch Readiness.
- Modulgrenzen sind bestaetigt: `parent_work` ist das einzige neue fachliche MVP-Modul; Identity, Admin, Sync, Mail-Outbox und Audit sind Portal-Foundation-Faehigkeiten; `fees`, `schedule`, `time_tracking` und vollstaendige `master_data`-Pflege bleiben ausserhalb des August-MVP.
- August-Scope ist freigegeben: Eltern koennen onboarden, einloggen, eigene Kinder sehen und Elternstunden einreichen; Team/Leitung koennen pruefen, korrigieren, auswerten und erinnern; Betrieb laeuft nach Homelab-Test auf dem Hetzner VPS.
- Provider-/Datenschutzentscheidungen sind fuer den Launch-Schnitt geklaert: Hetzner VPS fuer Produktivbetrieb, Resend fuer Portal-Mails ueber `portal.knirpsenstadt.de`, bestehendes Hetzner-Mailhosting fuer `@knirpsenstadt.de` bleibt unberuehrt. Datenschutzhinweis, Anbieterpruefung/AVV und Konto-Zuordnung zum Verein bleiben Launch-Readiness-Aufgaben vor Eltern-Rollout, aber keine offenen Architekturfragen.

## Executive Summary

Das Kita-Portal wird die langfristige gemeinsame Oberfläche für Eltern, Team, Leitung und Vorstand. Es ersetzt nicht sofort alle bestehenden Apps, sondern führt zuerst eine stabile Portal-Shell mit einheitlichem Login, Rollenmodell und dem Modul Elternstunden ein. Bestehende Apps bleiben kurzfristig lauffähig und werden später kontrolliert in das Portal überführt.

Das Zielbild ist ein modularer Monolith: ein überschaubares System mit klar getrennten fachlichen Modulen, einer gemeinsamen Identitätsschicht, einem gemeinsamen Stammdatenmodell und einem einheitlichen Deployment. Das reduziert Betriebsaufwand und Übergaberisiko, wenn die technische Verantwortung später nicht mehr bei einer einzelnen Person mit hoher IT-Expertise liegt.

Der MVP bis August 2026 umfasst:

- Portal-Startseite mit kurzer Begrüßung und Login-Maske.
- Authentifizierung mit Einladung, Passwort-Onboarding und Passwort-Reset.
- Rollenmodell für Admin, Leitung, Team und Eltern.
- Elternstunden-Modul mit Einreichung, Abnahme, Übersicht und Erinnerungen.
- Getrennter Betrieb auf einem neuen Hetzner VPS nach Homelab-Test.
- Saubere Trennung von Stammdaten und Elternstunden-Daten.

## Project Reality Check

Zwei Monate sind erreichbar, wenn der Launch-Schnitt diszipliniert bleibt. Codex Agents können große Teile der Implementierung schnell erzeugen, aber sie ersetzen nicht:

- Architekturentscheidungen,
- Datenmodell-Reviews,
- Sicherheits- und Rechteprüfung,
- Migrationstest,
- E-Mail-/DNS-/VPS-Betrieb,
- Vorstandspräsentation,
- Datenschutzabstimmung.

Deshalb ist der August-Launch realistisch als MVP, nicht als vollständige Neuplattform für alle bestehenden Kita-Apps.

## Product Vision

Das Portal soll perspektivisch alle internen und elternbezogenen Kita-Prozesse bündeln:

- Elternstunden
- Beiträge
- Dienstplan
- Zeiterfassung
- Stammdaten
- Gruppen und Ansprechpartner
- Erinnerungen und Auswertungen

Eltern sollen nur die für sie relevanten Module sehen. Team, Leitung und Vorstand sehen je nach Rolle zusätzliche interne Module. Die Dienstplan-App soll langfristig ebenfalls im Portal liegen, für Eltern aber nicht sichtbar sein. Eltern sollen später aus denselben Stammdaten sehen können, in welcher Gruppe ihr Kind ist und wer die zuständigen Ansprechpartnerinnen sind.

## Current Systems

### Existing `kita-apps`

- `backend-management`: Dienstplan und Zeiterfassung.
- `backend-fees`: Beiträge, Kinder, Eltern, Haushalte, Mitglieder, Import, Erinnerungen.
- `frontend/apps/dienstplan`: Dienstplan-App.
- `frontend/apps/zeiterfassung`: Zeiterfassung-App.
- `frontend/apps/beitraege`: Beitrags-App.
- Aktueller Produktivbetrieb läuft im privaten Homelab unter `kita.remer.cc`.

### Public Website

- Repo: `~/workspace/kita-astro`.
- Domain: `https://knirpsenstadt.de`.
- Hosting: Vercel Free Tier.
- CMS-Daten liegen in einem Git-Repo.
- Bleibt separat vom Portal.

### Mail/Domain

- Domain und bestehendes Mailhosting liegen bei Hetzner.
- `@knirpsenstadt.de` Mailboxes laufen weiter über das bestehende Hetzner-Setup.
- Portal-Versand läuft über Resend mit der Versanddomain `portal.knirpsenstadt.de`.

## Target Architecture

### Guiding Principle

Das Ziel ist nicht ein Zoo aus vielen Spezialdiensten. Das Ziel ist ein wartbarer modularer Monolith mit klaren Grenzen:

- Ein Portal-Frontend.
- Eine zentrale Identity-/Auth-Schicht.
- Ein gemeinsames Stammdatenmodell.
- Fachmodule mit klar getrennten Datenbereichen.
- Ein Deployment-Stack.
- Ein Runbook.

### Portal Shell

`portal.knirpsenstadt.de` zeigt im unauthentifizierten Zustand:

- Kita-/Portal-Branding.
- Kurzes "Hallo" bzw. Begrüßung.
- Login-Maske mit E-Mail und Passwort.
- Passwort-vergessen-Link.
- Kein öffentliches Marketing und keine Modulübersicht.

Nach Login zeigt das Portal nur die Module, die für die Rolle des Users freigeschaltet sind.

### Module

MVP:

- `parent_work`: Elternstunden.

Spätere Module:

- `schedule`: Dienstplan.
- `time_tracking`: Zeiterfassung.
- `fees`: Beiträge.
- `master_data`: Kinder, Eltern, Haushalte, Gruppen, Ansprechpartner.
- `admin`: Benutzer, Rollen, Einladungen, Systemzustand.

### Confirmed Module Boundaries

Fuer den August-MVP gelten diese Grenzen:

| Bereich | MVP-Verantwortung | Grenze |
| --- | --- | --- |
| Portal Shell | Login-geschuetzte Oberflaeche, Rollen-Navigation, Dashboard-Einstieg | keine oeffentliche Marketingseite, keine Migration aller Alt-Apps |
| Identity/Auth | Benutzer, Rollen, Einladungen, Passwort-Onboarding, Reset, Sessions | keine externen Identity Provider im MVP |
| Admin | Einladungen, Rollen, Sync-/Systemsicht soweit fuer Launch noetig | keine vollstaendige Self-Service-Verwaltung aller Sonderfaelle |
| Master Data | read-only Snapshots aus `backend-fees` in `portal.synced_*` | keine Schreibzugriffe auf `fees`, keine fuehrende Stammdatenpflege im Portal |
| Parent Work | Soll/Ist/Rest, Einreichung, Pruefung, Korrektur, Jahresuebersicht, Reminder | keine Datei-Uploads, keine automatischen Reminder ohne manuelle Preview |
| Fees | bleibt in bestehender Beitrags-App | kein Elternzugriff auf Beitraege/Zahlungen im August-MVP |
| Schedule/Time Tracking | bleiben in bestehenden Apps | keine Portal-Migration vor Launch |
| Email/Audit | transaktionale Portal-Mails, Outbox, relevante Audit-Events | kein allgemeines Newsletter- oder Messaging-System |

## Single Source of Truth

Kurzfristig bleibt `backend-fees` die führende Quelle für Kinder, Eltern und Haushalte, weil diese Daten dort bereits vorhanden sind.

Mittelfristig soll ein eigenes Stammdatenmodul entstehen:

- Kinder
- Eltern/Sorgeberechtigte
- Haushalte/Familien
- Gruppen
- Gruppenzuordnung von Kindern
- Ansprechpartnerinnen je Gruppe
- aktive/inaktive Zeiträume

Fachmodule dürfen diese Daten nutzen, aber nicht unkontrolliert duplizieren oder unabhängig ändern.

### MVP Data Boundary

Für den Elternstunden-MVP gilt:

- Elternstunden liegen im MVP im `portal`-Schema in eigenen `parent_work_*`-Tabellen; eine separate Datenbank ist kein Launch-Erfordernis.
- Es gibt keine Schreibrechte auf die bestehenden Beitrags-Stammdatentabellen.
- Stammdaten werden read-only synchronisiert.
- Sync-Fehler gehen in eine Quarantäne-Liste.
- Elternstunden speichern nur notwendige Snapshots und Source-IDs.

## Auth and Onboarding

Current implementation status:

- Backend login foundation exists in `backend-portal`: login, refresh, current-user, and logout endpoints under `/api/portal/v1/auth/*`.
- Backend auth uses active `portal.users`, `portal.user_roles`, bcrypt password hashes, JWT access/refresh tokens, and persisted hashed refresh tokens in `portal.refresh_tokens`.
- A temporary bootstrap path exists via `PORTAL_BOOTSTRAP_ADMIN_EMAIL` and `PORTAL_BOOTSTRAP_ADMIN_PASSWORD` to create the first active `ADMIN` user while invitation/onboarding is still pending.
- Invitation, password onboarding, password reset, email delivery, account lockout/rate limiting, and audit events for auth are still pending.

### Roles

| Rolle | Zweck |
| --- | --- |
| `ADMIN` | Systemadministration, Einladungen, Rollen, Sync, technische Einstellungen |
| `BEITRAG` | Stammdatenverwaltung, Berechnung der Beiträge |
| `VORSTAND`| Lesezugriff auf Statistiken, Kinder und Elternnamen einsehbar, keine weiteren persönlichen Daten |
| `LEITUNG` | Jahresübersicht, Soll-Overrides, Reminder, fachliche Steuerung |
| `TEAM` | Elternstunden prüfen, genehmigen, ablehnen. Dienstplan einsehen |
| `PARENT` | Eigene Kinder und eigene Elternstunden, Übersicht der eigenen Beiträge |

### Invitation Flow

1. Admin lädt Eltern per E-Mail ein.
2. Einladungslink öffnet ausschließlich Passwort-Setup.
3. Vor Abschluss des Passwort-Setups gibt es keinen Portalzugriff und keine Anzeige von Kinderdaten.
4. Wenn das Onboarding abgebrochen wird, bleibt der Einladungslink gültig.
5. Wenn ein Passwort gesetzt wurde, wird der Einladungslink ungültig.
6. Admin kann offene Einladungen jederzeit widerrufen.

Einladungslinks sind bewusst nicht zeitlich befristet. Das Risiko wird reduziert durch:

- Setup-only-Zugriff.
- Kein Zugriff auf Portal- oder Kinderdaten vor Passwortabschluss.
- Widerruf durch Admin.
- Audit-Log.
- Rate-Limits.

### Login

Nach Onboarding erfolgt Login mit:

- E-Mail
- Passwort

Passwortregeln:

- mindestens 8 Zeichen,
- Prüfung gegen bekannte kompromittierte Passwörter,
- Passwort-Hashing mit aktuellem, geeignetem Verfahren,
- keine Klartextpasswörter in Logs oder Mails.

### Password Reset

- Passwort-vergessen sendet einen Reset-Link per E-Mail.
- Antwort ist immer generisch, damit keine Account Enumeration möglich ist.
- Reset-Link ist kurzlebig und single-use.
- Nach erfolgreichem Reset werden bestehende Sessions/Refresh Tokens invalidiert.

## Email

Versand erfolgt über Resend Free.

Absender:

- `Kita Knirpsenstadt <noreply@portal.knirpsenstadt.de>`

DNS:

- Resend-DNS-Records werden für `portal.knirpsenstadt.de` gesetzt.
- Bestehendes Mailhosting für `@knirpsenstadt.de` darf nicht beeinträchtigt werden.

MVP-Mailtypen:

- Einladung
- Passwort-Reset
- Elternstunden-Erinnerung
- optional Abnahme-/Ablehnungsbenachrichtigung

Resend Free Limits müssen berücksichtigt werden:

- 100 E-Mails pro Tag,
- 3.000 E-Mails pro Monat,
- mehrere Empfänger zählen einzeln.

Reminder- und Invite-Batches müssen daher throttling- und retry-fähig sein.

## Elternstunden Module

### Rule Set

- Kita-Jahr: `01.08.-31.07.`
- Standard-Soll: 9 Stunden je aktivem Kind.
- Teiljahr:
  - Kita-Tertiale: `01.08.-30.11.`, `01.12.-31.03.`, `01.04.-31.07.`.
  - Je angefangenem Tertial 3 Stunden.
  - Mindestens 3 Stunden, wenn das Kind im Kita-Jahr aktiv war.
- Leitung kann Sollstunden pro Kind/Jahr überschreiben.
- Befreiung von Elternstunden bei aktueller oder vergangener Vorstandsarbeit

### Parent Features

- Dashboard je Kind:
  - Sollstunden
  - genehmigte Stunden
  - eingereichte offene Stunden
  - abgelehnte Stunden
  - Reststunden
- Eintrag erfassen:
  - Kind
  - Datum
  - Dauer
  - Kategorie
  - Beschreibung
- Historie mit Status und Ablehnungsgrund.
- Keine Datei-Uploads im MVP.

### Team/Leitung Features

- Abnahme-Queue.
- Eintrag genehmigen.
- Eintrag ablehnen mit Pflichtnotiz.
- Dauer bei Genehmigung korrigieren.
- Backfill alter Papierzettel.
- Jahresübersicht je Kind/Familie.
- Soll-Override.
- Reminder-Preview.
- Reminder-Versand.

### Entry Statuses

| Status | Bedeutung |
| --- | --- |
| `SUBMITTED` | Von Eltern eingereicht, wartet auf Prüfung |
| `APPROVED` | Angenommen und zählt zum Ist |
| `REJECTED` | Abgelehnt, zählt nicht |
| `VOIDED` | Nachträglich ungültig gemacht |

### Audit

Jede relevante Aktion wird auditierbar gespeichert:

- Einladung erstellt/widerrufen.
- Passwort-Onboarding abgeschlossen.
- Login-/Reset-relevante Sicherheitsereignisse.
- Elternstunden-Eintrag erstellt/geändert.
- Genehmigt/abgelehnt/korrigiert.
- Reminder versendet.
- Sync-Konflikt erzeugt/gelöst.

## Deployment Strategy

### Environments

| Umgebung | Zweck |
| --- | --- |
| Local | Entwicklung |
| Homelab | Staging, Vorstandspräsentation, Pre-Launch-Test |
| Hetzner VPS | Produktivbetrieb |

Vor August 2026 wird im Homelab getestet und präsentiert. Erst danach zieht der produktive Betrieb auf den VPS.

### VPS Target

- Anbieter: Hetzner.
- Neuer Cloud VPS muss zusätzlich gebucht werden.
- Startgröße: CX23, sofern bei Buchung weiterhin passend.
- Betrieb im Vereins-/Kita-Kontext, nicht als dauerhaft privates Einzelpersonen-Setup.

### Stack

Docker Compose Stack auf dem VPS:

- Caddy Reverse Proxy.
- Portal-Frontend.
- Portal-/Parent-Work-Backend.
- PostgreSQL `kita` DB mit `portal`, `fees` und `public` Schema.
- bestehende Kita-Backends, solange ihre Module noch nicht ins Portal migriert sind.
- Backup-Job.

GitHub Actions:

- baut Images,
- pusht nach GHCR,
- taggt mit Git SHA,
- Produktivdeploy nutzt SHA-Tags, nicht blind `latest`.

### DNS

- `knirpsenstadt.de` bleibt bei Vercel.
- `portal.knirpsenstadt.de` zeigt auf den Hetzner VPS.
- Caddy übernimmt HTTPS.

### Secrets

- Produktions-Secrets liegen nicht im Repo im Klartext.
- Ziel: SOPS-verschlüsselte Secret-Dateien im Deploy-Repo.
- Resend API Key, JWT/Session Secrets, DB-Passwörter und Backup-Zugang getrennt halten.

## Backup and Disaster Recovery

Gewählte DR-Stufe für MVP: best effort.

Das bedeutet:

- tägliche verschlüsselte PostgreSQL-Dumps,
- Upload zur Hetzner Storage Box,
- lokale Kurzzeit-Retention,
- Storage-Box-Retention,
- Prüfung, ob Dump und Upload erfolgreich waren,
- dokumentierter Restore-Prozess.

Nicht im MVP zugesagt:

- harte RPO/RTO-Garantie,
- automatischer Restore-Test,
- Hochverfügbarkeitssetup,
- Failover-VPS.

Risiko: Ohne regelmäßigen Restore-Test ist nicht bewiesen, dass ein Backup vollständig wiederherstellbar ist. Das wird bewusst akzeptiert, sollte aber nach Launch verbessert werden.

## Privacy and Governance

Vor Eltern-Rollout erforderlich:

- Datenschutzhinweis für das Portal.
- Auftragsverarbeitungsverträge bzw. Anbieterprüfung für Hetzner und Resend.
- Klärung, dass Provider-Accounts perspektivisch dem Verein bzw. der Kita zugeordnet sind.
- Rollen- und Berechtigungskonzept schriftlich dokumentieren.

Aufbewahrung:

- personenbezogene Elternstunden-Details 3 Kita-Jahre,
- danach löschen oder anonymisiert als Summe behalten.

## Delivery Plan

### Phase 0: Spec and Architecture Freeze

Ziel: belastbare Entscheidungen vor Implementierung.

Status: abgeschlossen am 2026-05-25.

Deliverables:

- Spec reviewed und in diesem Dokument als Phase-0-Entscheidung markiert.
- Datenmodell-Schnitt bestaetigt: `portal`-Schema mit Identity, read-only Sync-Snapshots, Parent-Work, Outbox und Audit ist der Startpunkt.
- Modulgrenzen bestaetigt: August-MVP ist Portal-Foundation plus `parent_work`; Alt-Apps bleiben ausserhalb des Launch-Schnitts.
- Launch-Schnitt bestaetigt: Homelab-Staging, Vorstand-Demo, dann Hetzner-VPS-Produktion fuer `portal.knirpsenstadt.de`.
- Datenschutz-/Providerpunkte fuer Architektur geklaert: Hetzner VPS, Resend-Subdomain-Versand und bestehendes Mailhosting koexistieren; Datenschutztext, AVV/Anbieterpruefung und Vereinskonto-Zuordnung bleiben Pflicht vor Eltern-Rollout.

### Phase 1: Portal Foundation

Ziel: Portal-Shell und Identity.

Deliverables:

- Portal-Frontend mit Startseite, Login, Passwort-Reset.
- Backend-Identity mit Rollen.
- Invite- und Onboarding-Flow.
- Resend-Integration.
- Admin-UI für Einladungen.

### Phase 2: Parent Work MVP

Ziel: Elternstunden vollständig nutzbar.

Deliverables:

- Stammdaten-Sync aus Beiträge-System.
- Quarantäne-Ansicht für Sync-Probleme.
- Eltern-Dashboard.
- Einreichformular.
- Team-Abnahme.
- Leitung-Übersicht.
- Reminder-Preview und Versand.
- Audit-Log.

### Phase 3: Homelab Staging and Vorstand Demo

Ziel: reale Abläufe vor Produktivumzug testen.

Deliverables:

- Homelab-Staging-Deployment.
- Testdaten oder echte Daten mit kontrolliertem Zugriff.
- Demo-Script für Vorstand:
  - Eltern-Onboarding,
  - Stunden einreichen,
  - Team genehmigt,
  - Leitung sieht Übersicht,
  - Reminder-Preview.
- Feedback einarbeiten.

### Phase 4: VPS Production Readiness

Ziel: Betrieb außerhalb des privaten Homelabs.

Deliverables:

- Hetzner VPS buchen.
- Basis-Hardening.
- DNS vorbereiten.
- Compose-Stack deployen.
- Backups zur Storage Box einrichten.
- bestehende Kita-App migrieren.
- Smoke Tests.
- `portal.knirpsenstadt.de` live schalten.

### Phase 5: Post-Launch Consolidation

Ziel: langfristige Portal-Architektur ausbauen.

Kandidaten:

- Dienstplan als Portal-Modul integrieren.
- Stammdatenmodul aus Beiträge-App herauslösen.
- Gruppen/Ansprechpartner für Eltern sichtbar machen.
- Beiträge-Modul ins Portal überführen.
- Backup-Restore-Test automatisieren.
- Betriebs-UI für weniger technische Nachfolger ausbauen.

## MVP Non-Goals

Nicht Teil des August-MVP:

- vollständiger Rewrite von Dienstplan, Zeiterfassung und Beiträge.
- Elternzugriff auf Beiträge oder Zahlungsdaten.
- Datei-Uploads für Elternstunden.
- automatische monatliche Reminder ohne manuelle Preview.
- Hochverfügbarkeit.
- garantierte RTO/RPO.
- vollständige Self-Service-Administration für alle Sonderfälle.

## Acceptance Criteria for August 2026 Launch

Das Portal ist launchbereit, wenn:

- Eltern per Einladung onboarden und Passwort setzen können.
- Eltern sich mit Passwort einloggen können.
- Passwort-Reset funktioniert.
- Eltern nur eigene Haushaltskinder sehen.
- Eltern Stunden einreichen können.
- Team/Leitung Stunden prüfen kann.
- Soll/Ist/Rest je Kind korrekt berechnet wird.
- Reminder-Preview und Versand funktionieren.
- Sync-Konflikte nicht zu falschem Elternzugriff führen.
- Audit-Log relevante Aktionen nachvollziehbar macht.
- Portal auf Hetzner VPS läuft.
- tägliches Backup erfolgreich zur Storage Box geschrieben wird.
- Vorstand den Prozess akzeptiert hat.

## Major Risks

| Risiko | Auswirkung | Gegenmaßnahme |
| --- | --- | --- |
| Scope wächst vor August | Launch rutscht | MVP strikt auf Portal + Elternstunden begrenzen |
| Falsche Stammdatenzuordnung | Datenschutzproblem | Sync-Quarantäne, keine Best-Effort-Freigabe bei Konflikten |
| Einladungslink wird weitergeleitet | ungewolltes Onboarding | Setup-only, Admin-Widerruf, Audit, kein Datenzugriff vor Passwort |
| Resend Free Limit erreicht | Mails verzögert | Queue, Throttling, Admin-Status |
| Kein Restore-Test | Backup nicht bewiesen | Risiko bewusst dokumentieren, nach Launch verbessern |
| Zu viele getrennte Systeme | schwer übergebbar | Zielbild modularer Monolith, Portal als gemeinsame Shell |
| Big-Bang-Rewrite | Tech Debt und Launchrisiko | evolutionäre Migration |

## Long-Term Target

Langfristig soll `kita-apps` ein übergabefähiges, dokumentiertes Portal-System sein:

- ein offizieller Betrieb auf Vereins-/Kita-Infrastruktur,
- eine Login- und Rollenlogik,
- eine zentrale Stammdatenbasis,
- klar getrennte fachliche Module,
- keine produktive Abhängigkeit vom privaten Homelab,
- dokumentierte Routineaufgaben,
- normale Administration über UI statt SSH.
