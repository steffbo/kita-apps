# UI-Bausteine Beiträge

Verbindliche Regeln für `frontend/apps/beitraege`. Referenz-Entwurf: `ui-mockup.html` (gleicher Ordner),
Referenz-Umsetzung: `src/pages/ChildrenPage.vue`. Ziel: Gleiches sieht gleich aus und verhält sich gleich,
kein unnötiger Text, hell und dunkel gleichwertig, Handy zuerst.

## Farben

- Nur semantische Tokens, keine Tailwind-Paletten (`red-*`, `amber-*`, `green-*`, `blue-*`, `emerald-*`,
  `orange-*`, `purple-*`, `sky-*`, `gray-*` usw.) und keine `dark:`-Varianten für Farben. Die Tokens
  schalten selbst um.
- Fläche/Text: `bg-background`, `bg-card`, `bg-muted`, `bg-accent` (Hover), `text-foreground`,
  `text-muted-foreground`, `border` (Standardrahmen).
- Akzent (Indigo): `bg-primary`/`text-primary-foreground` für die Hauptaktion, `bg-primary-soft` +
  `text-primary-text` für Auswahl/aktiv/Info.
- Status: `success` (bezahlt, erledigt, aktiv), `warning` (offen, Hinweis, braucht Aufmerksamkeit),
  `destructive` (überfällig, Fehler, Löschen). Je Ton `text-<ton>`, `bg-<ton>-soft`, `border-<ton>-line`.
  Neutrales (Info ohne Dringlichkeit, Zähler) bleibt grau.
- Lila/Orange/Blau als eigene Bedeutungen gibt es nicht mehr; auf die drei Status oder neutral abbilden.
- Farb-Helfer in `utils/` (`getFeeTypeColor`, `getConfidenceColor`, `statusBadgeClass`) liefern fertige
  `badge`-Klassen; am Element keine eigenen Badge-Maße dazuschreiben.

## Klassen (`src/assets/main.css`, `@layer components`)

| Klasse | Wofür |
| --- | --- |
| `btn` + `btn-primary` | eine Hauptaktion je Bereich (rechts bzw. zuletzt) |
| `btn` + `btn-secondary` | weitere Aktionen, Zeilenaktionen, „Abbrechen“ |
| `btn` + `btn-ghost` | leise Aktionen (Auswahl aufheben, Toolbar-Icons) |
| `btn` + `btn-danger` / `btn-danger-outline` | Löschen bestätigen / Löschen anstoßen |
| `btn-sm`, `btn-icon` | kleiner (h-8) / quadratisch, Icon-Button braucht `aria-label` |
| `badge` (+ `badge-primary|success|warning|danger`) | Status und Zähler, kurz |
| `note` (+ `note-info|success|warning|danger`) | Hinweis/Fehler im Inhalt; Icon zuerst, dann Text |
| `card` | Fläche mit Rahmen, `rounded-xl` |
| `chip` + `aria-pressed` | Filter über Listen (an/aus, ein Wert aus mehreren); Zähler in `span.chip-count` |
| `label`, `input` | Formularfeld (auch `select`, `textarea.input`) |
| `table` | jede Datentabelle in einer `card` (Kopf grau, Zeilenlinien, keine Zebrastreifen); Padding fest, Kopfzelle rechtsbündig mit `text-right` |

Icons in `btn`/`note` brauchen keine Größe (`<Plus />`), das setzt die Klasse.

## Komponenten (`src/components/ui/`)

- `PageHeader`: Titel (`h1`) links, Aktionen rechts (`#actions`), optional `back` für Detailseiten.
  Keine Untertitel wie „49 Kinder registriert“ oder Seitenbeschreibungen; Anzahl gehört an die Chips/Liste.
- `AppSheet`: jeder Dialog. Handy: Sheet von unten, Desktop: zentriert. Props `title` (auch `aria-label`,
  bleibt für Tests gleich), `size` `sm|md|lg`. Buttons in `#footer`: „Abbrechen“ (`btn-secondary`) zuerst,
  Hauptaktion zuletzt. Formulare: `<form id="x">` im Inhalt, Submit-Button mit `form="x"` im Footer.
  Esc, Klick daneben und Scroll-Sperre macht die Komponente; keine eigenen Esc-Handler mehr.
- `AppPagination`: Fußzeile seitenweiser Listen (`v-model:page-size`, `@go`); Zustand und Laden kommen aus
  `composables/usePagedList.ts`.
- `EmptyState`: leere Liste (`icon`, `text`, optional Aktion im Slot).
- Weiterhin: `SearchInput`, `SortTh`.

## Listen auf dem Handy

Breite Tabellen unter `sm` als Liste (`ul.divide-y sm:hidden`): Zeile = Titel (`font-medium`) + Unterzeile
(`text-[13px] text-muted-foreground`, Werte mit ` · ` getrennt) + Badges/Betrag rechts. Die Tabelle steht
daneben mit `hidden sm:block`. Kleine Tabellen dürfen bleiben (scrollen seitlich).

## Text

- Überschriften: `h1` `text-2xl font-semibold tracking-tight` (über `PageHeader`), Abschnitte
  `text-base font-semibold`. Kein `font-bold` für normale Texte.
- Hinweistexte nur, wenn sie eine Information tragen, die nicht schon im UI steht. Kurz und sachlich ohne
  Anrede („3 Kinder deaktivieren?“ statt „Möchten Sie wirklich …?“).
- Beschriftungen, auf die Tests zugreifen (Buttons, Labels, Dialog-Titel, Überschriften), nur ändern, wenn
  der Test mitgeändert wird.
