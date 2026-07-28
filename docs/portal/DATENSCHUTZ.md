# Portal Datenschutz, Provider und Rollen

## Status

| Attribut | Wert |
| --- | --- |
| Stand | 2026-05-25 |
| Ticket | Phase 0: Datenschutz & Provider-Pruefung |
| Scope | Portal-Datenschutzhinweis, AVV-/Providerpruefung Hetzner und Resend, Rollen- und Berechtigungskonzept, Provider-Kontenzuordnung |
| Portal | `portal.knirpsenstadt.de` |
| Verantwortlicher | Knirpsenstadt e.V. Panketal, Ahornallee 27, 16341 Panketal |
| Status | Dokumentiert und fuer MVP-Architektur freigegeben; Account-Nachweise muessen vor Eltern-Rollout in den Vereinsablaegen abgelegt werden |

Dieses Dokument ist der Phase-0-Handoff fuer Datenschutz, Provider und Rollen. Es ersetzt keine anwaltliche Pruefung, definiert aber den Umsetzungs- und Nachweisstand, den die Portal-Implementierung einhalten muss.

## Ergebnis der Providerpruefung

### Entscheidung

- Hetzner bleibt der Zielanbieter fuer den produktiven Portal-VPS und die Storage-Box-Backups.
- Resend bleibt der Zielanbieter fuer transaktionale Portal-E-Mails ueber `portal.knirpsenstadt.de`.
- Das bestehende Hetzner-Mailhosting fuer `@knirpsenstadt.de` bleibt getrennt und wird durch Resend-DNS fuer die Subdomain nicht ersetzt.
- Provider-Accounts duerfen perspektivisch nicht dauerhaft an eine Privatperson gebunden bleiben. Zielzustand ist ein Vereins-/Kita-Konto mit mindestens zwei administrativen Personen.

### Quellenstand

Die Providerpruefung basiert auf den offiziellen Anbieterquellen, abgerufen am 2026-05-25:

| Anbieter | Quelle | Relevanz |
| --- | --- | --- |
| Hetzner | `https://docs.hetzner.com/de/general/company-and-policy/data-protection-at-hetzner/` | AV-Vertrag im Kundenaccount, Subunternehmer, TOMs, Drittland-/Standortregeln |
| Hetzner | `https://www.hetzner.com/AV/DPA_de.pdf` | AV-Vertrag, Stand Vertragsmuster 1.1 vom 2025-02-10, TOMs Stand 2026-02-16 |
| Resend | `https://resend.com/security/gdpr` | GDPR-Hinweise, US-Speicherung, SCC-Hinweis, DPA-Verweis |
| Resend | `https://resend.com/legal/dpa` | Data Processing Addendum, SCCs, Details of Processing, TOMs |
| Resend | `https://resend.com/legal/subprocessors` | Subprozessorenliste, Stand 2025-12-31 |
| Resend | `https://resend.com/legal/terms-of-service` | Vertragseinbeziehung von Terms, Privacy Policy und DPA |

### Hetzner

| Punkt | Bewertung |
| --- | --- |
| Rolle | Auftragsverarbeiter fuer Hosting, Storage und technische Infrastruktur |
| Daten | Portal-Datenbank, Server-Logs, technische Backups, Deploy-/Betriebsdaten |
| AVV | Hetzner stellt ein AV-Vertragsmuster bereit; Abschluss erfolgt im Kundenaccount per Zustimmung, keine handschriftliche Signatur erforderlich |
| Standort | Fuer Cloud-Produkte bestimmt die gewaehlte Produkt-Location die Speicherung; fuer das Portal muss eine EU-Location gewaehlt werden |
| Subunternehmer | Offizielle Subunternehmerliste ist im AVV und bei Hetzner verlinkt |
| TOMs | Hetzner dokumentiert TOMs und laesst diese laut eigener Doku jaehrlich pruefen; bei geschlossenem AVV ist das Pruefprotokoll im Kundenportal abrufbar |
| Freigabe | Freigegeben fuer MVP-Produktion, wenn EU-Standort, AVV-Nachweis, MFA und Vereinsaccount erfuellt sind |

Pflichtvorgaben fuer die Umsetzung:

- VPS-Region in der EU waehlen; keine USA- oder Singapur-Location fuer Produktivdaten.
- Backups vor Upload zur Storage Box verschluesseln.
- Server-seitige Verschluesselung, Firewall, SSH-Zugriff, OS-Patches und Datenbank-Hardening bleiben Aufgabe des Portal-Betriebs.
- Hetzner-AVV im Vereins-/Kita-Account abschliessen oder nach Kontouebertragung erneut abschliessen.
- AVV-PDF, Subunternehmerliste und TOM-/Pruefnachweis in der Vereinsablage ablegen, nicht im Git-Repo.

