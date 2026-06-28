const flowSteps = Array.from(document.querySelectorAll('[data-screen]'))
const screens = Array.from(document.querySelectorAll('.screen'))
const title = document.querySelector('#screen-title')
const backButton = document.querySelector('[data-back]')
const stack = ['home']

function showScreen(name, push = true) {
  const target = document.querySelector(`#${name}-screen`)
  if (!target) return
  screens.forEach((screen) => screen.classList.toggle('active', screen === target))
  document.querySelectorAll('.flow-step').forEach((step) => {
    step.classList.toggle('active', step.dataset.screen === name)
  })
  title.textContent = target.dataset.title || '服链通'
  if (push && stack[stack.length - 1] !== name) {
    stack.push(name)
  }
  const scroller = document.querySelector('.phone-screen')
  scroller.scrollTop = 0
}

flowSteps.forEach((item) => {
  item.addEventListener('click', () => showScreen(item.dataset.screen))
})

backButton.addEventListener('click', () => {
  if (stack.length <= 1) {
    showScreen('home', false)
    return
  }
  stack.pop()
  showScreen(stack[stack.length - 1], false)
})

document.querySelectorAll('.filter-row button, .message-tabs button, .publish-types button, .role-switch button').forEach((button) => {
  button.addEventListener('click', () => {
    const group = button.parentElement
    group.querySelectorAll('button').forEach((item) => item.classList.remove('active'))
    button.classList.add('active')
  })
})
