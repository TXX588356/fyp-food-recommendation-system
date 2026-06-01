import { UnstyledButton } from "@mantine/core"
import "@/App.css"

export function OptionButton({
  label,
  selected,
  onClick,
  wide = false,
  disabled = false,
}: {
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
        selected ? 'ui-option-selected' : '',
        wide ? 'ui-option-wide' : '',
        disabled ? 'ui-option-disabled' : '',
      ].join(' ')}
      disabled={disabled}
      onClick={onClick}
    >
      <span>{label}</span>
    </UnstyledButton>
  )
}
