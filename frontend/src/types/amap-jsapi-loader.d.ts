import '@amap/amap-jsapi-loader'

// 官方 1.0.1 的运行时提供 reset，但其类型声明遗漏了该方法。
declare module '@amap/amap-jsapi-loader' {
    export function reset(): void
}
