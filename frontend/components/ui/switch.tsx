"use client";

// Switch is a small accessible on/off toggle used in admin forms (e.g. account
// status, super-admin). Controlled via `checked` / `onChange`.
export function Switch({
  checked,
  onChange,
  label,
  hint,
  disabled,
  tone = "accent",
}: {
  checked: boolean;
  onChange: (next: boolean) => void;
  label?: string;
  hint?: string;
  disabled?: boolean;
  tone?: "accent" | "danger";
}) {
  return (
    <label className={`switch-field${disabled ? " is-disabled" : ""}`}>
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        disabled={disabled}
        className={`switch switch-${tone}${checked ? " on" : ""}`}
        onClick={() => !disabled && onChange(!checked)}
      >
        <span className="switch-knob" />
      </button>
      {label || hint ? (
        <span className="switch-text">
          {label ? <span className="switch-label">{label}</span> : null}
          {hint ? <span className="switch-hint">{hint}</span> : null}
        </span>
      ) : null}
    </label>
  );
}
