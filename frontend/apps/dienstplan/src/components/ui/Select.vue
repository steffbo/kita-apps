<script setup lang="ts">
import { type HTMLAttributes, computed, ref } from 'vue';
import {
  SelectRoot,
  SelectTrigger,
  SelectValue,
  SelectPortal,
  SelectContent,
  SelectViewport,
  SelectItem,
  SelectItemText,
  SelectItemIndicator,
} from 'radix-vue';
import { Check, ChevronDown } from 'lucide-vue-next';
import { cn } from '@/lib/utils';

export interface SelectOption {
  value: string;
  label: string;
  disabled?: boolean;
}

const props = defineProps<{
  modelValue?: string;
  options: SelectOption[];
  placeholder?: string;
  class?: HTMLAttributes['class'];
  disabled?: boolean;
  required?: boolean;
  name?: string;
}>();

const emit = defineEmits<{
  'update:modelValue': [value: string];
}>();

// Local model for uncontrolled use
const localValue = ref(props.modelValue ?? '');

// Use computed to support both controlled and uncontrolled modes
const value = computed({
  get: () => props.modelValue ?? localValue.value,
  set: (val) => {
    localValue.value = val;
    emit('update:modelValue', val);
  }
});
</script>

<template>
  <SelectRoot v-model="value" :disabled="disabled" :required="required" :name="name">
    <SelectTrigger
      :class="cn(
        'flex h-10 w-full items-center justify-between rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 [&>span]:line-clamp-1',
        props.class
      )"
    >
      <SelectValue :placeholder="placeholder" />
      <ChevronDown class="h-4 w-4 opacity-50" />
    </SelectTrigger>

    <SelectPortal>
      <SelectContent
        class="relative z-50 max-h-96 min-w-32 overflow-hidden rounded-md border bg-popover text-popover-foreground shadow-md data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2"
        :side-offset="4"
        position="popper"
      >
        <SelectViewport class="p-1 max-h-[300px] w-full min-w-[var(--radix-select-trigger-width)]">
          <SelectItem
            v-for="option in options"
            :key="option.value"
            :value="option.value"
            :disabled="option.disabled"
            class="relative flex w-full cursor-default select-none items-center rounded-sm py-1.5 pl-8 pr-2 text-sm outline-none focus:bg-accent focus:text-accent-foreground data-[disabled]:pointer-events-none data-[disabled]:opacity-50"
          >
            <SelectItemIndicator class="absolute left-2 flex h-3.5 w-3.5 items-center justify-center">
              <Check class="h-4 w-4" />
            </SelectItemIndicator>
            <SelectItemText>{{ option.label }}</SelectItemText>
          </SelectItem>
        </SelectViewport>
      </SelectContent>
    </SelectPortal>
  </SelectRoot>
</template>
