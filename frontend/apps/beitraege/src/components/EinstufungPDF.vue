<script setup lang="ts">
import { computed, nextTick, ref } from 'vue';
import type { Einstufung, Child } from '@/api/types';
import { FileDown, Loader2 } from 'lucide-vue-next';
import printStyles from './EinstufungPDF.css?raw';

const props = defineProps<{
  einstufung: Einstufung;
}>();

const isGenerating = ref(false);
const pdfContainer = ref<HTMLElement | null>(null);

const child = computed(() => props.einstufung.child as Child | undefined);

// Compute the distinct fee columns for the letter
interface FeeColumn {
  label: string; // e.g. "Sept 25"
  careHours: number;
  careType: string; // "Krippe" or "Kindergarten"
  childcareFee: number;
  foodFee: number;
  membershipFee: number;
}

const feeColumns = computed<FeeColumn[]>(() => {
  const e = props.einstufung;
  const validFrom = new Date(e.validFrom);
  const startMonth = validFrom.getMonth(); // 0-based
  const startYear = validFrom.getFullYear();

  const cols: FeeColumn[] = [];

  const formatMonth = (month: number, year: number) => {
    return new Date(year, month).toLocaleString('de-DE', { month: 'short', year: '2-digit' });
  };

  const formatCareType = (ct: string) => {
    if (ct === 'krippe') return 'Krippe';
    if (ct === 'kindergarten') return 'Kindergarten';
    return ct.charAt(0).toUpperCase() + ct.slice(1);
  };

  // Column 1: First month (with membership fee)
  cols.push({
    label: formatMonth(startMonth, startYear),
    careHours: e.careHoursPerWeek,
    careType: formatCareType(e.careType),
    childcareFee: e.monthlyChildcareFee,
    foodFee: e.monthlyFoodFee,
    membershipFee: e.annualMembershipFee,
  });

  // Column 2: Second month (no membership fee)
  const m2 = startMonth + 1;
  const y2 = m2 > 11 ? startYear + 1 : startYear;
  cols.push({
    label: formatMonth(m2 % 12, y2),
    careHours: e.careHoursPerWeek,
    careType: formatCareType(e.careType),
    childcareFee: e.monthlyChildcareFee,
    foodFee: e.monthlyFoodFee,
    membershipFee: 0,
  });

  // Column 3: If child turns 3 within the next 12 months → beitragsfrei
  if (child.value && e.careType === 'krippe') {
    const birthDate = new Date(child.value.birthDate);
    const turnsThreeDate = new Date(birthDate.getFullYear() + 3, birthDate.getMonth(), birthDate.getDate());
    // The month the child transitions to Kindergarten (first full month after turning 3)
    let transMonth = turnsThreeDate.getMonth();
    let transYear = turnsThreeDate.getFullYear();
    // If birthday is not the first of the month, transition happens next month
    if (turnsThreeDate.getDate() > 1) {
      transMonth += 1;
      if (transMonth > 11) {
        transMonth = 0;
        transYear += 1;
      }
    }

    const transDate = new Date(transYear, transMonth, 1);
    const windowEnd = new Date(startYear, startMonth + 12, 1);

    if (transDate > validFrom && transDate <= windowEnd) {
      cols.push({
        label: formatMonth(transMonth, transYear),
        careHours: e.careHoursPerWeek,
        careType: 'Kindergarten',
        childcareFee: 0,
        foodFee: e.monthlyFoodFee,
        membershipFee: 0,
      });
    }
  }

  return cols;
});

const entryDateFormatted = computed(() => {
  if (!child.value?.entryDate) return '—';
  return new Date(child.value.entryDate).toLocaleDateString('de-DE');
});

const birthDateFormatted = computed(() => {
  if (!child.value?.birthDate) return '—';
  return new Date(child.value.birthDate).toLocaleDateString('de-DE');
});

const memberNumber = computed(() => child.value?.memberNumber ?? '—');

const childName = computed(() => {
  if (!child.value) return 'Vorname Nachname';
  return `${child.value.firstName} ${child.value.lastName}`;
});

const einstufungYear = computed(() => props.einstufung.year);

const feeRuleText = computed(() => {
  const e = props.einstufung;
  if (e.highestRateVoluntary) {
    return 'Die Einstufung erfolgte aufgrund der freiwilligen Anerkennung des Höchstsatzes.';
  }
  if (e.feeRule === 'beitragsfrei') {
    return `Die Einstufung wurde aufgrund der eingereichten Einkommensnachweise vorgenommen. Gemäß Elternentlastungsgesetz ist der Beitrag beitragsfrei. Bei Änderungen informiert uns bitte umgehend.`;
  }
  const ruleRef = e.feeRule.includes('Entlastung')
    ? 'nach dem Elternentlastungsgesetz 2023/2024'
    : 'nach der Elternbeitragssatzung';
  return `Die Einstufung wurde aufgrund der eingereichten Einkommensnachweise und ${ruleRef} vorgenommen. Bei Änderungen informiert uns bitte umgehend.`;
});

function formatEur(amount: number): string {
  return amount.toLocaleString('de-DE', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) + ' €';
}

