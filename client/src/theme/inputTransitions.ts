const transitionStyleId = 'transitions-p12'

const transitionStyles = `
:root {
  --shake-distance: 6px;
  --shake-overshoot: 4px;
  --shake-dur-a: 80ms;
  --shake-dur-b: 60ms;
  --shake-ease: cubic-bezier(0.22, 1, 0.36, 1);
  --revert-hold: 3000ms;
  --revert-dur: 280ms;
}

.t-input {
  transition:
    border-color 150ms ease-out,
    box-shadow 150ms ease-out,
    background 150ms ease-out;
  will-change: transform;
}

.t-input.is-error,
.t-input[data-error] {
  border-color: var(--danger);
  box-shadow: 0 0 0 4px rgba(180, 35, 24, 0.12);
  transition:
    border-color var(--revert-dur, 280ms) ease-out,
    box-shadow var(--revert-dur, 280ms) ease-out,
    background var(--revert-dur, 280ms) ease-out;
}

.t-error-msg,
.t-input-wrap .mantine-InputWrapper-error {
  opacity: 0;
  visibility: hidden;
  transition:
    opacity var(--revert-dur, 280ms) ease-out,
    visibility 0s linear var(--revert-dur, 280ms);
}

.t-input-wrap.is-error .t-error-msg,
.t-input-wrap:has(.t-input.is-error) .mantine-InputWrapper-error,
.t-input-wrap:has(.t-input[data-error]) .mantine-InputWrapper-error {
  opacity: 1;
  visibility: visible;
  transition:
    opacity var(--revert-dur, 280ms) ease-out,
    visibility 0s linear 0s;
}

.t-input.is-shaking {
  animation: t-input-shake calc(
      var(--shake-dur-a) * 2 + var(--shake-dur-b) * 2
    ) linear;
}

@keyframes t-input-shake {
  0% { transform: translateX(0); animation-timing-function: var(--shake-ease); }
  28.57% { transform: translateX(var(--shake-distance)); animation-timing-function: var(--shake-ease); }
  57.14% { transform: translateX(calc(var(--shake-distance) * -1)); animation-timing-function: var(--shake-ease); }
  78.57% { transform: translateX(var(--shake-overshoot)); animation-timing-function: var(--shake-ease); }
  100% { transform: translateX(0); }
}

@media (prefers-reduced-motion: reduce) {
  .t-input {
    animation: none !important;
    transform: none !important;
  }
}
`

if (typeof document !== 'undefined' && !document.getElementById(transitionStyleId)) {
  const style = document.createElement('style')
  style.id = transitionStyleId
  style.textContent = transitionStyles
  document.head.appendChild(style)
}

export const replayInputShake = (selector: string) => {
  window.requestAnimationFrame(() => {
    document.querySelectorAll<HTMLElement>(selector).forEach((element) => {
      element.classList.remove('is-shaking')
      void element.offsetWidth
      element.classList.add('is-shaking')
      element.addEventListener(
        'animationend',
        () => element.classList.remove('is-shaking'),
        { once: true },
      )
    })
  })
}
