const screens = Array.from(document.querySelectorAll('.screen'))
const title = document.querySelector('#screen-title')
const flowSteps = Array.from(document.querySelectorAll('.flow-step'))
const tabButtons = Array.from(document.querySelectorAll('.tabbar button'))
const toast = document.querySelector('#toast')
const historyStack = ['home']

document.addEventListener('click', (event) => {
  const toastButton = event.target.closest('[data-toast]')
  if (toastButton) {
    showToast(toastButton.dataset.toast)
    return
  }

  const screenButton = event.target.closest('[data-screen]')
  if (screenButton) {
    showScreen(screenButton.dataset.screen)
    return
  }

  if (event.target.closest('[data-back]')) {
    goBack()
  }
})

function showScreen(name, push = true) {
  const next = document.querySelector(`#${name}-screen`)
  if (!next) return

  screens.forEach((screen) => screen.classList.remove('active'))
  next.classList.add('active')
  title.textContent = next.dataset.title || '服链通'

  flowSteps.forEach((step) => {
    step.classList.toggle('active', step.dataset.screen === name)
  })

  tabButtons.forEach((button) => {
    button.classList.toggle('active', button.dataset.screen === name)
  })

  document.querySelector('.phone-screen').scrollTop = 0
  if (push && historyStack[historyStack.length - 1] !== name) {
    historyStack.push(name)
  }
}

function goBack() {
  if (historyStack.length <= 1) {
    showScreen('home', false)
    return
  }
  historyStack.pop()
  showScreen(historyStack[historyStack.length - 1], false)
}

function showToast(message) {
  toast.textContent = message
  toast.classList.add('show')
  window.clearTimeout(showToast.timer)
  showToast.timer = window.setTimeout(() => {
    toast.classList.remove('show')
  }, 1800)
}
