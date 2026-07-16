import { UnstyledButton } from "@mantine/core"
import type { ReactNode } from "react"

export function OptionButton({
  icon,
  label,
  selected,
  onClick,
  wide = false,
  disabled = false,
}: {
  icon?: ReactNode,
  label: string,
  selected: boolean,
  onClick: () => void,
  wide?: boolean,
  disabled?: boolean
}) {
  return (
     <UnstyledButton
      className={[
        'ui-option',
        icon ? 'ui-option-with-icon' : '',
        selected ? 'ui-option-selected' : '',
        wide ? 'ui-option-wide' : '',
        disabled ? 'ui-option-disabled' : '',
      ].join(' ')}
      disabled={disabled}
      onClick={onClick}
    >
      {icon && (
        <span className="ui-option-icon" aria-hidden="true">
          {icon}
        </span>
      )}
      <span>{label}</span>
    </UnstyledButton>
  )
}
