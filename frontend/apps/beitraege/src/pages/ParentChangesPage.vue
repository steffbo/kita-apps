<script setup lang="ts">
import ActivityList from '@/components/family/ActivityList.vue';
</script>

<template>
  <div>
    <div class="mb-6">
      <h1 class="text-2xl font-bold text-foreground">Änderungen von Eltern</h1>
      <p class="text-muted-foreground mt-1">
        Geänderte Kontakt- und Kinderdaten, gemeldete Elternstunden und Fehlermeldungen (neueste zuerst)
      </p>
    </div>
    <ActivityList :limit="100" />
  </div>
</template>