async function waitForPdfLayout() {
  await nextTick();

  if ('fonts' in document) {
    try {
      await document.fonts.ready;
    } catch {
      // Ignore font loading issues and continue with the current layout state.
    }
  }

  await new Promise<void>((resolve) => {
    requestAnimationFrame(() => {
      requestAnimationFrame(() => resolve());
    });
  });
}

async function generatePdf() {
  if (!pdfContainer.value) return;
  isGenerating.value = true;

  try {
    await waitForPdfLayout();
    const printWindow = window.open('', '_blank', 'width=960,height=1200');
    if (!printWindow) {
      console.error('Print window could not be opened');
      return;
    }

    const safeChildName = childName.value
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#39;');

    const title = `Einstufung_${einstufungYear.value}_${safeChildName.replace(/\s+/g, '_')}`;
    const documentHtml = `<!doctype html>
<html lang="de">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>${title}</title>
  <style>
${printStyles}

@page {
  size: A4;
  margin: 10mm 12mm 12mm 12mm;
}

html,
body {
  margin: 0;
  padding: 0;
  background: #fff;
}

body {
  -webkit-print-color-adjust: exact;
  print-color-adjust: exact;
  padding: 0;
}

.print-root {
  padding: 0;
  margin: 0;
}

  </style>
</head>
<body>
  <div class="print-root">${pdfContainer.value.outerHTML}</div>
  <script>
    window.addEventListener('load', function () {
      window.focus();
      setTimeout(function () {
        window.print();
      }, 40);
    });
    window.addEventListener('afterprint', function () {
      window.close();
    });
  <\/script>
</body>
</html>`;

    printWindow.document.open();
    printWindow.document.write(documentHtml);
    printWindow.document.close();
  } finally {
    isGenerating.value = false;
  }
}

defineExpose({ generatePdf });
</script>

