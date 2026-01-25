<script setup lang="ts">
import { AlertDialogTrigger, type AlertDialogTriggerProps } from 'reka-ui'

defineOptions({
  inheritAttrs: false,
})

const props = defineProps<AlertDialogTriggerProps>()
</script>

<template>
  <AlertDialogTrigger v-bind="{ ...props, ...$attrs }">
    <slot />
  </AlertDialogTrigger>
</template>
