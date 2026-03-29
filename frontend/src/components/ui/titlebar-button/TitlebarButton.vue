<!-- components/ui/titlebar-button/TitlebarButton.vue -->
<script setup lang="ts">
import type { PrimitiveProps } from "reka-ui"
import type { HTMLAttributes } from "vue"
import { Primitive } from "reka-ui"
import { cn } from "@/lib/utils"

interface Props extends PrimitiveProps {
  class?: HTMLAttributes["class"]
  ariaLabel?: string
}

const props = withDefaults(defineProps<Props>(), {
  as: "button",
})
</script>

<template>
  <Primitive
    :as="as"
    :as-child="asChild"
    type="button"
    :class="cn(
      'flex items-center justify-center shrink-0 transition-colors outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2',
      props.class
    )"
    :aria-label="ariaLabel"
  >
    <slot />
  </Primitive>
</template>