<template>
  <div>
    <!-- Download button -->
    <button
      @click="generatePdf"
      :disabled="isGenerating"
      class="inline-flex items-center gap-2 px-4 py-2 text-sm text-white bg-primary rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
    >
      <Loader2 v-if="isGenerating" class="h-4 w-4 animate-spin" />
      <FileDown v-else class="h-4 w-4" />
      PDF drucken / speichern
    </button>

    <!-- Hidden print layout -->
    <div class="pdf-stage" aria-hidden="true">
      <div ref="pdfContainer" class="page">

        <!-- page-header -->
        <div class="page-header">
          <div class="page-header__sender">
            Elternverein Kita Knirpsenstadt e.V. &middot; Ahornallee 27 &middot; 16341 Panketal
          </div>
          <div class="page-header__sub">Der Vorstand der Kita</div>
          <div class="page-header__rule"></div>
        </div>

        <!-- title -->
        <div class="title">
          Einstufung Elternbeiträge {{ einstufungYear }}
        </div>

        <!-- info-grid -->
        <div class="info-grid">
          <div class="info-grid__name">{{ childName }}</div>
          <div class="info-grid__fields">
            <div class="info-item">
              <div class="info-item__label">Geburtsdatum</div>
              <div class="info-item__value">{{ birthDateFormatted }}</div>
            </div>
            <div class="info-item">
              <div class="info-item__label">Besucht seit</div>
              <div class="info-item__value">{{ entryDateFormatted }}</div>
            </div>
            <div class="info-item">
              <div class="info-item__label">Mitgliedsnummer</div>
              <div class="info-item__value">{{ memberNumber }}</div>
            </div>
            <div class="info-item info-item--wide">
              <div class="info-item__label">Einrichtung</div>
              <div class="info-item__value">Kita Knirpsenstadt e.V., Ahornallee 27, 16341 Panketal</div>
            </div>
          </div>
        </div>

        <!-- section: Rechtstext -->
        <section class="section">
          <p class="body-text">
            Nach § 17 des Kindertagesstättengesetzes haben die Erziehungsberechtigten Beiträge zur
            Inanspruchnahme eines Platzes in der Kindertagesstätte zu entrichten. Dieser monatliche
            Elternbeitrag wird in Verbindung mit der Elternbeitragsordnung des Trägers ermittelt.
            Die Kindertagesstätte „Knirpsenstadt" in 16341 Panketal, Ahornallee 27 befindet sich in
            freier Trägerschaft des „Knirpsenstadt e.V. Panketal".
          </p>
          <p class="body-text">
            Berechnet wird nach wirtschaftlicher Leistungsfähigkeit (Nettoeinkommen im Jahr), dem
            Alter des Kindes und der beanspruchten Betreuungszeit. Eine Ermäßigung des Elternbeitrages
            wird auch nach der Anzahl der unterhaltspflichtigen Kinder gewährt (jedoch nicht nach dem
            Brandenburg Entlastungspaket).
          </p>
        </section>

        <!-- notice-box: Einstufungsgrundlage -->
        <div class="notice-box">
          <div class="notice-box__label">Grundlage der Einstufung</div>
          <div class="notice-box__text">{{ feeRuleText }}</div>
        </div>

        <!-- section: Beitragsübersicht -->
        <section class="section">
          <div class="section__heading">Monatliche Beiträge</div>

          <table class="fee-table">
            <thead>
              <tr>
                <th class="fee-table__col-label"></th>
                <th
                  v-for="(col, idx) in feeColumns"
                  :key="col.label"
                  class="fee-table__col-month"
                  :class="{ 'fee-table__col-month--first': idx === 0 }"
                >
                  <div class="fee-table__month-name">{{ col.label }}</div>
                  <div class="fee-table__month-sub">{{ col.careType }} &middot; {{ col.careHours }} h/Woche</div>
                </th>
              </tr>
            </thead>
            <tbody>
              <tr class="fee-table__row">
                <td class="fee-table__row-label">Platzgeld</td>
                <td
                  v-for="(col, idx) in feeColumns"
                  :key="col.label"
                  class="fee-table__amount"
                  :class="{ 'fee-table__amount--primary': idx === 0 }"
                >
                  {{ formatEur(col.childcareFee) }}
                </td>
              </tr>
              <tr class="fee-table__row">
                <td class="fee-table__row-label">Essensgeld</td>
                <td
                  v-for="(col, idx) in feeColumns"
                  :key="col.label"
                  class="fee-table__amount"
                  :class="{ 'fee-table__amount--primary': idx === 0 }"
                >
                  {{ formatEur(col.foodFee) }}
                </td>
              </tr>
              <tr v-if="feeColumns.some(c => c.membershipFee > 0)" class="fee-table__row fee-table__row--membership">
                <td class="fee-table__row-label fee-table__row-label--membership">Vereinsbeitrag (jährlich)</td>
                <td
                  v-for="(col, idx) in feeColumns"
                  :key="col.label"
                  class="fee-table__amount fee-table__amount--membership"
                  :class="{ 'fee-table__amount--primary': idx === 0 }"
                >
                  {{ col.membershipFee > 0 ? formatEur(col.membershipFee) : '—' }}
                </td>
              </tr>
            </tbody>
          </table>
        </section>

        <!-- payment-note -->
        <div class="payment-note">
          <span class="payment-note__marker">Hinweis &ndash;</span>
          <span class="payment-note__text">
            Bitte gleicht die Beträge für Mitgliedschaft, Betreuung und Essensgeld in
            <strong class="payment-note__emphasis">getrennten</strong> Zahlungen unter Angabe des
            Namens und der Mitgliedsnummer aus.
          </span>
        </div>

        <!-- section: Zahlungsbedingungen & Änderungspflicht -->
        <section class="section">
          <div class="section__heading">Zahlungsbedingungen</div>
          <p class="body-text">
            Der monatliche Beitrag wird am 5. eines jeden Monats fällig. Beiträge, die
            in Verzug sind, werden zusätzlich mit einer Mahngebühr von 10,00 € erhoben.
          </p>
          <p class="body-text">
            Der Vereinsbeitrag (derzeit 30,00 €) ist jährlich zu zahlen: Bei Vertragsbeginn sofort,
            ansonsten bis spätestens Ende des ersten Quartals. Nach Fristablauf wird ein Mahngeld
            von 5,00 € erhoben.
          </p>
          <div class="section__heading section__heading--sub">Änderungspflicht</div>
          <p class="body-text">
            Wenn sich das Nettoeinkommen im laufenden Jahr gegenüber dem Vorjahr (bzw. bei
            Selbständigen gegenüber der letzten Festsetzung) um mehr als 10 % verändert, ist dies
            unter Vorlage entsprechender Nachweise unverzüglich anzuzeigen.
          </p>
        </section>

        <!-- footer -->
        <footer class="footer">
          <div class="footer__rule"></div>
          <div class="footer__register">
            Kita Knirpsenstadt e.V. &middot; Vereinsregister VR 4217 beim Amtsgericht Frankfurt (Oder)
          </div>
          <div class="footer__columns">
            <div class="footer__col">
              <div class="footer__col-heading">Vorstandsmitglieder</div>
              <div class="footer__line">André Rüger (1. Vorsitzender)</div>
              <div class="footer__line">Sarah Thränhardt (2. Vorsitzende / Bauliches)</div>
              <div class="footer__line">Marcus Rehaag (Finanzen)</div>
              <div class="footer__line">Stefan Remer (Elternarbeit)</div>
              <div class="footer__line">Samantha Lahl (Schriftführerin)</div>
              <div class="footer__line">Dennis Braak (Personal)</div>
            </div>
            <div class="footer__col">
              <div class="footer__col-heading">Bankverbindung</div>
              <div class="footer__line">Knirpsenstadt e. V.</div>
              <div class="footer__line">IBAN: DE33 3702 0500 0003 3214 00</div>
              <div class="footer__line">BIC: BFSWDE33XXX</div>
              <div class="footer__line">Bank für Sozialwirtschaft AG</div>
            </div>
          </div>
          <div class="footer__legal">
            Rechtlich verbindliche Aussagen für den Verein trifft allein der Vorstand.
          </div>
        </footer>

      </div>
    </div>
  </div>
</template>

<style scoped src="./EinstufungPDF.css"></style>
