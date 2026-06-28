import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'list', component: () => import('@/views/TaskList.vue') },
    { path: '/tasks/:id', name: 'detail', component: () => import('@/views/TaskDetail.vue'), props: true },
  ],
})

export default router
