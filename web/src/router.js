import { createRouter, createWebHashHistory } from 'vue-router'
import CatalogPage from './pages/CatalogPage.vue'
import ServersPage from './pages/ServersPage.vue'
import ServerDetailPage from './pages/ServerDetailPage.vue'
import SettingsPage from './pages/SettingsPage.vue'

export const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', redirect: '/catalog' },
    { path: '/catalog', name: 'catalog', component: CatalogPage },
    { path: '/servers', name: 'servers', component: ServersPage },
    { path: '/servers/:id', name: 'server-detail', component: ServerDetailPage, props: true },
    { path: '/settings', name: 'settings', component: SettingsPage }
  ]
})
