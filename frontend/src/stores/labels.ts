import { reactive } from 'vue'
import { defineStore } from 'pinia'
import { LoadAppLabels } from '../../wailsjs/go/main/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'

interface LabelJob {
  running: boolean
  current: number
  total: number
  message: string
  results: Record<string, string>
}

export const useLabelStore = defineStore('labels', () => {
  const job = reactive<LabelJob>({
    running: false,
    current: 0,
    total: 0,
    message: '',
    results: {},
  })

  let unlisten: (() => void) | null = null

  function ensureListener() {
    if (unlisten) {
      return
    }
    unlisten = EventsOn('app:progress', (p: { stage: string; message: string; percent: number }) => {
      if (p.stage !== 'labels') {
        return
      }
      job.message = p.message
      const match = p.message.match(/(\d+)\s*\/\s*(\d+)/)
      if (match) {
        job.current = Number(match[1])
        job.total = Number(match[2])
      }
    })
  }

  async function load(serial: string, pkgs: string[]) {
    if (job.running || !pkgs.length) {
      return
    }

    ensureListener()
    job.running = true
    job.current = 0
    job.total = pkgs.length
    job.results = {}
    job.message = '准备加载…'

    try {
      const mapping = await LoadAppLabels(serial, pkgs)
      job.results = mapping
      job.current = pkgs.length
      job.message = `完成 ${Object.keys(mapping).length}/${pkgs.length}`
    } finally {
      job.running = false
    }
  }

  return { job, load }
})
