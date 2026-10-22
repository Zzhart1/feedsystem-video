// 极简轻提示（纯 JS 工具模块）：Toaster.success('xxx') / Toaster.error('xxx') / Toaster.info('xxx')
import { h, render } from 'vue'

let container = null
let seq = 0

function show(type, text) {
  if (!container) {
    container = document.createElement('div')
    container.className = 'toast-host'
    document.body.appendChild(container)
  }
  const id = ++seq
  const el = document.createElement('div')
  render(
    h(
      'div',
      {
        class: `toast toast-${type}`,
        style: `animation: toastIn .22s ease, toastOut .3s ease ${2.4}s forwards`,
      },
      text,
    ),
    el,
  )
  container.appendChild(el)
  setTimeout(() => {
    render(null, el)
    el.remove()
  }, 2900)
}

export default {
  success: (t) => show('success', t),
  error: (t) => show('error', t),
  info: (t) => show('info', t),
}
