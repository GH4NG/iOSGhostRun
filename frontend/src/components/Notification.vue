<template>
    <div class="fixed top-14 left-1/2 -translate-x-1/2 z-[9999] w-max max-w-[calc(100vw-2rem)] max-h-[calc(100dvh-5rem)] overflow-y-auto flex flex-col gap-2 pointer-events-none">
        <TransitionGroup enter-active-class="transition duration-300 ease-out"
            enter-from-class="opacity-0 -translate-y-2 scale-95" enter-to-class="opacity-100 translate-y-0 scale-100"
            leave-active-class="transition duration-200 ease-in" leave-from-class="opacity-100 translate-y-0 scale-100"
            leave-to-class="opacity-0 -translate-y-2 scale-95">
            <div v-for="notification in notifications" :key="notification.id"
                :class="getNotificationClass(notification.type)"
                class="px-4 py-3 rounded-lg shadow-lg text-sm font-medium flex items-center gap-3 pointer-events-auto backdrop-blur-md border animation-all">
                <component :is="getIcon(notification.type)" class="w-4 h-4 shrink-0" />
                <span class="flex-1 min-w-0 break-words">{{ notification.message }}</span>
            </div>
        </TransitionGroup>
    </div>
</template>

<script setup lang="ts">
import { useNotification, type NotificationType } from '../composables/useNotification'
import { CheckCircledIcon, CrossCircledIcon, ExclamationTriangleIcon, InfoCircledIcon } from '@radix-icons/vue'

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
    return {
        success: CheckCircledIcon,
        error: CrossCircledIcon,
        warning: ExclamationTriangleIcon,
        info: InfoCircledIcon
    }[type]
}
</script>
