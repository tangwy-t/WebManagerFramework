/**
 * 插件装配入口
 */
import type { App } from 'vue'
import type { Router } from 'vue-router'
import { pluginManager } from './plugin/manager'

/** 安装插件（注册 store、合并语言、执行 install） */
export function setupPlugins(app: App, router: Router): void {
  pluginManager.install(app, router)
}

/** 挂载插件（执行 mount） */
export function mountPlugins(): void {
  pluginManager.mount()
}