### Resend

| Punkt | Bewertung |
| --- | --- |
| Rolle | Auftragsverarbeiter fuer transaktionalen E-Mail-Versand |
| Vertragspartner | Plus Five Five, Inc., 2261 Market Street #5039, San Francisco, CA 94114 |
| Daten | Empfaenger-E-Mail-Adresse, technische E-Mail-Metadaten, notwendiger transaktionaler Nachrichteninhalt |
| DPA | Resend veroeffentlicht ein DPA; es wird laut DPA/Terms mit Annahme des Agreements bzw. durch Ausfuehrung verbindlich |
| Drittland | Resend speichert Customer Data in den USA und verweist auf SCCs; DPA und Subprozessoren sind vor Rollout zu exportieren/abzulegen |
| Subprozessoren | Die offizielle Liste enthaelt mehrere US-Subprozessoren; das ist fuer transaktionale Portal-Mails akzeptiert, solange Datenminimierung eingehalten wird |
| Freigabe | Freigegeben fuer MVP-E-Mails mit Datenminimierung, Tracking-Verzicht und Vereinsaccount-Nachweis |

Pflichtvorgaben fuer die Umsetzung:

- Resend nur fuer transaktionale Portal-Mails verwenden: Einladungen, Passwort-Reset, Elternstunden-Erinnerung und fachliche Statusbenachrichtigungen.
- Keine sensiblen Detaildaten in E-Mails. Elternstunden-E-Mails duerfen hoechstens auf offene Aktion oder Status hinweisen und ins Portal verlinken; Details stehen im Portal nach Login.
- Keine offenen Beitraege, Bankdaten, Gesundheitsdaten oder vertraulichen Freitexte ueber Resend versenden.
- Open-/Click-Tracking und Marketing-/Audience-Funktionen fuer das Portal nicht aktivieren.
- API-Key als separates Produktions-Secret fuehren und bei Kontouebertragung oder Admin-Wechsel rotieren.
- DPA, Terms, Privacy Policy und Subprozessorenliste aus dem Resend-Dashboard bzw. den Legal-Seiten in der Vereinsablage ablegen.

### Account-Zielzustand

| Konto | Zielzustand | Vor Eltern-Rollout erforderlich |
| --- | --- | --- |
| Hetzner | Vertrags-/Kundenkonto auf Knirpsenstadt e.V. Panketal oder vergleichbare offizielle Kita-/Vereinsidentitaet | Rechnungsdaten, Vertragsinhaber, MFA, mindestens zwei Admins, AVV-Nachweis |
| Resend | Workspace/Konto fuer Knirpsenstadt e.V. bzw. Kita Knirpsenstadt, nicht dauerhaft persoenliches Entwicklerkonto | Admin-/Recovery-Zugriff fuer Verein, DPA-Nachweis, Domain `portal.knirpsenstadt.de`, API-Key-Rotation |
| GitHub/GHCR | Technischer Build-/Registry-Zugriff darf bestehen, aber Produktiv-Secrets muessen uebergabefaehig dokumentiert sein | Secrets nicht an Einzelperson binden; Deploy- und Restore-Zugriff dokumentieren |

Nachweisregeln:

- Vertrags-/AVV-Dokumente gehoeren in eine nicht-oeffentliche Vereinsablage.
- Im Repo wird nur dokumentiert, welche Nachweise existieren muessen und wo die operative Verantwortung liegt.
- Nach jeder Provider-Aenderung dieses Dokument und `SPEC.md` aktualisieren.

## Rollen- und Berechtigungskonzept

### Grundsaetze

- Least Privilege: Jede Rolle sieht nur die Module und Daten, die sie fuer ihre Aufgabe braucht.
- Rollen steuern Modulzugang; Datenzugriff wird zusaetzlich durch Haushalts-, Kinder- und Modulregeln eingeschraenkt.
- Elternzugriff ist immer haushaltsgebunden ueber `portal.user_households`.
- Das Portal schreibt nicht in das bestehende `fees`-Schema; Stammdaten werden read-only nach `portal.synced_*` synchronisiert.
- Frontend-Routen sind nur UX-Gates. Jede produktive Backend-API muss Rollen und Row-Level-Zugriff serverseitig pruefen.
- Privilegierte Aenderungen, Sync-Konflikte und Elternstunden-Entscheidungen muessen in `portal.audit_events` nachvollziehbar sein.

