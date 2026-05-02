import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'dashboard',
      component: () => import('../views/DashboardView.vue'),
    },
    {
      path: '/instances',
      name: 'instances',
      component: () => import('../views/InstanceListView.vue'),
    },
    {
      path: '/instances/create',
      name: 'create-instance',
      component: () => import('../views/CreateInstanceView.vue'),
    },
    {
      path: '/instances/:id',
      name: 'instance-detail',
      component: () => import('../views/InstanceDetailView.vue'),
    },
  ],
})

export default router