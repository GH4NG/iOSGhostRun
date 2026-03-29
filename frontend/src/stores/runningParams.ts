import { defineStore } from 'pinia'
import { ref } from 'vue'

export interface RunningParams {
    speed: number
    speedVariance: number
    routeOffset: number
    loopCount: number
}

export const useRunningParamsStore = defineStore(
    'runningParams',
    () => {
        // 默认参数
        const DEFAULT_PARAMS: RunningParams = {
            speed: 8,
            speedVariance: 10,
            routeOffset: 2,
            loopCount: 1
        }

        const params = ref<RunningParams>(DEFAULT_PARAMS)

        const setParams = (newParams: Partial<RunningParams>) => {
            params.value = { ...params.value, ...newParams }
        }

        const resetParams = () => {
            params.value = DEFAULT_PARAMS
        }

        return {
            params,
            setParams,
            resetParams
        }
    },
    {
        persist: true
    }
)