### Rollen

| Rolle | Zweck | Typischer Personenkreis |
| --- | --- | --- |
| `ADMIN` | Technische und organisatorische Portaladministration | technische Admins, sehr kleiner Kreis |
| `BEITRAG` | Beitrags- und Stammdatenverwaltung in bestehenden/kuenftigen Beitragsmodulen | Kassen-/Beitragsverantwortliche |
| `VORSTAND` | Vereinsaufsicht und Auswertungen mit stark begrenztem Personenbezug | Vorstandsmitglieder |
| `LEITUNG` | Fachliche Steuerung der Kita-Ablaufe, Elternstunden-Soll, Reminder, Uebersichten | Kita-Leitung |
| `TEAM` | Operative Pruefung von Elternstunden und perspektivisch Dienstplan-/Teamfunktionen | Teammitglieder |
| `PARENT` | Eigener Haushalt, eigene Kinder und eigene Elternstunden | Eltern/Sorgeberechtigte |

### MVP-Modulzugriff

| Modul | `ADMIN` | `BEITRAG` | `VORSTAND` | `LEITUNG` | `TEAM` | `PARENT` |
| --- | --- | --- | --- | --- | --- | --- |
| Login, eigenes Profil, Logout | ja | ja | ja | ja | ja | ja |
| Dashboard | ja | ja | ja | ja | ja | ja |
| Administration: Einladungen, Rollen | ja | nein | nein | nein | nein | nein |
| Sync-/Systemstatus | ja | nein | lesen, falls fuer Vorstand freigegeben | lesen/fachlich klaeren | nein | nein |
| Read-only Stammdaten-Snapshots | ja | ja | Namen/Statistik, keine Kontaktdetails | ja | nur soweit fachlich noetig | nur eigener Haushalt |
| Elternstunden: eigene Einreichung | nein | nein | nein | nein | nein | ja |
| Elternstunden: eigene Historie | nein | nein | nein | nein | nein | ja |
| Elternstunden: Review Queue | ja | nein | nein | ja | ja | nein |
| Elternstunden: Genehmigen/Ablehnen | ja | nein | nein | ja | ja | nein |
| Elternstunden: Sollstunden-Override | ja | nein | nein | ja | nein | nein |
| Elternstunden: Auswertung | ja | nein | aggregiert/aufsichtsbezogen | ja | begrenzte Teamansicht | nur eigene Kinder |
| Reminder-Preview und Versand | ja | nein | nein | ja | nein | nein |
| Beitraege-Modul | geplant | geplant | nein im MVP | nein im MVP | nein | nein im MVP |
| Dienstplan/Zeiterfassung | geplant | nein | nein | geplant | geplant | nein |

### Konkrete Berechtigungsregeln fuer Elternstunden

- `PARENT` darf nur Kinder sehen, deren Haushalt ueber `portal.user_households` mit dem eigenen User verknuepft ist.
- `PARENT` darf Elternstunden nur fuer eigene Haushaltskinder erstellen und eigene Eintraege sehen.
- `PARENT` darf eingereichte Eintraege nach Abgabe nicht stillschweigend in genehmigte Historie umschreiben; Aenderungen nach Review muessen als neue Aktion oder auditierte Korrektur modelliert werden.
- `TEAM` darf eingereichte Elternstunden pruefen, genehmigen oder ablehnen, aber keine Sollstunden setzen.
- `LEITUNG` darf Sollstunden pro Kind/Jahr uebersteuern und Reminder vorbereiten/versenden.
- `VORSTAND` erhaelt im MVP keine Review Queue und keine Kontaktdaten. Zulässig sind Namen und aggregierte Auswertungen, soweit fuer Vereinsaufsicht erforderlich.
- `BEITRAG` sieht im MVP keine Elternstunden und keine Elternstunden-Review.
- `ADMIN` darf technische Korrekturen ausfuehren, soll fachliche Entscheidungen aber nach Moeglichkeit den Rollen `LEITUNG` oder `TEAM` ueberlassen.

### Rollenvergabe und Betrieb

- Rollen werden nicht durch Nutzer selbst vergeben.
- `ADMIN` ist die einzige Rolle, die Rollen und Einladungen im Portal verwalten darf.
- Neue `ADMIN`-Rollen brauchen vor Produktivbetrieb eine dokumentierte Freigabe durch Vorstand oder bestehende Admins.
- Rollen fuer ausgeschiedene Eltern, Teammitglieder oder Vorstaende werden zeitnah entzogen; Accounts koennen auf `DISABLED` gesetzt werden.
- Regelmaessiger Review: vor Kita-Jahresstart und bei Vorstands-/Teamwechseln Rollenliste exportieren und pruefen.

