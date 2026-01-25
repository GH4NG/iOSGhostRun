import { ref } from 'vue'

export type NotificationType = 'success' | 'error' | 'warning' | 'info'

interface Notification {
    id: string
    type: NotificationType
    message: string
    duration: number
}

// 全局通知列表
const notifications = ref<Notification[]>([])
let notificationId = 0

export function useNotification() {
    const showNotification = (options: { type?: NotificationType; message: string; duration?: number }) => {
        const id = `notification-${++notificationId}`
        const duration = options.duration ?? 3000

        const notification: Notification = {
            id,
            type: options.type || 'info',
            message: options.message,
            duration
        }

        notifications.value.push(notification)

        // 自动移除通知
        if (duration > 0) {
            setTimeout(() => {
                removeNotification(id)
            }, duration)
        }

        return id
    }

    const removeNotification = (id: string) => {
        const index = notifications.value.findIndex(n => n.id === id)
        if (index > -1) {
            notifications.value.splice(index, 1)
        }
    }

    const showSuccess = (message: string, duration?: number) => {
        return showNotification({
            type: 'success',
            message,
            duration
        })
    }

    const showError = (message: string, duration?: number) => {
        return showNotification({
            type: 'error',
            message,
            duration
        })
    }

    const showWarning = (message: string, duration?: number) => {
        return showNotification({
            type: 'warning',
            message,
            duration
        })
    }

    const showInfo = (message: string, duration?: number) => {
        return showNotification({
            type: 'info',
            message,
            duration
        })
    }

    return {
        notifications,
        showNotification,
        showSuccess,
        showError,
        showWarning,
        showInfo,
        removeNotification
    }
}
