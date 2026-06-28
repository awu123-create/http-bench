import { onUnmounted } from 'vue'

export function useSSE() {
  let es: EventSource | null = null

  function open(url: string, onMetric: (data: unknown) => void, onClose?: () => void) {
    close()
    es = new EventSource(url)
    es.addEventListener('metric', (e: MessageEvent) => {
      try {
        onMetric(JSON.parse(e.data))
      } catch {
        // ignore parse error
      }
    })
    es.onerror = () => {
      // EventSource 遇到错误/服务端关闭时触发，关闭连接并通知
      close()
      onClose?.()
    }
  }

  function close() {
    if (es) {
      es.close()
      es = null
    }
  }

  onUnmounted(close)

  return { open, close }
}
