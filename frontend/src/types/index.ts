export interface Instance {
  id: string
  name: string
  rpc_port: number
  rpc_secret: string
  dir: string
  status: 'running' | 'stopped' | 'error'
  pid: number
  config_json: Record<string, string>
  created_at: string
  updated_at: string
}

export interface TaskProgress {
  gid: string
  name: string
  status: string
  total_length: number
  completed_length: number
  download_speed: number
  upload_speed: number
  upload_length: number
  connections: number
  num_seeders: number
  error_code: string
  error_message: string
  dir: string
}

export interface TaskRecord {
  id: number
  instance_id: string
  gid: string
  name: string
  uris: string
  dir: string
  files_json: string
  total_length: number
  completed_length: number
  download_speed: number
  upload_length: number
  status: 'complete' | 'error'
  error_code: number
  error_message: string
  info_hash: string
  completed_at: string
}

export interface PaginatedResult {
  records: TaskRecord[]
  total: number
  page: number
  per_page: number
}

export interface GlobalStats {
  total_download_speed: number
  total_upload_speed: number
  num_active_downloads: number
  num_running_instances: number
}

export interface WSMessage {
  type: string
  instance_id?: string
  data?: any
}

export interface CreateInstanceRequest {
  name: string
  dir?: string
  options?: Record<string, string>
}

export interface AddTaskRequest {
  type: 'uri' | 'torrent' | 'metalink'
  uris?: string[]
  torrent?: string
  metalink?: string
  options?: Record<string, string>
}