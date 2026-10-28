// Seedance channel profile stored in credentials.seedance_profile.
// Mirrors backend/internal/service/seedance_profile.go: a preset supplies
// defaults and every non-empty field below overrides it.

export type SeedancePreset = 'ark' | 'openai_videos'
export type SeedanceBillingMode = 'tokens' | 'video'
// Rewrites the documented client body for the channel; '' passes it through.
export type SeedanceRequestFormat = '' | 'sd25' | 'opaigc'

export interface SeedanceProfileForm {
  preset: SeedancePreset
  create_path: string
  status_path: string
  content_path: string
  delete_path: string
  task_id_field: string
  status_field: string
  success_statuses: string
  billing_mode: '' | SeedanceBillingMode
  usage_tokens_field: string
  duration_field: string
  resolution_field: string
  request_duration_field: string
  request_resolution_field: string
  request_format: SeedanceRequestFormat
}

export type SeedanceTextField = Exclude<keyof SeedanceProfileForm, 'preset' | 'billing_mode' | 'request_format'>

export const SEEDANCE_PATH_FIELDS: SeedanceTextField[] = [
  'create_path',
  'status_path',
  'content_path',
  'delete_path'
]

export const SEEDANCE_RESPONSE_FIELDS: SeedanceTextField[] = [
  'task_id_field',
  'status_field',
  'success_statuses',
  'usage_tokens_field',
  'duration_field',
  'resolution_field',
  'request_duration_field',
  'request_resolution_field'
]

// Defaults shown as placeholders; keep in sync with seedancePresetProfile().
export const SEEDANCE_PRESET_DEFAULTS: Record<SeedancePreset, Record<SeedanceTextField, string> & { billing_mode: SeedanceBillingMode }> = {
  ark: {
    create_path: '/api/v3/contents/generations/tasks',
    status_path: '/api/v3/contents/generations/tasks/{task_id}',
    content_path: '',
    delete_path: '/api/v3/contents/generations/tasks/{task_id}',
    task_id_field: 'id',
    status_field: 'status',
    success_statuses: 'succeeded',
    billing_mode: 'tokens',
    usage_tokens_field: 'usage.completion_tokens',
    duration_field: 'duration',
    resolution_field: 'resolution',
    request_duration_field: 'duration',
    request_resolution_field: 'resolution'
  },
  openai_videos: {
    create_path: '/v1/videos',
    status_path: '/v1/videos/{task_id}',
    content_path: '/v1/videos/{task_id}/content',
    delete_path: '',
    task_id_field: 'id',
    status_field: 'status',
    success_statuses: 'completed',
    billing_mode: 'video',
    usage_tokens_field: '',
    duration_field: 'seconds,duration',
    resolution_field: 'resolution,size',
    request_duration_field: 'seconds,duration',
    request_resolution_field: 'resolution,size'
  }
}

export const emptySeedanceProfileForm = (preset: SeedancePreset = 'ark'): SeedanceProfileForm => ({
  preset,
  create_path: '',
  status_path: '',
  content_path: '',
  delete_path: '',
  task_id_field: '',
  status_field: '',
  success_statuses: '',
  billing_mode: '',
  usage_tokens_field: '',
  duration_field: '',
  resolution_field: '',
  request_duration_field: '',
  request_resolution_field: '',
  request_format: ''
})

const asString = (value: unknown): string => (typeof value === 'string' ? value.trim() : '')

export const readSeedanceProfile = (credentials?: Record<string, unknown>): SeedanceProfileForm => {
  const raw = credentials?.seedance_profile
  if (raw === null || typeof raw !== 'object' || Array.isArray(raw)) {
    return emptySeedanceProfileForm()
  }
  const source = raw as Record<string, unknown>
  const form = emptySeedanceProfileForm(source.preset === 'openai_videos' ? 'openai_videos' : 'ark')
  for (const key of [...SEEDANCE_PATH_FIELDS, ...SEEDANCE_RESPONSE_FIELDS]) {
    if (key === 'success_statuses') continue
    form[key] = asString(source[key])
  }
  if (Array.isArray(source.success_statuses)) {
    form.success_statuses = source.success_statuses.map(asString).filter(Boolean).join(',')
  }
  if (source.billing_mode === 'tokens' || source.billing_mode === 'video') {
    form.billing_mode = source.billing_mode
  }
  if (source.request_format === 'sd25' || source.request_format === 'opaigc') {
    form.request_format = source.request_format
  }
  return form
}

// buildSeedanceProfile keeps only explicit overrides so preset defaults can
// evolve on the backend without stale copies saved in every account.
export const buildSeedanceProfile = (form: SeedanceProfileForm): Record<string, unknown> => {
  const profile: Record<string, unknown> = { preset: form.preset }
  for (const key of [...SEEDANCE_PATH_FIELDS, ...SEEDANCE_RESPONSE_FIELDS]) {
    if (key === 'success_statuses') continue
    const value = form[key].trim()
    if (value) profile[key] = value
  }
  const statuses = form.success_statuses
    .split(',')
    .map((value) => value.trim())
    .filter(Boolean)
  if (statuses.length > 0) profile.success_statuses = statuses
  if (form.billing_mode) profile.billing_mode = form.billing_mode
  if (form.request_format) profile.request_format = form.request_format
  return profile
}

// Returns an i18n key for the first problem, or '' when the profile is valid.
// Same rules as ResolveSeedanceProfile so admins see errors before saving.
export const validateSeedanceProfile = (form: SeedanceProfileForm): string => {
  for (const key of SEEDANCE_PATH_FIELDS) {
    const value = form[key].trim()
    if (!value) continue
    if (!value.startsWith('/')) return 'admin.accounts.seedance.errors.pathSlash'
    if (key !== 'create_path' && !value.includes('{task_id}')) {
      return 'admin.accounts.seedance.errors.pathTaskId'
    }
  }
  return ''
}

export const applySeedanceProfile = (
  credentials: Record<string, unknown>,
  enabled: boolean,
  form: SeedanceProfileForm
) => {
  if (!enabled) {
    delete credentials.seedance_profile
    return
  }
  credentials.seedance_profile = buildSeedanceProfile(form)
}