### Implementierungsstand

| Bereich | Stand 2026-05-25 |
| --- | --- |
| Rollentypen | `ADMIN`, `BEITRAG`, `VORSTAND`, `LEITUNG`, `TEAM`, `PARENT` existieren in Migration, Domain und Frontend-Typen |
| Frontend-Navigation | Modul-Sichtbarkeit ist in `frontend/apps/portal/src/lib/modules.ts` rollenbasiert definiert |
| Frontend-Routen | Ready-Routen fuer `parent_work`, `review` und `admin` haben Rollen-Metadata in `frontend/apps/portal/src/router/index.ts` |
| Backend-Auth | Login, Refresh, Me und Logout existieren; Rollen kommen aus `portal.user_roles` |
| Backend-Fach-APIs | Parent-work-, Invite-, Onboarding-, Password-reset- und Sync-Endpunkte sind noch nicht implementiert |
| Backend-Permissions | Muss mit den fachlichen APIs serverseitig umgesetzt werden; Frontend-Gates reichen nicht aus |

## Datenschutzhinweis fuer das Portal

Der folgende Text ist der Portal-Datenschutzhinweis fuer die Veroeffentlichung bzw. Abstimmung. Platzhalter in eckigen Klammern muessen vor Eltern-Rollout final ergaenzt werden.

### Datenschutzhinweis Kita-Portal

#### 1. Verantwortlicher

Verantwortlich fuer die Datenverarbeitung im Kita-Portal ist:

Knirpsenstadt e.V. Panketal<br>
Ahornallee 27<br>
16341 Panketal

Vertreten durch den Vorstand.

Kontakt: `[E-Mail-Adresse des Vereins/der Kita ergaenzen]`

Datenschutzkontakt: `[Datenschutzkontakt ergaenzen, falls abweichend]`

#### 2. Zweck des Portals

Das Kita-Portal unter `portal.knirpsenstadt.de` dient der digitalen Zusammenarbeit zwischen Kita, Verein, Team und Eltern. Im MVP werden insbesondere Benutzerkonten, Login, Einladungen, Passwort-Zuruecksetzung, Rollen, Haushalts-/Kind-Zuordnung und Elternstunden verarbeitet.

#### 3. Welche Daten verarbeitet werden

Je nach Rolle und Nutzung koennen folgende Daten verarbeitet werden:

- Kontodaten: Name, E-Mail-Adresse, Rollen, Account-Status.
- Zugangsdaten: Passwort-Hash, Refresh-Token-Hash, Login-/Logout-Zeitpunkte und technische Sicherheitsereignisse.
- Stammdaten-Snapshots: Haushalte, Eltern, Kinder, Zuordnungen und relevante Kita-Stammdaten aus dem Beitrags-/Verwaltungssystem.
- Elternstunden: Sollstunden, eingereichte Taetigkeiten, Datum, Dauer, Status, Pruefentscheidung, Korrektur- oder Ablehnungsnotizen.
- E-Mail-Daten: Empfaengeradresse, Mailtyp, Versandstatus und notwendiger Inhalt transaktionaler Portal-Mails.
- Protokoll- und Auditdaten: technische Logs, Audit-Events zu sicherheits- und fachlich relevanten Aktionen.

#### 4. Zwecke und Rechtsgrundlagen

Die Daten werden verarbeitet, um:

- Benutzerkonten bereitzustellen und Zugriff zu schuetzen.
- Eltern nur die eigenen Haushalts- und Kinderdaten anzuzeigen.
- Elternstunden einzureichen, zu pruefen, auszuwerten und zu erinnern.
- Einladungen und Passwort-Zuruecksetzungen zu versenden.
- Missbrauch, Fehlzugriffe und technische Stoerungen zu erkennen.
- Kita- und Vereinsprozesse nachweisbar und geordnet abzuwickeln.

Rechtsgrundlagen sind insbesondere Art. 6 Abs. 1 lit. b DSGVO fuer die Durchfuehrung des Betreuungs-/Vereinsverhaeltnisses, Art. 6 Abs. 1 lit. f DSGVO fuer sicheren Betrieb, Nachvollziehbarkeit und Missbrauchsschutz sowie, soweit einschlaegig, Art. 6 Abs. 1 lit. c DSGVO fuer rechtliche Pflichten.

#### 5. Empfaenger und Auftragsverarbeiter

Innerhalb der Kita/des Vereins erhalten nur berechtigte Personen Zugriff gemaess Rollen- und Berechtigungskonzept.

