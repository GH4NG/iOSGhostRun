/**
 * 跑步控制参数的本地存储管理
 */

export interface RunningParams {
    speed: number
    speedVariance: number
    routeOffset: number
    loopCount: number
}

const STORAGE_KEY = 'running-params'

// 默认参数
const DEFAULT_PARAMS: RunningParams = {
    speed: 8,
    speedVariance: 10,
    routeOffset: 2,
    loopCount: 1
}

/**
 * 从 localStorage 读取跑步参数
 */
export function loadRunningParams(): RunningParams {
    try {
        const stored = localStorage.getItem(STORAGE_KEY)
        if (stored) {
            const parsed = JSON.parse(stored)
            // 合并默认值和存储值，确保所有字段都存在
            return { ...DEFAULT_PARAMS, ...parsed }
        }
    } catch (error) {
        console.warn('Failed to load running params from storage:', error)
    }
    return DEFAULT_PARAMS
}

/**
 * 保存跑步参数到 localStorage
 */
export function saveRunningParams(params: RunningParams): void {
    try {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(params))
    } catch (error) {
        console.warn('Failed to save running params to storage:', error)
    }
}

/**
 * 重置为默认参数
 */
export function resetRunningParams(): RunningParams {
    try {
        localStorage.removeItem(STORAGE_KEY)
    } catch (error) {
        console.warn('Failed to reset running params:', error)
    }
    return DEFAULT_PARAMS
}
