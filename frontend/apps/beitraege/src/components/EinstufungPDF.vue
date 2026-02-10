<script setup lang="ts">
import { computed, ref } from 'vue';
import type { Einstufung, Child } from '@/api/types';
import { FileDown, Loader2 } from 'lucide-vue-next';

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

async function generatePdf() {
  if (!pdfContainer.value) return;
  isGenerating.value = true;

  try {
    const html2pdf = (await import('html2pdf.js')).default;
    const opt = {
      margin: [10, 12, 15, 12],
      filename: `Einstufung_${einstufungYear.value}_${childName.value.replace(/\s/g, '_')}.pdf`,
      image: { type: 'jpeg', quality: 0.98 },
      html2canvas: { scale: 2, useCORS: true },
      jsPDF: { unit: 'mm', format: 'a4', orientation: 'portrait' as const },
    };

    await html2pdf().set(opt).from(pdfContainer.value).save();
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
      PDF herunterladen
    </button>

    <!-- Hidden PDF content (rendered off-screen for html2pdf) -->
    <div class="fixed left-[-9999px] top-0">
      <div
        ref="pdfContainer"
        style="width: 186mm; font-family: 'Segoe UI', -apple-system, BlinkMacSystemFont, Arial, sans-serif; font-size: 10.5px; color: #1a1a1a; line-height: 1.5; padding: 0; background: #fff;"
      >
        <!-- Header -->
        <div style="margin-bottom: 24px;">
          <div style="font-size: 9px; color: #666; margin-bottom: 2px; letter-spacing: 0.3px;">
            Elternverein Kita Knirpsenstadt e.V. · Ahornallee 27 · 16341 Panketal
          </div>
          <div style="font-size: 10px; color: #888; font-weight: 500;">
            Der Vorstand der Kita
          </div>
        </div>

        <!-- Title -->
        <div style="font-size: 20px; font-weight: 700; color: #16a34a; margin-bottom: 20px; letter-spacing: -0.3px;">
          Einstufung Elternbeiträge {{ einstufungYear }}
        </div>

        <!-- Child info card -->
        <div style="background: linear-gradient(135deg, #f0fdf4 0%, #dcfce7 100%); border-left: 4px solid #16a34a; padding: 16px 18px; margin-bottom: 20px; border-radius: 6px;">
          <div style="font-size: 13px; font-weight: 700; color: #15803d; margin-bottom: 12px;">
            {{ childName }}
          </div>
          <div style="display: flex; flex-wrap: wrap; gap: 16px; font-size: 10px;">
            <div style="flex: 1; min-width: 120px;">
              <div style="color: #666; margin-bottom: 3px;">Geburtsdatum</div>
              <div style="font-weight: 600; color: #1a1a1a;">{{ birthDateFormatted }}</div>
            </div>
            <div style="flex: 1; min-width: 120px;">
              <div style="color: #666; margin-bottom: 3px;">Besucht seit</div>
              <div style="font-weight: 600; color: #1a1a1a;">{{ entryDateFormatted }}</div>
            </div>
            <div style="flex: 1; min-width: 120px;">
              <div style="color: #666; margin-bottom: 3px;">Mitgliedsnummer</div>
              <div style="font-weight: 600; color: #1a1a1a;">{{ memberNumber }}</div>
            </div>
            <div style="flex: 2; min-width: 180px;">
              <div style="color: #666; margin-bottom: 3px;">Einrichtung</div>
              <div style="font-weight: 600; color: #1a1a1a; line-height: 1.3;">
                Kita Knirpsenstadt e.V.<br>
                Ahornallee 27, 16341 Panketal
              </div>
            </div>
          </div>
        </div>

        <!-- Legal intro -->
        <div style="font-size: 10px; margin-bottom: 16px; text-align: justify; color: #444; line-height: 1.6;">
          Nach § 17 des Kindertagesstättengesetzes haben die Erziehungsberechtigten Beiträge zur Inanspruchnahme eines Platzes in
          der Kindertagesstätte zu entrichten. Dieser monatliche Elternbeitrag wird in Verbindung mit der Elternbeitragsordnung des Trägers ermittelt.
          Die Kindertagesstätte „Knirpsenstadt" in 16341 Panketal, Ahornallee 27 befindet sich in freier Trägerschaft des „Knirpsenstadt e.V. Panketal".
        </div>
        <div style="font-size: 10px; margin-bottom: 18px; text-align: justify; color: #444; line-height: 1.6;">
          Berechnet wird nach wirtschaftlicher Leistungsfähigkeit (Nettoeinkommen im Jahr), dem Alter des Kindes und der beanspruchten Betreuungszeit. Eine
          Ermäßigung des Elternbeitrages wird auch nach der Anzahl der unterhaltspflichtigen Kinder gewährt (jedoch nicht nach dem Brandenburg Entlastungspaket).
        </div>

        <!-- Einstufung basis box -->
        <div style="background: #f0fdf4; border: 2px solid #16a34a; border-radius: 6px; padding: 12px 16px; margin-bottom: 20px; text-align: center;">
          <div style="font-size: 10.5px; font-weight: 600; color: #15803d; line-height: 1.5;">
            {{ feeRuleText }}
          </div>
        </div>

        <!-- Fee breakdown header -->
        <div style="font-size: 13px; font-weight: 700; color: #1a1a1a; margin-bottom: 12px; border-bottom: 2px solid #16a34a; padding-bottom: 6px;">
          Monatliche Beiträge
        </div>

        <!-- Fee columns as cards -->
        <div style="display: flex; gap: 10px; margin-bottom: 18px;">
          <div
            v-for="(col, idx) in feeColumns"
            :key="col.label"
            :style="{
              flex: 1,
              background: idx === 0 ? '#fff7ed' : '#f9fafb',
              border: idx === 0 ? '2px solid #f97316' : '1px solid #e5e7eb',
              borderRadius: '6px',
              padding: '12px',
            }"
          >
            <div style="text-align: center; margin-bottom: 10px;">
              <div :style="{ fontSize: '11px', fontWeight: 700, color: idx === 0 ? '#ea580c' : '#16a34a', marginBottom: '2px' }">
                {{ col.label }}
              </div>
              <div style="fontSize: '9px', color: '#666;">
                {{ col.careType }} · {{ col.careHours }}h/Woche
              </div>
            </div>
            <div style="border-top: 1px solid #e5e7eb; padding-top: 8px;">
              <div style="display: flex; justify-content: space-between; align-items: baseline; margin-bottom: 6px;">
                <span style="fontSize: '9px'; color: '#666';">Platzgeld</span>
                <span style="fontSize: '12px'; fontWeight: 700; color: '#1a1a1a';">{{ formatEur(col.childcareFee) }}</span>
              </div>
              <div style="display: flex; justify-content: space-between; align-items: baseline; margin-bottom: 6px;">
                <span style="fontSize: '9px'; color: '#666';">Essensgeld</span>
                <span style="fontSize: '11px'; fontWeight: 600; color: '#1a1a1a';">{{ formatEur(col.foodFee) }}</span>
              </div>
              <div v-if="col.membershipFee > 0" style="display: flex; justify-content: space-between; align-items: baseline; padding-top: 6px; border-top: 1px dashed #e5e7eb;">
                <span style="fontSize: '9px'; color: '#ea580c'; fontWeight: 600;">Vereinsbeitrag (jährlich)</span>
                <span style="fontSize: '11px'; fontWeight: 700; color: '#ea580c';">{{ formatEur(col.membershipFee) }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Payment notice -->
        <div style="background: #fef2f2; border: 2px solid #dc2626; border-radius: 6px; padding: 10px 14px; margin-bottom: 18px;">
          <div style="text-align: center; color: #991b1b; font-weight: 700; font-size: 10.5px; line-height: 1.5;">
            ⚠️ Bitte gleicht die Beträge für Mitgliedschaft, Betreuung und Essensgeld in <span style="text-decoration: underline;">getrennten</span> Zahlungen unter Angabe des Namens und der Mitgliedsnummer aus.
          </div>
        </div>

        <!-- Payment terms -->
        <div style="font-size: 10px; color: #444; line-height: 1.6; margin-bottom: 8px; text-align: justify;">
          <strong>Zahlungsbedingungen:</strong> Der monatliche Beitrag wird am 5. eines jeden Monats fällig. Beiträge, die einen Monat in Verzug sind, werden zusätzlich mit einer Mahngebühr von 10,00 € erhoben.
        </div>
        <div style="font-size: 10px; color: #444; line-height: 1.6; margin-bottom: 8px; text-align: justify;">
          Der Vereinsbeitrag (derzeit 30,00 €) ist jährlich zu zahlen: Bei Vertragsbeginn sofort, ansonsten bis spätestens Ende des ersten Quartals. Nach Fristablauf wird ein Mahngeld von 5,00 € erhoben.
        </div>
        <div style="font-size: 10px; color: #444; line-height: 1.6; margin-bottom: 20px; text-align: justify;">
          <strong>Änderungspflicht:</strong> Wenn sich das Nettoeinkommen im laufenden Jahr gegenüber dem Vorjahr (bzw. bei Selbständigen gegenüber der letzten Festsetzung) um mehr als 10 % verändert, ist dies unter Vorlage entsprechender Nachweise unverzüglich anzuzeigen.
        </div>

        <!-- Footer -->
        <div style="border-top: 2px solid #e5e7eb; padding-top: 12px; margin-top: 24px;">
          <div style="font-size: 9px; color: #666; margin-bottom: 8px; font-weight: 600;">
            Kita Knirpsenstadt e.V. · Vereinsregister VR 4217 beim Amtsgericht Frankfurt (Oder)
          </div>
          <div style="display: flex; gap: 20px; font-size: 8px; color: #666; line-height: 1.4;">
            <div style="flex: 1;">
              <div style="font-weight: 700; margin-bottom: 4px; color: #1a1a1a;">Vorstandsmitglieder</div>
              André Rüger (1. Vorsitzender)<br>
              Sarah Thielandt (2. Vorsitzende / Bauliches)<br>
              Marcus Rehberg (Kassenwart)<br>
              Stefan Remer (Elternarbeit)<br>
              Samantha Lahl (Schriftführer)<br>
              Dennis Braak (Personal)
            </div>
            <div style="flex: 1;">
              <div style="font-weight: 700; margin-bottom: 4px; color: #1a1a1a;">Bankverbindung</div>
              Knirpsenstadt e. V.<br>
              IBAN: DE53 3702 0500 0003 3714 00<br>
              BIC: BFSWDE33XXX<br>
              Bank für Sozialwirtschaft AG
            </div>
          </div>
          <div style="margin-top: 8px; font-size: 8px; color: #888; font-style: italic;">
            Rechtlich verbindliche Aussagen für den Verein trifft allein der Vorstand.
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