Fuer den technischen Betrieb werden Auftragsverarbeiter eingesetzt:

- Hetzner Online GmbH fuer Hosting, Server, Storage/Backups und Infrastruktur.
- Plus Five Five, Inc. / Resend fuer transaktionalen E-Mail-Versand.

Mit den Anbietern sind Auftragsverarbeitungsvertraege bzw. Data Processing Addenda abzuschliessen und in der Vereinsablage nachzuweisen.

#### 6. Drittlanduebermittlung

Das Portal-Hosting bei Hetzner muss auf einer EU-Location betrieben werden.

Resend speichert nach eigener Anbieterinformation Customer Data in den USA. Fuer den E-Mail-Versand werden deshalb nur die fuer transaktionale Mails notwendigen Daten uebermittelt. Grundlage fuer die Drittlanduebermittlung sind die im Resend-DPA vorgesehenen Schutzmechanismen, insbesondere Standardvertragsklauseln. Portal-Mails werden datenarm gestaltet und enthalten keine vertraulichen Detaildaten, die erst nach Login im Portal angezeigt werden sollen.

#### 7. Speicherdauer

- Konten werden fuer die Dauer der Portalnutzung und danach nur solange gespeichert, wie dies fuer Nachweis-, Sicherheits- oder Verwaltungszwecke erforderlich ist.
- Refresh Tokens werden nur als Hash gespeichert und bei Logout, Rotation, Passwort-Reset oder Account-Deaktivierung widerrufen bzw. geloescht.
- Elternstunden-Details werden fuer drei Kita-Jahre aufbewahrt und danach geloescht oder nur anonymisiert/aggregiert weitergefuehrt.
- E-Mail- und technische Versandprotokolle werden nur solange gespeichert, wie sie fuer Zustellung, Fehleranalyse und Nachweis erforderlich sind.
- Audit-Logs werden solange aufbewahrt, wie sie fuer Nachvollziehbarkeit, Sicherheit und rechtliche Interessen erforderlich sind.

Die technischen Loeschjobs fuer diese Aufbewahrungsregeln sind nach aktueller Implementierung noch zu ergaenzen.

#### 8. Sicherheit

Das Portal nutzt rollenbasierten Zugriff, serverseitige Berechtigungspruefungen, verschluesselte Transportwege, gehashte Passwoerter und widerrufbare Refresh-Token-Sessions. Produktiv-Secrets werden nicht im Repository gespeichert. Backups muessen verschluesselt abgelegt werden.

#### 9. Rechte der betroffenen Personen

Betroffene Personen koennen nach Massgabe der DSGVO Auskunft, Berichtigung, Loeschung, Einschraenkung der Verarbeitung, Datenuebertragbarkeit und Widerspruch verlangen. Anfragen koennen an den oben genannten Kontakt gerichtet werden.

Ausserdem besteht ein Beschwerderecht bei einer zustaendigen Datenschutzaufsichtsbehoerde.

#### 10. Keine oeffentliche Marketingseite und kein Tracking im MVP

Das Portal ist eine login-geschuetzte Arbeitsoberflaeche. Im MVP ist keine oeffentliche Marketingseite, kein Werbetracking und keine Analysefunktion fuer Eltern vorgesehen. Im Browser werden nur technisch notwendige Informationen fuer die Anmeldung und Sitzung gespeichert.

## Launch-Checkliste Datenschutz

Vor dem Eltern-Rollout muessen folgende Punkte erledigt sein:

- Hetzner-Konto auf Verein/Kita ausrichten oder Uebergabeplan dokumentieren.
- Hetzner-AVV im richtigen Konto abschliessen und PDF/Nachweise in Vereinsablage speichern.
- Hetzner-VPS in EU-Location buchen und Produktivdaten nicht in Drittland-Regionen betreiben.
- Resend-Konto/Workspace auf Verein/Kita ausrichten oder Uebergabeplan dokumentieren.
- Resend-DPA/Terms/Subprozessorenliste als Nachweis speichern.
- Resend-Tracking fuer Portal-Mails deaktiviert lassen.
- Produktions-API-Key und Domain-Zugriffe uebergabefaehig dokumentieren.
- Datenschutzhinweis mit finaler Kontaktadresse pruefen und vor Login/Onboarding verlinken.
- Backend-Permissions fuer neue APIs serverseitig gemaess Rollenmatrix umsetzen.
- Loesch-/Aufbewahrungsjobs fuer Elternstunden, E-Mail-Outbox und Audit/Logs konkretisieren.
