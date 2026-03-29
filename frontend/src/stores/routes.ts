import { defineStore } from 'pinia'
import { ref } from 'vue'

export interface RoutePoint {
    lat: number
    lon: number
}

export interface SavedRoute {
    name: string
    points: RoutePoint[]
    createdAt: number
}

export const useRoutesStore = defineStore(
    'routes',
    () => {
        const routes = ref<SavedRoute[]>([])

        /**
         * 列出所有路线
         */
        const listRoutes = (): SavedRoute[] => {
            return routes.value
        }

        /**
         * 保存路线
         */
        const saveRoute = (name: string, points: RoutePoint[]): void => {
            const existing = routes.value.findIndex(r => r.name === name)

            const route: SavedRoute = {
                name,
                points,
                createdAt: Date.now()
            }

            if (existing >= 0) {
                routes.value[existing] = route
            } else {
                routes.value.push(route)
            }
        }

        /**
         * 加载路线
         */
        const loadRoute = (name: string): RoutePoint[] | null => {
            const route = routes.value.find(r => r.name === name)
            return route?.points ?? null
        }

        /**
         * 删除路线
         */
        const deleteRoute = (name: string): void => {
            routes.value = routes.value.filter(r => r.name !== name)
        }

        /**
         * 保存上次路线
         */
        const saveLastRoute = (points: RoutePoint[]): void => {
            const route: SavedRoute = {
                name: 'last_route',
                points,
                createdAt: Date.now()
            }

            const existing = routes.value.findIndex(r => r.name === route.name)
            if (existing >= 0) {
                routes.value[existing] = route
            } else {
                routes.value.push(route)
            }
        }

        /**
         * 获取上次路线
         */
        const getLastRoute = (): RoutePoint[] | null => {
            return loadRoute('last_route')
        }

        return {
            routes,
            listRoutes,
            saveRoute,
            loadRoute,
            deleteRoute,
            saveLastRoute,
            getLastRoute
        }
    },
    {
        persist: true
    }
)
