<template>
    <div class="fixed top-6 left-1/2 -translate-x-1/2 z-[9999] flex flex-col gap-2 pointer-events-none">
        <Transition-group enter-active-class="transition duration-300 ease-out"
            enter-from-class="opacity-0 -translate-y-2 scale-95" enter-to-class="opacity-100 translate-y-0 scale-100"
            leave-active-class="transition duration-200 ease-in" leave-from-class="opacity-100 translate-y-0 scale-100"
            leave-to-class="opacity-0 -translate-y-2 scale-95">
            <div v-for="notification in notifications" :key="notification.id"
                :class="getNotificationClass(notification.type)"
                class="px-4 py-3 rounded-lg shadow-lg text-sm font-medium flex items-center gap-3 pointer-events-auto backdrop-blur-md border animation-all">
                <component :is="getIcon(notification.type)" class="w-4 h-4 shrink-0" />
                <span class="flex-1">{{ notification.message }}</span>
            </div>
        </Transition-group>
    </div>
</template>

<script setup lang="ts">
import { useNotification, type NotificationType } from '../composables/useNotification'

const { notifications } = useNotification()

function getNotificationClass(type: NotificationType): string {
    switch (type) {
        case 'success':
            return 'bg-emerald-500/90 text-emerald-50 border-emerald-400/50'
        case 'error':
            return 'bg-destructive/90 text-red-50 border-red-400/50'
        case 'warning':
            return 'bg-amber-500/90 text-amber-50 border-amber-400/50'
        case 'info':
            return 'bg-blue-500/90 text-blue-50 border-blue-400/50'
        default:
            return 'bg-gray-500/90 text-gray-50 border-gray-400/50'
    }
}

function getIcon(type: NotificationType) {
    switch (type) {
        case 'success':
            return SuccessIcon
        case 'error':
            return ErrorIcon
        case 'warning':
            return WarningIcon
        case 'info':
            return InfoIcon
        default:
            return InfoIcon
    }
}

const SuccessIcon = {
    template: `
    <svg fill="currentColor" viewBox="0 0 20 20">
      <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
    </svg>
  `
}

const ErrorIcon = {
    template: `
    <svg fill="currentColor" viewBox="0 0 20 20">
      <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd" />
    </svg>
  `
}

const WarningIcon = {
    template: `
    <svg fill="currentColor" viewBox="0 0 20 20">
      <path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd" />
    </svg>
  `
}

const InfoIcon = {
    template: `
    <svg fill="currentColor" viewBox="0 0 20 20">
      <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clip-rule="evenodd" />
    </svg>
  `
}
</script>
