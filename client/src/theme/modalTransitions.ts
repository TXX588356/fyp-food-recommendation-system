import type { ModalProps } from '@mantine/core'

const transitionStyleId = 'transitions-p7'

const transitionStyles = `
:root {
  --modal-open-dur: 250ms;
  --modal-close-dur: 150ms;
  --modal-scale: 0.96;
  --modal-scale-close: 0.96;
  --modal-ease: cubic-bezier(0.22, 1, 0.36, 1);
}

.t-modal {
  transform-origin: center;
  will-change: transform, opacity;
}

@media (prefers-reduced-motion: reduce) {
  .t-modal {
    transition: none !important;
  }
}
`

if (typeof document !== 'undefined' && !document.getElementById(transitionStyleId)) {
  const style = document.createElement('style')
  style.id = transitionStyleId
  style.textContent = transitionStyles
  document.head.appendChild(style)
}

export const modalTransitionProps: ModalProps['transitionProps'] = {
  transition: {
    common: {
      transformOrigin: 'center',
    },
    in: {
      opacity: 1,
      transform: 'scale(1)',
    },
    out: {
      opacity: 0,
      transform: 'scale(var(--modal-scale))',
    },
    transitionProperty: 'transform, opacity',
  },
  duration: 250,
  exitDuration: 150,
  timingFunction: 'cubic-bezier(0.22, 1, 0.36, 1)',
}

export const modalContentClassName = (...classes: Array<string | undefined>) => (
  ['t-modal', ...classes].filter(Boolean).join(' ')
)
