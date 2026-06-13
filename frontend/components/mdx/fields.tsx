import { ReactNode } from "react";

function FieldRow({
  name,
  type,
  required,
  deprecated,
  defaultValue,
  children
}: {
  name?: string;
  type?: string;
  required?: boolean;
  deprecated?: boolean;
  defaultValue?: string;
  children?: ReactNode;
}) {
  return (
    <div className="mdx-field">
      <div className="mdx-field__head">
        {name ? <code className="mdx-field__name">{name}</code> : null}
        {type ? <span className="mdx-field__type">{type}</span> : null}
        {required ? <span className="mdx-field__req">必填</span> : null}
        {deprecated ? <span className="mdx-field__dep">废弃</span> : null}
        {defaultValue ? <span className="mdx-field__default">默认 {defaultValue}</span> : null}
      </div>
      {children ? <div className="mdx-field__body">{children}</div> : null}
    </div>
  );
}

export function ParamField({
  path,
  query,
  body,
  header,
  type,
  required,
  deprecated,
  default: def,
  children
}: {
  path?: string;
  query?: string;
  body?: string;
  header?: string;
  type?: string;
  required?: boolean;
  deprecated?: boolean;
  default?: string;
  children?: ReactNode;
}) {
  return (
    <FieldRow name={path || query || body || header} type={type} required={required} deprecated={deprecated} defaultValue={def}>
      {children}
    </FieldRow>
  );
}

export function ResponseField({
  name,
  type,
  required,
  deprecated,
  default: def,
  children
}: {
  name?: string;
  type?: string;
  required?: boolean;
  deprecated?: boolean;
  default?: string;
  children?: ReactNode;
}) {
  return (
    <FieldRow name={name} type={type} required={required} deprecated={deprecated} defaultValue={def}>
      {children}
    </FieldRow>
  );
}

// Field is a generic alias.
export const Field = ResponseField;

export function Response({ children }: { children: ReactNode }) {
  return <div className="mdx-response">{children}</div>;
}
