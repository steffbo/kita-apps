<script setup lang="ts">
import AppSheet from '@/components/ui/AppSheet.vue';
import type { EmailLog } from '@/api/types';
import { formatDateTime } from '@/utils/format';
import { formatEmailType } from '@/utils/reminders';

defineProps<{ log: EmailLog }>();
defineEmits<{ close: [] }>();
</script>

<template>
  <AppSheet title="Gesendete E-Mail" size="lg" @close="$emit('close')">
    <div>
      <p class="mb-4 text-sm text-muted-foreground">{{ log.subject }}</p>
      <div>
        <dl class="grid gap-4 text-sm sm:grid-cols-2">
          <div>
            <dt class="font-medium text-muted-foreground">Zeitpunkt</dt>
            <dd class="mt-1 text-foreground">{{ formatDateTime(log.sentAt ?? '') }}</dd>
          </div>
          <div>
            <dt class="font-medium text-muted-foreground">Typ</dt>
            <dd class="mt-1 text-foreground">{{ formatEmailType(log.emailType ?? '') }}</dd>
          </div>
          <div class="sm:col-span-2">
            <dt class="font-medium text-muted-foreground">Empfänger</dt>
            <dd class="mt-1 break-all text-foreground">{{ log.toEmail }}</dd>
          </div>
          <div class="sm:col-span-2">
            <dt class="font-medium text-muted-foreground">Betreff</dt>
            <dd class="mt-1 text-foreground">{{ log.subject }}</dd>
          </div>
        </dl>

        <pre class="mt-5 whitespace-pre-wrap rounded-lg border bg-muted p-4 text-sm leading-6 text-foreground">{{ log.body || '-' }}</pre>
      </div>
    </div>
    <template #footer>
      <button type="button" class="btn btn-secondary" @click="$emit('close')">Schließen</button>
    </template>
  </AppSheet>
</template>
