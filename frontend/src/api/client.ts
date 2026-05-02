import axios from 'axios'
import type {
  Instance,
  CreateInstanceRequest,
  TaskProgress,
  PaginatedResult,
  GlobalStats,
  AddTaskRequest,
} from '../types'

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
})

api.interceptors.response.use(
  (response) => response.data,
  (error) => {
    throw error.response?.data || { error: error.message }
  },
)

export const listInstances = (): Promise<Instance[]> => api.get('/instances')
export const getInstance = (id: string): Promise<Instance> => api.get(`/instances/${id}`)
export const createInstance = (data: CreateInstanceRequest): Promise<Instance> => api.post('/instances', data)
export const updateInstance = (id: string, data: Partial<CreateInstanceRequest>): Promise<Instance> => api.put(`/instances/${id}`, data)
export const deleteInstance = (id: string): Promise<void> => api.delete(`/instances/${id}`)
export const startInstance = (id: string): Promise<Instance> => api.post(`/instances/${id}/start`)
export const stopInstance = (id: string): Promise<Instance> => api.post(`/instances/${id}/stop`)
export const restartInstance = (id: string): Promise<Instance> => api.post(`/instances/${id}/restart`)

export const listActiveTasks = (id: string): Promise<TaskProgress[]> => api.get(`/instances/${id}/tasks/active`)
export const listWaitingTasks = (id: string): Promise<TaskProgress[]> => api.get(`/instances/${id}/tasks/waiting`)
export const listStoppedTasks = (id: string): Promise<TaskProgress[]> => api.get(`/instances/${id}/tasks/stopped`)
export const addTask = (id: string, data: AddTaskRequest): Promise<{ gid: string } | { gids: string[] }> => api.post(`/instances/${id}/tasks`, data)
export const getTaskStatus = (id: string, gid: string): Promise<TaskProgress> => api.get(`/instances/${id}/tasks/${gid}`)
export const pauseTask = (id: string, gid: string): Promise<void> => api.post(`/instances/${id}/tasks/${gid}/pause`)
export const unpauseTask = (id: string, gid: string): Promise<void> => api.post(`/instances/${id}/tasks/${gid}/unpause`)
export const removeTask = (id: string, gid: string, deleteFiles?: boolean): Promise<void> =>
  api.delete(`/instances/${id}/tasks/${gid}`, { params: deleteFiles ? { delete_files: 'true' } : {} })

export const listHistory = (id: string, page?: number, perPage?: number): Promise<PaginatedResult> =>
  api.get(`/instances/${id}/history`, { params: { page, per_page: perPage } })
export const deleteHistoryRecord = (id: string, gid: string, deleteFiles?: boolean): Promise<void> =>
  api.delete(`/instances/${id}/history/${gid}`, { params: deleteFiles ? { delete_files: 'true' } : {} })

export const getGlobalStats = (): Promise<GlobalStats> => api.get('/stats')

export default api